package main

import (
	"fmt"
	"log/slog"
	"os"

	"rulem/internal/config"
	"rulem/internal/tui"
	"rulem/internal/tui/setupmenu"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// TODO reenable once we have a proper logging setup
	// // Initialize logging
	// if len(os.Getenv("DEBUG")) > 0 {
	// 	f, err := tea.LogToFile("debug.log", "debug")
	// 	if err != nil {
	// 		slog.Error("Failed to open log file", "error", err)
	// 		os.Exit(1)
	// 	}
	// 	defer f.Close()
	// }
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		// Level: slog.LevelDebug,
		Level: slog.LevelError,
	}))
	slog.SetDefault(logger)

	// Check if first run and handle setup
	if config.IsFirstRun() {
		if err := runFirstTimeSetup(); err != nil {
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
	model := tui.NewMainModel(cfg)
	program := tea.NewProgram(model, tea.WithAltScreen())
	if err := program.Start(); err != nil {
		slog.Error("Error starting TUI program", "error", err)
		os.Exit(1)
	}

}

func runFirstTimeSetup() error {
	menu := setupmenu.NewSetupModel()
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
