package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestConfigPath(t *testing.T) {
	t.Log("Testing ConfigPath using go-app-paths")

	// Test that ConfigPaths returns valid paths
	primary, err := ConfigPath()
	fmt.Println(primary)
	if err != nil {
		t.Fatalf("Failed to get config path: %s", err)
	}

	// Both paths should be non-empty
	if primary == "" {
		t.Error("Primary config path should not be empty")
	}

	// Path should be absolute or relative
	if !filepath.IsAbs(primary) && !strings.HasPrefix(primary, ".") {
		t.Errorf("Primary path should be absolute or relative, got: %s", primary)
	}

	// The path should contain "rulem"
	if !strings.Contains(primary, "rulem") {
		t.Errorf("Primary path should contain 'rulem', got: %s", primary)
	}

	// Primary should end with config.yaml
	if !strings.HasSuffix(primary, "config.yaml") {
		t.Errorf("Primary path should end with 'config.yaml', got: %s", primary)
	}

	t.Logf("Primary config path: %s", primary)
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
