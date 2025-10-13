package setupmenu

import (
	"os"
	"path/filepath"
	"rulem/internal/config"
	"testing"
)

// SetTestConfigPath sets the RULEM_CONFIG_PATH environment variable to a temporary directory
// and returns a cleanup function. This prevents tests from overwriting the user's real config.
func SetTestConfigPath(t *testing.T) (configPath string, cleanup func()) {
	t.Helper()

	// Create a temporary directory for the test config
	tempDir, err := os.MkdirTemp("", "rulem-test-config-*")
	if err != nil {
		t.Fatalf("Failed to create temp config dir: %v", err)
	}

	// Set the environment variable
	configPath = filepath.Join(tempDir, "config.yaml")
	originalPath := os.Getenv("RULEM_CONFIG_PATH")
	err = os.Setenv("RULEM_CONFIG_PATH", configPath)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to set RULEM_CONFIG_PATH: %v", err)
	}

	// Return cleanup function
	cleanup = func() {
		// Restore original environment variable
		if originalPath == "" {
			os.Unsetenv("RULEM_CONFIG_PATH")
		} else {
			os.Setenv("RULEM_CONFIG_PATH", originalPath)
		}
		// Clean up temp directory
		os.RemoveAll(tempDir)
	}

	return configPath, cleanup
}

// CreateTempStorageDir creates a temporary directory for test storage
// and ensures it gets cleaned up after the test
func CreateTempStorageDir(t *testing.T, prefix string) string {
	t.Helper()

	// Get user home directory for a more realistic test path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home dir: %v", err)
	}

	tempDir, err := os.MkdirTemp(homeDir, prefix)
	if err != nil {
		t.Fatalf("Failed to create temp storage dir: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	return tempDir
}

// FileExists checks if a file or directory exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// LoadTestConfig loads the config from the test config path
func LoadTestConfig(t *testing.T) (*config.Config, error) {
	t.Helper()
	return config.Load()
}
