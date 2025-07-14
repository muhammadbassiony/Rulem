## Relevant Files

- `cmd/rulemig/main.go` - Entry point for the CLI tool.
- `internal/ui/fileviewer.go` - Implements the file viewer interface using Go charm libraries.
- `internal/ui/fileviewer_test.go` - Unit tests for the file viewer interface.
- `internal/core/migrator.go` - Core logic for copying, symlinking, and validating instruction files.
- `internal/core/migrator_test.go` - Unit tests for migrator logic.
- `internal/config/config.go` - Handles configuration management (storage location, etc.).
- `internal/config/config_test.go` - Unit tests for configuration management.
- `internal/validation/validator.go` - Validates Markdown instruction files against required schema.
- `internal/validation/validator_test.go` - Unit tests for file validation logic.
- `README.md` - Project overview, installation, usage, and contribution guidelines.
- `ARCHITECTURE.md` - Detailed technical architecture and design decisions.
- `Taskfile.yml` - Defines all local development, build, test, and lint commands for Task.

### Notes

- Unit tests should typically be placed alongside the code files they are testing (e.g., `fileviewer.go` and `fileviewer_test.go` in the same directory).
- Use `go test ./...` to run all tests in the project.
- Use `task` to run all local development commands (see README.md for details).

## Tasks

- [ ] 1.0 Set up project structure and configuration management
- [x] 1.1 Initialize Go module and project directory structure
  - [x] 1.2 Implement configuration file handling (YAML/JSON)
  - [x] 1.3 Implement logic for setting and updating storage location
  - [x] 1.4 Add unit tests for configuration management
  - [x] 2.1 Integrate Go charm libraries (bubbletea, lipgloss)
  - [x] 2.2 Display list of available instruction files with names and descriptions
  - [x] 2.3 Allow user to select one or more files for migration
  - [x] 2.4 Add unit tests for file viewer interface
  - [x] 3.1 Prompt user to choose between copy or symlink for selected files
  - [x] 3.2 Implement logic for copying files to target locations
  - [x] 3.3 Implement logic for creating symlinks to centralized files
  - [x] 3.4 Prevent overwriting existing files without confirmation
  - [x] 3.5 Add unit tests for migration logic
- [ ] 4.0 Implement instruction file editing and validation
  - [x] 4.1 Launch user’s preferred terminal editor for file editing
  - [x] 4.2 Validate Markdown files against required structure/schema
  - [x] 4.3 Display clear error messages for invalid files
  - [x] 4.4 Add unit tests for validation logic
- [ ] 5.0 Integrate support for multiple AI assistants and file formats/locations
  - [x] 5.1 Implement logic for mapping instruction files to correct locations for each supported AI assistant
  - [x] 5.2 Allow user to select target AI assistant/format during migration
  - [x] 5.3 Add support for default and custom locations
  - [x] 5.4 Add unit tests for assistant/format mapping logic
- [ ] 6.0 Implement user prompts, error handling, and terminal UI with Go charm libraries
  - [x] 6.1 Implement interactive prompts for all user decisions
  - [x] 6.2 Display error/success messages in the terminal UI
  - [x] 6.3 Ensure smooth navigation and return to main menu after actions
  - [x] 6.4 Add unit tests for UI and error handling
- [x] Documentation: Add README.md and ARCHITECTURE.md files
