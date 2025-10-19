package repository

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rulem/internal/logging"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
)

func TestGitSource_Prepare_InitialClone_Success(t *testing.T) {
	// Test that GitSource can successfully perform initial clone
	tempDir := t.TempDir()
	clonePath := filepath.Join(tempDir, "test-repo")
	remoteURL := "https://github.com/octocat/Hello-World.git"
	logger, _ := logging.NewTestLogger()

	gs := NewGitSource(remoteURL, nil, clonePath)
	gotPath, err := gs.Prepare(logger)

	if err != nil {
		t.Fatalf("Prepare() unexpected error for initial clone: %v", err)
	}

	// Should return the clone path
	if gotPath != clonePath {
		t.Errorf("Prepare() returned path = %s, want %s", gotPath, clonePath)
	}

	// Verify the repository was cloned by checking if it exists and is a git repo
	_, err = git.PlainOpen(clonePath)
	if err != nil {
		t.Errorf("Prepare() should have cloned repository at %s, got error: %v", clonePath, err)
	}

	// Directory should exist and contain git repository
	if _, err := os.Stat(clonePath); os.IsNotExist(err) {
		t.Errorf("Prepare() clone directory does not exist: %s", clonePath)
	}

	// Should be a valid git repository
	if _, err := gs.getGitRemoteURL(clonePath); err != nil {
		t.Errorf("Prepare() cloned directory is not a valid git repository: %v", err)
	}
}

func TestGitSource_Prepare_InitialClone_WithBranch(t *testing.T) {
	// Test that GitSource can clone with specific branch
	tempDir := t.TempDir()
	clonePath := filepath.Join(tempDir, "test-repo-branch")
	remoteURL := "https://github.com/octocat/Hello-World.git"
	branch := "master"
	logger, _ := logging.NewTestLogger()

	gs := NewGitSource(remoteURL, &branch, clonePath)
	gotPath, err := gs.Prepare(logger)

	if err != nil {
		t.Fatalf("Prepare() unexpected error for branch clone: %v", err)
	}

	if gotPath != clonePath {
		t.Errorf("Prepare() returned path = %s, want %s", gotPath, clonePath)
	}

	// Verify the repository was cloned
	_, err = git.PlainOpen(clonePath)
	if err != nil {
		t.Errorf("Prepare() should have cloned repository at %s, got error: %v", clonePath, err)
	}

	// Directory should exist and contain git repository
	if _, err := os.Stat(clonePath); os.IsNotExist(err) {
		t.Errorf("Prepare() clone directory does not exist: %s", clonePath)
	}
}

func TestGitSource_Prepare_FetchRefresh_Success(t *testing.T) {
	// Test that GitSource can successfully fetch updates from existing clone
	tempDir := t.TempDir()
	clonePath := filepath.Join(tempDir, "existing-repo")
	remoteURL := "https://github.com/octocat/Hello-World.git"
	logger, _ := logging.NewTestLogger()

	// First clone should succeed
	gs := NewGitSource(remoteURL, nil, clonePath)
	_, err := gs.Prepare(logger)
	if err != nil {
		t.Fatalf("First Prepare() failed: %v", err)
	}

	// Verify repository was cloned
	_, err = git.PlainOpen(clonePath)
	if err != nil {
		t.Fatalf("First Prepare() should have cloned repository at %s, got error: %v", clonePath, err)
	}

	// Second prepare should fetch updates
	_, err = gs.Prepare(logger)
	if err != nil {
		t.Fatalf("Second Prepare() failed: %v", err)
	}

	// Repository should still exist and be a git repo after second prepare
	_, err = git.PlainOpen(clonePath)
	if err != nil {
		t.Errorf("Second Prepare() should maintain git repository at %s, got error: %v", clonePath, err)
	}
}

