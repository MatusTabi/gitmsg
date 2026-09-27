// Package app coordinates commit-message generation.
package app

import (
	"context"

	"github.com/MatusTabi/gitmsg/internal/prompt"
	"github.com/MatusTabi/gitmsg/internal/types"
)

// DiffSource retrieves a staged Git diff.
type DiffSource interface {
	StagedDiff(context.Context) types.Result[string]
}

// Generator generates a commit message from a prompt.
type Generator interface {
	Generate(context.Context, string) types.Result[types.CommitMessage]
}

// Run retrieves the staged diff, builds the prompt, and generates a message.
func Run(ctx context.Context, source DiffSource, generator Generator) types.Result[types.CommitMessage] {
	diff := source.StagedDiff(ctx)
	if !diff.IsSuccess() {
		return types.Failure[types.CommitMessage](diff.Error())
	}

	return generator.Generate(ctx, prompt.BuildCommitMessagePrompt(diff.Value()))
}
