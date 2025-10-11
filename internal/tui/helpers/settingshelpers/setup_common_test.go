package settingshelpers

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
)

func TestGetRepositoryTypeOptions(t *testing.T) {
	options := GetRepositoryTypeOptions()

	if len(options) != 2 {
		t.Errorf("Expected 2 repository type options, got %d", len(options))
	}

	// Check local option
	if options[0].Type != "local" {
		t.Errorf("Expected first option to be 'local', got %s", options[0].Type)
	}
	if options[0].Title == "" {
		t.Error("Local option title should not be empty")
	}
	if options[0].Desc == "" {
		t.Error("Local option description should not be empty")
	}

	// Check GitHub option
	if options[1].Type != "github" {
		t.Errorf("Expected second option to be 'github', got %s", options[1].Type)
	}
	if options[1].Title == "" {
		t.Error("GitHub option title should not be empty")
	}
	if options[1].Desc == "" {
		t.Error("GitHub option description should not be empty")
	}
}

func TestValidateGitHubURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid HTTPS URL with .git",
			url:     "https://github.com/user/repo.git",
			wantErr: false,
		},
		{
			name:    "valid HTTPS URL without .git",
			url:     "https://github.com/user/repo",
			wantErr: false,
		},
		{
			name:    "valid HTTP URL",
			url:     "http://github.com/user/repo.git",
			wantErr: false,
		},
		{
			name:    "valid SSH URL",
			url:     "git@github.com:user/repo.git",
			wantErr: false,
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			url:     "   ",
			wantErr: true,
		},
		{
			name:    "no protocol",
			url:     "github.com/user/repo",
			wantErr: true,
		},
		{
			name:    "HTTPS without path",
			url:     "https://github.com",
			wantErr: true,
		},
		{
			name:    "HTTPS with only slash",
			url:     "https://github.com/",
			wantErr: true,
		},
		{
			name:    "SSH without colon",
			url:     "git@github.com/user/repo",
			wantErr: true,
		},
		{
			name:    "SSH without path",
			url:     "git@github.com:",
			wantErr: true,
		},
		{
			name:    "valid URL with subdomain",
			url:     "https://gitlab.example.com/user/repo.git",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGitHubURL(tt.url)

			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestValidateBranchName(t *testing.T) {
	tests := []struct {
		name    string
		branch  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid branch name",
			branch:  "main",
			wantErr: false,
		},
		{
			name:    "valid branch with hyphen",
			branch:  "feature-branch",
			wantErr: false,
		},
		{
			name:    "valid branch with slash",
			branch:  "feature/my-feature",
			wantErr: false,
		},
		{
			name:    "valid branch with numbers",
			branch:  "release-1.2.3",
			wantErr: false,
		},
		{
			name:    "empty branch (allowed)",
			branch:  "",
			wantErr: false,
		},
		{
			name:    "whitespace only (treated as empty)",
			branch:  "   ",
			wantErr: false,
		},
		{
			name:    "branch with spaces",
			branch:  "my branch",
			wantErr: true,
			errMsg:  "cannot contain spaces",
		},
		{
			name:    "branch starting with slash",
			branch:  "/feature",
			wantErr: true,
			errMsg:  "cannot start or end with /",
		},
		{
			name:    "branch ending with slash",
			branch:  "feature/",
			wantErr: true,
			errMsg:  "cannot start or end with /",
		},
		{
			name:    "branch with consecutive dots",
			branch:  "feature..fix",
			wantErr: true,
			errMsg:  "consecutive dots",
		},
		{
			name:    "branch with tilde",
			branch:  "feature~1",
			wantErr: true,
			errMsg:  "cannot contain '~'",
		},
		{
			name:    "branch with caret",
			branch:  "feature^",
			wantErr: true,
			errMsg:  "cannot contain '^'",
		},
		{
			name:    "branch with colon",
			branch:  "feature:test",
			wantErr: true,
			errMsg:  "cannot contain ':'",
		},
		{
			name:    "branch with question mark",
			branch:  "feature?",
			wantErr: true,
			errMsg:  "cannot contain '?'",
		},
		{
			name:    "branch with asterisk",
			branch:  "feature*",
			wantErr: true,
			errMsg:  "cannot contain '*'",
		},
		{
			name:    "branch with bracket",
			branch:  "feature[test]",
			wantErr: true,
			errMsg:  "cannot contain '['",
		},
		{
			name:    "just a dot",
			branch:  ".",
			wantErr: true,
			errMsg:  "invalid branch name",
		},
		{
			name:    "ending with .lock",
			branch:  "feature.lock",
			wantErr: true,
			errMsg:  "invalid branch name",
		},
		{
			name:    "valid underscore",
			branch:  "feature_test",
			wantErr: false,
		},
		{
			name:    "valid single dot",
			branch:  "v1.0",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBranchName(tt.branch)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errMsg)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestDeriveClonePath(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantName string // Expected repository name in path
	}{
		{
			name:     "HTTPS URL with .git",
			url:      "https://github.com/user/my-repo.git",
			wantName: "my-repo",
		},
		{
			name:     "HTTPS URL without .git",
			url:      "https://github.com/user/my-repo",
			wantName: "my-repo",
		},
		{
			name:     "SSH URL",
			url:      "git@github.com:user/my-repo.git",
			wantName: "my-repo",
		},
		{
			name:     "URL with hyphens",
			url:      "https://github.com/user/my-cool-repo.git",
			wantName: "my-cool-repo",
		},
		{
			name:     "URL with underscores",
			url:      "https://github.com/user/my_repo.git",
			wantName: "my_repo",
		},
		{
			name:     "empty URL (fallback to default dir)",
			url:      "",
			wantName: "rulem", // Should return default storage dir which contains "rulem"
		},
		{
			name:     "invalid URL (fallback to default dir)",
			url:      "not-a-valid-url",
			wantName: "rulem", // Should return default storage dir
		},
		{
			name:     "URL with numbers",
			url:      "https://github.com/user/repo-v2.git",
			wantName: "repo-v2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := DeriveClonePath(tt.url)

			// Check that path is not empty
			if path == "" {
				t.Error("DeriveClonePath returned empty path")
			}

			// Check that path contains the expected name
			if !strings.Contains(path, tt.wantName) {
				t.Errorf("Expected path to contain '%s', got: %s", tt.wantName, path)
			}
		})
	}
}

