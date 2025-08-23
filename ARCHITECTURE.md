# Architecture Overview: rulem

This document describes the architecture, design decisions, and extensibility model for the rulem CLI tool.

## Overview

rulem is built as a modular Go application following clean architecture principles. It uses a Terminal User Interface (TUI) built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lipgloss](https://github.com/charmbracelet/lipgloss) for styling, providing an interactive way to manage AI assistant rule files.

## 1. High-Level Architecture

rulem is organized as a modular Go application with a clear separation of concerns:

### Directory Structure

```
rulem/
├── cmd/rulem/              # CLI entry point and main loop
├── internal/
│   ├── config/            # Configuration management (YAML)
│   ├── logging/           # Structured logging system
│   ├── tui/               # Terminal User Interface components
│   │   ├── components/    # Reusable UI components (layout, filepicker)
│   │   ├── helpers/       # UI utility functions and context
│   │   ├── setupmenu/     # First-time setup wizard
│   │   ├── saverulesmodel/# Save rules file functionality
│   │   ├── importrulesmenu/# Import rules with copy/symlink
│   │   ├── settingsmenu/  # Configuration settings UI
│   │   └── styles/        # TUI styling and theming
│   ├── filemanager/       # File system operations and storage management
│   └── editors/           # Editor integration and detection
├── pkg/
│   └── fileops/           # Core file operations (copy, symlink, validation)
└── tasks/                 # Development tasks and documentation
```

### Key Design Principles

- **Separation of Concerns**: Each package has a single, well-defined responsibility
- **Dependency Injection**: Configuration and dependencies are injected rather than hardcoded
- **Error Handling**: Comprehensive error handling with user-friendly messages
- **Testability**: Modular design enables comprehensive unit testing
- **Extensibility**: Plugin-like architecture for adding new AI assistants and formats

## 2. Key Components

### CLI Entry Point (`cmd/rulem/main.go`)

**Responsibilities:**
- Application initialization and startup sequence
- First-time setup detection and handling
- Configuration loading and validation
- TUI initialization with Bubble Tea
- Graceful error handling and shutdown

**Key Functions:**
- `main()`: Orchestrates the complete application lifecycle
- `runFirstTimeSetup()`: Handles initial user configuration

### Configuration Management (`internal/config/`)

**Responsibilities:**
- YAML-based configuration persistence
- Platform-specific config file location handling (XDG Base Directory spec)
- Configuration validation and migration
- Secure storage directory management

**Key Types:**
- `Config`: Main configuration structure with storage paths and metadata
- `LoadConfigMsg`, `SaveConfigMsg`, `ReloadConfigMsg`: Bubble Tea message types for async operations

**Key Functions:**
- `Load()`: Loads configuration from standard location
- `Save()`: Persists configuration changes
- `IsFirstRun()`: Detects first-time application usage

### Terminal User Interface (`internal/tui/`)

**Architecture:** State-based TUI with centralized navigation

**Core Components:**
- `MainModel`: Root TUI model implementing the Bubble Tea pattern
- `AppState`: Enumeration of application states (Menu, Settings, Import, etc.)
- `MenuItemModel`: Interface for feature-specific models

**Key Features:**
- **State Management**: Clean separation between different application views
- **Navigation System**: Message-based transitions between states
- **Error Handling**: Consistent error display and recovery mechanisms
- **Responsive Design**: Dynamic layout adaptation to terminal dimensions

**UI Models:**
- `SetupModel`: First-time setup wizard
- `SettingsModel`: Configuration management interface
- `SaveRulesModel`: Save rules file functionality
- `ImportRulesModel`: Import rules with copy/symlink options

### File Management (`internal/filemanager/`)

**Responsibilities:**
- Secure storage directory creation and management
- File system operations with proper permissions
- Storage path resolution and validation
- Directory scanning and file discovery

**Key Functions:**
- `CreateSecureStorageRoot()`: Creates storage directories with proper permissions
- `GetDefaultStorageDir()`: Determines platform-appropriate default storage location

### File Operations (`pkg/fileops/`)

**Responsibilities:**
- Core file operations (copy, symlink, validation)
- Cross-platform file system compatibility
- Error handling for file system operations
- File content validation and processing

**Key Functions:**
- `CopyFile()`: Safe file copying with overwrite protection
- `CreateSymlink()`: Cross-platform symbolic link creation
- `ValidateFile()`: File content validation against schemas

### Logging System (`internal/logging/`)

**Responsibilities:**
- Structured logging throughout the application
- Configurable log levels and output formats
- Integration with Bubble Tea message system
- Debug information and user action tracking

**Key Features:**
- **Structured Logs**: JSON-formatted logs with context
- **User Action Tracking**: Logs user interactions for analytics and debugging
- **State Transition Logging**: Tracks TUI state changes for debugging

## 3. Data Flow

### Application Startup Flow

1. **CLI Entry Point** (`main()`)
   - Initialize logging system
   - Check for first-time setup
   - Load user configuration
   - Initialize TUI with Bubble Tea
   - Start main application loop

2. **First-Time Setup** (if needed)
   - Launch setup wizard TUI
   - Guide user through configuration
   - Create initial config file
   - Set up storage directories

### Main Application Flow

1. **TUI Initialization**
   - Create MainModel with configuration
   - Set up menu items and navigation
   - Initialize layout and styling
   - Start Bubble Tea program loop

2. **Menu Navigation**
   - Display main menu with available options
   - Handle user input (navigation, selection, filtering)
   - Transition to selected feature state
   - Create and initialize feature-specific model

3. **Feature Operation** (e.g., Import Rules)
   - Initialize feature-specific model
   - Display feature interface
   - Handle user interactions
   - Perform file operations
   - Show progress and results
   - Return to main menu

### File Operation Flow

1. **File Selection**
   - Scan storage directory for available files
   - Display file browser interface
   - Handle user selection and filtering

2. **Migration Process**
   - User selects migration method (copy/symlink)
   - User selects target assistant/format
   - Validate source file format
   - Perform file operation with safety checks
   - Update UI with progress and results

3. **File Editing** (if needed)
   - Launch user's preferred editor
   - Wait for editor to close
   - Re-validate file after editing
   - Continue with migration or show errors

### Configuration Flow

1. **Load Configuration**
   - Determine config file path (platform-specific)
   - Read and parse YAML configuration
   - Validate configuration values
   - Apply defaults for missing values

2. **Update Configuration**
   - User modifies settings through TUI
   - Validate new configuration values
   - Save changes to disk
   - Send reload message to refresh application state

## 4. TUI Architecture

### State-Based Design

The TUI follows a state machine pattern where the application can be in one of several states:

- **StateMenu**: Main navigation menu
- **StateSettings**: Configuration management
- **StateSaveRules**: Save rules file interface
- **StateImportCopy**: Import rules with copy/symlink
- **StateFetchGithub**: GitHub integration interface
- **StateError**: Error display and recovery
- **StateComingSoon**: Placeholder for unimplemented features

### Message System

Communication between components uses Bubble Tea's message system:

- **Navigation Messages**: `NavigateMsg` for state transitions
- **Error Messages**: `ErrorMsg` for error handling
- **Configuration Messages**: `ReloadConfigMsg` for config updates
- **User Input**: `tea.KeyMsg`, `tea.WindowSizeMsg` for UI events

### Model Hierarchy

```
MainModel (Root)
├── Menu List (Bubble Tea list component)
├── Layout Component (Consistent styling)
├── Active Feature Model (State-dependent)
│   ├── SettingsModel
│   ├── SaveRulesModel
│   ├── ImportRulesModel
│   └── FetchGithubModel
└── UI State (Error, loading, etc.)
```

### Layout System

- **Responsive Design**: Adapts to terminal dimensions
- **Consistent Styling**: Uses Lipgloss for cross-platform styling
- **Component-Based**: Reusable UI components (layout, filepicker)
- **Error Handling**: Integrated error display and recovery

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
