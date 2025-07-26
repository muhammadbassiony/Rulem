package main

import (
	"fmt"
	"log/slog"
	"os"

	"rulem/internal/config"
	"rulem/internal/logging"
	"rulem/internal/tui"
	"rulem/internal/tui/setupmenu"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// setup logging
	appLogger := logging.NewAppLogger()

	// Check if first run and handle setup
	if config.IsFirstRun() {
		if err := runFirstTimeSetup(appLogger); err != nil {
			slog.Error("Setup failed", "error", err)
			os.Exit(1)
		}
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Error loading config", "error", err)
		os.Exit(1)
	}

	slog.Info("Configuration loaded successfully.", cfg.StorageDir, cfg.InitTime)

	// Initialize TUI application
	model := tui.NewMainModel(cfg, appLogger)
	program := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		slog.Error("Error starting TUI program", "error", err)
		os.Exit(1)
	}

}

func runFirstTimeSetup(logger *logging.AppLogger) error {
	menu := setupmenu.NewSetupModel(logger)
	program := tea.NewProgram(menu)

	finalModel, err := program.Run()
	if err != nil {
		return fmt.Errorf("setup failed: %w", err)
	}

	// Extract setup results and save config
	setup := finalModel.(setupmenu.SetupModel)
	if setup.Cancelled {
		return fmt.Errorf("setup cancelled by user")
	}

	return nil
}
