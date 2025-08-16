package config

import (
	"fmt"
	"os"
	"path/filepath"
	"rulem/internal/filemanager"
	"rulem/internal/logging"
	"time"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"
)

// TUI Messages for async operations
type LoadConfigMsg struct {
	Config *Config
	Error  error
}

type SaveConfigMsg struct {
	Error error
}

const APP_NAME = "rulem" // application name used for config directory

// Config holds user configuration for rulem.
type Config struct {
	// StorageDir is the directory where rulem stores its rule files.
	StorageDir string `yaml:"storage_dir"`
	Version    string `yaml:"version"`   // Track config version
	InitTime   int64  `yaml:"init_time"` // Unix timestamp of first setup
}

// ConfigPath returns the standard config file paths for the current platform
func ConfigPath() (string, error) {
	configDir := filepath.Join(xdg.ConfigHome, APP_NAME)
	configPath := filepath.Join(configDir, "config.yaml")

	logging.Debug("Determined config paths", "path", configPath)
	return configPath, nil
}

// Load loads the config from the standard location
// If no config exists, it returns an error indicating first run is needed
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
	primary, err := ConfigPath()
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
func DefaultConfig() Config {
	path := filemanager.GetDefaultStorageDir()
	logging.Debug("Using default storage directory", "path", path)

	return Config{
		StorageDir: path,
		Version:    "1.0",
		InitTime:   0, // Will be set during first save
	}
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

// SetStorageDir updates the storage directory and saves the config
func (c *Config) SetStorageDir(newDir string) error {
	c.StorageDir = newDir
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

// UpdateStorageDir updates the storage directory, ensures it exists, and saves the config
func UpdateStorageDir(cfg *Config, newStorageDir string) error {
	cfg.StorageDir = newStorageDir

	// Ensure storage directory exists
	root, err := filemanager.CreateSecureStorageRoot(cfg.StorageDir)
	if err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}
	defer root.Close()

	// Save the config
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	logging.Info("Configuration updated successfully", "storage_dir", cfg.StorageDir)
	return nil
}

// CreateNewConfig initializes a new configuration with the specified storage directory
func CreateNewConfig(storageDir string) error {
	// Create the default config
	cfg := DefaultConfig()
	cfg.StorageDir = storageDir

	// Ensure storage directory exists
	root, err := filemanager.CreateSecureStorageRoot(cfg.StorageDir)
	if err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}
	defer root.Close()

	// Save the config to the standard location
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	logging.Info("Configuration created successfully", "storage_dir", cfg.StorageDir)
	return nil
}
