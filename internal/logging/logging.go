// Package logging provides structured logging for the rulem application.
//
// This package wraps the Charm bracelet log library to provide consistent,
// structured logging throughout the application. It supports multiple log levels,
// key-value pair metadata, and integration with the Bubble Tea TUI framework.
//
// Key features:
//   - Structured logging with key-value pairs for better debugging
//   - Multiple log levels (Debug, Info, Warn, Error)
//   - Singleton pattern for easy access across the application
//   - Bubble Tea integration for logging UI events and state changes
//   - User action tracking for analytics and debugging
//   - Configurable output formatting and destinations
//
// The AppLogger struct is the main logging interface, providing methods for:
//   - Standard log levels with structured metadata
//   - User action logging for tracking user interactions
//   - State transition logging for TUI debugging
//   - Message logging for Bubble Tea event tracking
//
// Usage patterns:
//   - Use package-level functions for simple logging: logging.Info("message")
//   - Use AppLogger instance for more control: logger.Info("message", "key", "value")
//   - Include relevant context in key-value pairs for better debugging
//
// Log levels:
//   - Debug: Detailed debugging information (disabled in production)
//   - Info: General information about application flow
//   - Warn: Warning conditions that don't prevent operation
//   - Error: Error conditions that may impact functionality
package logging

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
)

type AppLogger struct {
	logger *log.Logger
	debug  bool
}

var (
	defaultLogger *AppLogger
	once          sync.Once
	debugMode     bool
)

// Initialize sets up the debug mode for the global logger.
// This should be called once at application startup before any logging occurs.
func Initialize(debug bool) {
	debugMode = debug
}

// GetDefault returns the default logger instance (singleton).
// Uses sync.Once to ensure thread-safe initialization.
func GetDefault() *AppLogger {
	once.Do(func() {
		defaultLogger = newAppLoggerWithDebugMode(debugMode)
	})
	return defaultLogger
}

// Package-level convenience functions for quick logging
func Info(msg string, keyvals ...any) {
	GetDefault().Info(msg, keyvals...)
}

func Warn(msg string, keyvals ...any) {
	GetDefault().Warn(msg, keyvals...)
}

func Error(msg string, keyvals ...any) {
	GetDefault().Error(msg, keyvals...)
}

func Debug(msg string, keyvals ...any) {
	GetDefault().Debug(msg, keyvals...)
}

func LogMessage(msg tea.Msg) {
	GetDefault().LogMessage(msg)
}

func LogPerformance(operation string, start time.Time) {
	GetDefault().LogPerformance(operation, start)
}

func newAppLoggerWithDebugMode(debug bool) *AppLogger {

	var logger *log.Logger

	if debug {
		// Debug mode: Always log to current directory and clear on each run
		// This design choice helps with:
		// 1. User issue debugging: Easy to find logs in current directory
		// 2. Development workflow: Fresh logs for each debug session
		// 3. Support requests: Clear, isolated log files for specific runs
		logPath := "rulem.log"

		// Always clear the log file when debug is enabled for fresh debugging
		// This ensures each debug session starts with a clean slate
		openFlags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC

		logFile, err := os.OpenFile(logPath, openFlags, 0o644)
		if err != nil {
			panic(fmt.Sprintf("Failed to create debug log file at %s: %v", logPath, err))
		}

		logger = log.NewWithOptions(logFile, log.Options{
			ReportCaller:    true,
			ReportTimestamp: true,
			TimeFormat:      time.Kitchen,
			Prefix:          "Rulem",
			CallerOffset:    1, // Skip wrapper function calls
		})
		logger.SetLevel(log.DebugLevel)

		fmt.Printf("Debug logging enabled. Log file: %s\n", logPath)
		logger.Info("Debug logging enabled", "log_file", logPath)

	} else {
		// Production: Log errors to stderr only
		logger = log.NewWithOptions(os.Stderr, log.Options{
			ReportTimestamp: true,
			TimeFormat:      time.RFC3339,
			Prefix:          "Rulem",
			CallerOffset:    1, // Skip wrapper function calls
		})
		logger.SetLevel(log.WarnLevel)
	}

	return &AppLogger{
		logger: logger,
		debug:  debug,
	}
}

// Log application events
func (al *AppLogger) Info(msg string, keyvals ...any) {
	al.logger.Info(msg, keyvals...)
}

func (al *AppLogger) Warn(msg string, keyvals ...any) {
	al.logger.Warn(msg, keyvals...)
}

func (al *AppLogger) Error(msg string, keyvals ...any) {
	al.logger.Error(msg, keyvals...)
}

func (al *AppLogger) Debug(msg string, keyvals ...any) {
	if al.debug {
		al.logger.Debug(msg, keyvals...)
	}
}

// Log a bubbletea message (debug only)
func (al *AppLogger) LogMessage(msg tea.Msg) {
	if !al.debug {
		return
	}

	al.logger.Debug("Message received",
		"type", fmt.Sprintf("%T", msg),
		"content", fmt.Sprintf("%+v", msg),
	)
}

// Pretty print any object (replaces spew)
func (al *AppLogger) DebugObject(name string, obj any) {
	if al.debug {
		al.logger.Debug("Object dump", "name", name, "object", fmt.Sprintf("%+v", obj))
	}
}

// Log performance metrics
func (al *AppLogger) LogPerformance(operation string, start time.Time) {
	if al.debug {
		duration := time.Since(start)
		al.logger.Debug("Performance",
			"operation", operation,
			"duration", duration,
		)
	}
}

// Log state transitions for debugging
func (al *AppLogger) LogStateTransition(component, from, to string) {
	if al.debug {
		al.logger.Debug("State transition",
			"component", component,
			"from", from,
			"to", to,
		)
	}
}

// Log user actions for debugging
func (al *AppLogger) LogUserAction(action, context string) {
	if al.debug {
		al.logger.Debug("User action",
			"action", action,
			"context", context,
		)
	}
}

// Testing Helper - NewTestLogger creates a logger that writes to a buffer for testing
func NewTestLogger() (*AppLogger, *bytes.Buffer) {
	var buf bytes.Buffer

	logger := log.NewWithOptions(&buf, log.Options{
		ReportTimestamp: false, // Easier to test without timestamps
		ReportCaller:    false,
		Prefix:          "Test",
		CallerOffset:    1, // Skip wrapper function calls
	})
	logger.SetLevel(log.DebugLevel)

	return &AppLogger{
		logger: logger,
		debug:  true,
	}, &buf
}
