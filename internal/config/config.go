package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"rulemig/internal/filemanager"
	"runtime"
	"time"

	"gopkg.in/yaml.v3"
)

const APP_NAME = "rulemig"

// Config holds user configuration for rulemig.
type Config struct {
	// StorageDir is the directory where instruction files are stored.
	StorageDir string `yaml:"storage_dir"`
	Version    string `yaml:"version"`   // Track config version
	InitTime   int64  `yaml:"init_time"` // Unix timestamp of first setup
}

// ConfigPaths returns the standard config file paths for the current platform
func ConfigPaths() (primary, fallback string) {
	slog.Info("Determining config paths for platform:", "os", runtime.GOOS)
	switch runtime.GOOS {
	case "darwin": // macOS
		home, _ := os.UserHomeDir()
		// Primary: XDG config (modern standard)
		if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
			primary = filepath.Join(xdgConfig, APP_NAME, "config.yaml")
		} else {
			primary = filepath.Join(home, ".config", APP_NAME, "config.yaml")
		}
		// Fallback: traditional dotfile
		fallback = filepath.Join(home, "."+APP_NAME+".yaml")

	case "windows":
		// Windows uses APPDATA
		if appData := os.Getenv("APPDATA"); appData != "" {
			primary = filepath.Join(appData, APP_NAME, "config.yaml")
		} else {
			home, _ := os.UserHomeDir()
			primary = filepath.Join(home, "AppData", "Roaming", APP_NAME, "config.yaml")
		}
		// No fallback on Windows
		fallback = primary

	case "linux":
	default:
		home, _ := os.UserHomeDir()
		// Primary: XDG config (Linux standard)
		if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
			primary = filepath.Join(xdgConfig, APP_NAME, "config.yaml")
		} else {
			primary = filepath.Join(home, ".config", APP_NAME, "config.yaml")
		}
		// Fallback: traditional dotfile
		fallback = filepath.Join(home, "."+APP_NAME+".yaml")
	}
	return
}

// Load loads the config from the standard location
// If no config exists, it returns an error indicating first run is needed
func Load() (*Config, error) {
	configPath, exists := FindConfigFile()
	slog.Info("Loading config from:", "path", configPath)
	if !exists {
		return nil, fmt.Errorf("no configuration found, first-time setup required")
	}

	return LoadFrom(configPath)
}

// LoadFrom loads config from a specific path
func LoadFrom(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	var cfg Config
	dec := yaml.NewDecoder(f)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// FindConfigFile returns the path to an existing config file, or the primary path if none exists
func FindConfigFile() (string, bool) {
	primary, fallback := ConfigPaths()

	// Check primary location first
	if _, err := os.Stat(primary); err == nil {
		slog.Info("Config found at primary path:", "path", primary)
		return primary, true
	}

	// Check fallback location
	if fallback != primary {
		if _, err := os.Stat(fallback); err == nil {
			return fallback, true
		}
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
	slog.Info("Using default storage directory:", "path", path)

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