func TestGitSource_Prepare_PATAuth_Required(t *testing.T) {
	// Test that GitSource handles PAT authentication correctly
	tempDir := t.TempDir()
	clonePath := filepath.Join(tempDir, "private-repo")
	// Use a hypothetical private repository URL
	remoteURL := "https://github.com/private/secure-repo.git"
	logger, _ := logging.NewTestLogger()

	gs := NewGitSource(remoteURL, nil, clonePath)

	// Without PAT configured, should fail with auth error
	_, err := gs.Prepare(logger)
	if err == nil {
		t.Fatalf("Prepare() should fail without PAT for private repo")
	}

	// Error should mention authentication
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "auth") && !strings.Contains(errStr, "token") && !strings.Contains(errStr, "credential") {
		t.Errorf("Prepare() error should mention authentication issue, got: %v", err)
	}

	// Error should guide user to Settings
	if !strings.Contains(errStr, "settings") {
		t.Errorf("Prepare() error should guide user to Settings, got: %v", err)
	}
}

func TestGitSource_Prepare_DirtyWorkingTree_Message(t *testing.T) {
	// Test that GitSource detects dirty working tree and provides guidance
	tempDir := t.TempDir()
	clonePath := filepath.Join(tempDir, "dirty-repo")
	remoteURL := "https://github.com/octocat/Hello-World.git"
	logger, _ := logging.NewTestLogger()

	// First, clone the repository
	gs := NewGitSource(remoteURL, nil, clonePath)
	_, err := gs.Prepare(logger)
	if err != nil {
		t.Fatalf("Initial clone failed: %v", err)
	}

	// Create a dirty file in the working tree
	testFile := filepath.Join(clonePath, "dirty-file.txt")
	if err := os.WriteFile(testFile, []byte("uncommitted changes"), 0644); err != nil {
		t.Fatalf("Failed to create dirty file: %v", err)
	}

	// Second prepare should succeed even with dirty state
	// (GitSource doesn't fail on dirty working tree, it just skips updates)
	_, err = gs.Prepare(logger)
	if err != nil {
		t.Fatalf("Prepare() with dirty tree failed: %v", err)
	}

	// Verify dirty state can be detected
	isDirty, err := CheckGithubRepositoryStatus(clonePath)
	if err != nil {
		t.Fatalf("CheckGithubRepositoryStatus() failed: %v", err)
	}
	if !isDirty {
		t.Errorf("CheckGithubRepositoryStatus() should detect dirty working tree")
	}

	// Dirty file should still exist (no data loss)
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Errorf("Prepare() should not delete dirty files")
	}
}

func TestGitSource_Prepare_InvalidURL(t *testing.T) {
	// Test that GitSource handles invalid remote URLs
	tempDir := t.TempDir()
	clonePath := filepath.Join(tempDir, "invalid-repo")
	logger, _ := logging.NewTestLogger()

	testCases := []struct {
		name      string
		remoteURL string
	}{
		{
			name:      "empty URL",
			remoteURL: "",
		},
		{
			name:      "malformed URL",
			remoteURL: "not-a-valid-url",
		},
		{
			name:      "non-existent repository",
			remoteURL: "https://github.com/nonexistent/nonexistent.git",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gs := NewGitSource(tc.remoteURL, nil, clonePath)
			_, err := gs.Prepare(logger)

			if err == nil {
				t.Errorf("Prepare() should fail for invalid URL: %s", tc.remoteURL)
			}

			// Should provide meaningful error message
			if err.Error() == "" {
				t.Errorf("Prepare() should provide meaningful error message")
			}
		})
	}
}

func TestGitSource_Prepare_DirectoryConflict(t *testing.T) {
	// Test that GitSource handles directory conflicts correctly
	tempDir := t.TempDir()
	clonePath := filepath.Join(tempDir, "conflict-repo")
	remoteURL := "https://github.com/octocat/Hello-World.git"
	logger, _ := logging.NewTestLogger()

	// Create a directory with different content
	if err := os.MkdirAll(clonePath, 0755); err != nil {
		t.Fatalf("Failed to create conflict directory: %v", err)
	}
	conflictFile := filepath.Join(clonePath, "existing-file.txt")
	if err := os.WriteFile(conflictFile, []byte("existing content"), 0644); err != nil {
		t.Fatalf("Failed to create conflict file: %v", err)
	}

	gs := NewGitSource(remoteURL, nil, clonePath)
	_, err := gs.Prepare(logger)

	if err == nil {
		t.Fatalf("Prepare() should fail for directory conflict")
	}

	// Should provide guidance about manual resolution
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "conflict") && !strings.Contains(errStr, "exists") && !strings.Contains(errStr, "resolve") {
		t.Errorf("Prepare() error should mention conflict/resolution, got: %v", err)
	}

	// Original file should still exist (no data loss)
	if _, err := os.Stat(conflictFile); os.IsNotExist(err) {
		t.Errorf("Prepare() should not delete existing files during conflict")
	}
}

