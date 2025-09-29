// Package config provides configuration management for the rulem application.
//
// This package handles loading, saving, and managing user configuration settings
// for the rulem CLI tool. It supports YAML-based configuration files stored in
// platform-standard locations using the XDG Base Directory specification.
//
// Key features:
//   - Automatic detection of configuration file locations
//   - First-run setup detection and handling
//   - Secure storage directory management
//   - Configuration validation and error handling
//   - Integration with Bubble Tea for async configuration operations
//
// Configuration is stored in YAML format and includes:
//   - Central repository location and optional sync metadata
//   - Configuration version tracking
//   - Initialization timestamp
//
// The package provides both synchronous and asynchronous (Bubble Tea command)
// interfaces for configuration operations, making it suitable for both CLI
// and TUI contexts.
//
// Default configuration location:
//   - Linux/macOS: ~/.config/rulem/config.yaml
//   - Windows: %APPDATA%\rulem\config.yaml
//
// Environment variables:
//   - RULEM_CONFIG_PATH: Override default config path (primarily for testing)
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"rulem/internal/logging"
	"rulem/internal/repository"
	"time"

	"github.com/adrg/xdg"
	tea "github.com/charmbracelet/bubbletea"
	"gopkg.in/yaml.v3"
)

// TUI Messages for async operations

// LoadConfigMsg is sent when a configuration loading operation completes.
// It contains either the loaded configuration or an error if loading failed.
type LoadConfigMsg struct {
	Config *Config
	Error  error
}

// SaveConfigMsg is sent when a configuration saving operation completes.
// It contains an error if the save operation failed, or nil on success.
type SaveConfigMsg struct {
	Error error
}

// ReloadConfigMsg is sent when a configuration reload operation completes.
// This is typically used after settings changes to refresh the application state.
type ReloadConfigMsg struct {
	Config *Config
	Error  error
}

const AppName = "rulem" // application name used for config directory

// Config holds user configuration for rulem.
//
// This struct represents the complete configuration state of the application,
// including storage locations, version information, and initialization metadata.
// The configuration is persisted as YAML and loaded at application startup.
//
// Fields:
//   - Central.Path: The directory where rulem stores or clones rule files
//   - Central.RemoteURL: Optional Git remote used to populate the local cache
//   - Version: Configuration schema version for handling upgrades
//   - InitTime: Unix timestamp when the configuration was first created
type Config struct {
	Version  string                             `yaml:"version"`   // Track config version
	InitTime int64                              `yaml:"init_time"` // Unix timestamp of first setup
	Central  repository.CentralRepositoryConfig `yaml:"central"`
}

// Path returns the standard config file paths for the current platform
// Can be overridden with RULEM_CONFIG_PATH environment variable for testing
func Path() (string, error) {
	// Check for test override first
	if testPath := os.Getenv("RULEM_CONFIG_PATH"); testPath != "" {
		logging.Debug("Using test config path from environment", "path", testPath)
		return testPath, nil
	}

	configDir := filepath.Join(xdg.ConfigHome, AppName)
	configPath := filepath.Join(configDir, "config.yaml")

	logging.Debug("Determined config paths", "path", configPath)
	return configPath, nil
}

// Load loads the config from the standard location.
// If no config exists, it returns an error indicating first run is needed.
//
// This is the primary entry point for loading configuration at application startup.
// It automatically determines the correct config file path based on the platform
// and XDG Base Directory specification.
//
// Returns:
//   - *Config: The loaded configuration if successful
//   - error: An error if loading fails or if first-time setup is required
func Load() (*Config, error) {
	configPath, exists := FindConfigFile()
	logging.Debug("Loading config from", "path", configPath)
	if !exists {
		return nil, fmt.Errorf("no configuration found, first-time setup required")
	}

	return LoadFrom(configPath)
}

// LoadFrom loads config from a specific path
func LoadFrom(path string) (*Config, error) {
	logging.Info("Reading config file from: ", "path", path)
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	var cfg Config
	logging.Info("Decoding config file")
	dec := yaml.NewDecoder(f)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// FindConfigFile returns the path to an existing config file, and whether it exists.
func FindConfigFile() (string, bool) {
	primary, err := Path()
	if err != nil {
		logging.Error("Failed to get config path", "error", err)
		return "", false
	}

	// Check primary location first
	if _, err := os.Stat(primary); err == nil {
		logging.Debug("Config found at primary path", "path", primary)
		return primary, true
	}

	// Return primary path for new config
	return primary, false
}

// IsFirstRun checks if this is the first time the application is run
func IsFirstRun() bool {
	_, exists := FindConfigFile()
	return !exists
}

// DefaultConfig returns a Config with sensible defaults.
// This creates a new configuration instance with platform-appropriate default values,
// including a default storage directory determined by the filemanager package.
//
// Returns:
//   - Config: A configuration struct with default values set
func DefaultConfig() Config {
	path := repository.GetDefaultStorageDir()
	logging.Debug("Using default storage directory", "path", path)

	cfg := Config{
		Version:  "1.0",
		InitTime: 0, // Will be set during first save
		Central: repository.CentralRepositoryConfig{
			Path: path,
		},
	}
	return cfg
}

// Save writes the config to the standard location
func (c *Config) Save() error {
	configPath, _ := FindConfigFile()
	return c.SaveTo(configPath)
}

// SaveTo writes the config to a specific path
func (c *Config) SaveTo(path string) error {
	// Set init time if this is the first save
	if c.InitTime == 0 {
		c.InitTime = time.Now().Unix()
	}

	if c.Central.Path == "" {
		return fmt.Errorf("central path must be set before saving config")
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create file with restrictive permissions (600) for security
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer f.Close()

	enc := yaml.NewEncoder(f)
	defer enc.Close()

	if err := enc.Encode(c); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// SetCentralPath updates the central repository path and saves the config.
func (c *Config) SetCentralPath(newPath string) error {
	c.Central.Path = newPath
	return c.Save()
}

// LoadConfig loads configuration and returns a message for TUI
func LoadConfig() (*Config, error) {
	return Load()
}

// SaveConfig saves configuration for TUI operations
func SaveConfig(cfg *Config) error {
	return cfg.Save()
}

// ReloadConfig reloads configuration from disk for TUI operations
// This is useful when the config has been updated by another component
func ReloadConfig() tea.Cmd {
	return func() tea.Msg {
		cfg, err := Load()
		return ReloadConfigMsg{Config: cfg, Error: err}
	}
}

// UpdateCentralPath updates the central repository path, ensures it exists, and saves the config.
func UpdateCentralPath(cfg *Config, newPath string) error {
	cfg.Central.Path = newPath

	// Ensure central repository path exists with secure permissions
	root, err := repository.EnsureLocalStorageDirectory(cfg.Central.Path)
	if err != nil {
		return fmt.Errorf("failed to prepare central repository directory: %w", err)
	}
	defer root.Close()

	// Save the config
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	logging.Info("Configuration updated successfully", "central_path", cfg.Central.Path)
	return nil
}

// CreateNewConfig initializes a new configuration with the specified storage directory
func CreateNewConfig(storageDir string) error {
	// Create the default config
	cfg := DefaultConfig()
	cfg.Central.Path = storageDir

	// Ensure storage directory exists
	root, err := repository.EnsureLocalStorageDirectory(cfg.Central.Path)
	if err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}
	defer root.Close()

	// Save the config to the standard location
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	logging.Info("Configuration created successfully", "central_path", cfg.Central.Path)
	return nil
}
