// Package codex generates commit messages through the Codex CLI.
package codex

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/MatusTabi/gitmsg/internal/types"
)

const (
	defaultTimeout = time.Minute
	maxDiagnostic  = 4096
)

var (
	ErrCodexNotFound   = errors.New("Codex CLI not found.\n\nInstall it with:\n  npm install -g @openai/codex\n\nThen authenticate with:\n  codex login")
	ErrCreateWorkspace = errors.New("Couldn't prepare a temporary workspace for Codex.\n\nCheck your temporary directory permissions and try again.")
	ErrCodexTimedOut   = errors.New("Codex took too long to respond.\n\nCheck your connection and try again.")
	ErrCodexFailed     = errors.New("Codex couldn't generate a commit message.\n\nAuthenticate with:\n  codex login\n\nThen try again.")
	ErrReadResponse    = errors.New("Couldn't read the response from Codex.\n\nTry running gitmsg again. If the problem continues, update Codex and authenticate with `codex login`.")
	ErrInvalidMessage  = errors.New("Codex returned an invalid commit message.\n\nTry running gitmsg again. If the problem continues, update Codex and authenticate with `codex login`.")
	ErrEmptyMessage    = errors.New("Codex returned an empty commit message.\n\nTry running gitmsg again. If the problem continues, authenticate with `codex login`.")
	conventionalCommit = regexp.MustCompile(`^(feat|fix|docs|refactor|test|build|ci|chore|perf|style)(\([^\s()]+\))?!?:\s+\S.*$`)
)

type executableFinder func(string) (string, error)
type commandExecutor func(context.Context, string, []string, string) ([]byte, error)

// Provider generates commit messages with the Codex CLI.
type Provider struct {
	lookPath executableFinder
	execute  commandExecutor
	timeout  time.Duration
}

// New creates a Codex provider using the executable found on PATH.
func New() Provider {
	return Provider{
		lookPath: exec.LookPath,
		execute:  execute,
		timeout:  defaultTimeout,
	}
}

// Generate sends prompt to Codex and returns a validated commit message.
func (p Provider) Generate(ctx context.Context, prompt string) types.Result[types.CommitMessage] {
	codexPath, err := p.lookPath("codex")
	if err != nil {
		return types.Failure[types.CommitMessage](ErrCodexNotFound)
	}

	dir, err := os.MkdirTemp("", "gitmsg-codex-")
	if err != nil {
		return types.Failure[types.CommitMessage](types.WithMessage(ErrCreateWorkspace.Error(), err))
	}
	defer os.RemoveAll(dir)

	requestCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	outputPath := filepath.Join(dir, "message.txt")
	args := []string{
		"exec",
		"--sandbox", "read-only",
		"--skip-git-repo-check",
		"--output-last-message", outputPath,
		prompt,
	}
	stderr, err := p.execute(requestCtx, codexPath, args, dir)
	if err != nil {
		if errors.Is(requestCtx.Err(), context.Canceled) {
			return types.Failure[types.CommitMessage](context.Canceled)
		}
		if errors.Is(requestCtx.Err(), context.DeadlineExceeded) {
			return types.Failure[types.CommitMessage](types.WithMessage(ErrCodexTimedOut.Error(), context.DeadlineExceeded))
		}

		diagnostic := sanitizeDiagnostic(stderr, prompt)
		if diagnostic == "" {
			return types.Failure[types.CommitMessage](ErrCodexFailed)
		}
		return types.Failure[types.CommitMessage](fmt.Errorf("%s\n\nDetails:\n%s", ErrCodexFailed, diagnostic))
	}

	output, err := os.ReadFile(outputPath)
	if err != nil {
		return types.Failure[types.CommitMessage](types.WithMessage(ErrReadResponse.Error(), err))
	}

	message, err := normalizeMessage(string(output))
	if err != nil {
		return types.Failure[types.CommitMessage](err)
	}
	return types.Success(types.CommitMessage(message))
}

func execute(ctx context.Context, command string, args []string, dir string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = dir
	cmd.Stdout = io.Discard

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stderr.Bytes(), err
}

func sanitizeDiagnostic(stderr []byte, prompt string) string {
	diagnostic := strings.TrimSpace(string(stderr))
	diagnostic = strings.ReplaceAll(diagnostic, prompt, "[redacted]")
	if len(diagnostic) > maxDiagnostic {
		diagnostic = diagnostic[:maxDiagnostic] + "..."
	}
	return diagnostic
}

func normalizeMessage(output string) (string, error) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "```") {
			continue
		}

		line = strings.TrimSpace(strings.TrimPrefix(line, ">"))
		line = strings.Trim(line, "`")
		if conventionalCommit.MatchString(line) {
			return line, nil
		}
		return "", ErrInvalidMessage
	}

	return "", ErrEmptyMessage
}
