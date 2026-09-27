// Package git reads staged changes without modifying the repository.
package git

import (
	"context"
	"errors"
	"os/exec"
	"strings"

	"github.com/MatusTabi/gitmsg/internal/types"
)

const MaxDiffSize = 200 * 1024

var (
	ErrNotRepository  = errors.New("Not a Git repository.\n\nRun gitmsg from inside a Git repository.")
	ErrNoStagedDiff   = errors.New("No staged changes found.\n\nStage changes with `git add <file>` first.")
	ErrReadStagedDiff = errors.New("Couldn't read staged changes.\n\nCheck your Git repository and try again.")
	ErrDiffTooLarge   = errors.New("Staged changes are too large.\n\nSplit them into smaller commits and try again.")
)

type commandRunner func(context.Context, string, ...string) ([]byte, error)

// Source reads the staged diff from the current Git repository.
type Source struct {
	run commandRunner
}

// NewSource creates a staged-diff source that invokes Git from PATH.
func NewSource() Source {
	return Source{run: runCommand}
}

// StagedDiff verifies the repository and returns its staged diff.
func (s Source) StagedDiff(ctx context.Context) types.Result[string] {
	inside, err := s.run(ctx, "git", "rev-parse", "--is-inside-work-tree")
	if err != nil && ctx.Err() != nil {
		return types.Failure[string](ctx.Err())
	}
	if err != nil || strings.TrimSpace(string(inside)) != "true" {
		return types.Failure[string](ErrNotRepository)
	}

	diff, err := s.run(ctx, "git", "diff", "--cached", "--no-ext-diff")
	if err != nil {
		if ctx.Err() != nil {
			return types.Failure[string](ctx.Err())
		}
		return types.Failure[string](types.WithMessage(ErrReadStagedDiff.Error(), err))
	}
	if len(diff) == 0 {
		return types.Failure[string](ErrNoStagedDiff)
	}
	if len(diff) > MaxDiffSize {
		return types.Failure[string](ErrDiffTooLarge)
	}

	return types.Success(string(diff))
}

func runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}
