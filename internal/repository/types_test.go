package repository

import (
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

// TestRepositoryEntry_IsLocal tests the IsLocal method
func TestRepositoryEntry_IsLocal(t *testing.T) {
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
			expected: true,
		},
		{
			name: "github repository",
			repo: RepositoryEntry{
				Type:      RepositoryTypeGitHub,
				Path:      "/home/user/cached-rules",
				RemoteURL: stringPtr("https://github.com/user/rules.git"),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.repo.IsLocal()
			if got != tt.expected {
				t.Errorf("IsLocal() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestRepositoryEntry_GetRemoteURL tests the GetRemoteURL method
func TestRepositoryEntry_GetRemoteURL(t *testing.T) {
	tests := []struct {
		name     string
		repo     RepositoryEntry
		expected string
	}{
		{
			name: "github repository with URL",
			repo: RepositoryEntry{
				Type:      RepositoryTypeGitHub,
				RemoteURL: stringPtr("https://github.com/user/repo.git"),
			},
			expected: "https://github.com/user/repo.git",
		},
		{
			name: "local repository without URL",
			repo: RepositoryEntry{
				Type: RepositoryTypeLocal,
			},
			expected: "",
		},
		{
			name: "repository with nil URL",
			repo: RepositoryEntry{
				Type:      RepositoryTypeGitHub,
				RemoteURL: nil,
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.repo.GetRemoteURL()
			if got != tt.expected {
				t.Errorf("GetRemoteURL() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestRepositoryEntry_GetBranch tests the GetBranch method
func TestRepositoryEntry_GetBranch(t *testing.T) {
	tests := []struct {
		name     string
		repo     RepositoryEntry
		expected string
	}{
		{
			name: "repository with branch",
			repo: RepositoryEntry{
				Branch: stringPtr("main"),
			},
			expected: "main",
		},
		{
			name: "repository with nil branch",
			repo: RepositoryEntry{
				Branch: nil,
			},
			expected: "",
		},
		{
			name: "repository with empty branch",
			repo: RepositoryEntry{
				Branch: stringPtr(""),
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.repo.GetBranch()
			if got != tt.expected {
				t.Errorf("GetBranch() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestRepositoryEntry_String tests the String method
func TestRepositoryEntry_String(t *testing.T) {
	tests := []struct {
		name     string
		repo     RepositoryEntry
		contains []string
	}{
		{
			name: "local repository",
			repo: RepositoryEntry{
				ID:   "local-123",
				Name: "Local Rules",
				Type: RepositoryTypeLocal,
				Path: "/home/user/rules",
			},
			contains: []string{"local-123", "Local Rules", "local", "/home/user/rules"},
		},
		{
			name: "github repository",
			repo: RepositoryEntry{
				ID:        "github-456",
				Name:      "GitHub Rules",
				Type:      RepositoryTypeGitHub,
				Path:      "/cache/rules",
				RemoteURL: stringPtr("https://github.com/user/repo.git"),
			},
			contains: []string{"github-456", "GitHub Rules", "github", "https://github.com/user/repo.git"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.repo.String()
			for _, expected := range tt.contains {
				if !strings.Contains(got, expected) {
					t.Errorf("String() = %q, want it to contain %q", got, expected)
				}
			}
		})
	}
}

// TestRepositoryType_String tests the String method
func TestRepositoryType_String(t *testing.T) {
	tests := []struct {
		name     string
		typ      RepositoryType
		expected string
	}{
		{
			name:     "local type",
			typ:      RepositoryTypeLocal,
			expected: "local",
		},
		{
			name:     "github type",
			typ:      RepositoryTypeGitHub,
			expected: "github",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.typ.String()
			if got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestRepositoryType_IsValid tests the IsValid method
func TestRepositoryType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		typ      RepositoryType
		expected bool
	}{
		{
			name:     "valid local type",
			typ:      RepositoryTypeLocal,
			expected: true,
		},
		{
			name:     "valid github type",
			typ:      RepositoryTypeGitHub,
			expected: true,
		},
		{
			name:     "invalid type",
			typ:      "gitlab",
			expected: false,
		},
		{
			name:     "empty type",
			typ:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.typ.IsValid()
			if got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
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

// TestPreparedRepository_ID tests the ID convenience method
func TestPreparedRepository_ID(t *testing.T) {
	pr := PreparedRepository{
		Entry: RepositoryEntry{
			ID: "test-123",
		},
	}

	if pr.ID() != "test-123" {
		t.Errorf("ID() = %q, want %q", pr.ID(), "test-123")
	}
}

// TestPreparedRepository_Name tests the Name convenience method
func TestPreparedRepository_Name(t *testing.T) {
	pr := PreparedRepository{
		Entry: RepositoryEntry{
			Name: "Test Repository",
		},
	}

	if pr.Name() != "Test Repository" {
		t.Errorf("Name() = %q, want %q", pr.Name(), "Test Repository")
	}
}

// TestPreparedRepository_Type tests the Type convenience method
func TestPreparedRepository_Type(t *testing.T) {
	pr := PreparedRepository{
		Entry: RepositoryEntry{
			Type: RepositoryTypeLocal,
		},
	}

	if pr.Type() != RepositoryTypeLocal {
		t.Errorf("Type() = %q, want %q", pr.Type(), RepositoryTypeLocal)
	}
}

// TestPreparedRepository_IsRemote tests the IsRemote convenience method
func TestPreparedRepository_IsRemote(t *testing.T) {
	tests := []struct {
		name     string
		pr       PreparedRepository
		expected bool
	}{
		{
			name: "local repository",
			pr: PreparedRepository{
				Entry: RepositoryEntry{
					Type: RepositoryTypeLocal,
				},
			},
			expected: false,
		},
		{
			name: "github repository",
			pr: PreparedRepository{
				Entry: RepositoryEntry{
					Type: RepositoryTypeGitHub,
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pr.IsRemote()
			if got != tt.expected {
				t.Errorf("IsRemote() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestPreparedRepository_WasSynced tests the WasSynced method
func TestPreparedRepository_WasSynced(t *testing.T) {
	tests := []struct {
		name     string
		pr       PreparedRepository
		expected bool
	}{
		{
			name: "successful sync",
			pr: PreparedRepository{
				SyncResult: RepositorySyncResult{
					Status: SyncStatusSuccess,
				},
			},
			expected: true,
		},
		{
			name: "failed sync",
			pr: PreparedRepository{
				SyncResult: RepositorySyncResult{
					Status: SyncStatusFailed,
				},
			},
			expected: false,
		},
		{
			name: "skipped sync",
			pr: PreparedRepository{
				SyncResult: RepositorySyncResult{
					Status: SyncStatusSkipped,
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pr.WasSynced()
			if got != tt.expected {
				t.Errorf("WasSynced() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestPreparedRepository_HasError tests the HasError method
func TestPreparedRepository_HasError(t *testing.T) {
	tests := []struct {
		name     string
		pr       PreparedRepository
		expected bool
	}{
		{
			name: "failed sync",
			pr: PreparedRepository{
				SyncResult: RepositorySyncResult{
					Status: SyncStatusFailed,
				},
			},
			expected: true,
		},
		{
			name: "successful sync",
			pr: PreparedRepository{
				SyncResult: RepositorySyncResult{
					Status: SyncStatusSuccess,
				},
			},
			expected: false,
		},
		{
			name: "skipped sync",
			pr: PreparedRepository{
				SyncResult: RepositorySyncResult{
					Status: SyncStatusSkipped,
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pr.HasError()
			if got != tt.expected {
				t.Errorf("HasError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestPreparedRepository_WasSkipped tests the WasSkipped method
func TestPreparedRepository_WasSkipped(t *testing.T) {
	tests := []struct {
		name     string
		pr       PreparedRepository
		expected bool
	}{
		{
			name: "skipped sync",
			pr: PreparedRepository{
				SyncResult: RepositorySyncResult{
					Status: SyncStatusSkipped,
				},
			},
			expected: true,
		},
		{
			name: "successful sync",
			pr: PreparedRepository{
				SyncResult: RepositorySyncResult{
					Status: SyncStatusSuccess,
				},
			},
			expected: false,
		},
		{
			name: "failed sync",
			pr: PreparedRepository{
				SyncResult: RepositorySyncResult{
					Status: SyncStatusFailed,
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pr.WasSkipped()
			if got != tt.expected {
				t.Errorf("WasSkipped() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestPreparedRepository_String tests the String method
func TestPreparedRepository_String(t *testing.T) {
	pr := PreparedRepository{
		Entry: RepositoryEntry{
			ID:   "test-123",
			Name: "Test Repo",
		},
		LocalPath: "/tmp/test",
		SyncResult: RepositorySyncResult{
			Status: SyncStatusSuccess,
		},
	}

	str := pr.String()
	expectedSubstrings := []string{"test-123", "Test Repo", "/tmp/test", "Success"}

	for _, expected := range expectedSubstrings {
		if !strings.Contains(str, expected) {
			t.Errorf("String() = %q, want it to contain %q", str, expected)
		}
	}
}

// Helper functions

func stringPtr(s string) *string {
	return &s
}

func int64Ptr(i int64) *int64 {
	return &i
}
