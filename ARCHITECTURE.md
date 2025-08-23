# Architecture Overview: rulem

## Overview

rulem is built as a modular Go application following clean architecture principles. It uses a Terminal User Interface (TUI) built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lipgloss](https://github.com/charmbracelet/lipgloss) for styling, providing an interactive way to manage AI assistant rule files.

## 1. High-Level Architecture

### Directory Structure

```
rulem/
├── cmd/rulem/                  # CLI entry point and main loop
├── internal/
│   ├── config/                # Configuration management (YAML)
│   ├── logging/               # Structured logging system
│   ├── tui/                   # Terminal User Interface components
│   │   ├── components/       # Reusable UI components (layout, filepicker)
│   │   ├── helpers/          # UI utility functions and context
│   │   ├── setupmenu/        # First-time setup wizard
│   │   ├── saverulesmodel/   # Save rules file functionality
│   │   ├── importrulesmenu/  # Import rules with copy/symlink
│   │   ├── settingsmenu/     # Configuration settings UI
│   │   └── styles/           # TUI styling and theming
│   ├── filemanager/          # File system operations and storage management
│   └── editors/              # Editor integration and detection
├── pkg/
│   └── fileops/              # Core file operations (copy, symlink, validation)
└── tasks/                    # Development tasks and documentation
```

## 2. Key Components

### CLI Entry Point (`cmd/rulem/main.go`)

**Responsibilities:**
- Application initialization and startup sequence
- First-time setup detection and handling
- Configuration loading and validation
- TUI initialization with Bubble Tea

### Configuration Management (`internal/config/`)

**Responsibilities:**
- YAML-based configuration persistence
- Platform-specific config file location handling (XDG Base Directory spec)
- Configuration validation and migration
- Secure storage directory management

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

### File Operations (`pkg/fileops/`)

**Responsibilities:**
- Core file operations (copy, symlink, validation)
- Cross-platform file system compatibility
- Error handling for file system operations
- File content validation and processing

### Editor Integration (`internal/editors/`)

**Responsibilities:**
- Contains editor rule file configurations
