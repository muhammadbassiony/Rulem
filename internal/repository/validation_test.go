package repository

import (
	"strings"
	"testing"
)

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
		{
			name: "local repo with last sync time",
			repo: RepositoryEntry{
				ID:           "test-repo-1234567890",
				Name:         "Test Repo",
				Type:         RepositoryTypeLocal,
				Path:         "/tmp/test",
				LastSyncTime: int64Ptr(1234567890),
				CreatedAt:    1234567890,
			},
			expectErr: "should not have a last_sync_time",
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

// TestValidateRepositoryName tests the ValidateRepositoryName function
func TestValidateRepositoryName(t *testing.T) {
	tests := []struct {
		name      string
		repoName  string
		wantErr   bool
		expectErr string
	}{
		{
			name:     "valid name",
			repoName: "My Rules",
			wantErr:  false,
		},
		{
			name:      "empty name",
			repoName:  "",
			wantErr:   true,
			expectErr: "cannot be empty",
		},
		{
			name:      "whitespace only",
			repoName:  "   ",
			wantErr:   true,
			expectErr: "cannot be empty",
		},
		{
			name:      "name too long",
			repoName:  strings.Repeat("a", 101),
			wantErr:   true,
			expectErr: "too long",
		},
		{
			name:      "control characters",
			repoName:  "Test\x00Repo",
			wantErr:   true,
			expectErr: "invalid control characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRepositoryName(tt.repoName)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.expectErr != "" && !strings.Contains(err.Error(), tt.expectErr) {
					t.Errorf("expected error containing %q, got: %v", tt.expectErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}
		})
	}
}

// TestValidateRepositoryPath tests the ValidateRepositoryPath function
func TestValidateRepositoryPath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantErr   bool
		expectErr string
	}{
		{
			name:    "valid path",
			path:    "/home/user/rules",
			wantErr: false,
		},
		{
			name:      "empty path",
			path:      "",
			wantErr:   true,
			expectErr: "cannot be empty",
		},
		{
			name:      "whitespace only",
			path:      "   ",
			wantErr:   true,
			expectErr: "cannot be empty",
		},
		{
			name:      "null bytes",
			path:      "/home/user\x00/rules",
			wantErr:   true,
			expectErr: "null bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRepositoryPath(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.expectErr != "" && !strings.Contains(err.Error(), tt.expectErr) {
					t.Errorf("expected error containing %q, got: %v", tt.expectErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}
		})
	}
}

// Note: Tests for ValidateAllRepositories are in multi_test.go
// as they test multi-repository orchestration functionality