func TestFormatPATDisplay(t *testing.T) {
	tests := []struct {
		name string
		pat  string
		want string
	}{
		{
			name: "typical PAT",
			pat:  "ghp_1234567890abcdef1234567890abcdef12345678",
			want: "ghp_••••••••••••••••••••••••••••••••••••5678",
		},
		{
			name: "short PAT",
			pat:  "short",
			want: "•••••",
		},
		{
			name: "very short PAT",
			pat:  "abc",
			want: "•••",
		},
		{
			name: "empty PAT",
			pat:  "",
			want: "",
		},
		{
			name: "PAT with spaces (trimmed)",
			pat:  "  ghp_1234567890  ",
			want: "ghp_••••••7890",
		},
		{
			name: "minimum length for partial display",
			pat:  "123456789",
			want: "1234•6789",
		},
		{
			name: "fine-grained PAT",
			pat:  "github_pat_11ABCDEFGHIJKLMNOPQRSTUVWXYZ",
			want: "gith•••••••••••••••••••••••••••••••WXYZ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatPATDisplay(tt.pat)

			if got != tt.want {
				t.Errorf("FormatPATDisplay() = %q, want %q", got, tt.want)
			}

			// Verify masked characters are present (if PAT is long enough)
			trimmedPAT := strings.TrimSpace(tt.pat)
			if len(trimmedPAT) > 8 && !strings.Contains(got, "•") {
				t.Error("Expected masked PAT to contain bullet character '•'")
			}
		})
	}
}

