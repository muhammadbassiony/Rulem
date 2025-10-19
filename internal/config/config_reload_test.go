package config

import (
	"os"
	"path/filepath"
	"rulem/internal/repository"
	"testing"
	"time"
)

func TestReloadConfig(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Set environment variable to use test config path
	oldConfigPath := os.Getenv("RULEM_CONFIG_PATH")
	os.Setenv("RULEM_CONFIG_PATH", configPath)
	defer func() {
		if oldConfigPath != "" {
			os.Setenv("RULEM_CONFIG_PATH", oldConfigPath)
		} else {
			os.Unsetenv("RULEM_CONFIG_PATH")
		}
	}()

	// Get user home directory for valid paths
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("Cannot get home directory: %v", err)
	}

	// Create initial config with valid path under home
	initialCentralPath := filepath.Join(homeDir, "test-rulem-initial")
	initialConfig := Config{
		Version:  "1.0",
		InitTime: time.Now().Unix(),
		Repositories: []repository.RepositoryEntry{
			{
				ID:        "test-repo-123456",
				Name:      "Test Repo",
				Type:      repository.RepositoryTypeLocal,
				CreatedAt: time.Now().Unix(),
				Path:      initialCentralPath,
			},
		},
	}

	// Save initial config
	err = initialConfig.SaveTo(configPath)
	if err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	// Test ReloadConfig command
	reloadCmd := ReloadConfig()
	if reloadCmd == nil {
		t.Fatal("ReloadConfig returned nil command")
	}

	// Execute the command to get the message
	msg := reloadCmd()

	// Verify message type
	reloadMsg, ok := msg.(ReloadConfigMsg)
	if !ok {
		t.Fatalf("Expected ReloadConfigMsg, got %T", msg)
	}

	// Verify no error occurred
	if reloadMsg.Error != nil {
		t.Fatalf("ReloadConfig returned error: %v", reloadMsg.Error)
	}

	// Verify config was loaded correctly
	if reloadMsg.Config == nil {
		t.Fatal("ReloadConfig returned nil config")
	}

	if len(reloadMsg.Config.Repositories) == 0 {
		t.Fatal("Expected at least one repository")
	}

	if reloadMsg.Config.Repositories[0].Path != initialCentralPath {
		t.Errorf("Expected central path '%s', got '%s'", initialCentralPath, reloadMsg.Config.Repositories[0].Path)
	}

	if reloadMsg.Config.Version != "1.0" {
		t.Errorf("Expected Version '1.0', got '%s'", reloadMsg.Config.Version)
	}

	// Test reload after config modification
	// Modify config on disk
	modifiedCentralPath := filepath.Join(homeDir, "test-rulem-modified")
	modifiedConfig := Config{
		Version:  "1.1",
		InitTime: initialConfig.InitTime,
		Repositories: []repository.RepositoryEntry{
			{
				ID:        "test-repo-123456",
				Name:      "Test Repo Modified",
				Type:      repository.RepositoryTypeLocal,
				CreatedAt: time.Now().Unix(),
				Path:      modifiedCentralPath,
			},
		},
	}

	err = modifiedConfig.SaveTo(configPath)
	if err != nil {
		t.Fatalf("Failed to save modified config: %v", err)
	}

	// Reload again
	reloadCmd2 := ReloadConfig()
	msg2 := reloadCmd2()
	reloadMsg2 := msg2.(ReloadConfigMsg)

	// Verify changes were detected
	if reloadMsg2.Error != nil {
		t.Fatalf("ReloadConfig returned error on second load: %v", reloadMsg2.Error)
	}

	if len(reloadMsg2.Config.Repositories) == 0 {
		t.Fatal("Expected at least one repository in reloaded config")
	}

	if reloadMsg2.Config.Repositories[0].Path != modifiedCentralPath {
		t.Errorf("Expected modified central path '%s', got '%s'", modifiedCentralPath, reloadMsg2.Config.Repositories[0].Path)
	}

	if reloadMsg2.Config.Version != "1.1" {
		t.Errorf("Expected modified Version '1.1', got '%s'", reloadMsg2.Config.Version)
	}
}

