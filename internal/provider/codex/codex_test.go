package codex

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestGenerateRejectsMissingCodex(t *testing.T) {
	provider := Provider{
		lookPath: func(string) (string, error) { return "", errors.New("missing") },
		execute:  execute,
		timeout:  time.Minute,
	}

	result := provider.Generate(context.Background(), "prompt")
	if result.IsSuccess() {
		t.Fatal("Generate() succeeded")
	}
	if !errors.Is(result.Error(), ErrCodexNotFound) {
		t.Fatalf("Generate() error = %v, want ErrCodexNotFound", result.Error())
	}
	for _, instruction := range []string{"npm install -g @openai/codex", "codex login"} {
		if !strings.Contains(result.Error().Error(), instruction) {
			t.Errorf("Generate() error does not contain %q: %v", instruction, result.Error())
		}
	}
}

func TestGenerateInvokesCodexInTemporaryWorkspace(t *testing.T) {
	var gotCommand string
	var gotArgs []string
	var gotDir string
	provider := Provider{
		lookPath: func(string) (string, error) { return "/fake/codex", nil },
		execute: func(_ context.Context, command string, args []string, dir string) ([]byte, error) {
			gotCommand = command
			gotArgs = args
			gotDir = dir
			outputPath := args[len(args)-2]
			return nil, os.WriteFile(outputPath, []byte("feat(cli): generate messages\n"), 0o600)
		},
		timeout: time.Minute,
	}

	result := provider.Generate(context.Background(), "staged diff")
	if !result.IsSuccess() {
		t.Fatalf("Generate() error = %v", result.Error())
	}
	if result.Value() != "feat(cli): generate messages" {
		t.Fatalf("Generate() = %q", result.Value())
	}
	if gotCommand != "/fake/codex" {
		t.Fatalf("command = %q", gotCommand)
	}
	if gotDir == "" {
		t.Fatal("Codex working directory was empty")
	}
	if _, err := os.Stat(gotDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temporary directory still exists or could not be checked: %v", err)
	}
	joined := strings.Join(gotArgs, "\x00")
	for _, want := range []string{"exec", "--sandbox", "read-only", "--skip-git-repo-check", "staged diff"} {
		if !strings.Contains(joined, want) {
			t.Errorf("arguments do not contain %q: %q", want, gotArgs)
		}
	}
	if strings.Contains(joined, "--ask-for-approval") {
		t.Errorf("arguments contain unsupported --ask-for-approval flag: %q", gotArgs)
	}
}

func TestGenerateIncludesSanitizedDiagnostics(t *testing.T) {
	prompt := "secret staged diff"
	provider := Provider{
		lookPath: func(string) (string, error) { return "/fake/codex", nil },
		execute: func(context.Context, string, []string, string) ([]byte, error) {
			return []byte("authentication failed: " + prompt), errors.New("failed")
		},
		timeout: time.Minute,
	}

	result := provider.Generate(context.Background(), prompt)
	if result.IsSuccess() {
		t.Fatal("Generate() succeeded")
	}
	if strings.Contains(result.Error().Error(), prompt) {
		t.Fatalf("Generate() leaked prompt: %v", result.Error())
	}
	if !strings.Contains(result.Error().Error(), "authentication failed") {
		t.Fatalf("Generate() error = %v", result.Error())
	}
}

func TestGenerateTimesOut(t *testing.T) {
	provider := Provider{
		lookPath: func(string) (string, error) { return "/fake/codex", nil },
		execute: func(ctx context.Context, _ string, _ []string, _ string) ([]byte, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
		timeout: time.Millisecond,
	}

	result := provider.Generate(context.Background(), "prompt")
	if !errors.Is(result.Error(), context.DeadlineExceeded) {
		t.Fatalf("Generate() error = %v, want deadline exceeded", result.Error())
	}
}

func TestGeneratePropagatesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	provider := Provider{
		lookPath: func(string) (string, error) { return "/fake/codex", nil },
		execute: func(ctx context.Context, _ string, _ []string, _ string) ([]byte, error) {
			return nil, ctx.Err()
		},
		timeout: time.Minute,
	}

	result := provider.Generate(ctx, "prompt")
	if !errors.Is(result.Error(), context.Canceled) {
		t.Fatalf("Generate() error = %v, want canceled", result.Error())
	}
}

func TestNormalizeMessage(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    string
		wantErr string
	}{
		{name: "plain", output: "fix: repair parser\n", want: "fix: repair parser"},
		{name: "fenced", output: "```\nfeat(api)!: change response\n```\n", want: "feat(api)!: change response"},
		{name: "quoted", output: "> docs: explain usage\n", want: "docs: explain usage"},
		{name: "invalid type", output: "deps: upgrade library", wantErr: "invalid commit message"},
		{name: "invalid structure", output: "fix parser", wantErr: "invalid commit message"},
		{name: "empty", output: "\n", wantErr: "empty commit message"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeMessage(test.output)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("normalizeMessage() error = %v, want %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeMessage() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("normalizeMessage() = %q, want %q", got, test.want)
			}
		})
	}
}
