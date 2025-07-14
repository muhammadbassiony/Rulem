# Architecture Overview: rulemig

This document describes the architecture, design decisions, and extensibility model for the rulemig CLI tool.

## 1. High-Level Architecture

rulemig is organized as a modular Go application with a clear separation of concerns:

- **cmd/rulemig/**: CLI entry point, argument parsing, and main loop
- **internal/config/**: Configuration management (YAML/JSON), storage location logic
- **internal/ui/**: Terminal UI (TUI) components using Bubble Tea and Lipgloss
- **internal/core/**: Core migration, symlink, and assistant mapping logic
- **internal/validation/**: Markdown file validation logic

## 2. Key Components

### CLI Entry Point (`cmd/rulemig/main.go`)
- Handles startup, config loading, and user prompts
- Orchestrates UI and core logic

### Configuration (`internal/config/config.go`)
- Loads and saves YAML config (default: `~/.config/rulemig/config.yaml`)
- Manages storage locations for each assistant
- Provides update and validation methods

### Terminal UI (`internal/ui/fileviewer.go`)
- Implements file browser and selection using Bubble Tea
- Handles navigation, selection, and error/success messages
- Integrates with core logic for migration actions

### Core Logic (`internal/core/migrator.go`, `assistantmap.go`)
- Handles file copy/symlink operations
- Maps instruction files to correct locations for each assistant
- Prevents overwrites and confirms user actions
- Extensible for new assistants/formats

### Validation (`internal/validation/validator.go`)
- Validates Markdown files for required structure/schema
- Provides clear error messages for invalid files

## 3. Data Flow

1. User launches CLI → Config is loaded
2. File viewer displays available instruction files
3. User selects files and migration method (copy/symlink)
4. User selects target assistant/format
5. Core migrator performs operation, validates files, and updates UI
6. User can edit files in terminal editor; validation is re-run

## 4. Extensibility

- **Adding a new AI assistant:**
  - Update `assistantmap.go` with new mapping logic and default locations
  - Add any required validation rules in `validator.go`
  - Update config schema if needed
- **Supporting new file formats:**
  - Extend validation logic
  - Update migration logic to handle new formats

## 5. Error Handling & User Experience

- All errors are surfaced in the TUI with clear, actionable messages
- Overwrites require explicit confirmation
- Invalid files prompt user to edit or abort
- Smooth navigation and return to main menu after actions

## 6. Testing

- Unit tests for all core logic, config, UI, and validation
- Run with `go test ./...`
- Tests are located alongside their respective code files

## 7. Design Decisions

- **Bubble Tea & Lipgloss** chosen for modern, maintainable TUI
- **YAML config** for human-readability and extensibility
- **Modular internal packages** for testability and maintainability
- **Editor integration** for seamless file editing

## 8. Future Improvements

- Plugin system for third-party assistant support
- Remote sync/integration with cloud storage
- Richer validation and linting for instruction files
- Enhanced accessibility and keyboard shortcuts

## 9. References

- [Bubble Tea Documentation](https://github.com/charmbracelet/bubbletea)
- [Lipgloss Documentation](https://github.com/charmbracelet/lipgloss)
- [Go Modules](https://blog.golang.org/using-go-modules)

---

For questions or contributions, see [README.md](./README.md) or open an issue on GitHub.
