# gitmsg Agent Context

## Project Purpose

`gitmsg` is a lightweight, open-source Go CLI that generates meaningful Git commit messages from staged changes. It uses an already configured AI provider through that provider's CLI or API, so users do not need another AI subscription or credentials managed by this project.

The primary command is `gitmsg`: it should quickly print one concise Conventional Commit message to standard output.

## Product Principles

- Keep the default workflow simple: one command generates a message.
- Support AI providers through a shared interface; provider implementations must be independent of Git and CLI presentation.
- Be non-invasive: never modify source files, stage changes, create commits, or run autonomous actions in a user's repository.
- Process only staged changes obtained with `git diff --cached`.
- Send providers only the information necessary to generate the commit message.
- Reuse provider authentication. Do not store, manage, or ask for AI credentials.
- Prefer a small, dependency-light, cross-platform Go binary. Favor the standard library unless a dependency has a clear need.

## MVP Scope

The MVP generates a Conventional Commit message from staged changes. It must:

1. Verify the current directory is in a Git repository.
2. Retrieve and validate staged changes with `git diff --cached`.
3. Resolve a selected or configured provider.
4. Build a structured prompt containing the staged diff and commit-message instructions.
5. Ask the provider for a concise commit message.
6. Print that message to standard output.

Supported initial providers:

- OpenAI Codex through the Codex CLI.
- Claude Code through the Claude CLI.
- Gemini through the Gemini CLI.

Only one installed and configured provider is required for `gitmsg` to work.

Out of scope unless explicitly requested:

- Creating commits or staging files.
- Git hooks.
- Interactive commit editors.
- Background services.
- Custom AI model hosting.
- Complex configuration systems.

## CLI Contract

- `gitmsg`: generate using the default provider.
- `gitmsg --provider codex`: generate using a specified provider.
- `gitmsg --copy`: copy the generated message to the clipboard.
- `gitmsg config set provider codex`: set the default provider.
- `gitmsg --version`: print the installed version.
- `gitmsg --help`: print usage information.

Keep CLI behavior predictable. Use clear, actionable errors for missing Git repositories, empty staged diffs, unavailable providers, and provider failures. Do not leak a staged diff in errors unless necessary.

## Architecture Boundaries

Organize code into focused components, generally beneath `cmd/` and `internal/`:

- CLI: argument parsing, user interaction, and output.
- Git: repository validation and staged diff extraction.
- Provider: a common commit-message generation interface and provider adapters.
- Prompt: prompt construction and Conventional Commit formatting instructions.
- Config: user preferences, including the default provider.

Keep dependencies directional: CLI coordinates components; Git does not know providers; providers do not know Git or CLI concerns. Adding a provider must not require changes to Git behavior.

Use safe subprocess execution: pass arguments directly rather than through a shell, capture useful diagnostics, and honor command failures. Keep operating-system-specific behavior isolated, especially clipboard and installation logic.

## Development Guidance

- Make the smallest change that satisfies the requested behavior.
- Preserve the MVP boundary; do not add speculative configuration or provider features.
- Write tests for behavior that can fail independently, especially Git command handling, provider selection, prompt construction, configuration, and error paths.
- Keep generated commit messages concise and Conventional Commits compliant by default.
- Before completing code changes, run the relevant Go tests. Run `go test ./...` when the repository supports it.

## Distribution

Releases are precompiled GitHub Release binaries built with GitHub Actions and GoReleaser for macOS, Linux, and Windows. The Unix installation script should detect OS and architecture, download the appropriate release, verify its checksum, and install to a user-accessible location without administrator privileges where possible. Windows uses a separate installation path.
