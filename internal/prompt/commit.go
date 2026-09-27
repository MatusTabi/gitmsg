// Package prompt builds provider prompts from staged diffs.
package prompt

import "fmt"

// BuildCommitMessagePrompt instructs a provider to return one Conventional Commit message.
func BuildCommitMessagePrompt(diff string) string {
	return fmt.Sprintf(`Generate exactly one concise Conventional Commit message for the staged Git diff below.

Return only the commit message. Do not use Markdown, code fences, explanations, or additional lines.
Use one of these types: feat, fix, docs, refactor, test, build, ci, chore, perf, style.
Use this format: type(optional-scope)!: subject. The scope and ! are optional.

Staged diff:
%s`, diff)
}