func TestGitSource_Prepare_EmptyPath_Error(t *testing.T) {
	// Test that GitSource handles empty clone path
	remoteURL := "https://github.com/octocat/Hello-World.git"
	logger, _ := logging.NewTestLogger()

	gs := NewGitSource(remoteURL, nil, "")
	_, err := gs.Prepare(logger)

	if err == nil {
		t.Fatalf("Prepare() should fail for empty clone path")
	}

	// Should provide meaningful error about path
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "path") {
		t.Errorf("Prepare() error should mention path issue, got: %v", err)
	}
}

func TestGitSource_Prepare_PATExpired_Error(t *testing.T) {
	// Test that GitSource handles expired PAT with clear error message
	tempDir := t.TempDir()
	clonePath := filepath.Join(tempDir, "expired-pat-repo")
	remoteURL := "https://github.com/private/secure-repo.git"
	logger, _ := logging.NewTestLogger()

	// Set up credential manager with an invalid/expired token
	credMgr := NewCredentialManager()
	// This will fail validation, simulating an expired token scenario
	invalidToken := "expired_token_123"
	// Validation should fail, but we'll store it anyway to simulate an expired token
	// In real scenarios, validation would prevent storage, but this simulates a token that became invalid after storage
	_ = credMgr.StoreGitHubToken(invalidToken)

	gs := NewGitSource(remoteURL, nil, clonePath)
	_, err := gs.Prepare(logger)

	if err == nil {
		t.Fatalf("Prepare() should fail with expired/invalid PAT")
	}

	// Should provide guidance about updating token in Settings
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "token") && !strings.Contains(errStr, "expired") && !strings.Contains(errStr, "invalid") {
		t.Errorf("Prepare() error should mention token issue, got: %v", err)
	}

	if !strings.Contains(errStr, "settings") {
		t.Errorf("Prepare() error should guide to Settings, got: %v", err)
	}
}

func TestGetDefaultStorageDir(t *testing.T) {
	tests := []struct {
		name string
		want func(string) bool // validation function
	}{
		{
			name: "returns non-empty path",
			want: func(path string) bool {
				return path != ""
			},
		},
		{
			name: "returns absolute path",
			want: func(path string) bool {
				return filepath.IsAbs(path)
			},
		},
		{
			name: "contains rulem in path",
			want: func(path string) bool {
				return strings.Contains(path, "rulem")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetDefaultStorageDir()
			if !tt.want(got) {
				t.Errorf("GetDefaultStorageDir() = %v, validation failed", got)
			}
		})
	}
}

