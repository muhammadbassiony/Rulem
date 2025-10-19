package repository

import (
	"rulem/internal/logging"
	"strings"
	"testing"
)

// TestRepositoryEntry_IsRemote tests the IsRemote method
func TestRepositoryEntry_IsRemote(t *testing.T) {
	tests := []struct {
		name     string
		repo     RepositoryEntry
		expected bool
	}{
		{
			name: "local repository",
			repo: RepositoryEntry{
				Type: RepositoryTypeLocal,
				Path: "/home/user/rules",
			},
			expected: false,
		},
		{
			name: "github repository",
			repo: RepositoryEntry{
				Type:      RepositoryTypeGitHub,
				Path:      "/home/user/cached-rules",
				RemoteURL: stringPtr("https://github.com/user/rules.git"),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.repo.IsRemote()
			if got != tt.expected {
				t.Errorf("IsRemote() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestPrepareRepository_LocalSource tests preparing a local repository
func TestPrepareRepository_LocalSource(t *testing.T) {
	tempDir := t.TempDir()
	logger, _ := logging.NewTestLogger()

	repo := RepositoryEntry{
		ID:        "test-repo-123",
		Name:      "Test Repo",
		Type:      RepositoryTypeLocal,
		Path:      tempDir,
		CreatedAt: 1234567890,
	}

	localPath, syncInfo, err := PrepareRepository(repo, logger)
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

// TestPrepareRepository_InvalidLocalPath tests error handling for invalid paths
func TestPrepareRepository_InvalidLocalPath(t *testing.T) {
	logger, _ := logging.NewTestLogger()

	repo := RepositoryEntry{
		ID:        "test-repo-123",
		Name:      "Test Repo",
		Type:      RepositoryTypeLocal,
		Path:      "/nonexistent/directory/that/should/not/exist",
		CreatedAt: 1234567890,
	}

	_, _, err := PrepareRepository(repo, logger)
	if err == nil {
		t.Fatal("Expected error for invalid local path")
	}

	// Should get an error about the directory not existing
	if !strings.Contains(strings.ToLower(err.Error()), "does not exist") {
		t.Errorf("Expected 'does not exist' error, got: %v", err)
	}
}

// TestPrepareRepository_WithNilLogger tests operation with nil logger
func TestPrepareRepository_WithNilLogger(t *testing.T) {
	tempDir := t.TempDir()

	repo := RepositoryEntry{
		ID:        "test-repo-123",
		Name:      "Test Repo",
		Type:      RepositoryTypeLocal,
		Path:      tempDir,
		CreatedAt: 1234567890,
	}

	// Should work with nil logger (logging calls are guarded)
	localPath, syncInfo, err := PrepareRepository(repo, nil)
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

// TestValidateRepositoryEntry_ValidEntries tests validation of valid repository entries
func TestValidateRepositoryEntry_ValidEntries(t *testing.T) {
	tests := []struct {
		name string
		repo RepositoryEntry
	}{
		{
			name: "valid local repository",
			repo: RepositoryEntry{
				ID:        "local-repo-1234567890",
				Name:      "Local Repository",
				Type:      RepositoryTypeLocal,
				Path:      "/home/user/rules",
				CreatedAt: 1234567890,
			},
		},
		{
			name: "valid github repository",
			repo: RepositoryEntry{
				ID:        "github-repo-1234567890",
				Name:      "GitHub Repository",
				Type:      RepositoryTypeGitHub,
				Path:      "/home/user/.local/share/rulem/repo",
				RemoteURL: stringPtr("https://github.com/user/repo.git"),
				CreatedAt: 1234567890,
			},
		},
		{
			name: "github repository with branch",
			repo: RepositoryEntry{
				ID:        "github-repo-1234567890",
				Name:      "GitHub Repository",
				Type:      RepositoryTypeGitHub,
				Path:      "/home/user/.local/share/rulem/repo",
				RemoteURL: stringPtr("https://github.com/user/repo.git"),
				Branch:    stringPtr("main"),
				CreatedAt: 1234567890,
			},
		},
		{
			name: "github repository with last sync time",
			repo: RepositoryEntry{
				ID:           "github-repo-1234567890",
				Name:         "GitHub Repository",
				Type:         RepositoryTypeGitHub,
				Path:         "/home/user/.local/share/rulem/repo",
				RemoteURL:    stringPtr("https://github.com/user/repo.git"),
				LastSyncTime: int64Ptr(1234567890),
				CreatedAt:    1234567890,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRepositoryEntry(tt.repo)
			if err != nil {
				t.Errorf("expected no error for valid repository, got: %v", err)
			}
		})
	}
}

// TestValidateRepositoryEntry_InvalidID tests ID validation
func TestValidateRepositoryEntry_InvalidID(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		expectErr string
	}{
		{
			name:      "empty ID",
			id:        "",
			expectErr: "cannot be empty",
		},
		{
			name:      "ID without dash",
			id:        "noDashHere",
			expectErr: "invalid repository ID format",
		},
		{
			name:      "ID with non-numeric timestamp",
			id:        "repo-name-abc",
			expectErr: "timestamp must be numeric",
		},
		{
			name:      "ID ending with dash",
			id:        "repo-name-",
			expectErr: "missing timestamp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := RepositoryEntry{
				ID:        tt.id,
				Name:      "Test Repo",
				Type:      RepositoryTypeLocal,
				Path:      "/tmp/test",
				CreatedAt: 1234567890,
			}

			err := ValidateRepositoryEntry(repo)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("expected error containing %q, got: %v", tt.expectErr, err)
			}
		})
	}
}

// TestValidateRepositoryEntry_InvalidName tests name validation
func TestValidateRepositoryEntry_InvalidName(t *testing.T) {
	tests := []struct {
		name      string
		repoName  string
		expectErr string
	}{
		{
			name:      "empty name",
			repoName:  "",
			expectErr: "cannot be empty",
		},
		{
			name:      "whitespace-only name",
			repoName:  "   ",
			expectErr: "cannot be empty",
		},
		{
			name:      "name too long",
			repoName:  strings.Repeat("a", 101),
			expectErr: "too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := RepositoryEntry{
				ID:        "test-repo-1234567890",
				Name:      tt.repoName,
				Type:      RepositoryTypeLocal,
				Path:      "/tmp/test",
				CreatedAt: 1234567890,
			}

			err := ValidateRepositoryEntry(repo)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("expected error containing %q, got: %v", tt.expectErr, err)
			}
		})
	}
}

// TestValidateRepositoryEntry_InvalidType tests type validation
func TestValidateRepositoryEntry_InvalidType(t *testing.T) {
	tests := []struct {
		name      string
		repoType  RepositoryType
		expectErr string
	}{
		{
			name:      "empty type",
			repoType:  "",
			expectErr: "invalid repository type",
		},
		{
			name:      "invalid type",
			repoType:  "gitlab",
			expectErr: "invalid repository type",
		},
		{
			name:      "uppercase type",
			repoType:  "LOCAL",
			expectErr: "invalid repository type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := RepositoryEntry{
				ID:        "test-repo-1234567890",
				Name:      "Test Repo",
				Type:      tt.repoType,
				Path:      "/tmp/test",
				CreatedAt: 1234567890,
			}

			err := ValidateRepositoryEntry(repo)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("expected error containing %q, got: %v", tt.expectErr, err)
			}
		})
	}
}

