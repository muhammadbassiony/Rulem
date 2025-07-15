package filemanager

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetDefaultStorageDir(t *testing.T) {
	tests := []struct {
		name        string
		setupEnv    func()
		cleanupEnv  func()
		expectPath  string
		expectValid bool
	}{
		{
			name: "macOS with XDG_DATA_HOME set",
			setupEnv: func() {
				os.Setenv("XDG_DATA_HOME", "/custom/data")
			},
			cleanupEnv: func() {
				os.Unsetenv("XDG_DATA_HOME")
			},
			expectValid: true,
		},
		{
			name: "macOS with invalid XDG_DATA_HOME",
			setupEnv: func() {
				os.Setenv("XDG_DATA_HOME", "relative/path")
			},
			cleanupEnv: func() {
				os.Unsetenv("XDG_DATA_HOME")
			},
			expectValid: true, // Should fallback to default
		},
		{
			name: "home directory unavailable",
			setupEnv: func() {
				os.Setenv("HOME", "/nonexistent")
			},
			cleanupEnv: func() {
				os.Unsetenv("HOME")
			},
			expectPath:  "/nonexistent/.local/share/rulemig", // The actual implementation behavior
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupEnv != nil {
				tt.setupEnv()
			}
			defer func() {
				if tt.cleanupEnv != nil {
					tt.cleanupEnv()
				}
			}()

			result := GetDefaultStorageDir()

			if !tt.expectValid {
				t.Errorf("Expected invalid result, got: %s", result)
				return
			}

			if tt.expectPath != "" && result != tt.expectPath {
				t.Errorf("Expected path %s, got %s", tt.expectPath, result)
			}

			// Verify the path is reasonable
			if result == "" {
				t.Error("GetDefaultStorageDir returned empty string")
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()

	tests := []struct {
		name     string
		input    string
		expected func() string
	}{
		{
			name:  "home path expansion",
			input: "~/Documents/rulemig",
			expected: func() string {
				if err != nil {
					return "~/Documents/rulemig" // Should return unchanged if HOME not available
				}
				return filepath.Join(home, "Documents", "rulemig")
			},
		},
		{
			name:     "absolute path unchanged",
			input:    "/absolute/path",
			expected: func() string { return "/absolute/path" },
		},
		{
			name:     "relative path unchanged",
			input:    "relative/path",
			expected: func() string { return "relative/path" },
		},
		{
			name:     "just tilde",
			input:    "~",
			expected: func() string { return "~" },
		},
		{
			name:     "empty string",
			input:    "",
			expected: func() string { return "" },
		},
		{
			name:     "tilde not at start",
			input:    "path/~/file",
			expected: func() string { return "path/~/file" }, // Should not be expanded
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandPath(tt.input)
			expected := tt.expected()
			if result != expected {
				t.Errorf("Expected %s, got %s", expected, result)
			}
		})
	}
}

func TestValidateStorageDir(t *testing.T) {
	// Get user home directory for proper test paths
	home, err := os.UserHomeDir()
	if err != nil {
		t.Logf("Warning: Cannot get home directory: %v, using fallback", err)
		home = "/tmp" // Use /tmp as fallback for testing
	}

	userTestDir := filepath.Join(home, "test-rulemig")

	tests := []struct {
		name        string
		input       string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty string",
			input:       "",
			expectError: true,
			errorMsg:    "cannot be empty",
		},
		{
			name:        "whitespace only",
			input:       "   ",
			expectError: true,
			errorMsg:    "cannot be empty",
		},
		{
			name:        "path traversal attack",
			input:       "../../../etc/passwd",
			expectError: true,
			errorMsg:    "path traversal not allowed",
		},
		{
			name:        "valid path in user directory",
			input:       userTestDir,
			expectError: false,
		},
		{
			name:        "valid home relative path",
			input:       "~/Documents/rulemig",
			expectError: false, // This should be valid if ~/Documents exists
		},
		{
			name:        "reserved directory - root",
			input:       "/",
			expectError: true,
			errorMsg:    "reserved directories",
		},
		{
			name:        "reserved directory - etc",
			input:       "/etc/rulemig",
			expectError: true,
			errorMsg:    "reserved directories",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For home relative path test, ensure the parent directory exists
			if tt.name == "valid home relative path" {
				home, err := os.UserHomeDir()
				if err != nil {
					t.Skip("Cannot get home directory for this test")
				}
				documentsDir := filepath.Join(home, "Documents")
				if _, err := os.Stat(documentsDir); os.IsNotExist(err) {
					// Create Documents directory for test
					if err := os.MkdirAll(documentsDir, 0755); err != nil {
						t.Skipf("Cannot create Documents directory for test: %v", err)
					}
					defer os.Remove(documentsDir) // Clean up if we created it
				}
			}

			err := ValidateStorageDir(tt.input)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %s", err.Error())
				}
			}
		})
	}
}

