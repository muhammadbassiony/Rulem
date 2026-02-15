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
	"os/signal"
	"rulem/internal/config"
	"rulem/internal/logging"
	"rulem/internal/tui"
	"rulem/internal/tui/helpers"
	"rulem/internal/tui/setupmenu"
	"runtime"
	"syscall"

	mcp "rulem/internal/mcp"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// Version info (set by GoReleaser)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var (
	debugMode bool
	appLogger *logging.AppLogger
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "rulem",
	Short: "AI Assistant Instruction Manager",
	Long: `rulem is a command-line tool for managing and organizing AI assistant
instruction files across different environments and formats.

Key features:
• Modern terminal user interface built with Bubble Tea
• Cross-platform support (macOS, Linux, Windows)
• Copy and symlink support for file management
• Support for multiple AI assistants (GitHub Copilot, Cursor, Claude, etc.)
• Centralized instruction file management
• Version control friendly`,
	Example: `  # Launch the interactive TUI (default behavior)
  rulem

  # Start with debug logging enabled
  rulem --debug

  # Start the MCP server
  rulem mcp

  # Show version information
  rulem version

Note: Debug logs are saved to ./rulem.log in the current directory`,
	RunE: runTUI,
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  "Print detailed version information including build commit and date",
	Run: func(cmd *cobra.Command, args []string) {
		initLogger() // Initialize logger for debug output if needed
		fmt.Printf("rulem %s (%s) built on %s\n", version, commit, date)
	},
}

// mcpCmd represents the MCP server command
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the Model Context Protocol server",
	Long: `Start the Model Context Protocol (MCP) server to enable integration
with AI assistants that support the MCP standard.

This allows rulem to be used as a context provider for AI assistants,
giving them access to your organized instruction files.

The server communicates via stdin/stdout using JSON-RPC as per MCP specification.`,
	RunE: runMCPServer,
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&debugMode, "debug", "d", false, "Enable debug logging")

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(mcpCmd)

	// Hide the help command and completion command in the main help output
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:    "help [command]",
		Short:  "Help about any command",
		Hidden: true,
		Run:    rootCmd.HelpFunc(),
	})

	// Hide the completion command
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
}

// initLogger initializes the logger based on debug mode
func initLogger() {
	// Initialize the global singleton logger
	logging.Initialize(debugMode)

	// Get reference to the singleton for local use
	appLogger = logging.GetDefault()

	if debugMode {
		appLogger.Debug("Debug mode enabled")
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runTUI handles the main TUI execution (default behavior)
func runTUI(cmd *cobra.Command, args []string) error {
	// Initialize logger based on debug flag
	initLogger()

	// Check if first run and handle setup
	if config.IsFirstRun() {
		appLogger.Debug("First run detected, starting setup")
		if err := runFirstTimeSetup(appLogger); err != nil {
			return fmt.Errorf("setup failed: %w", err)
		}
		appLogger.Debug("First run setup completed successfully")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}
	if cfg == nil {
		return fmt.Errorf("configuration is nil after loading")
	}
	appLogger.Info("Configuration loaded successfully", "init_time", cfg.InitTime)

	// Initialize TUI application with panic recovery
	model := tui.NewMainModel(cfg, appLogger)
	program := tea.NewProgram(model, tea.WithAltScreen(), tea.WithoutCatchPanics())

	appLogger.Debug("Starting TUI program")
	return runWithRecovery(func() error {
		_, err := program.Run()
		return err
	}, appLogger, "TUI program")
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
	logger.Debug("Initializing first-time setup UI")
	ctx := helpers.NewUIContext(0, 0, nil, logger) // Dimensions will be set by tea program
	menu := setupmenu.NewSetupModel(ctx)
	program := tea.NewProgram(menu, tea.WithAltScreen(), tea.WithoutCatchPanics())

	logger.Debug("Running setup program")
	var finalModel tea.Model
	err := runWithRecovery(func() error {
		var err error
		finalModel, err = program.Run()
		return err
	}, logger, "setup program")
	if err != nil {
		return fmt.Errorf("setup program failed: %w", err)
	}

	// Extract setup results and save config
	setup := finalModel.(*setupmenu.SetupModel)
	if setup.Cancelled {
		logger.Debug("Setup was cancelled by user")
		return fmt.Errorf("setup cancelled by user")
	}

	logger.Debug("Setup completed successfully")
	return nil
}

// runWithRecovery runs a function with panic recovery and graceful terminal restoration
func runWithRecovery(fn func() error, logger *logging.AppLogger, operation string) error {
	defer func() {
		if r := recover(); r != nil {
			// Restore terminal in case of panic
			fmt.Print("\033[?25h") // Show cursor
			fmt.Print("\033[2J")   // Clear screen
			fmt.Print("\033[H")    // Move cursor to home

			// Get stack trace
			buf := make([]byte, 1024*64)
			buf = buf[:runtime.Stack(buf, false)]

			// Log detailed error information to file
			logger.Error("Panic occurred during "+operation,
				"panic", r,
				"stack", string(buf),
			)

			// Show professional error message to user
			fmt.Fprintf(os.Stderr, "\n🚨 Application Error\n")
			fmt.Fprintf(os.Stderr, "The %s encountered an unexpected error and needs to close.\n\n", operation)

			if debugMode {
				fmt.Fprintf(os.Stderr, "Debug information:\n")
				fmt.Fprintf(os.Stderr, "  Error: %v\n", r)
				fmt.Fprintf(os.Stderr, "  Log file: ./rulem.log\n\n")
				fmt.Fprintf(os.Stderr, "Please check the log file for detailed stack trace information.\n")
			} else {
				fmt.Fprintf(os.Stderr, "To get detailed error information, run with --debug flag:\n")
				fmt.Fprintf(os.Stderr, "  rulem --debug\n\n")
			}

			fmt.Fprintf(os.Stderr, "If this problem persists, please report it as a bug.\n")

			// Force exit without returning to avoid Cobra's error handling
			os.Exit(1)
		}
	}()

	return fn()
}

// runMCPServer handles the MCP server execution
func runMCPServer(cmd *cobra.Command, args []string) error {
	// Initialize logger based on debug flag
	initLogger()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}
	if cfg == nil {
		return fmt.Errorf("configuration is nil after loading")
	}

	// Create and start MCP server
	appLogger.Info("Starting MCP server")
	server := mcp.NewServer(cfg, appLogger)
	if server == nil {
		return fmt.Errorf("failed to initialize MCP server")
	}

	appLogger.Debug("MCP server initialized, starting communication loop")

	// Set up signal handling for graceful shutdown

	// Create channel to listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- runWithRecovery(func() error {
			return server.Start()
		}, appLogger, "MCP server")
	}()

	// Wait for either server error or shutdown signal
	select {
	case err := <-errChan:
		if err != nil {
			appLogger.Error("MCP server error", "error", err)
			return err
		}
	case sig := <-sigChan:
		appLogger.Info("Received shutdown signal", "signal", sig)

		// Graceful shutdown
		if err := server.Stop(); err != nil {
			appLogger.Error("Error during server shutdown", "error", err)
		} else {
			appLogger.Info("MCP server stopped gracefully")
		}
	}

	return nil
}