func TestParseGitURL(t *testing.T) {
	tests := []struct {
		name      string
		gitURL    string
		wantHost  string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		// Valid GitHub URLs
		{
			name:      "github ssh url with .git",
			gitURL:    "git@github.com:owner/repo.git",
			wantHost:  "github.com",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "github ssh url without .git",
			gitURL:    "git@github.com:owner/repo",
			wantHost:  "github.com",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "github https url with .git",
			gitURL:    "https://github.com/owner/repo.git",
			wantHost:  "github.com",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "github https url without .git",
			gitURL:    "https://github.com/owner/repo",
			wantHost:  "github.com",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "github http url",
			gitURL:    "http://github.com/owner/repo.git",
			wantHost:  "github.com",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		// Valid custom Git server URLs
		{
			name:      "custom git server ssh",
			gitURL:    "git@git.company.com:team/project.git",
			wantHost:  "git.company.com",
			wantOwner: "team",
			wantRepo:  "project",
			wantErr:   false,
		},
		{
			name:      "custom git server https",
			gitURL:    "https://git.company.com/team/project.git",
			wantHost:  "git.company.com",
			wantOwner: "team",
			wantRepo:  "project",
			wantErr:   false,
		},
		{
			name:      "gitlab ssh url",
			gitURL:    "git@gitlab.com:namespace/project.git",
			wantHost:  "gitlab.com",
			wantOwner: "namespace",
			wantRepo:  "project",
			wantErr:   false,
		},
		{
			name:      "bitbucket https url",
			gitURL:    "https://bitbucket.org/user/repository.git",
			wantHost:  "bitbucket.org",
			wantOwner: "user",
			wantRepo:  "repository",
			wantErr:   false,
		},
		// Edge cases with special characters
		{
			name:      "repo name with hyphens",
			gitURL:    "https://github.com/my-org/my-repo-name.git",
			wantHost:  "github.com",
			wantOwner: "my-org",
			wantRepo:  "my-repo-name",
			wantErr:   false,
		},
		{
			name:      "repo name with underscores",
			gitURL:    "git@github.com:my_org/my_repo.git",
			wantHost:  "github.com",
			wantOwner: "my_org",
			wantRepo:  "my_repo",
			wantErr:   false,
		},
		{
			name:      "repo name with dots",
			gitURL:    "https://github.com/user/repo.name.git",
			wantHost:  "github.com",
			wantOwner: "user",
			wantRepo:  "repo.name",
			wantErr:   false,
		},
		{
			name:      "repo name with numbers",
			gitURL:    "git@github.com:org123/repo456.git",
			wantHost:  "github.com",
			wantOwner: "org123",
			wantRepo:  "repo456",
			wantErr:   false,
		},
		// URLs with extra whitespace
		{
			name:      "url with leading whitespace",
			gitURL:    "  https://github.com/owner/repo.git",
			wantHost:  "github.com",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "url with trailing whitespace",
			gitURL:    "https://github.com/owner/repo.git  ",
			wantHost:  "github.com",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "url with surrounding whitespace",
			gitURL:    "  git@github.com:owner/repo.git  ",
			wantHost:  "github.com",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		// Invalid URLs
		{
			name:    "empty url",
			gitURL:  "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			gitURL:  "   ",
			wantErr: true,
		},
		{
			name:    "invalid url format",
			gitURL:  "not-a-url",
			wantErr: true,
		},
		{
			name:    "url missing owner and repo",
			gitURL:  "https://github.com/",
			wantErr: true,
		},
		{
			name:    "url missing repo",
			gitURL:  "https://github.com/owner",
			wantErr: true,
		},
		{
			name:    "url missing owner",
			gitURL:  "https://github.com//repo.git",
			wantErr: true,
		},
		{
			name:    "ssh url with wrong format",
			gitURL:  "git@github.com/owner/repo.git",
			wantErr: true,
		},
		{
			name:    "url without host",
			gitURL:  "https:///owner/repo.git",
			wantErr: true,
		},
		{
			name:    "malformed ssh url",
			gitURL:  "git@:owner/repo.git",
			wantErr: true,
		},
		{
			name:    "url with only slashes",
			gitURL:  "https://github.com///",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ParseGitURL(tt.gitURL)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseGitURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if info.Host != tt.wantHost {
					t.Errorf("ParseGitURL() Host = %v, want %v", info.Host, tt.wantHost)
				}
				if info.Owner != tt.wantOwner {
					t.Errorf("ParseGitURL() Owner = %v, want %v", info.Owner, tt.wantOwner)
				}
				if info.Repo != tt.wantRepo {
					t.Errorf("ParseGitURL() Repo = %v, want %v", info.Repo, tt.wantRepo)
				}
			}
		})
	}
}

