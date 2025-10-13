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
	"regexp"
	"rulem/internal/logging"
	"rulem/internal/repository"
	"strings"
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
//   - Version: Configuration schema version (kept for informational purposes)
//   - InitTime: Unix timestamp when the configuration was first created
//   - Repositories: Array of configured repositories (replaces single Central field)
//
// Note: RepositoryEntry is defined in the repository package as it's a domain entity.
// Config package consumes repository domain types for persistence.
type Config struct {
	Version      string                       `yaml:"version"`      // Track config version (informational only)
	InitTime     int64                        `yaml:"init_time"`    // Unix timestamp of first setup
	Repositories []repository.RepositoryEntry `yaml:"repositories"` // Configured repositories (replaces Central)
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
// This creates a new configuration instance with an empty repositories array.
// Repositories should be added through the setup wizard or settings menu.
//
// Returns:
//   - Config: A configuration struct with default values set
func DefaultConfig() Config {
	cfg := Config{
		Version:      "1.0",
		InitTime:     0,                              // Will be set during first save
		Repositories: []repository.RepositoryEntry{}, // Empty array - repositories added through setup
	}
	return cfg
}

// FindRepositoryByID searches for a repository by its unique ID.
// Uses linear search which is acceptable for the expected scale (≤10 repositories).
//
// Parameters:
//   - id: The unique repository identifier to search for
//
// Returns:
//   - *repository.RepositoryEntry: Pointer to the found repository entry
//   - error: Error if repository is not found
func (c *Config) FindRepositoryByID(id string) (*repository.RepositoryEntry, error) {
	for i := range c.Repositories {
		if c.Repositories[i].ID == id {
			return &c.Repositories[i], nil
		}
	}
	return nil, fmt.Errorf("repository not found: %s", id)
}

// FindRepositoryByName searches for a repository by its display name.
// Uses case-insensitive matching for user convenience.
//
// Parameters:
//   - name: The repository display name to search for (case-insensitive)
//
// Returns:
//   - *repository.RepositoryEntry: Pointer to the found repository entry
//   - error: Error if repository is not found
func (c *Config) FindRepositoryByName(name string) (*repository.RepositoryEntry, error) {
	lowerName := strings.ToLower(name)
	for i := range c.Repositories {
		if strings.ToLower(c.Repositories[i].Name) == lowerName {
			return &c.Repositories[i], nil
		}
	}
	return nil, fmt.Errorf("repository not found: %s", name)
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

	// No validation on Repositories - empty array is valid (user hasn't configured yet)

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

// GenerateRepositoryID generates a unique, human-readable identifier for a repository.
// The ID format is "sanitized-name-timestamp" (e.g., "personal-rules-1728756432").
//
// This design provides several benefits:
//   - Human-readable: Contains the repository name for debugging
//   - Guaranteed uniqueness: Timestamp ensures no collisions even with same name
//   - Sortable: Natural chronological ordering
//   - Config-friendly: Short enough for YAML readability
//
// Parameters:
//   - name: The repository display name to sanitize
//   - timestamp: Unix timestamp (typically time.Now().Unix())
//
// Returns:
//   - string: Unique repository ID in format "sanitized-name-timestamp"
//
// Example:
//
//	id := GenerateRepositoryID("Personal Rules", 1728756432)
//	// Returns: "personal-rules-1728756432"
func GenerateRepositoryID(name string, timestamp int64) string {
	sanitized := sanitizeNameForID(name)
	return fmt.Sprintf("%s-%d", sanitized, timestamp)
}

// sanitizeNameForID converts a repository name to a safe, URL-friendly format for ID generation.
// This function implements the original sanitization logic that was previously in the repository package.
//
// The sanitization process:
//  1. Convert to lowercase
//  2. Replace non-alphanumeric characters with dashes using regex
//  3. Trim leading and trailing dashes
//  4. Fallback to "repo" if empty after sanitization
//
// Parameters:
//   - name: The repository display name to sanitize
//
// Returns:
//   - string: Sanitized name containing only lowercase letters, numbers, and dashes
//
// Examples:
//
//	sanitizeNameForID("Personal Rules")      // "personal-rules"
//	sanitizeNameForID("My@Project#123!")     // "my-project-123"
//	sanitizeNameForID("!!!@@@")              // "repo"
//	sanitizeNameForID("")                    // "repo"
//	sanitizeNameForID("  spaces  ")          // "spaces"
func sanitizeNameForID(name string) string {
	// Convert to lowercase
	sanitized := strings.ToLower(name)

	// Replace non-alphanumeric characters with dash
	// Pattern: [^a-z0-9]+ matches one or more characters that are NOT lowercase letters or digits
	re := regexp.MustCompile(`[^a-z0-9]+`)
	sanitized = re.ReplaceAllString(sanitized, "-")

	// Trim leading and trailing dashes
	sanitized = strings.Trim(sanitized, "-")

	// Fallback to "repo" if empty after sanitization
	if sanitized == "" {
		sanitized = "repo"
	}

	return sanitized
}
