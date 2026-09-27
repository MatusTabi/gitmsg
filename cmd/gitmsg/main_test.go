package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/MatusTabi/gitmsg/internal/types"
)

func TestMain(m *testing.M) {
	if os.Getenv("GITMSG_FAKE_CODEX") == "1" {
		os.Exit(runFakeCodex())
	}

	os.Exit(m.Run())
}

func runFakeCodex() int {
	args := os.Args[1:]
	outputPath := ""
	for i, arg := range args {
		if arg == "--output-last-message" && i+1 < len(args) {
			outputPath = args[i+1]
			break
		}
	}
	if outputPath == "" || len(args) == 0 {
		return 2
	}

	prompt := args[len(args)-1]
	if capturePath := os.Getenv("GITMSG_CAPTURE"); capturePath != "" {
		content := fmt.Sprintf("%s\n%s", mustGetwd(), prompt)
		if err := os.WriteFile(capturePath, []byte(content), 0o600); err != nil {
			return 3
		}
	}

	return writeFakeResponse(outputPath, os.Getenv("GITMSG_FAKE_OUTPUT"))
}

func mustGetwd() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return dir
}

func writeFakeResponse(path, response string) int {
	if response == "" {
		response = "feat: generate commit message"
	}
	if err := os.WriteFile(path, []byte(response+"\n"), 0o600); err != nil {
		return 4
	}
	return 0
}

