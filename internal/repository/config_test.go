package repository

import (
	"rulem/internal/logging"
	"strings"
	"testing"
)

func TestCentralRepositoryConfig_IsRemote(t *testing.T) {
	tests := []struct {
		name     string
		config   CentralRepositoryConfig
		expected bool
	}{
		{
			name: "local repository (nil remote URL)",
			config: CentralRepositoryConfig{
				Path:      "/home/user/rules",
				RemoteURL: nil,
			},
			expected: false,
		},
		{
			name: "local repository (empty remote URL)",
			config: CentralRepositoryConfig{
				Path:      "/home/user/rules",
				RemoteURL: stringPtr(""),
			},
			expected: false,
		},
		{
			name: "remote repository (GitHub)",
			config: CentralRepositoryConfig{
				Path:      "/home/user/cached-rules",
				RemoteURL: stringPtr("https://github.com/user/rules.git"),
			},
			expected: true,
		},
		{
			name: "remote repository (custom Git server)",
			config: CentralRepositoryConfig{
				Path:      "/home/user/cached-rules",
				RemoteURL: stringPtr("git@git.company.com:team/rules.git"),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.IsRemote()
			if got != tt.expected {
				t.Errorf("IsRemote() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPrepareRepository_LocalSource(t *testing.T) {
	tempDir := t.TempDir()
	logger, _ := logging.NewTestLogger()

	cfg := CentralRepositoryConfig{
		Path:      tempDir,
		RemoteURL: nil, // Local mode
	}

	localPath, syncInfo, err := PrepareRepository(cfg, logger)
	if err != nil {
		t.Fatalf("PrepareRepository failed: %v", err)
	}

	// Should return the absolute path
	if localPath != tempDir {
		t.Errorf("Expected path '%s', got '%s'", tempDir, localPath)
	}

	// LocalSource should not set sync flags
	if syncInfo.Cloned || syncInfo.Updated || syncInfo.Dirty {
		t.Errorf("LocalSource should not set sync flags, got: %+v", syncInfo)
	}

	// Should have a user-friendly message
	if !strings.Contains(strings.ToLower(syncInfo.Message), "local") {
		t.Errorf("Expected message about local repository, got: %q", syncInfo.Message)
	}
}

func TestPrepareRepository_InvalidLocalPath(t *testing.T) {
	logger, _ := logging.NewTestLogger()

	cfg := CentralRepositoryConfig{
		Path:      "/nonexistent/directory/that/should/not/exist",
		RemoteURL: nil, // Local mode
	}

	_, _, err := PrepareRepository(cfg, logger)
	if err == nil {
		t.Fatal("Expected error for invalid local path")
	}

	// Should get an error about the directory not existing
	if !strings.Contains(strings.ToLower(err.Error()), "does not exist") {
		t.Errorf("Expected 'does not exist' error, got: %v", err)
	}
}

func TestPrepareRepository_GitMode_WithoutAuth(t *testing.T) {
	tempDir := t.TempDir()
	logger, _ := logging.NewTestLogger()

	// Use a public repository that should be accessible without authentication
	remoteURL := "https://github.com/octocat/Hello-World.git"
	cfg := CentralRepositoryConfig{
		Path:      tempDir,
		RemoteURL: &remoteURL, // Git mode
	}

	// Should use GitSource and succeed for public repository
	localPath, syncInfo, err := PrepareRepository(cfg, logger)
	if err != nil {
		t.Fatalf("PrepareRepository failed: %v", err)
	}

	// Should return the configured path
	if localPath != tempDir {
		t.Errorf("Expected path '%s', got '%s'", tempDir, localPath)
	}

	// GitSource should set clone flag for initial clone
	if !syncInfo.Cloned {
		t.Errorf("Expected Cloned=true for initial GitSource clone, got %+v", syncInfo)
	}

	// Should have GitSource success message
	if !strings.Contains(strings.ToLower(syncInfo.Message), "clone") {
		t.Errorf("Expected clone message, got: %q", syncInfo.Message)
	}
}

func TestPrepareRepository_WithNilLogger(t *testing.T) {
	tempDir := t.TempDir()

	cfg := CentralRepositoryConfig{
		Path:      tempDir,
		RemoteURL: nil, // Local mode
	}

	// Should work with nil logger (logging calls are guarded)
	localPath, syncInfo, err := PrepareRepository(cfg, nil)
	if err != nil {
		t.Fatalf("PrepareRepository with nil logger failed: %v", err)
	}

	if localPath != tempDir {
		t.Errorf("Expected path '%s', got '%s'", tempDir, localPath)
	}

	if syncInfo.Message == "" {
		t.Error("Expected sync info message even with nil logger")
	}
}

func TestPrepareRepository_EmptyPath(t *testing.T) {
	logger, _ := logging.NewTestLogger()

	cfg := CentralRepositoryConfig{
		Path:      "", // Empty path
		RemoteURL: nil,
	}

	_, _, err := PrepareRepository(cfg, logger)
	if err == nil {
		t.Fatal("Expected error for empty path")
	}

	// Should get validation error about empty path
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "empty") && !strings.Contains(errStr, "cannot be empty") {
		t.Errorf("Expected error about empty path, got: %v", err)
	}
}

func TestCentralRepositoryConfig_WithAllFields(t *testing.T) {
	remoteURL := "https://github.com/user/repo.git"
	branch := "main"
	lastSync := int64(1234567890)

	cfg := CentralRepositoryConfig{
		Path:         "/home/user/cached-repo",
		RemoteURL:    &remoteURL,
		Branch:       &branch,
		LastSyncTime: &lastSync,
	}

	// Should be detected as remote
	if !cfg.IsRemote() {
		t.Error("Expected IsRemote() to return true for config with remote URL")
	}

	// Check that all fields are preserved in the config
	if cfg.Path != "/home/user/cached-repo" {
		t.Errorf("Path not preserved: got %s", cfg.Path)
	}
	if cfg.RemoteURL == nil || *cfg.RemoteURL != remoteURL {
		t.Errorf("RemoteURL not preserved: got %v", cfg.RemoteURL)
	}
	if cfg.Branch == nil || *cfg.Branch != branch {
		t.Errorf("Branch not preserved: got %v", cfg.Branch)
	}
	if cfg.LastSyncTime == nil || *cfg.LastSyncTime != lastSync {
		t.Errorf("LastSyncTime not preserved: got %v", cfg.LastSyncTime)
	}
}

func TestNewLocalSource(t *testing.T) {
	path := "/home/user/rules"

	source := NewLocalSource(path)

	if source.Path != path {
		t.Errorf("Expected Path to be %s, got %s", path, source.Path)
	}
}

func TestNewGitSource(t *testing.T) {
	remoteURL := "https://github.com/user/repo.git"
	branch := "main"
	localPath := "/tmp/cached-repo"

	source := NewGitSource(remoteURL, &branch, localPath)

	if source.RemoteURL != remoteURL {
		t.Errorf("Expected RemoteURL to be %s, got %s", remoteURL, source.RemoteURL)
	}

	if source.Branch == nil || *source.Branch != branch {
		t.Errorf("Expected Branch to be %s, got %v", branch, source.Branch)
	}

	if source.Path != localPath {
		t.Errorf("Expected Path to be %s, got %s", localPath, source.Path)
	}
}

func TestNewGitSource_NilBranch(t *testing.T) {
	remoteURL := "https://github.com/user/repo.git"
	localPath := "/tmp/cached-repo"

	source := NewGitSource(remoteURL, nil, localPath)

	if source.RemoteURL != remoteURL {
		t.Errorf("Expected RemoteURL to be %s, got %s", remoteURL, source.RemoteURL)
	}

	if source.Branch != nil {
		t.Errorf("Expected Branch to be nil, got %v", source.Branch)
	}

	if source.Path != localPath {
		t.Errorf("Expected Path to be %s, got %s", localPath, source.Path)
	}
}

func TestGitSource_Prepare_RequiresAuth(t *testing.T) {
	source := NewGitSource("https://github.com/user/repo.git", nil, "/tmp/test")
	logger, _ := logging.NewTestLogger()

	_, _, err := source.Prepare(logger)
	if err == nil {
		t.Error("GitSource.Prepare should require authentication")
	}

	// Should mention authentication requirement
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "auth") && !strings.Contains(errStr, "token") {
		t.Errorf("Expected error to mention authentication, got: %v", err)
	}
}

// Helper function to create a string pointer
func stringPtr(s string) *string {
	return &s
}
