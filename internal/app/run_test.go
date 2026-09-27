package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/MatusTabi/gitmsg/internal/types"
)

type fakeSource struct {
	result types.Result[string]
}

func (s fakeSource) StagedDiff(context.Context) types.Result[string] {
	return s.result
}

type fakeGenerator struct {
	prompt string
	result types.Result[types.CommitMessage]
	called bool
}

func (g *fakeGenerator) Generate(_ context.Context, prompt string) types.Result[types.CommitMessage] {
	g.called = true
	g.prompt = prompt
	return g.result
}

func TestRunBuildsPromptAndGeneratesMessage(t *testing.T) {
	generator := &fakeGenerator{result: types.Success(types.CommitMessage("feat: add generator"))}
	result := Run(context.Background(), fakeSource{result: types.Success("diff --git a/a b/a")}, generator)

	if !result.IsSuccess() {
		t.Fatalf("Run() error = %v", result.Error())
	}
	if result.Value() != "feat: add generator" {
		t.Fatalf("Run() = %q", result.Value())
	}
	if !generator.called {
		t.Fatal("generator was not called")
	}
	if !strings.Contains(generator.prompt, "diff --git a/a b/a") {
		t.Fatalf("prompt does not contain staged diff: %q", generator.prompt)
	}
}

func TestRunDoesNotGenerateWhenDiffFails(t *testing.T) {
	want := errors.New("no staged changes")
	generator := &fakeGenerator{result: types.Success(types.CommitMessage("feat: ignored"))}
	result := Run(context.Background(), fakeSource{result: types.Failure[string](want)}, generator)

	if !errors.Is(result.Error(), want) {
		t.Fatalf("Run() error = %v, want %v", result.Error(), want)
	}
	if generator.called {
		t.Fatal("generator was called after diff failure")
	}
}