// TestValidateRepositoryEntry_InvalidCreatedAt tests CreatedAt validation
func TestValidateRepositoryEntry_InvalidCreatedAt(t *testing.T) {
	tests := []struct {
		name      string
		createdAt int64
	}{
		{
			name:      "zero timestamp",
			createdAt: 0,
		},
		{
			name:      "negative timestamp",
			createdAt: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := RepositoryEntry{
				ID:        "test-repo-1234567890",
				Name:      "Test Repo",
				Type:      RepositoryTypeLocal,
				Path:      "/tmp/test",
				CreatedAt: tt.createdAt,
			}

			err := ValidateRepositoryEntry(repo)
			if err == nil {
				t.Fatalf("expected error for invalid CreatedAt, got nil")
			}
			if !strings.Contains(err.Error(), "created_at") {
				t.Errorf("expected error about created_at, got: %v", err)
			}
		})
	}
}

// TestValidateRepositoryEntry_InvalidPath tests path validation
func TestValidateRepositoryEntry_InvalidPath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		expectErr string
	}{
		{
			name:      "empty path",
			path:      "",
			expectErr: "path cannot be empty",
		},
		{
			name:      "whitespace-only path",
			path:      "   ",
			expectErr: "path cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := RepositoryEntry{
				ID:        "test-repo-1234567890",
				Name:      "Test Repo",
				Type:      RepositoryTypeLocal,
				Path:      tt.path,
				CreatedAt: 1234567890,
			}

			err := ValidateRepositoryEntry(repo)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("expected error containing %q, got: %v", tt.expectErr, err)
			}
		})
	}
}

