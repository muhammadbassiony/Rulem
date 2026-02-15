# rulem: AI Assistant Instruction Manager

[![Release](https://img.shields.io/github/v/release/muhammadbassiony/rulem)](https://github.com/muhammadbassiony/rulem/releases) [![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A compact CLI for managing instruction files and MCP tooling across multiple repositories. Built with Go and Bubble Tea, rulem keeps each repository’s instructions, settings, and MCP tools in sync while sharing credentials, storage, and background scans.

## Quick orientation

- **Multi-repo aware**: Each repository gets its own instructions and settings inside the TUI, but all share the same credentials and MCP registry.
- **Primary workflows**: Launch `rulem` for the TUI, `rulem mcp` for the MCP server, and use the menu actions to save/import rules, refresh GitHub repos, or edit repository metadata.
- **Safety guards**: Git-backed flows run dirty-state checks before mutating branches, clone paths, or deleting a repo.

## Quick start

1. Install (see below), then run `rulem` to launch the interactive UI.
2. Add or select a repository, use the main menu to save/import rules, or open the settings menu to edit metadata.
3. Run `rulem mcp` to expose your rule files as MCP tools for connected IDEs/assistants.
4. When developing, `go test ./...` and `go run ./cmd/rulem` are the canonical verification/build loops (see `hacking.md` for more).

## Installation

- **Homebrew (macOS)**: `brew tap muhammadbassiony/rulem && brew install rulem`
- **Snap/Linux**: `sudo snap install rulem`
- **Binary**: Download the latest release from [GitHub Releases](https://github.com/muhammadbassiony/rulem/releases/latest)
- **Source**: `git clone https://github.com/muhammadbassiony/rulem && go build -o rulem ./cmd/rulem`

## Configuration

Config lives at `~/.config/rulem/config.yaml`. The CLI initializes it on first run and exposes it through the TUI settings menu (storage paths, version info, timestamps). Refer to the settings menu flows in `internal/tui/settingsmenu` when adjusting how multiple repositories are handled simultaneously.

## MCP integration

- Start the MCP server with `rulem mcp` (add `--debug` for verbose logging).
- Rule files with YAML frontmatter are auto-registered as MCP tools; each repo contributes tools that share the stored PAT/token.
- Use MCP inspectors (e.g., `mcp-inspector`) to confirm tool registration and invocation flows.

## Development highlights

- **Prereqs**: Go 1.24+. Task is optional (`task build/test/lint`).
- **Common commands**: `go mod tidy`, `go test ./...`, `go build ./cmd/rulem`, `go run ./cmd/rulem`.
- **Testing note**: Tests that mutate config must call `helpers.SetTestConfigPath(t)` to avoid touching your real settings.
- **Code layout**: Core logic lives under `cmd/rulem`, `internal/` (config, filemanager, logging, tui), and `pkg/fileops`. The settings menu and its multi-repo flows are documented in `internal/tui/settingsmenu/README.md` and `hacking.md`.
