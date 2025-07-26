package logging

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
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
)

// GetDefault returns the default logger instance (singleton-like for convenience)
func GetDefault() *AppLogger {
	once.Do(func() {
		defaultLogger = NewAppLogger()
	})
	return defaultLogger
}

// Package-level convenience functions for quick logging
func Info(msg string, keyvals ...interface{}) {
	GetDefault().Info(msg, keyvals...)
}

func Warn(msg string, keyvals ...interface{}) {
	GetDefault().Warn(msg, keyvals...)
}

func Error(msg string, keyvals ...interface{}) {
	GetDefault().Error(msg, keyvals...)
}

func Debug(msg string, keyvals ...interface{}) {
	GetDefault().Debug(msg, keyvals...)
}

func LogMessage(msg tea.Msg) {
	GetDefault().LogMessage(msg)
}

func LogPerformance(operation string, start time.Time) {
	GetDefault().LogPerformance(operation, start)
}

func NewAppLogger() *AppLogger {
	debug := os.Getenv("DEBUG") != ""

	var logger *log.Logger

	if debug {
		// Development: Log to file, clear on each run
		cwd, err := os.Getwd()
		if err != nil {
			panic(fmt.Sprintf("Failed to get current working directory: %v", err))
		}

		logPath := filepath.Join(cwd, "rulem.log")

		// Clear the log file on each run for development
		logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			panic(fmt.Sprintf("Failed to create debug log file: %v", err))
		}

		logger = log.NewWithOptions(logFile, log.Options{
			ReportCaller:    true,
			ReportTimestamp: true,
			TimeFormat:      time.Kitchen,
			Prefix:          "Rulem",
		})
		logger.SetLevel(log.DebugLevel)

		logger.Info("Debug logging enabled", "log_file", logPath)

	} else {
		// Production: Log errors to stderr only
		logger = log.NewWithOptions(os.Stderr, log.Options{
			ReportTimestamp: true,
			TimeFormat:      time.RFC3339,
			Prefix:          "Rulem",
		})
		logger.SetLevel(log.WarnLevel)
	}

	return &AppLogger{
		logger: logger,
		debug:  debug,
	}
}

// Log application events
func (al *AppLogger) Info(msg string, keyvals ...interface{}) {
	al.logger.Info(msg, keyvals...)
}

func (al *AppLogger) Warn(msg string, keyvals ...interface{}) {
	al.logger.Warn(msg, keyvals...)
}

func (al *AppLogger) Error(msg string, keyvals ...interface{}) {
	al.logger.Error(msg, keyvals...)
}

func (al *AppLogger) Debug(msg string, keyvals ...interface{}) {
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
func (al *AppLogger) DebugObject(name string, obj interface{}) {
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
	})
	logger.SetLevel(log.DebugLevel)

	return &AppLogger{
		logger: logger,
		debug:  true,
	}, &buf
}