func TestReloadConfigError(t *testing.T) {
	// Set non-existent config path
	tempDir := t.TempDir()
	nonExistentPath := filepath.Join(tempDir, "nonexistent", "config.yaml")

	oldConfigPath := os.Getenv("RULEM_CONFIG_PATH")
	os.Setenv("RULEM_CONFIG_PATH", nonExistentPath)
	defer func() {
		if oldConfigPath != "" {
			os.Setenv("RULEM_CONFIG_PATH", oldConfigPath)
		} else {
			os.Unsetenv("RULEM_CONFIG_PATH")
		}
	}()

	// Test ReloadConfig with non-existent file
	reloadCmd := ReloadConfig()
	msg := reloadCmd()

	reloadMsg, ok := msg.(ReloadConfigMsg)
	if !ok {
		t.Fatalf("Expected ReloadConfigMsg, got %T", msg)
	}

	// Verify error is reported
	if reloadMsg.Error == nil {
		t.Fatal("Expected error when config file doesn't exist, got nil")
	}

	// Verify config is nil on error
	if reloadMsg.Config != nil {
		t.Fatal("Expected nil config on error, got non-nil")
	}
}

func TestReloadConfigIntegration(t *testing.T) {
	// This test simulates the TUI workflow where config is updated and then reloaded

	// Create temporary directory for test config
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Set environment variable to use test config path
	oldConfigPath := os.Getenv("RULEM_CONFIG_PATH")
	os.Setenv("RULEM_CONFIG_PATH", configPath)
	defer func() {
		if oldConfigPath != "" {
			os.Setenv("RULEM_CONFIG_PATH", oldConfigPath)
		} else {
			os.Unsetenv("RULEM_CONFIG_PATH")
		}
	}()

	// Get user home directory for valid paths
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("Cannot get home directory: %v", err)
	}

	// Create initial config (simulating main model startup)
	originalCentralPath := filepath.Join(homeDir, "test-rulem-original")
	initialConfig := DefaultConfig()
	initialConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:        "test-repo-123456",
			Name:      "Original Repo",
			Type:      repository.RepositoryTypeLocal,
			CreatedAt: time.Now().Unix(),
			Path:      originalCentralPath,
		},
	}
	err = initialConfig.Save()
	if err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	// Load config (simulating main model loading config)
	loadedConfig, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify loaded config
	if len(loadedConfig.Repositories) == 0 {
		t.Fatal("Expected at least one repository")
	}
	if loadedConfig.Repositories[0].Path != originalCentralPath {
		t.Errorf("Expected loaded central path '%s', got '%s'", originalCentralPath, loadedConfig.Repositories[0].Path)
	}

	// Update config (simulating settings menu update)
	updatedCentralPath := filepath.Join(homeDir, "test-rulem-updated")
	loadedConfig.Repositories[0].Path = updatedCentralPath
	loadedConfig.Repositories[0].Name = "Updated Repo"
	err = loadedConfig.Save()
	if err != nil {
		t.Fatalf("Failed to save updated config: %v", err)
	}

	// Reload config (simulating main model receiving ReloadConfigMsg)
	reloadCmd := ReloadConfig()
	msg := reloadCmd()
	reloadMsg := msg.(ReloadConfigMsg)

	if reloadMsg.Error != nil {
		t.Fatalf("ReloadConfig failed: %v", reloadMsg.Error)
	}

	// Verify the reloaded config has the updated values
	if len(reloadMsg.Config.Repositories) == 0 {
		t.Fatal("Expected at least one repository in reloaded config")
	}
	if reloadMsg.Config.Repositories[0].Path != updatedCentralPath {
		t.Errorf("Expected reloaded central path '%s', got '%s'", updatedCentralPath, reloadMsg.Config.Repositories[0].Path)
	}
	if reloadMsg.Config.Repositories[0].Name != "Updated Repo" {
		t.Errorf("Expected reloaded repository name 'Updated Repo', got '%s'", reloadMsg.Config.Repositories[0].Name)
	}
}