func TestGitSource_validateCloneDirectory(t *testing.T) {
	// Create temporary directories for testing
	tempDir := t.TempDir()

	tests := []struct {
		name                string
		setup               func() (clonePath, expectedRemoteURL string)
		expectedStatus      DirectoryStatus
		expectedErrContains string
	}{
		{
			name: "directory doesn't exist",
			setup: func() (string, string) {
				nonExistentPath := filepath.Join(tempDir, "nonexistent")
				return nonExistentPath, "git@github.com:user/repo.git"
			},
			expectedStatus: DirectoryStatusEmpty,
		},
		{
			name: "empty directory",
			setup: func() (string, string) {
				emptyPath := filepath.Join(tempDir, "empty")
				err := os.MkdirAll(emptyPath, 0755)
				if err != nil {
					t.Fatalf("Failed to create empty directory: %v", err)
				}
				return emptyPath, "git@github.com:user/repo.git"
			},
			expectedStatus: DirectoryStatusEmpty,
		},
		{
			name: "path is a file, not directory",
			setup: func() (string, string) {
				filePath := filepath.Join(tempDir, "notadir")
				f, err := os.Create(filePath)
				if err != nil {
					t.Fatalf("Failed to create file: %v", err)
				}
				f.Close()
				return filePath, "git@github.com:user/repo.git"
			},
			expectedStatus:      DirectoryStatusError,
			expectedErrContains: "not a directory",
		},
		{
			name: "directory with non-git content",
			setup: func() (string, string) {
				nonGitPath := filepath.Join(tempDir, "nongit")
				err := os.MkdirAll(nonGitPath, 0755)
				if err != nil {
					t.Fatalf("Failed to create directory: %v", err)
				}
				// Add some non-git file
				f, err := os.Create(filepath.Join(nonGitPath, "somefile.txt"))
				if err != nil {
					t.Fatalf("Failed to create file: %v", err)
				}
				f.Close()
				return nonGitPath, "git@github.com:user/repo.git"
			},
			expectedStatus:      DirectoryStatusConflict,
			expectedErrContains: "non-git content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := GitSource{}
			clonePath, expectedRemoteURL := tt.setup()
			status, err := gs.validateCloneDirectory(clonePath, expectedRemoteURL)

			if status != tt.expectedStatus {
				t.Errorf("validateCloneDirectory() status = %v, want %v", status, tt.expectedStatus)
			}
			if tt.expectedErrContains != "" {
				if err == nil {
					t.Errorf("validateCloneDirectory() expected error containing %q but got none", tt.expectedErrContains)
				} else if !strings.Contains(err.Error(), tt.expectedErrContains) {
					t.Errorf("validateCloneDirectory() error = %v, want error containing %q", err, tt.expectedErrContains)
				}
			} else {
				if err != nil {
					t.Errorf("validateCloneDirectory() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestDirectoryStatus_String(t *testing.T) {
	tests := []struct {
		status DirectoryStatus
		want   string
	}{
		{DirectoryStatusEmpty, "empty or doesn't exist"},
		{DirectoryStatusSameRepo, "same git repository"},
		{DirectoryStatusDifferentRepo, "different git repository"},
		{DirectoryStatusConflict, "contains non-git content"},
		{DirectoryStatusError, "validation error"},
		{DirectoryStatus(999), "unknown status"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.status.String()
			if got != tt.want {
				t.Errorf("DirectoryStatus.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGitSource_normalizeGitURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "github ssh url",
			input: "git@github.com:owner/repo.git",
			want:  "github.com/owner/repo",
		},
		{
			name:  "github https url",
			input: "https://github.com/owner/repo.git",
			want:  "github.com/owner/repo",
		},
		{
			name:  "github https without .git",
			input: "https://github.com/owner/repo",
			want:  "github.com/owner/repo",
		},
		{
			name:  "custom server ssh",
			input: "git@git.company.com:team/project.git",
			want:  "git.company.com/team/project",
		},
		{
			name:  "custom server https",
			input: "https://git.company.com/team/project.git",
			want:  "git.company.com/team/project",
		},
		{
			name:  "http url",
			input: "http://git.example.com/user/repo",
			want:  "git.example.com/user/repo",
		},
		{
			name:  "already normalized",
			input: "github.com/owner/repo",
			want:  "github.com/owner/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := GitSource{}
			got := gs.normalizeGitURL(tt.input)
			if got != tt.want {
				t.Errorf("normalizeGitURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGitSource_getGitRemoteURL(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name           string
		setup          func() string
		expectedURL    string
		expectedErrMsg string
	}{
		{
			name: "valid git repository with origin remote",
			setup: func() string {
				// Create a real Git repository using go-git
				repoPath := filepath.Join(tempDir, "valid-repo")

				// Initialize repository
				repo, err := git.PlainInit(repoPath, false)
				if err != nil {
					t.Fatalf("Failed to init repository: %v", err)
				}

				// Add origin remote
				_, err = repo.CreateRemote(&config.RemoteConfig{
					Name: "origin",
					URLs: []string{"git@github.com:user/test-repo.git"},
				})
				if err != nil {
					t.Fatalf("Failed to create remote: %v", err)
				}

				return repoPath
			},
			expectedURL: "git@github.com:user/test-repo.git",
		},
		{
			name: "valid git repository with https origin",
			setup: func() string {
				repoPath := filepath.Join(tempDir, "https-repo")

				repo, err := git.PlainInit(repoPath, false)
				if err != nil {
					t.Fatalf("Failed to init repository: %v", err)
				}

				_, err = repo.CreateRemote(&config.RemoteConfig{
					Name: "origin",
					URLs: []string{"https://github.com/user/test-repo.git"},
				})
				if err != nil {
					t.Fatalf("Failed to create remote: %v", err)
				}

				return repoPath
			},
			expectedURL: "https://github.com/user/test-repo.git",
		},
		{
			name: "git repository without origin remote",
			setup: func() string {
				repoPath := filepath.Join(tempDir, "no-origin")

				_, err := git.PlainInit(repoPath, false)
				if err != nil {
					t.Fatalf("Failed to init repository: %v", err)
				}

				return repoPath
			},
			expectedErrMsg: "cannot get origin remote",
		},
		{
			name: "git repository with multiple URLs",
			setup: func() string {
				repoPath := filepath.Join(tempDir, "multi-urls")

				repo, err := git.PlainInit(repoPath, false)
				if err != nil {
					t.Fatalf("Failed to init repository: %v", err)
				}

				// Create remote with multiple URLs (first one should be returned)
				_, err = repo.CreateRemote(&config.RemoteConfig{
					Name: "origin",
					URLs: []string{
						"git@github.com:user/primary.git",
						"git@backup.com:user/backup.git",
					},
				})
				if err != nil {
					t.Fatalf("Failed to create remote: %v", err)
				}

				return repoPath
			},
			expectedURL: "git@github.com:user/primary.git",
		},
		{
			name: "not a git repository",
			setup: func() string {
				nonGitPath := filepath.Join(tempDir, "not-git")
				err := os.MkdirAll(nonGitPath, 0755)
				if err != nil {
					t.Fatalf("Failed to create directory: %v", err)
				}
				return nonGitPath
			},
			expectedErrMsg: "directory is not a git repository",
		},
		{
			name: "nonexistent directory",
			setup: func() string {
				return filepath.Join(tempDir, "nonexistent")
			},
			expectedErrMsg: "directory is not a git repository",
		},
		{
			name: "bare repository with origin",
			setup: func() string {
				bareRepoPath := filepath.Join(tempDir, "bare-repo")

				// Initialize bare repository
				repo, err := git.PlainInit(bareRepoPath, true)
				if err != nil {
					t.Fatalf("Failed to init bare repository: %v", err)
				}

				// Add origin remote
				_, err = repo.CreateRemote(&config.RemoteConfig{
					Name: "origin",
					URLs: []string{"git@example.com:org/bare-repo.git"},
				})
				if err != nil {
					t.Fatalf("Failed to create remote: %v", err)
				}

				return bareRepoPath
			},
			expectedURL: "git@example.com:org/bare-repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := GitSource{}
			repoPath := tt.setup()

			url, err := gs.getGitRemoteURL(repoPath)

			if tt.expectedErrMsg != "" {
				if err == nil {
					t.Errorf("getGitRemoteURL() expected error containing %q but got none", tt.expectedErrMsg)
				} else if !strings.Contains(err.Error(), tt.expectedErrMsg) {
					t.Errorf("getGitRemoteURL() error = %v, want error containing %q", err, tt.expectedErrMsg)
				}
				if url != "" {
					t.Errorf("getGitRemoteURL() expected empty URL on error but got %q", url)
				}
			} else {
				if err != nil {
					t.Errorf("getGitRemoteURL() unexpected error: %v", err)
				}
				if url != tt.expectedURL {
					t.Errorf("getGitRemoteURL() = %q, want %q", url, tt.expectedURL)
				}
			}
		})
	}
}

func TestGetDefaultStorageDirConsistency(t *testing.T) {
	// This test verifies that GetDefaultStorageDir returns consistent results
	first := GetDefaultStorageDir()
	second := GetDefaultStorageDir()

	if first != second {
		t.Errorf("GetDefaultStorageDir should return consistent results, got %v and %v", first, second)
	}
	if !filepath.IsAbs(first) {
		t.Errorf("default storage dir should be absolute, got %v", first)
	}
	if !strings.Contains(first, "rulem") {
		t.Errorf("default storage dir should contain 'rulem', got %v", first)
	}
}

func TestCheckRepositoryStatus(t *testing.T) {
	logger, _ := logging.NewTestLogger()

	t.Run("NonExistentRepository", func(t *testing.T) {
		// Test with non-existent path
		_, err := CheckGithubRepositoryStatus("/nonexistent/path")
		if err == nil {
			t.Error("Expected error for non-existent repository")
		}
	})

	t.Run("NotAGitRepository", func(t *testing.T) {
		// Test with a directory that's not a git repo
		tempDir := t.TempDir()
		_, err := CheckGithubRepositoryStatus(tempDir)
		if err == nil {
			t.Error("Expected error for non-git directory")
		}
	})

	t.Run("CleanRepository", func(t *testing.T) {
		// Clone a real repository for testing
		tempDir := t.TempDir()
		clonePath := filepath.Join(tempDir, "clean-repo")
		remoteURL := "https://github.com/octocat/Hello-World.git"

		gs := NewGitSource(remoteURL, nil, clonePath)
		_, err := gs.Prepare(logger)
		if err != nil {
			t.Skipf("Cannot clone repository for testing (network issue?): %v", err)
		}

		// Check status - should be clean
		isDirty, err := CheckGithubRepositoryStatus(clonePath)
		if err != nil {
			t.Fatalf("Failed to check repository status: %v", err)
		}

		if isDirty {
			t.Error("Expected clean repository, but got dirty status")
		}
	})

	t.Run("DirtyRepository", func(t *testing.T) {
		// Clone a repository and modify it
		tempDir := t.TempDir()
		clonePath := filepath.Join(tempDir, "dirty-repo")
		remoteURL := "https://github.com/octocat/Hello-World.git"

		gs := NewGitSource(remoteURL, nil, clonePath)
		_, err := gs.Prepare(logger)
		if err != nil {
			t.Skipf("Cannot clone repository for testing (network issue?): %v", err)
		}

		// Modify a file to make it dirty
		testFile := filepath.Join(clonePath, "README")
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read test file: %v", err)
		}

		// Append some content
		newContent := append(content, []byte("\nTest modification")...)
		if err := os.WriteFile(testFile, newContent, 0644); err != nil {
			t.Fatalf("Failed to modify test file: %v", err)
		}

		// Check status - should be dirty
		isDirty, err := CheckGithubRepositoryStatus(clonePath)
		if err != nil {
			t.Fatalf("Failed to check repository status: %v", err)
		}

		if !isDirty {
			t.Error("Expected dirty repository, but got clean status")
		}
	})
}
