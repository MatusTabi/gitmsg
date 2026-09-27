package prompt

import (
	"strings"
	"testing"
)

func TestBuildCommitMessagePrompt(t *testing.T) {
	diff := "diff --git a/main.go b/main.go\n+new line"
	prompt := BuildCommitMessagePrompt(diff)

	for _, want := range []string{
		"Return only the commit message.",
		"feat, fix, docs, refactor, test, build, ci, chore, perf, style",
		"type(optional-scope)!: subject",
		diff,
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt does not contain %q", want)
		}
	}
}
