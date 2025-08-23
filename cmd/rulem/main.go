// Package main is the entry point for the rulem CLI application.
//
// This package initializes the application, handles first-time setup,
// loads configuration, and starts the Terminal User Interface (TUI).
// The application follows this startup sequence:
//
// 1. Initialize logging system
// 2. Check for first-time setup and run if needed
// 3. Load user configuration from disk
// 4. Initialize and start the TUI with Bubble Tea
// 5. Handle graceful shutdown on exit
//
// The main function serves as the orchestrator, delegating specific
// functionality to appropriate internal packages while maintaining
// overall application flow and error handling.
package main

import (
	"fmt"
	"os"

	"rulem/internal/config"
	"rulem/internal/logging"
	"rulem/internal/tui"
	"rulem/internal/tui/helpers"
	"rulem/internal/tui/setupmenu"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// setup logging
	appLogger := logging.NewAppLogger()

	// Check if first run and handle setup
	if config.IsFirstRun() {
		if err := runFirstTimeSetup(appLogger); err != nil {
			appLogger.Error("Setup failed", "error", err)
			os.Exit(1)
		}
	}
	appLogger.Debug("First run setup completed successfully.")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		appLogger.Error("Error loading config", "error", err)
		os.Exit(1)
	}
	if cfg == nil {
		appLogger.Error("Error loading config", "error", err)
		os.Exit(1)
	}

	appLogger.Info("Configuration loaded successfully.", cfg.StorageDir, cfg.InitTime)

	// Initialize TUI application
	model := tui.NewMainModel(cfg, appLogger)
	program := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		appLogger.Error("Error starting TUI program", "error", err)
		os.Exit(1)
	}

}

// runFirstTimeSetup handles the initial application setup for new users.
//
// This function is called automatically when the application detects that
// it's being run for the first time (no configuration file exists).
// It launches an interactive setup wizard using Bubble Tea that guides
// the user through:
//
// - Choosing a storage directory for rule files
// - Setting up initial configuration
// - Creating necessary directories and permissions
//
// The setup process runs in full-screen mode and provides a user-friendly
// interface for configuration. If the user cancels the setup, the function
// returns an error to prevent the application from continuing without config.
//
// Parameters:
//   - logger: Logger instance for recording setup progress and errors
//
// Returns:
//   - error: Nil if setup completed successfully, or an error if setup failed or was cancelled
func runFirstTimeSetup(logger *logging.AppLogger) error {
	ctx := helpers.NewUIContext(0, 0, nil, logger) // Dimensions will be set by tea program
	menu := setupmenu.NewSetupModel(ctx)
	program := tea.NewProgram(menu, tea.WithAltScreen())

	finalModel, err := program.Run()
	if err != nil {
		return fmt.Errorf("setup failed: %w", err)
	}

	// Extract setup results and save config
	setup := finalModel.(*setupmenu.SetupModel)
	if setup.Cancelled {
		return fmt.Errorf("setup cancelled by user")
	}

	return nil
}
