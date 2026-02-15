# Hacking Guide

## Local setup

1. Clone the repository and enter it: `git clone https://github.com/muhammadbassiony/rulem && cd rulem`.
2. Ensure Go 1.24+ is installed (`go version`).
3. Install Task (optional): `brew install go-task/tap/go-task` or use your package manager.
4. Use `go install ./...` to fetch any tooling dependencies.
5. Initialize your config if needed by running `rulem` once and walking through the setup wizard; this creates `~/.config/rulem/config.yaml`.

## Developing locally

- Use `task test` or `go test ./...` to run unit tests. Prefer `helpers.SetTestConfigPath(t)` in tests that write configs so you don’t override your personal settings.
- Run `task build` or `go build ./cmd/rulem` to compile the binary.
- The TUI is under `internal/tui`; most flows live inside `internal/tui/settingsmenu`. Read `internal/tui/settingsmenu/README.md` for flow summaries.
- Multi-repo support: the TUI can manage multiple repositories simultaneously—each repo has its own item in the settings menu, but shared PAT/credentials and storage paths.

## Common commands

```bash
go mod tidy
go test ./...
go build ./cmd/rulem
go run ./cmd/rulem
task lint    # if you use Task
```

## Debugging

- Logs are written to `rulem.log` when you run the CLI with `--debug`.
- Use the MCP inspector (`npm install -g @modelcontextprotocol/inspector` + `mcp-inspector`) to verify MCP tools.
- Check the `internal/tui/settingsmenu` flows when working on settings UI logic—each flow (add/edit/delete/refresh) has dedicated state machines.

## Notes

- When testing GitHub flows (refresh, branch change, delete), the settings menu runs dirty-state checks; mock `repository.CheckGitHubRepositoryStatus` to control those paths.
- Always re-run `rulem` after editing configs to ensure background scanners rebuild `preparedRepos`.
