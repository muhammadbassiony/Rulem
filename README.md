# rulem: AI Assistant Instruction Manager

[![Release](https://img.shields.io/github/v/release/muhammadbassiony/rulem)](https://github.com/muhammadbassiony/rulem/releases)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A command-line tool for managing and organizing AI assistant instruction files across different environments and formats. Built with Go and featuring a modern terminal user interface.

## Table of Contents

- [Motivation & Use Cases](#motivation--use-cases)
- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
  - [Supported AI Assistants](#supported-ai-assistants)
- [Development](#development)
  - [Prerequisites](#prerequisites)
  - [Common Commands](#common-commands)
  - [Project Structure](#project-structure)
- [Contributing](#contributing)

## Motivation & Use Cases

My main motivations for creating `rulem` are:
- **Inter-Tool/IDE Compatibility**: Easily switch between different AI tools and IDEs such as GitHub Copilot, Cursor, and Claude
- **Centralized Management**: Store all instruction files in one organized location
- **Reusability**: Reuse and adapt instructions across multiple projects without duplication, rulem also supports symlinking to keep files up to date as you constantly improve them.
- **Version Control**: Having a single source of truth for instruction files that can be tracked with version control systems.
- Experimentation & adpoting more AI powered development

## Features

- Modern terminal user interface built with Bubble Tea
- Cross-platform support (macOS, Linux, Windows)
- Secure file operations with validation
- Copy and symlink support for file management
- Extensible architecture for new AI assistants

## Installation

### Homebrew (macOS)
```bash
brew tap muhammadbassiony/rulem
brew install rulem
```

### Snap (Linux)
```bash
sudo snap install rulem
```

### Binary Download
Download the latest binary from [GitHub Releases](https://github.com/muhammadbassiony/rulem/releases/latest)

### Build from Source
```bash
git clone https://github.com/muhammadbassiony/rulem.git
cd rulem
go build -o rulem ./cmd/rulem
```

## Quick Start

1. **First Run**: Launch rulem and complete the setup wizard
   ```bash
   rulem
   ```

2. **Save Instructions**: Store your instruction files in the central repository
   - Navigate to "Save rules file"
   - Select your instruction file
   - Choose the appropriate AI assistant type

3. **Import to Project**: Apply instructions to a new project
   - Navigate to "Import rules (Copy)"
   - Select the instruction file
   - Choose copy or symlink mode

4. **Configure Settings**: Customize storage locations and preferences
   - Navigate to "Update settings"
   - Modify configuration as needed


## Configuration

Configuration is stored in `~/.config/rulem/config.yaml`:

```yaml
storage_dir: "/Users/username/.rulem"
version: "1.0"
init_time: 1640995200
```

for now it only contains the location of the storage directory, but more settings will be added in the future.

### Supported AI Assistants

- GitHub Copilot (both general and custom instructions)
- Cursor
- Claude code
- Gemini CLI
- Opencode

## Development

### Prerequisites
- Go 1.24+
- [Task](https://taskfile.dev/) (optional, for development tasks)

### Common Commands

```bash
# Development workflow
go mod tidy          # Install dependencies
go build ./cmd/rulem # Build binary
go test ./...        # Run tests
go run ./cmd/rulem   # Run from source

# With Task (recommended)
task build           # Build binary
task test            # Run tests
task lint            # Run linters
```

### Project Structure

```
rulem/
├── cmd/rulem/           # CLI entry point
├── internal/
│   ├── config/         # Configuration management
│   ├── filemanager/    # File operations
│   ├── logging/        # Logging system
│   └── tui/            # Terminal UI components
├── pkg/fileops/        # Core file operations
└── tasks/              # Development documentation
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.