func TestGetRepositoryTypeName(t *testing.T) {
	tests := []struct {
		name     string
		isRemote bool
		want     string
	}{
		{
			name:     "local repository",
			isRemote: false,
			want:     "Local Directory",
		},
		{
			name:     "remote repository",
			isRemote: true,
			want:     "GitHub Repository",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetRepositoryTypeName(tt.isRemote)
			if got != tt.want {
				t.Errorf("GetRepositoryTypeName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetManualCleanupWarning(t *testing.T) {
	tests := []struct {
		name       string
		fromGitHub bool
		oldPath    string
		newPath    string
		wantText   []string // Text that should be in the warning
	}{
		{
			name:       "GitHub to Local",
			fromGitHub: true,
			oldPath:    "/home/user/.local/share/rulem",
			newPath:    "/home/user/local-rules",
			wantText: []string{
				"Manual Cleanup Required",
				"/home/user/.local/share/rulem",
				"NOT be deleted automatically",
				"rm -rf",
				"/home/user/local-rules",
			},
		},
		{
			name:       "Local to GitHub",
			fromGitHub: false,
			oldPath:    "/home/user/local-rules",
			newPath:    "/home/user/.local/share/rulem",
			wantText: []string{
				"Note",
				"/home/user/local-rules",
				"remain unchanged",
				"NOT be deleted automatically",
				"/home/user/.local/share/rulem",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetManualCleanupWarning(tt.fromGitHub, tt.oldPath, tt.newPath)

			if got == "" {
				t.Error("GetManualCleanupWarning returned empty string")
			}

			for _, want := range tt.wantText {
				if !strings.Contains(got, want) {
					t.Errorf("Warning should contain %q, got:\n%s", want, got)
				}
			}
		})
	}
}

func TestGetDirtyStateWarning(t *testing.T) {
	repoPath := "/home/user/.local/share/rulem"
	warning := GetDirtyStateWarning(repoPath)

	expectedTexts := []string{
		"Uncommitted Changes",
		repoPath,
		"git add",
		"git commit",
		"git push",
		"git reset --hard HEAD",
		"NOT automatically clean",
		"manually",
	}

	for _, expected := range expectedTexts {
		if !strings.Contains(warning, expected) {
			t.Errorf("Warning should contain %q, got:\n%s", expected, warning)
		}
	}
}

func TestResetTextInputForState(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("old value")
	ti.Placeholder = "old placeholder"

	cmd := ResetTextInputForState(&ti, "new value", "new placeholder", textinput.EchoNormal)

	// Check that text input was reset properly
	if ti.Value() != "new value" {
		t.Errorf("Expected value 'new value', got %q", ti.Value())
	}

	if ti.Placeholder != "new placeholder" {
		t.Errorf("Expected placeholder 'new placeholder', got %q", ti.Placeholder)
	}

	if ti.EchoMode != textinput.EchoNormal {
		t.Errorf("Expected EchoMode Normal, got %v", ti.EchoMode)
	}

	// Check that it returns a command
	if cmd == nil {
		t.Error("Expected ResetTextInputForState to return a command")
	}
}

func TestResetTextInputForState_PasswordMode(t *testing.T) {
	ti := textinput.New()

	cmd := ResetTextInputForState(&ti, "secret", "Enter password", textinput.EchoPassword)

	if ti.EchoMode != textinput.EchoPassword {
		t.Errorf("Expected EchoMode Password, got %v", ti.EchoMode)
	}

	if cmd == nil {
		t.Error("Expected command to be returned")
	}
}

func TestValidateAndExpandLocalPath(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		path        string
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid absolute path",
			path:    tmpDir,
			wantErr: false,
		},
		{
			name:        "empty path",
			path:        "",
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name:        "whitespace only",
			path:        "   ",
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name:        "relative path without home",
			path:        "relative/path",
			wantErr:     true,
			errContains: "must be absolute",
		},
		{
			name:        "parent does not exist",
			path:        "/nonexistent/parent/directory",
			wantErr:     true,
			errContains: "parent directory does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expanded, err := ValidateAndExpandLocalPath(tt.path)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errContains)
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errContains, err.Error())
				}
				if expanded != "" {
					t.Error("Expected empty path on error")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if expanded == "" {
					t.Error("Expected non-empty expanded path")
				}
				// For valid paths, the expanded path should be absolute
				if !strings.HasPrefix(expanded, "/") && !strings.Contains(expanded, ":") {
					t.Errorf("Expected absolute path, got: %s", expanded)
				}
			}
		})
	}
}
