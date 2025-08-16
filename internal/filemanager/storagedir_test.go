package filemanager

import (
	"os"
	"path/filepath"
	"rulem/pkg/fileops"
	"strings"
	"testing"
)

// TestGetDefaultStorageDir tests the GetDefaultStorageDir function
func TestGetDefaultStorageDir(t *testing.T) {
	result := GetDefaultStorageDir()

	// Should not be empty
	if result == "" {
		t.Error("GetDefaultStorageDir returned empty string")
	}

	// Should contain rulem
	if !strings.Contains(result, "rulem") {
		t.Errorf("GetDefaultStorageDir should contain 'rulem', got: %s", result)
	}

	// Should be an absolute path (in most cases)
	if !filepath.IsAbs(result) && !strings.HasPrefix(result, ".rulem") {
		t.Errorf("GetDefaultStorageDir should return absolute path or .rulem fallback, got: %s", result)
	}
}

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
			expected: "~",
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

// TestCreateSecureStorageRoot tests the CreateSecureStorageRoot function
func TestCreateSecureStorageRoot(t *testing.T) {
	tempDir := createTempTestDir(t, "secure-root-test-")

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
			name:      "valid path in temp directory",
			input:     filepath.Join(tempDir, "secure-storage"),
			wantError: true,
		},
		{
			name:      "path outside home directory",
			input:     "/tmp/outside-home",
			wantError: true,
			errorMsg:  "within your home directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				cleanup := tt.setup()
				defer cleanup()
			}

			root, err := CreateSecureStorageRoot(tt.input)
			if root != nil {
				defer root.Close()
			}

			if tt.wantError {
				if err == nil {
					t.Errorf("CreateSecureStorageRoot(%q) expected error but got none", tt.input)
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("CreateSecureStorageRoot(%q) error = %v, want error containing %q",
						tt.input, err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("CreateSecureStorageRoot(%q) unexpected error: %v", tt.input, err)
				}
				if root == nil {
					t.Errorf("CreateSecureStorageRoot(%q) returned nil root without error", tt.input)
				}
			}
		})
	}
}

// Integration test for complete workflow
func TestIntegrationWorkflow(t *testing.T) {
	t.Run("complete workflow", func(t *testing.T) {
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

		// Cleanup
		os.RemoveAll(testDir)
	})
}

func BenchmarkGetDefaultStorageDir(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetDefaultStorageDir()
	}
}
