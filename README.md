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
- [MCP Integration](#mcp-integration)
  - [How MCP Works](#how-mcp-works)
  - [Tool Registration](#tool-registration)
  - [Example Tool Usage](#example-tool-usage)
  - [Integration with Popular Code Editors](#integration-with-popular-code-editors)
  - [Writing Effective Rule Descriptions](#writing-effective-rule-descriptions)
  - [Developing and Testing the MCP Server](#developing-and-testing-the-mcp-server)
  - [Why Rules as Tools](#why-rules-as-tools)
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
- **Better Integration in AI Assisted Development**: By integrating with the Model Context Protocol (MCP), rulem allows AI assistants to access your organized instruction files as tools and resources, enhancing their ability to assist you effectively.
- **Experimentation** & adpoting more AI powered development

For an example of a centralized rule repository in action, see [muhammadbassiony/Rulefiles](https://github.com/muhammadbassiony/Rulefiles).

## Features

- Modern terminal user interface built with Bubble Tea
- Cross-platform support (macOS, Linux, Windows)
- Model Context Protocol (MCP) server for AI assistant integration
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

2. **Get Help**: View all available commands and options
   ```bash
   rulem --help
   ```

3. **Debug Mode**: Enable detailed logging for troubleshooting
   ```bash
   rulem --debug
   ```

4. **Save Instructions**: Store your instruction files in the central repository
   - Navigate to "Save rules file"
   - Select your instruction file
   - Choose the appropriate AI assistant type

5. **Import to Project**: Apply instructions to a new project
   - Navigate to "Import rules (Copy)"
   - Select the instruction file
   - Choose copy or symlink mode

6. **Configure Settings**: Customize storage locations and preferences
   - Navigate to "Update settings"
   - Modify configuration as needed

For examples of rule files and repository organization, see [muhammadbassiony/Rulefiles](https://github.com/muhammadbassiony/Rulefiles).


## Configuration

Configuration is stored in `~/.config/rulem/config.yaml`:

```yaml
storage_dir: "/Users/username/.rulem"
version: "1.0"
init_time: 1640995200
```

for now it only contains the location of the storage directory, but more settings will be added in the future.

## MCP Integration

rulem includes a Model Context Protocol (MCP) server that allows AI assistants to access your organized instruction files as tools and resources.

### Starting the MCP Server

```bash
# Start MCP server (communicates via stdin/stdout)
rulem mcp

# Start with debug logging
rulem mcp --debug
```

### How MCP Works

The rulem MCP server automatically scans your rule files and registers them as dynamic tools based on their YAML frontmatter. Each rule file with valid frontmatter (containing at least a `description` field) becomes an MCP tool that AI assistants can invoke to access the rule content.

### Tool Registration

- **Automatic Discovery**: Tools are registered at server startup by scanning the configured storage directory
- **Naming**: Tool names are generated from the rule file's frontmatter `name` field or filename, with sanitization for safety
- **Descriptions**: Tool descriptions combine the rule's description with application context (if `applyTo` is specified)
- **Content Access**: When invoked, tools return the full content of the rule file (excluding frontmatter)

### Example Tool Usage

If you have a rule file `react-best-practices.md` with frontmatter:
```yaml
description: "React development best practices"
applyTo: "frontend development"
```

It becomes an MCP tool named `react_best_practices` with description "Use this for React development best practices (apply to: frontend development)".

For a practical example of a centralized rule repository integrated with this CLI tool and MCP server, see [muhammadbassiony/Rulefiles](https://github.com/muhammadbassiony/Rulefiles).

### Integration with Popular Code Editors

For most AI IDEs and CLI tools that support MCP, you can configure them to use rulem as an MCP server running locally through stdio. For most tools, the configuration is similar. You will need to add a setting like:
```json
  "rulem": {
    "command": ["rulem", "mcp"],
    "env": {}
  }
```

An example from Zed in your settings file:
```json
  "context_servers": {
    "Rulem": {
      "enabled": true,
      "source": "custom",
      "command": "rulem",
      "args": ["mcp"]
    }
  }
```

### Writing Effective Rule Descriptions

To maximize the likelihood that LLMs will use your rule files, write clear, specific, and actionable descriptions in the YAML frontmatter:

- **Be Specific**: Clearly state the context or scenario where the rule applies (e.g., "Use this for debugging React component state issues")
- **Action-Oriented**: Start with verbs like "Use this for", "Apply this when", or "Follow these guidelines for"
- **Include Context**: Mention technologies, frameworks, or problem types (e.g., "React", "API design", "error handling")
- **Concise but Descriptive**: Keep under 200 characters while being informative
- **Indicate Scope**: Use the `applyTo` field to specify broader application areas

**Example of an effective description:**
```yaml
description: "Use this for implementing secure API authentication with JWT tokens"
applyTo: "backend development"
```

This makes it more likely the LLM will invoke the tool when working on authentication-related tasks.

### Developing and Testing the MCP Server

For development and testing of the MCP server, you can use the MCP Inspector tool to interact with the server and see how AI assistants perceive your rule files.

#### Using MCP Inspector

1. **Install MCP Inspector**:
   ```bash
   npm install -g @modelcontextprotocol/inspector
   ```

2. **Start the Inspector**:
   ```bash
   mcp-inspector
   ```
   This opens a web interface at `http://localhost:5173` (or similar).

3. **Connect to rulem MCP Server**:
   - In the Inspector, configure the server with command: `rulem mcp`
   - The Inspector will start the server and display available tools

4. **Test Tool Invocation**:
   - Use the web interface to call tools and see the responses
   - This shows exactly how AI assistants will see and interact with your rule files
   - Verify tool names, descriptions, and content delivery

The Inspector is invaluable for debugging tool registration, testing rule file parsing, and ensuring your rules are presented correctly to AI assistants.

For more information, see the [MCP Inspector Documentation](https://modelcontextprotocol.io/docs/tools/inspector).

### Why Rules as Tools

We chose to implement rule files as MCP tools rather than resources or prompts for several key reasons:

#### Design Rationale

- **Active Invocation**: Tools allow AI assistants to actively request rule content when needed, rather than having static access to all rules
- **Contextual Relevance**: LLMs can decide when to use specific rules based on the conversation context and task at hand
- **Selective Access**: Prevents information overload by allowing assistants to pull only relevant rules
- **Dynamic Content**: Tools can return processed content (e.g., excluding frontmatter) and adapt descriptions

#### Technical Benefits

- **Better Performance**: Only loads rule content when requested, reducing memory usage
- **Security**: Content is validated and sanitized before delivery
- **Flexibility**: Easy to add metadata-driven features like conditional application

#### MCP Support

This approach leverages MCP's tool capabilities, which are widely supported across clients. For a list of MCP-compatible clients, see the [official MCP clients page](https://modelcontextprotocol.io/clients).

For detailed integration examples and troubleshooting, see [docs/mcp-integration.md](docs/mcp-integration.md).

## CLI Commands

### Basic Usage
```bash
# Launch interactive TUI (default)
rulem

# Show version information
rulem version

# Enable debug logging
rulem --debug

# Get help for any command
rulem --help
rulem [command] --help
```

### Available Commands

- `rulem` - Launch the interactive Terminal User Interface
- `rulem version` - Display version information
- `rulem mcp` - Start Model Context Protocol server for AI assistant integration


### Flags

- `-d, --debug` - Enable detailed debug logging to `rulem.log`
- `-h, --help` - Show help information



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
