package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConfigPaths(t *testing.T) {
	t.Log("Testing ConfigPaths for different platforms")

	// Save original environment
	originalHome := os.Getenv("HOME")
	originalXDGConfig := os.Getenv("XDG_CONFIG_HOME")

	defer func() {
		os.Setenv("HOME", originalHome)
		if originalXDGConfig != "" {
			os.Setenv("XDG_CONFIG_HOME", originalXDGConfig)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	tests := []struct {
		name           string
		setupEnv       func()
		expectPrimary  func() string
		expectFallback func() string
	}{
		{
			name: "default macOS paths",
			setupEnv: func() {
				os.Setenv("HOME", "/Users/test")
				os.Unsetenv("XDG_CONFIG_HOME")
			},
			expectPrimary: func() string {
				return "/Users/test/.config/rulemig/config.yaml"
			},
			expectFallback: func() string {
				return "/Users/test/.rulemig.yaml"
			},
		},
		{
			name: "macOS with XDG_CONFIG_HOME",
			setupEnv: func() {
				os.Setenv("HOME", "/Users/test")
				os.Setenv("XDG_CONFIG_HOME", "/custom/config")
			},
			expectPrimary: func() string {
				return "/custom/config/rulemig/config.yaml"
			},
			expectFallback: func() string {
				return "/Users/test/.rulemig.yaml"
			},
		},
		{
			name: "Linux with XDG_CONFIG_HOME",
			setupEnv: func() {
				os.Setenv("HOME", "/home/test")
				os.Setenv("XDG_CONFIG_HOME", "/custom/config")
			},
			expectPrimary: func() string {
				return "/custom/config/rulemig/config.yaml"
			},
			expectFallback: func() string {
				return "/home/test/.rulemig.yaml"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()

			primary, fallback := ConfigPaths()

			if primary != tt.expectPrimary() {
				t.Errorf("Expected primary path %s, got %s", tt.expectPrimary(), primary)
			}

			if fallback != tt.expectFallback() {
				t.Errorf("Expected fallback path %s, got %s", tt.expectFallback(), fallback)
			}
		})
	}
}

func TestConfigSaveLoad(t *testing.T) {
	t.Log("Testing Config Saving and Loading")

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Create test config
	originalConfig := Config{
		StorageDir: "/test/storage",
		Version:    "1.0",
		InitTime:   time.Now().Unix(),
	}

	// Test Save
	if err := originalConfig.SaveTo(configPath); err != nil {
		t.Fatalf("Failed to save config: %s", err)
	}

	// Test Load
	loadedConfig, err := LoadFrom(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %s", err)
	}

	// Verify content
	if loadedConfig.StorageDir != originalConfig.StorageDir {
		t.Errorf("StorageDir mismatch: expected %s, got %s", originalConfig.StorageDir, loadedConfig.StorageDir)
	}

	if loadedConfig.Version != originalConfig.Version {
		t.Errorf("Version mismatch: expected %s, got %s", originalConfig.Version, loadedConfig.Version)
	}

	if loadedConfig.InitTime != originalConfig.InitTime {
		t.Errorf("InitTime mismatch: expected %d, got %d", originalConfig.InitTime, loadedConfig.InitTime)
	}
}

func TestConfigInitTime(t *testing.T) {
	t.Log("Testing Config InitTime on Save")

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	config := Config{
		StorageDir: "/test",
		Version:    "1.0",
		// InitTime not set (0)
	}

	before := time.Now().Unix()
	if err := config.SaveTo(configPath); err != nil {
		t.Fatalf("Failed to save config: %s", err)
	}
	after := time.Now().Unix()

	// InitTime should be set during save
	if config.InitTime < before || config.InitTime > after {
		t.Errorf("InitTime %d should be between %d and %d", config.InitTime, before, after)
	}
}

func TestConfigFilePermissions(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	config := DefaultConfig()
	if err := config.SaveTo(configPath); err != nil {
		t.Fatalf("Failed to save config: %s", err)
	}

	// Check file permissions
	fileInfo, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %s", err)
	}

	mode := fileInfo.Mode()
	if mode&0077 != 0 {
		t.Errorf("Config file should not be readable by group/others, got mode %o", mode)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Version == "" {
		t.Error("Default config should have a version")
	}

	if config.StorageDir == "" {
		t.Error("Default config should have a storage directory")
	}

	if config.InitTime != 0 {
		t.Error("Default config InitTime should be 0 (will be set on save)")
	}
}

// Error handling tests
func TestConfigErrorHandling(t *testing.T) {
	t.Run("load non-existent file", func(t *testing.T) {
		_, err := LoadFrom("/non/existent/file.yaml")
		if err == nil {
			t.Error("Should error when loading non-existent file")
		}
	})

	t.Run("load invalid YAML", func(t *testing.T) {
		tempDir := t.TempDir()
		invalidFile := filepath.Join(tempDir, "invalid.yaml")
		os.WriteFile(invalidFile, []byte("invalid: yaml: content: ["), 0644)

		_, err := LoadFrom(invalidFile)
		if err == nil {
			t.Error("Should error when loading invalid YAML")
		}
	})

	t.Run("save to read-only directory", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping test as root user")
		}

		config := DefaultConfig()
		err := config.SaveTo("/root/config.yaml")
		if err == nil {
			t.Error("Should error when saving to read-only directory")
		}
	})
}
