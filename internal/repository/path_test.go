package repository

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
)

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

func TestDeriveClonePath(t *testing.T) {
	tests := []struct {
		name         string
		remoteURL    string
		wantRepoName string
		wantErr      bool
	}{
		{
			name:         "github ssh url",
			remoteURL:    "git@github.com:owner/repo.git",
			wantRepoName: "repo",
			wantErr:      false,
		},
		{
			name:         "github https url",
			remoteURL:    "https://github.com/owner/repo.git",
			wantRepoName: "repo",
			wantErr:      false,
		},
		{
			name:         "github url without .git suffix",
			remoteURL:    "https://github.com/owner/repo",
			wantRepoName: "repo",
			wantErr:      false,
		},
		{
			name:         "custom git server",
			remoteURL:    "git@custom.com:namespace/project.git",
			wantRepoName: "project",
			wantErr:      false,
		},
		{
			name:      "empty remote url",
			remoteURL: "",
			wantErr:   true,
		},
		{
			name:      "invalid remote url",
			remoteURL: "not-a-url",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DeriveClonePath(tt.remoteURL)
			if tt.wantErr {
				if err == nil {
					t.Errorf("DeriveClonePath() expected error but got none")
				}
				if got != "" {
					t.Errorf("DeriveClonePath() expected empty string but got %v", got)
				}
			} else {
				if err != nil {
					t.Errorf("DeriveClonePath() unexpected error: %v", err)
				}
				// Verify the path ends with the expected repo name
				expectedPath := filepath.Join(GetDefaultStorageDir(), tt.wantRepoName)
				if got != expectedPath {
					t.Errorf("DeriveClonePath() = %v, want %v", got, expectedPath)
				}
				// Verify it's an absolute path
				if !filepath.IsAbs(got) {
					t.Errorf("DeriveClonePath() should return absolute path, got %v", got)
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

func TestValidateCloneDirectory(t *testing.T) {
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
			clonePath, expectedRemoteURL := tt.setup()
			status, err := ValidateCloneDirectory(clonePath, expectedRemoteURL)

			if status != tt.expectedStatus {
				t.Errorf("ValidateCloneDirectory() status = %v, want %v", status, tt.expectedStatus)
			}
			if tt.expectedErrContains != "" {
				if err == nil {
					t.Errorf("ValidateCloneDirectory() expected error containing %q but got none", tt.expectedErrContains)
				} else if !strings.Contains(err.Error(), tt.expectedErrContains) {
					t.Errorf("ValidateCloneDirectory() error = %v, want error containing %q", err, tt.expectedErrContains)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateCloneDirectory() unexpected error: %v", err)
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

func TestNormalizeGitURL(t *testing.T) {
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
			got := normalizeGitURL(tt.input)
			if got != tt.want {
				t.Errorf("normalizeGitURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetGitRemoteURL(t *testing.T) {
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
			repoPath := tt.setup()

			url, err := getGitRemoteURL(repoPath)

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