func TestGitmsgEndToEnd(t *testing.T) {
	binary := buildGitmsg(t)
	repository := createRepository(t, true)
	fakeDir := createFakeCodex(t)
	capturePath := filepath.Join(t.TempDir(), "capture.txt")

	stdout, stderr, exitCode := runGitmsg(t, binary, repository, map[string]string{
		"PATH":               fakeDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		"GITMSG_FAKE_CODEX":  "1",
		"GITMSG_FAKE_OUTPUT": "```\nfeat(cli): generate a commit message\n```",
		"GITMSG_CAPTURE":     capturePath,
	})
	if exitCode != 0 {
		t.Fatalf("gitmsg exited %d; stderr: %s", exitCode, stderr)
	}
	if stdout != "feat(cli): generate a commit message\n" {
		t.Fatalf("stdout = %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	captured, err := os.ReadFile(capturePath)
	if err != nil {
		t.Fatalf("read Codex capture: %v", err)
	}
	parts := strings.SplitN(string(captured), "\n", 2)
	if len(parts) != 2 {
		t.Fatalf("invalid Codex capture: %q", captured)
	}
	if filepath.Clean(parts[0]) == filepath.Clean(repository) {
		t.Fatalf("Codex ran in repository directory %q", repository)
	}
	if !strings.Contains(parts[1], "staged content") {
		t.Fatalf("Codex prompt did not contain staged diff: %q", parts[1])
	}
	if strings.Contains(parts[1], "unstaged secret") {
		t.Fatalf("Codex prompt leaked unstaged content: %q", parts[1])
	}
}

func TestRunReturnsInterruptCodeForCancellation(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := runWith(
		context.Background(),
		&stdout,
		&stderr,
		fakeDiffSource{result: types.Failure[string](context.Canceled)},
		fakeCommitGenerator{},
	)
	if exitCode != 130 {
		t.Fatalf("runWith() = %d, want 130", exitCode)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("runWith() wrote stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestGitmsgRejectsEmptyStagedDiff(t *testing.T) {
	binary := buildGitmsg(t)
	repository := createRepository(t, false)

	_, stderr, exitCode := runGitmsg(t, binary, repository, nil)
	if exitCode != 1 {
		t.Fatalf("gitmsg exited %d, want 1; stderr: %s", exitCode, stderr)
	}
	if !strings.Contains(stderr, "No staged changes found.") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestGitmsgRejectsUnavailableCodex(t *testing.T) {
	binary := buildGitmsg(t)
	repository := createRepository(t, true)
	gitOnlyDir := createGitOnlyPath(t)

	_, stderr, exitCode := runGitmsg(t, binary, repository, map[string]string{
		"PATH": gitOnlyDir,
	})
	if exitCode != 1 {
		t.Fatalf("gitmsg exited %d, want 1; stderr: %s", exitCode, stderr)
	}
	if !strings.Contains(stderr, "Codex CLI not found.") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestGitmsgRejectsInvalidCodexOutput(t *testing.T) {
	binary := buildGitmsg(t)
	repository := createRepository(t, true)
	fakeDir := createFakeCodex(t)

	_, stderr, exitCode := runGitmsg(t, binary, repository, map[string]string{
		"PATH":               fakeDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		"GITMSG_FAKE_CODEX":  "1",
		"GITMSG_FAKE_OUTPUT": "update parser",
	})
	if exitCode != 1 {
		t.Fatalf("gitmsg exited %d, want 1; stderr: %s", exitCode, stderr)
	}
	if !strings.Contains(stderr, "Codex returned an invalid commit message.") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func buildGitmsg(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "gitmsg")
	if runtime.GOOS == "windows" {
		path += ".exe"
	}

	cmd := exec.Command("go", "build", "-o", path, ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build gitmsg: %v\n%s", err, output)
	}
	return path
}

func createRepository(t *testing.T, stageFile bool) string {
	t.Helper()
	repository := t.TempDir()
	runCommand(t, repository, "git", "init")
	runCommand(t, repository, "git", "config", "user.email", "test@example.com")
	runCommand(t, repository, "git", "config", "user.name", "Test User")

	stagedPath := filepath.Join(repository, "staged.txt")
	if err := os.WriteFile(stagedPath, []byte("staged content\n"), 0o600); err != nil {
		t.Fatalf("write staged fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repository, "unstaged.txt"), []byte("unstaged secret\n"), 0o600); err != nil {
		t.Fatalf("write unstaged fixture: %v", err)
	}
	if stageFile {
		runCommand(t, repository, "git", "add", "staged.txt")
	}
	return repository
}

func createFakeCodex(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	name := "codex"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	copyExecutable(t, os.Args[0], filepath.Join(directory, name))
	return directory
}

func createGitOnlyPath(t *testing.T) string {
	t.Helper()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("find git: %v", err)
	}
	directory := t.TempDir()
	name := "git"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	copyExecutable(t, gitPath, filepath.Join(directory, name))
	return directory
}

func copyExecutable(t *testing.T, source, destination string) {
	t.Helper()
	content, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read executable %q: %v", source, err)
	}
	if err := os.WriteFile(destination, content, 0o755); err != nil {
		t.Fatalf("write executable %q: %v", destination, err)
	}
}

func runGitmsg(t *testing.T, binary, directory string, overrides map[string]string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(binary)
	cmd.Dir = directory
	cmd.Env = environmentWith(overrides)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		return stdout.String(), stderr.String(), 0
	}
	if exitError, ok := err.(*exec.ExitError); ok {
		return stdout.String(), stderr.String(), exitError.ExitCode()
	}
	t.Fatalf("run gitmsg: %v", err)
	return "", "", 0
}

func environmentWith(overrides map[string]string) []string {
	if len(overrides) == 0 {
		return os.Environ()
	}

	values := make(map[string]string)
	for _, entry := range os.Environ() {
		key, value, found := strings.Cut(entry, "=")
		if found {
			values[key] = value
		}
	}
	for key, value := range overrides {
		values[key] = value
	}

	environment := make([]string, 0, len(values))
	for key, value := range values {
		environment = append(environment, key+"="+value)
	}
	return environment
}

func runCommand(t *testing.T, directory, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = directory
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, output)
	}
}

type fakeDiffSource struct {
	result types.Result[string]
}

func (s fakeDiffSource) StagedDiff(context.Context) types.Result[string] {
	return s.result
}

type fakeCommitGenerator struct{}

func (fakeCommitGenerator) Generate(context.Context, string) types.Result[types.CommitMessage] {
	return types.Success(types.CommitMessage("feat: ignored"))
}
