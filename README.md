# rulem: AI Assistant Instruction Manager CLI

[![Go Report Card](https://goreportcard.com/badge/github.com/yourusername/rulem)](https://goreportcard.com/report/github.com/yourusername/rulem)
[![GoDoc](https://godoc.org/github.com/yourusername/rulem?status.svg)](https://godoc.org/github.com/yourusername/rulem)

rulem is a command-line tool for managing and organizing rule files on your machine. It provides a robust TUI (Terminal User Interface) for browsing, editing, and moving rule files between different environments and formats with ease.

Built with modern Go and terminal UI frameworks, rulem helps developers maintain consistent AI assistant configurations across multiple tools and environments.

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Usage](#usage)
- [Configuration](#configuration)
- [Supported AI Assistants](#supported-ai-assistants)
- [Local Development](#local-development)
- [API Reference](#api-reference)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [License](#license)

## Features

- **🖥️ Modern TUI**: Browse and select instruction files with a modern TUI built with Bubble Tea and Lipgloss
- **🔄 Migration Tools**: Seamlessly migrate files between different AI assistants and formats
- **📋 File Operations**: Copy or symlink files to target locations with intelligent conflict resolution
- **✏️ Editor Integration**: Edit instruction files in your preferred terminal editor with automatic validation
- **✅ Schema Validation**: Validate Markdown files for required structure and schema compliance
- **🛡️ Safety First**: Prevent accidental overwrites with confirmation prompts and dry-run mode
- **📁 Flexible Storage**: Supports default and custom storage locations with secure directory management
- **🔌 Extensible**: Easy to add support for new assistants and formats
- **🧪 Well Tested**: Comprehensive unit tests for all core logic and UI components
- **📊 GitHub Integration**: Fetch rule files directly from GitHub repositories and gists

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

## Quick Start

1. **Install** rulem using one of the methods below
2. **Run** the application: `./rulem`
3. **Complete setup** wizard on first run (choose your storage directory)
4. **Start managing** your AI assistant rule files!

## Usage

### Basic Usage

```sh
./rulem
```

### Interactive Navigation

- **↑/↓ or j/k**: Navigate the file viewer
- **Enter**: Select menu item or confirm action
- **Space**: Select/deselect files for migration
- **/**: Filter menu items by typing
- **q**: Quit from main menu
- **Esc**: Go back or cancel current operation
- **Ctrl+C**: Force quit

### Main Menu Options

1. **💾 Save rules file**: Save a rules file from current directory to the central rules repository
2. **📄 Import rules (Copy)**: Import a rule file from the central repository to current directory
3. **⬇️ Fetch rules from Github**: Download a new rules file from a GitHub repository or gist
4. **⚙️ Update settings**: Modify your Rulem configuration settings

### Example Workflow

```bash
# 1. Save current directory's rules to repository
./rulem
# Navigate to "💾 Save rules file" and select your rule file

# 2. Switch to a different project
cd /path/to/other/project

# 3. Import rules for a different AI assistant
./rulem
# Navigate to "📄 Import rules (Copy)"
# Select the rule file you want to import
# Choose your target AI assistant (e.g., Cursor, GitHub Copilot)
# Select copy or symlink mode
```

### Advanced Usage

#### Custom Storage Directory

```bash
# Use a custom storage directory
export RULEM_CONFIG_PATH="/custom/path/config.yaml"
./rulem
```

#### Editor Integration

Set your preferred editor:

```bash
export EDITOR="code --wait"  # VS Code
# or
export EDITOR="vim"
# or
export EDITOR="nano"
```

The editor will be used when you need to modify rule files during the migration process.

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

## API Reference

### Configuration

rulem stores configuration in YAML format at:
- **Linux/macOS**: `~/.config/rulem/config.yaml`
- **Windows**: `%APPDATA%\rulem\config.yaml`

Example configuration:

```yaml
storage_dir: "/Users/username/.rulem"
version: "1.0"
init_time: 1640995200
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `RULEM_CONFIG_PATH` | Override default config file path | Platform-specific |
| `EDITOR` | Terminal editor for file editing | `nano` or system default |

### Exit Codes

- `0`: Success
- `1`: General error (setup failed, config error, etc.)

## Troubleshooting

### Common Issues

#### "No configuration found" Error

**Problem**: rulem exits with "no configuration found, first-time setup required"

**Solution**: Run `./rulem` and complete the setup wizard, or manually create a config file.

#### Permission Denied

**Problem**: Unable to create storage directory or access files

**Solution**:
```bash
# Fix permissions on storage directory
chmod 755 ~/.rulem
# Or choose a different storage location in settings
```

#### Terminal Display Issues

**Problem**: TUI doesn't render correctly or shows garbled text

**Solution**:
- Ensure your terminal supports Unicode and 256 colors
- Try a different terminal (iTerm2, Alacritty, etc.)
- Check that `TERM` environment variable is set correctly

#### Editor Not Found

**Problem**: "Editor not found" when trying to edit files

**Solution**:
```bash
# Set your preferred editor
export EDITOR="code --wait"  # VS Code
export EDITOR="vim"          # Vim
export EDITOR="nano"         # Nano
```

### Getting Help

1. **Check logs**: Enable debug logging by setting `RUST_LOG=debug`
2. **View config**: Check `~/.config/rulem/config.yaml` for configuration issues
3. **Test with defaults**: Try running with default settings to isolate problems
4. **Report issues**: [Open an issue](https://github.com/yourusername/rulem/issues) with:
   - Your operating system and terminal
   - rulem version (`./rulem --version`)
   - Steps to reproduce the issue
   - Error messages and logs

### Performance Issues

- **Slow startup**: Check storage directory size and disk performance
- **UI lag**: Ensure terminal window isn't too large (try resizing smaller)
- **Memory usage**: Monitor with `top` or Activity Monitor during operation

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
