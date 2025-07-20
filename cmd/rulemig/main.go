package main

import (
	"fmt"
	"log/slog"
	"os"

	"rulemig/internal/config"
	"rulemig/internal/tui"

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
		Level: slog.LevelDebug,
		// Level: slog.LevelError,
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
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Configuration loaded successfully.", cfg.StorageDir, cfg.InitTime)
}

func runFirstTimeSetup() error {
	fmt.Println("Welcome to rulemig! Let's set up your configuration.")

	setupModel := tui.NewSetupModel()
	program := tea.NewProgram(setupModel)

	finalModel, err := program.Run()
	if err != nil {
		return fmt.Errorf("setup failed: %w", err)
	}

	// Extract setup results and save config
	setup := finalModel.(tui.SetupModel)
	if setup.Cancelled {
		return fmt.Errorf("setup cancelled by user")
	}

	return nil
}
