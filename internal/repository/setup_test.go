package repository

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"rulem/pkg/fileops"
)

// TestExpandPath tests the ExpandPath function
func TestExpandPath(t *testing.T) {
	home := getHomeDir(t)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "home directory expansion",
			input:    "~/Documents",
			expected: filepath.Join(home, "Documents"),
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "just tilde",
			input:    "~",
			expected: home,
		},
		{
			name:     "tilde not at start",
			input:    "dir/~/file",
			expected: "dir/~/file",
		},
		{
			name:     "multiple tildes",
			input:    "~/~/file",
			expected: filepath.Join(home, "~/file"),
		},
		{
			name:     "tilde with no slash",
			input:    "~file",
			expected: "~file",
		},
		{
			name:     "absolute path",
			input:    "/absolute/path",
			expected: "/absolute/path",
		},
		{
			name:     "relative path",
			input:    "relative/path",
			expected: "relative/path",
		},
		{
			name:     "home with nested directories",
			input:    "~/Documents/Projects/rulem",
			expected: filepath.Join(home, "Documents", "Projects", "rulem"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fileops.ExpandPath(tt.input)
			if result != tt.expected {
				t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestExpandPathEdgeCases tests edge cases for ExpandPath
func TestExpandPathEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		setupEnv func() func() // setup function returns cleanup function
		validate func(t *testing.T, result string)
	}{
		{
			name:  "unicode characters",
			input: "~/测试/rülemig",
			setupEnv: func() func() {
				return func() {} // no setup needed
			},
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "测试") || !strings.Contains(result, "rülemig") {
					t.Errorf("Unicode characters not preserved in path: %s", result)
				}
			},
		},
		{
			name:  "spaces in path",
			input: "~/my documents/rule mig",
			setupEnv: func() func() {
				return func() {}
			},
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "my documents") || !strings.Contains(result, "rule mig") {
					t.Errorf("Spaces not preserved in path: %s", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setupEnv()
			defer cleanup()

			result := fileops.ExpandPath(tt.input)
			tt.validate(t, result)
		})
	}
}

// reservedStoragePath returns a path rulem's storage policy refuses on this
// platform, so the reserved-directory case is exercised everywhere.
func reservedStoragePath() string {
	if runtime.GOOS == "windows" {
		return `C:\\Windows\\rulem-rules`
	}
	return "/etc/rulem-rules"
}

// TestEnsureLocalStorageDirectory tests the EnsureLocalStorageDirectory function
func TestEnsureLocalStorageDirectory(t *testing.T) {
	tempDir := createTempTestDir(t, "local-storage-test-")

	tests := []struct {
		name      string
		input     string
		wantError bool
		errorMsg  string
		setup     func() func()
	}{
		{
			name:      "empty input",
			input:     "",
			wantError: true,
			errorMsg:  "cannot be empty",
		},
		{
			// BEHAVIOUR CHANGE (fileops os.Root migration): the function now
			// delegates to fileops.OpenDir, rulem's single storage policy, and
			// creates the directory. It no longer applies a home-directory-only
			// rule of its own - the setup flow that calls it has always allowed
			// any non-reserved absolute path.
			name:      "creates a storage directory and returns a handle",
			input:     filepath.Join(tempDir, "secure-storage"),
			wantError: false,
		},
		{
			name:      "reserved system directory is refused",
			input:     reservedStoragePath(),
			wantError: true,
			// ValidateStoragePath reports reserved directories through
			// ValidatePathSecurity, which phrases every rejection this way.
			errorMsg: "not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				cleanup := tt.setup()
				defer cleanup()
			}

			dir, err := EnsureLocalStorageDirectory(tt.input)
			if dir != nil {
				defer func() { _ = dir.Close() }()
			}

			if tt.wantError {
				if err == nil {
					t.Errorf("EnsureLocalStorageDirectory(%q) expected error but got none", tt.input)
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("EnsureLocalStorageDirectory(%q) error = %v, want error containing %q",
						tt.input, err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("EnsureLocalStorageDirectory(%q) unexpected error: %v", tt.input, err)
				}
				if dir == nil {
					t.Errorf("EnsureLocalStorageDirectory(%q) returned nil handle without error", tt.input)
				}
			}
		})
	}
}

// Integration test for complete local storage workflow
func TestLocalStorageIntegrationWorkflow(t *testing.T) {
	t.Run("complete local storage workflow", func(t *testing.T) {
		// Step 1: Get default directory
		defaultDir := GetDefaultStorageDir()
		if defaultDir == "" {
			t.Fatal("GetDefaultStorageDir returned empty string")
		}

		// Step 2: Use a test subdirectory
		testDir := filepath.Join(defaultDir, "integration-test")

		// Step 2.5: Ensure parent directory exists
		parentDir := filepath.Dir(testDir)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			t.Fatalf("Failed to create parent directory: %v", err)
		}

		// Step 3: Validate the directory using helper function
		err := fileops.ValidateStoragePath(testDir)
		if err != nil {
			t.Fatalf("ValidateStorageDir failed: %v", err)
		}

		// Step 4: Create the directory and test writability using fileops functions
		if err := fileops.EnsureDirectoryExists(testDir); err != nil {
			t.Fatalf("EnsureDirectoryExists failed: %v", err)
		}

		if err := fileops.ValidateDirectoryWritable(testDir); err != nil {
			t.Fatalf("ValidateDirectoryWritable failed: %v", err)
		}

		// Step 5: Verify directory exists and is usable
		if _, err := os.Stat(testDir); err != nil {
			t.Fatalf("Created directory doesn't exist: %v", err)
		}

		// Step 6: Test file operations in created directory
		testFile := filepath.Join(testDir, "test-rule.md")
		testContent := []byte("# Test Rule\n\nThis is a test markdown file.")

		err = os.WriteFile(testFile, testContent, 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		// Step 7: Verify file content
		readContent, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read test file: %v", err)
		}

		if string(readContent) != string(testContent) {
			t.Errorf("File content mismatch: got %q, want %q", readContent, testContent)
		}

		// Clean up
		if err := os.RemoveAll(testDir); err != nil {
			t.Logf("Warning: Failed to clean up test directory: %v", err)
		}
	})
}

// Helper function to get home directory for tests
func getHomeDir(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}
	return home
}

// Helper function to create temp test directory
func createTempTestDir(t *testing.T, pattern string) string {
	t.Helper()
	tempDir, err := os.MkdirTemp("", pattern)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Warning: Failed to clean up temp directory %s: %v", tempDir, err)
		}
	})
	return tempDir
}