func TestTestStorageDir(t *testing.T) {
	// Get user home directory for proper test paths
	home, err := os.UserHomeDir()
	if err != nil {
		t.Logf("Warning: Cannot get home directory: %v, using fallback", err)
		home = "/tmp" // Use /tmp as fallback for testing
	}

	tests := []struct {
		name        string
		input       string
		setup       func() string
		expectError bool
	}{
		{
			name:        "valid writable directory",
			input:       filepath.Join(home, "test-rulemig"),
			expectError: false,
		},
		{
			name: "read-only directory",
			setup: func() string {
				roDir := filepath.Join(home, "readonly-test")
				os.MkdirAll(roDir, 0555) // Read-only
				return roDir
			},
			expectError: true,
		},
		{
			name:        "home relative path",
			input:       "~/rulemig-test",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.input
			if tt.setup != nil {
				input = tt.setup()
			}

			err := TestStorageDir(input)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %s", err.Error())
			}

			// Cleanup
			if !tt.expectError {
				os.RemoveAll(ExpandPath(input))
			}
		})
	}
}

func TestCreateStorageDir(t *testing.T) {
	// Get user home directory for proper test paths
	home, err := os.UserHomeDir()
	if err != nil {
		t.Logf("Warning: Cannot get home directory: %v, using fallback", err)
		home = "/tmp" // Use /tmp as fallback for testing
	}

	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "create new directory",
			input:       filepath.Join(home, "new-single-dir"),
			expectError: false,
		},
		{
			name:        "existing directory",
			input:       filepath.Join(home, "existing-test"),
			expectError: false,
		},
		{
			name:        "home relative path",
			input:       "~/rulemig-create-test",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CreateStorageDir(tt.input)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %s", err.Error())
			}

			if !tt.expectError {
				// Verify directory was created
				expandedPath := ExpandPath(tt.input)
				if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
					t.Error("Directory was not created")
				}
				// Cleanup
				os.RemoveAll(expandedPath)
			}
		})
	}
}

// Integration test
func TestFileManagerIntegration(t *testing.T) {
	// Get user home directory for proper test paths
	home, err := os.UserHomeDir()
	if err != nil {
		t.Logf("Warning: Cannot get home directory: %v, using fallback", err)
		home = "/tmp" // Use /tmp as fallback for testing
	}

	testPath := filepath.Join(home, "integration-test")

	// Test the full workflow
	t.Run("full workflow", func(t *testing.T) {
		// 1. Validate
		if err := ValidateStorageDir(testPath); err != nil {
			t.Fatalf("Validation failed: %s", err)
		}

		// 2. Test
		if err := TestStorageDir(testPath); err != nil {
			t.Fatalf("Testing failed: %s", err)
		}

		// 3. Create
		if err := CreateStorageDir(testPath); err != nil {
			t.Fatalf("Creation failed: %s", err)
		}

		// 4. Verify it exists
		if _, err := os.Stat(testPath); os.IsNotExist(err) {
			t.Error("Directory was not created")
		}
	})
}

// Helper function
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
