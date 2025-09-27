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
	primary, err := Path()
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

func TestConfigCentralScenarios(t *testing.T) {
	t.Log("Testing Config persistence and migration for central repo settings")

	tempDir := t.TempDir()

	cases := []struct {
		name    string
		prepare func(path string) error
		assert  func(t *testing.T, cfg *Config)
	}{
		{
			name: "default config round-trip",
			prepare: func(path string) error {
				cfg := DefaultConfig()
				cfg.InitTime = time.Now().Unix()
				return cfg.SaveTo(path)
			},
			assert: func(t *testing.T, cfg *Config) {
				if cfg.Central.Path == "" {
					t.Error("expected central path to be set")
				}
				if cfg.Central.RemoteURL != nil {
					t.Errorf("expected remote URL to be nil, got %v", *cfg.Central.RemoteURL)
				}
				if cfg.Central.Branch != nil {
					t.Errorf("expected branch to be nil, got %v", *cfg.Central.Branch)
				}
				if cfg.Central.LastSyncTime != nil && *cfg.Central.LastSyncTime == 0 {
					t.Error("last sync time should be nil or non-zero when set")
				}
			},
		},
		{
			name: "remote settings persistence",
			prepare: func(path string) error {
				branch := "main"
				remote := "git@github.com:owner/repo.git"
				lastSync := time.Now().Unix()
				cfg := Config{
					Version: "2.0",
					Central: CentralRepositoryConfig{
						Path:         "/tmp/rules/cache",
						RemoteURL:    &remote,
						Branch:       &branch,
						LastSyncTime: &lastSync,
					},
				}
				return cfg.SaveTo(path)
			},
			assert: func(t *testing.T, cfg *Config) {
				if cfg.Central.Path != "/tmp/rules/cache" {
					t.Errorf("expected central path /tmp/rules/cache, got %s", cfg.Central.Path)
				}
				if cfg.Central.RemoteURL == nil || *cfg.Central.RemoteURL != "git@github.com:owner/repo.git" {
					t.Errorf("expected remote URL to persist, got %+v", cfg.Central.RemoteURL)
				}
				if cfg.Central.Branch == nil || *cfg.Central.Branch != "main" {
					t.Errorf("expected branch to persist, got %+v", cfg.Central.Branch)
				}
				if cfg.Central.LastSyncTime == nil || *cfg.Central.LastSyncTime == 0 {
					t.Errorf("expected last sync time to persist, got %+v", cfg.Central.LastSyncTime)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			configPath := filepath.Join(tempDir, tc.name+".yaml")
			if err := tc.prepare(configPath); err != nil {
				t.Fatalf("failed to prepare config file: %v", err)
			}

			cfg, err := LoadFrom(configPath)
			if err != nil {
				t.Fatalf("failed to load config: %v", err)
			}

			tc.assert(t, cfg)
		})
	}
}

func TestConfigInitTime(t *testing.T) {
	t.Log("Testing Config InitTime on Save")

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	config := Config{
		Version: "1.0",
		Central: CentralRepositoryConfig{
			Path: "/test",
		},
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

	if config.Central.Path == "" {
		t.Error("Default config should have a central path")
	}

	if config.InitTime != 0 {
		t.Error("Default config InitTime should be 0 (will be set on save)")
	}
}

func TestConfigPathEnvironmentOverride(t *testing.T) {
	t.Log("Testing ConfigPath environment variable override")

	// Save original environment
	originalPath := os.Getenv("RULEM_CONFIG_PATH")
	defer func() {
		if originalPath == "" {
			os.Unsetenv("RULEM_CONFIG_PATH")
		} else {
			os.Setenv("RULEM_CONFIG_PATH", originalPath)
		}
	}()

	// Test with environment variable set
	testPath := "/tmp/test-rulem-config.yaml"
	err := os.Setenv("RULEM_CONFIG_PATH", testPath)
	if err != nil {
		t.Fatalf("Failed to set environment variable: %v", err)
	}

	configPath, err := Path()
	if err != nil {
		t.Fatalf("Failed to get config path: %v", err)
	}

	if configPath != testPath {
		t.Errorf("Expected config path %s, got %s", testPath, configPath)
	}

	// Test with environment variable unset
	os.Unsetenv("RULEM_CONFIG_PATH")
	configPath, err = Path()
	if err != nil {
		t.Fatalf("Failed to get config path: %v", err)
	}

	if configPath == testPath {
		t.Error("Config path should not match test path when environment variable is unset")
	}

	if !strings.Contains(configPath, "rulem") {
		t.Errorf("Config path should contain 'rulem', got: %s", configPath)
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
		if err := os.WriteFile(invalidFile, []byte("invalid: yaml: content: ["), 0644); err != nil {
			t.Fatalf("Failed to write invalid YAML file: %v", err)
		}

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
