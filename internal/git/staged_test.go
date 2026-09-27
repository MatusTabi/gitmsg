package git

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestStagedDiff(t *testing.T) {
	var calls [][]string
	source := Source{run: func(_ context.Context, name string, args ...string) ([]byte, error) {
		calls = append(calls, append([]string{name}, args...))
		switch len(calls) {
		case 1:
			return []byte("true\n"), nil
		case 2:
			return []byte("diff --git a/file b/file\n"), nil
		default:
			t.Fatal("unexpected Git command")
			return nil, nil
		}
	}}

	result := source.StagedDiff(context.Background())
	if !result.IsSuccess() {
		t.Fatalf("StagedDiff() error = %v", result.Error())
	}
	if result.Value() != "diff --git a/file b/file\n" {
		t.Fatalf("StagedDiff() = %q", result.Value())
	}

	want := [][]string{
		{"git", "rev-parse", "--is-inside-work-tree"},
		{"git", "diff", "--cached", "--no-ext-diff"},
	}
	if len(calls) != len(want) {
		t.Fatalf("ran %d commands, want %d", len(calls), len(want))
	}
	for i := range want {
		if strings.Join(calls[i], "\x00") != strings.Join(want[i], "\x00") {
			t.Errorf("command %d = %q, want %q", i, calls[i], want[i])
		}
	}
}

func TestStagedDiffRejectsNonRepository(t *testing.T) {
	source := Source{run: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return nil, errors.New("not a git repository")
	}}

	result := source.StagedDiff(context.Background())
	if !errors.Is(result.Error(), ErrNotRepository) {
		t.Fatalf("StagedDiff() error = %v, want ErrNotRepository", result.Error())
	}
}

func TestStagedDiffRejectsEmptyDiff(t *testing.T) {
	call := 0
	source := Source{run: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		call++
		if call == 1 {
			return []byte("true\n"), nil
		}
		return nil, nil
	}}

	result := source.StagedDiff(context.Background())
	if !errors.Is(result.Error(), ErrNoStagedDiff) {
		t.Fatalf("StagedDiff() error = %v, want ErrNoStagedDiff", result.Error())
	}
}

func TestStagedDiffRejectsLargeDiff(t *testing.T) {
	call := 0
	source := Source{run: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		call++
		if call == 1 {
			return []byte("true\n"), nil
		}
		return []byte(strings.Repeat("a", MaxDiffSize+1)), nil
	}}

	result := source.StagedDiff(context.Background())
	if result.IsSuccess() {
		t.Fatal("StagedDiff() succeeded for an oversized diff")
	}
	if !errors.Is(result.Error(), ErrDiffTooLarge) {
		t.Fatalf("StagedDiff() error = %v, want ErrDiffTooLarge", result.Error())
	}
}

func TestStagedDiffWrapsDiffFailure(t *testing.T) {
	call := 0
	want := errors.New("git failed")
	source := Source{run: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		call++
		if call == 1 {
			return []byte("true\n"), nil
		}
		return nil, want
	}}

	result := source.StagedDiff(context.Background())
	if !errors.Is(result.Error(), want) {
		t.Fatalf("StagedDiff() error = %v, want wrapped %v", result.Error(), want)
	}
	if result.Error().Error() != ErrReadStagedDiff.Error() {
		t.Fatalf("StagedDiff() message = %q, want %q", result.Error(), ErrReadStagedDiff)
	}
}

func TestStagedDiffPropagatesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	source := Source{run: func(context.Context, string, ...string) ([]byte, error) {
		return nil, context.Canceled
	}}

	result := source.StagedDiff(ctx)
	if !errors.Is(result.Error(), context.Canceled) {
		t.Fatalf("StagedDiff() error = %v, want canceled", result.Error())
	}
}
