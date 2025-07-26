# rulem: AI Assistant Instruction Manager CLI

rulem is a command-line tool for managing and organizing rule files on your machine. It provides a robust TUI (Terminal User Interface) for browsing, editing, and moving rule files between different environments and formats with ease.

## Features

- Browse and select instruction files with a modern TUI (Bubble Tea, Lipgloss)
- Migrate files between different AI assistants and formats
- Copy or symlink files to target locations
- Edit instruction files in your preferred terminal editor
- Validate Markdown files for required structure/schema
- Prevent accidental overwrites with confirmation prompts
- Supports default and custom storage locations
- Extensible for new assistants and formats
- Comprehensive unit tests for all core logic and UI

## Installation

### Prerequisites
- Go 1.21 or later
- (Optional) $EDITOR environment variable set for editing files

### Build from Source
```sh
git clone https://github.com/yourusername/rulem.git
cd rulem
go build -o rulem ./cmd/rulem
```

### Run Tests
```sh
go test ./...
```

## Usage

```sh
./rulem
```

- Use arrow keys to navigate the file viewer
- Press <space> to select files for migration
- Choose between copy or symlink when prompted
- Select the target AI assistant/format
- Edit files in your terminal editor as needed
- Follow on-screen prompts for validation and error handling

## Configuration

- Configuration is stored in YAML (default: `~/.config/rulem/config.yaml`)
- Set or update storage locations for rule files
- Supports both default and custom locations for each assistant

## Supported AI Assistants

- GitHub Copilot
- Cursor
- Claude
- Gemini-CLI
- Opencode
- (Easily extensible for more)

## Local Development

This project uses [Taskfile](https://taskfile.dev/) for common development workflows. Taskfile provides a simple, cross-platform way to run build, test, lint, and other commands without Make.

### Prerequisites
- [Go 1.21+](https://go.dev/doc/install)
- [Task](https://taskfile.dev/installation/) (install with `brew install go-task/tap/go-task` or see Taskfile docs)

### Common Commands

All commands below are run from the project root:

| Command                | Description                                 |
|------------------------|---------------------------------------------|
| `task build`           | Build the rulem CLI binary                  |
| `task run`             | Run the CLI interactively                   |
| `task test`            | Run all unit tests                          |
| `task lint`            | Lint the codebase with golangci-lint        |
| `task tidy`            | Clean up go.mod and go.sum                  |
| `task clean`           | Remove built binaries and temp files        |
| `task dev`             | Build, lint, and test in sequence           |

#### Example Workflow

1. Install dependencies: `go mod tidy`
2. Build the CLI: `task build`
3. Run tests: `task test`
4. Lint the code: `task lint`
5. Run the CLI: `task run`
6. Clean up: `task clean`

See `Taskfile.yml` for all available tasks and details.

## Contributing

Contributions are welcome! Please open issues or pull requests for bug fixes, new features, or improvements.

## License

MIT License. See [LICENSE](LICENSE) for details.

## Acknowledgements

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) for TUI
- [Lipgloss](https://github.com/charmbracelet/lipgloss) for styling
- Inspired by the needs of multi-assistant power users