// TestValidateRepositoryEntry_GitHubWithoutRemoteURL tests GitHub validation
func TestValidateRepositoryEntry_GitHubWithoutRemoteURL(t *testing.T) {
	tests := []struct {
		name      string
		remoteURL *string
		expectErr string
	}{
		{
			name:      "nil remote URL",
			remoteURL: nil,
			expectErr: "must have a remote URL",
		},
		{
			name:      "empty remote URL",
			remoteURL: stringPtr(""),
			expectErr: "must have a remote URL",
		},
		{
			name:      "whitespace-only remote URL",
			remoteURL: stringPtr("   "),
			expectErr: "must have a remote URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := RepositoryEntry{
				ID:        "test-repo-1234567890",
				Name:      "Test Repo",
				Type:      RepositoryTypeGitHub,
				Path:      "/tmp/test",
				RemoteURL: tt.remoteURL,
				CreatedAt: 1234567890,
			}

			err := ValidateRepositoryEntry(repo)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("expected error containing %q, got: %v", tt.expectErr, err)
			}
		})
	}
}

// TestValidateRepositoryEntry_GitHubWithEmptyBranch tests branch validation
func TestValidateRepositoryEntry_GitHubWithEmptyBranch(t *testing.T) {
	repo := RepositoryEntry{
		ID:        "test-repo-1234567890",
		Name:      "Test Repo",
		Type:      RepositoryTypeGitHub,
		Path:      "/tmp/test",
		RemoteURL: stringPtr("https://github.com/user/repo.git"),
		Branch:    stringPtr(""),
		CreatedAt: 1234567890,
	}

	err := ValidateRepositoryEntry(repo)
	if err == nil {
		t.Fatalf("expected error for empty branch, got nil")
	}
	if !strings.Contains(err.Error(), "branch") {
		t.Errorf("expected error about branch, got: %v", err)
	}
}

// TestValidateRepositoryEntry_GitHubWithInvalidLastSyncTime tests LastSyncTime validation
func TestValidateRepositoryEntry_GitHubWithInvalidLastSyncTime(t *testing.T) {
	tests := []struct {
		name         string
		lastSyncTime *int64
	}{
		{
			name:         "zero timestamp",
			lastSyncTime: int64Ptr(0),
		},
		{
			name:         "negative timestamp",
			lastSyncTime: int64Ptr(-1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := RepositoryEntry{
				ID:           "test-repo-1234567890",
				Name:         "Test Repo",
				Type:         RepositoryTypeGitHub,
				Path:         "/tmp/test",
				RemoteURL:    stringPtr("https://github.com/user/repo.git"),
				LastSyncTime: tt.lastSyncTime,
				CreatedAt:    1234567890,
			}

			err := ValidateRepositoryEntry(repo)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), "last_sync_time") {
				t.Errorf("expected error about last_sync_time, got: %v", err)
			}
		})
	}
}

// TestValidateRepositoryEntry_LocalWithGitHubFields tests that local repos don't have GitHub fields
func TestValidateRepositoryEntry_LocalWithGitHubFields(t *testing.T) {
	tests := []struct {
		name      string
		repo      RepositoryEntry
		expectErr string
	}{
		{
			name: "local repo with remote URL",
			repo: RepositoryEntry{
				ID:        "test-repo-1234567890",
				Name:      "Test Repo",
				Type:      RepositoryTypeLocal,
				Path:      "/tmp/test",
				RemoteURL: stringPtr("https://github.com/user/repo.git"),
				CreatedAt: 1234567890,
			},
			expectErr: "should not have a remote URL",
		},
		{
			name: "local repo with branch",
			repo: RepositoryEntry{
				ID:        "test-repo-1234567890",
				Name:      "Test Repo",
				Type:      RepositoryTypeLocal,
				Path:      "/tmp/test",
				Branch:    stringPtr("main"),
				CreatedAt: 1234567890,
			},
			expectErr: "should not have a branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRepositoryEntry(tt.repo)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("expected error containing %q, got: %v", tt.expectErr, err)
			}
		})
	}
}

// TestRepositoryType_Constants tests that type constants are defined correctly
func TestRepositoryType_Constants(t *testing.T) {
	if RepositoryTypeLocal != "local" {
		t.Errorf("RepositoryTypeLocal should be 'local', got: %q", RepositoryTypeLocal)
	}
	if RepositoryTypeGitHub != "github" {
		t.Errorf("RepositoryTypeGitHub should be 'github', got: %q", RepositoryTypeGitHub)
	}
}

// TestNewLocalSource tests local source creation
func TestNewLocalSource(t *testing.T) {
	path := "/home/user/rules"

	source := NewLocalSource(path)

	if source.Path != path {
		t.Errorf("Expected Path to be %s, got %s", path, source.Path)
	}
}

// TestNewGitSource tests git source creation
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

// TestNewGitSource_NilBranch tests git source creation with nil branch
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

// Helper functions

func stringPtr(s string) *string {
	return &s
}

func int64Ptr(i int64) *int64 {
	return &i
}
