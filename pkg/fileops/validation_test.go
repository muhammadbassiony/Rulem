package fileops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Tests for ValidatePathSecurity

func TestValidatePathSecurity(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectError bool
		errorText   string
	}{
		{
			name:        "valid simple path",
			path:        "simple/path/file.txt",
			expectError: false,
		},
		{
			name:        "valid absolute path",
			path:        "/absolute/path/file.txt",
			expectError: false,
		},
		{
			name:        "empty path",
			path:        "",
			expectError: true,
			errorText:   "path cannot be empty",
		},
		{
			name:        "whitespace only path",
			path:        "   \t\n  ",
			expectError: true,
			errorText:   "path cannot be empty",
		},
		{
			name:        "path traversal with ..",
			path:        "../../../etc/passwd",
			expectError: true,
			errorText:   "path traversal not allowed",
		},
		{
			name:        "path traversal in middle",
			path:        "valid/../../etc/passwd",
			expectError: true,
			errorText:   "path traversal not allowed",
		},
		{
			name:        "path traversal after cleaning",
			path:        "valid/../../../etc/passwd",
			expectError: true,
			errorText:   "path traversal not allowed",
		},
		{
			name:        "legitimate .. usage",
			path:        "file..txt",
			expectError: true,
			errorText:   "path traversal not allowed",
		},
		{
			name:        "single dot",
			path:        "./file.txt",
			expectError: false,
		},
		{
			name:        "multiple slashes",
			path:        "path//to///file.txt",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePathSecurity(tt.path)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("Expected error containing %q, got: %v", tt.errorText, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// Tests for ValidateCWDPath

func TestValidateCWDPath(t *testing.T) {
	tests := []struct {
		name        string
		destPath    string
		expectError bool
		errorText   string
	}{
		{
			name:        "valid relative path",
			destPath:    "subdir/file.txt",
			expectError: false,
		},
		{
			name:        "valid nested relative path",
			destPath:    "deep/nested/path/file.txt",
			expectError: false,
		},
		{
			name:        "empty destination path",
			destPath:    "",
			expectError: true,
			errorText:   "destination path cannot be empty",
		},
		{
			name:        "absolute path",
			destPath:    "/absolute/path/file.txt",
			expectError: true,
			errorText:   "must be relative to current working directory",
		},
		{
			name:        "path traversal with ..",
			destPath:    "../escape.txt",
			expectError: true,
			errorText:   "path traversal not allowed",
		},
		{
			name:        "path traversal in middle",
			destPath:    "valid/../escape.txt",
			expectError: true,
			errorText:   "path traversal not allowed in destination path",
		},
		{
			name:        "current directory reference",
			destPath:    "./file.txt",
			expectError: false,
		},
		{
			name:        "file in current directory",
			destPath:    "file.txt",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCWDPath(tt.destPath)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("Expected error containing %q, got: %v", tt.errorText, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// Tests for ValidateFileInDirectory

func TestValidateFileInDirectory(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Create test files
	validFile := createTestFile(t, tempDir, "valid.txt", "content")
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subDir: %v", err)
	}
	nestedFile := createTestFile(t, subDir, "nested.txt", "nested content")

	// Create directory for testing
	testDir := filepath.Join(tempDir, "testdir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("Failed to create testDir: %v", err)
	}

	t.Run("valid file in directory", func(t *testing.T) {
		err := ValidateFileInDirectory(validFile, tempDir)
		if err != nil {
			t.Errorf("Expected no error for valid file, got: %v", err)
		}
	})

	t.Run("valid nested file in directory", func(t *testing.T) {
		err := ValidateFileInDirectory(nestedFile, tempDir)
		if err != nil {
			t.Errorf("Expected no error for nested file, got: %v", err)
		}
	})

	t.Run("file outside directory", func(t *testing.T) {
		outsideDir := createTempDir(t)
		defer os.RemoveAll(outsideDir)
		outsideFile := createTestFile(t, outsideDir, "outside.txt", "content")

		err := ValidateFileInDirectory(outsideFile, tempDir)
		if err == nil {
			t.Error("Expected error for file outside directory")
		}
		if !strings.Contains(err.Error(), "not within base directory") {
			t.Errorf("Expected 'not within base directory' error, got: %v", err)
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")

		err := ValidateFileInDirectory(nonExistentFile, tempDir)
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
		if !strings.Contains(err.Error(), "does not exist") {
			t.Errorf("Expected 'does not exist' error, got: %v", err)
		}
	})

	t.Run("path is directory", func(t *testing.T) {
		err := ValidateFileInDirectory(testDir, tempDir)
		if err == nil {
			t.Error("Expected error when path is directory")
		}
		if !strings.Contains(err.Error(), "directory, not a file") {
			t.Errorf("Expected 'directory, not a file' error, got: %v", err)
		}
	})

	t.Run("empty file path", func(t *testing.T) {
		err := ValidateFileInDirectory("", tempDir)
		if err == nil {
			t.Error("Expected error for empty file path")
		}
	})

	t.Run("empty base directory", func(t *testing.T) {
		err := ValidateFileInDirectory(validFile, "")
		if err == nil {
			t.Error("Expected error for empty base directory")
		}
	})
}

// Tests for SanitizeFilename

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		expected    string
		expectError bool
		errorText   string
	}{
		{
			name:        "simple filename",
			filename:    "file.txt",
			expected:    "file.txt",
			expectError: false,
		},
		{
			name:        "filename with spaces",
			filename:    "my file.txt",
			expected:    "my file.txt",
			expectError: false,
		},
		{
			name:        "path traversal attack",
			filename:    "../../../etc/passwd",
			expected:    "passwd",
			expectError: false,
		},
		{
			name:        "filename with forward slash",
			filename:    "folder/file.txt",
			expected:    "file.txt",
			expectError: false,
		},
		{
			name:        "filename with backslash",
			filename:    "folder\\file.txt",
			expected:    "folder\\file.txt",
			expectError: false,
		},
		{
			name:        "empty filename",
			filename:    "",
			expectError: true,
			errorText:   "filename cannot be empty",
		},
		{
			name:        "just dots",
			filename:    "..",
			expectError: true,
			errorText:   "invalid filename after sanitization",
		},
		{
			name:        "single dot",
			filename:    ".",
			expectError: true,
			errorText:   "invalid filename after sanitization",
		},
		{
			name:        "whitespace only",
			filename:    "   ",
			expectError: true,
			errorText:   "invalid filename after sanitization",
		},
		{
			name:        "complex path with dots",
			filename:    "../../folder/../file..name.txt",
			expected:    "filename.txt",
			expectError: false,
		},
		{
			name:        "filename becomes empty after sanitization",
			filename:    "../..",
			expectError: true,
			errorText:   "invalid filename after sanitization",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SanitizeFilename(tt.filename)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("Expected error containing %q, got: %v", tt.errorText, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				} else if result != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

// Tests for ValidateFileAccess

func TestValidateFileAccess(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Create test files
	readableFile := createTestFile(t, tempDir, "readable.txt", "content")
	writableFile := createTestFile(t, tempDir, "writable.txt", "content")

	// Create directory for testing
	testDir := filepath.Join(tempDir, "testdir")
	os.Mkdir(testDir, 0755)

	t.Run("readable file check", func(t *testing.T) {
		err := ValidateFileAccess(readableFile, false)
		if err != nil {
			t.Errorf("Expected no error for readable file, got: %v", err)
		}
	})

	t.Run("writable file check", func(t *testing.T) {
		err := ValidateFileAccess(writableFile, true)
		if err != nil {
			t.Errorf("Expected no error for writable file, got: %v", err)
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")

		err := ValidateFileAccess(nonExistentFile, false)
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
		if !strings.Contains(err.Error(), "does not exist") {
			t.Errorf("Expected 'does not exist' error, got: %v", err)
		}
	})

	t.Run("directory instead of file", func(t *testing.T) {
		err := ValidateFileAccess(testDir, false)
		if err == nil {
			t.Error("Expected error when path is directory")
		}
		if !strings.Contains(err.Error(), "directory, not a file") {
			t.Errorf("Expected 'directory, not a file' error, got: %v", err)
		}
	})

	// Platform-specific tests for file permissions
	if os.Getenv("CI") == "" { // Skip permission tests in CI environments
		t.Run("unreadable file", func(t *testing.T) {
			unreadableFile := createTestFile(t, tempDir, "unreadable.txt", "content")
			if err := os.Chmod(unreadableFile, 0000); err != nil {
				t.Skip("Cannot change file permissions")
			}
			defer func() {
				if err := os.Chmod(unreadableFile, 0644); err != nil {
					t.Logf("warning: failed to restore permissions: %v", err)
				}
			}()

			err := ValidateFileAccess(unreadableFile, false)
			if err == nil {
				t.Error("Expected error for unreadable file")
			}
		})
	}
}

// Integration tests

func TestValidationIntegration(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	t.Run("complete validation workflow", func(t *testing.T) {
		// Create a test file
		testFile := createTestFile(t, tempDir, "integration_test.txt", "integration content")

		// Test path security
		err := ValidatePathSecurity(testFile)
		if err != nil {
			t.Errorf("Path security validation failed: %v", err)
		}

		// Test file in directory validation
		err = ValidateFileInDirectory(testFile, tempDir)
		if err != nil {
			t.Errorf("File in directory validation failed: %v", err)
		}

		// Test file access validation
		err = ValidateFileAccess(testFile, false)
		if err != nil {
			t.Errorf("File access validation failed: %v", err)
		}

		// Test filename sanitization
		_, err = SanitizeFilename(filepath.Base(testFile))
		if err != nil {
			t.Errorf("Filename sanitization failed: %v", err)
		}
	})

	t.Run("security validation prevents attacks", func(t *testing.T) {
		maliciousPaths := []string{
			"../../../etc/passwd",
			"..\\..\\..\\Windows\\System32",
			"/etc/shadow",
		}

		for _, maliciousPath := range maliciousPaths {
			err := ValidatePathSecurity(maliciousPath)
			if err == nil {
				t.Errorf("Security validation should have rejected: %s", maliciousPath)
			}
		}
	})
}

// TestIsReservedDirectory tests the isReservedDirectory helper function
func TestIsReservedDirectory(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "unix root directory",
			path:     "/",
			expected: !isWindows(),
		},
		{
			name:     "unix etc directory",
			path:     "/etc",
			expected: !isWindows(),
		},
		{
			name:     "unix bin directory",
			path:     "/bin",
			expected: !isWindows(),
		},
		{
			name:     "windows system directory",
			path:     "C:\\Windows",
			expected: isWindows(),
		},
		{
			name:     "windows program files",
			path:     "C:\\Program Files",
			expected: isWindows(),
		},
		{
			name:     "windows system32",
			path:     "C:\\System32",
			expected: isWindows(),
		},
		{
			name:     "safe temp directory",
			path:     os.TempDir(),
			expected: false,
		},
		{
			name:     "user home directory",
			path:     func() string { home, _ := os.UserHomeDir(); return home }(),
			expected: false,
		},
		{
			name:     "nested system directory",
			path:     "/etc/passwd",
			expected: !isWindows(),
		},
		{
			name:     "case insensitive windows",
			path:     "c:\\windows",
			expected: isWindows(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsReservedDirectory(tt.path)
			if result != tt.expected {
				t.Errorf("isReservedDirectory(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

// TestGetReservedDirectories tests the getReservedDirectories helper function
func TestGetReservedDirectories(t *testing.T) {
	dirs := getReservedDirectories()

	// Should not be empty
	if len(dirs) == 0 {
		t.Error("getReservedDirectories should return non-empty slice")
	}

	// Should contain common system directories
	hasUnixDirs := false
	hasWindowsDirs := false

	for _, dir := range dirs {
		// Check for Unix directories
		if dir == "/" || dir == "/etc" || dir == "/bin" {
			hasUnixDirs = true
		}
		// Check for Windows directories
		if strings.Contains(dir, "C:\\Windows") || strings.Contains(dir, "Program Files") {
			hasWindowsDirs = true
		}
	}

	// Should always have Unix directories (for cross-platform safety)
	if !hasUnixDirs {
		t.Error("getReservedDirectories should contain Unix system directories")
	}

	// On Windows, should also have Windows directories
	if isWindows() && !hasWindowsDirs {
		t.Error("getReservedDirectories should contain Windows system directories on Windows")
	}

	// Check for duplicates
	seen := make(map[string]bool)
	for _, dir := range dirs {
		if seen[dir] {
			t.Errorf("getReservedDirectories returned duplicate directory: %s", dir)
		}
		seen[dir] = true
	}
}

// Benchmark tests

func BenchmarkValidatePathSecurity(b *testing.B) {
	testPath := "valid/path/to/file.txt"
	for range b.N {
		if err := ValidatePathSecurity(testPath); err != nil {
			b.Logf("ValidatePathSecurity returned error: %v", err)
		}
	}
}

func BenchmarkSanitizeFilename(b *testing.B) {
	testFilename := "valid_filename.txt"
	for range b.N {
		SanitizeFilename(testFilename)
	}
}

// Tests for ValidateDirectoryWritable

func TestValidateDirectoryWritable(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name      string
		dirPath   string
		setup     func() string
		wantError bool
		errorText string
	}{
		{
			name:      "valid writable directory",
			setup:     func() string { return filepath.Join(tempDir, "writable") },
			wantError: false,
		},
		{
			name:      "directory gets created if missing",
			setup:     func() string { return filepath.Join(tempDir, "new_dir") },
			wantError: false,
		},
		{
			name:      "nested directory creation",
			setup:     func() string { return filepath.Join(tempDir, "nested", "deep", "dir") },
			wantError: false,
		},
		{
			name: "file exists with same name",
			setup: func() string {
				filePath := filepath.Join(tempDir, "existing_file")
				if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
					t.Fatalf("Failed to write file: %v", err)
				}
				return filePath
			},
			wantError: true,
			errorText: "cannot create directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dirPath := tt.setup()

			err := ValidateDirectoryWritable(dirPath)

			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateDirectoryWritable(%q) expected error but got none", dirPath)
					return
				}
				if tt.errorText != "" && !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("ValidateDirectoryWritable(%q) error = %v, want error containing %q",
						dirPath, err, tt.errorText)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateDirectoryWritable(%q) unexpected error: %v", dirPath, err)
				}
				// Verify directory exists and is writable
				if stat, err := os.Stat(dirPath); err != nil {
					t.Errorf("Directory should exist after validation: %v", err)
				} else if !stat.IsDir() {
					t.Errorf("Path should be a directory")
				}
			}
		})
	}
}

// Tests for ValidatePathInHome

func TestValidatePathInHome(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot determine home directory")
	}

	tests := []struct {
		name      string
		path      string
		wantError bool
		errorText string
		expected  string
	}{
		{
			name:      "path within home",
			path:      filepath.Join(homeDir, "Documents", "file.txt"),
			wantError: false,
			expected:  filepath.Join("Documents", "file.txt"),
		},
		{
			name:      "home directory itself",
			path:      homeDir,
			wantError: false,
			expected:  ".",
		},
		{
			name:      "path outside home",
			path:      "/etc/passwd",
			wantError: true,
			errorText: "path is outside home directory",
		},
		{
			name:      "path traversal outside home",
			path:      filepath.Join(homeDir, "..", "..", "etc", "passwd"),
			wantError: true,
			errorText: "path is outside home directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidatePathInHome(tt.path)

			if tt.wantError {
				if err == nil {
					t.Errorf("ValidatePathInHome(%q) expected error but got none", tt.path)
					return
				}
				if tt.errorText != "" && !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("ValidatePathInHome(%q) error = %v, want error containing %q",
						tt.path, err, tt.errorText)
				}
			} else {
				if err != nil {
					t.Errorf("ValidatePathInHome(%q) unexpected error: %v", tt.path, err)
				}
				if result != tt.expected {
					t.Errorf("ValidatePathInHome(%q) = %q, want %q", tt.path, result, tt.expected)
				}
			}
		})
	}
}

// Tests for ValidateStoragePath

func TestValidateStoragePath(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot determine home directory")
	}

	tests := []struct {
		name      string
		path      string
		wantError bool
		errorText string
	}{
		{
			name:      "empty path",
			path:      "",
			wantError: true,
			errorText: "storage directory cannot be empty",
		},
		{
			name:      "whitespace only",
			path:      "   \t\n  ",
			wantError: true,
			errorText: "storage directory cannot be empty",
		},
		{
			name:      "valid home relative path",
			path:      "~/Documents/myapp",
			wantError: false,
		},
		{
			name:      "valid absolute path in temp",
			path:      tempDir,
			wantError: false,
		},
		{
			name:      "path traversal attack",
			path:      "../../../etc/passwd",
			wantError: true,
			errorText: "path traversal not allowed",
		},
		{
			name:      "relative path not from home",
			path:      "relative/path",
			wantError: true,
			errorText: "path must be absolute or relative to home directory",
		},
		{
			name:      "system directory",
			path:      "/etc",
			wantError: true,
			errorText: "path traversal not allowed",
		},
		{
			name:      "user ssh directory",
			path:      filepath.Join(homeDir, ".ssh"),
			wantError: true,
			errorText: "path traversal not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStoragePath(tt.path)

			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateStoragePath(%q) expected error but got none", tt.path)
					return
				}
				if tt.errorText != "" && !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("ValidateStoragePath(%q) error = %v, want error containing %q",
						tt.path, err, tt.errorText)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateStoragePath(%q) unexpected error: %v", tt.path, err)
				}
			}
		})
	}
}

// Tests for ValidateContentSecurity

func TestValidateContentSecurity(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
		errorText   string
	}{
		{
			name:        "clean content",
			content:     "This is clean content",
			expectError: false,
		},
		{
			name:        "content with newlines",
			content:     "Content with\nnewlines",
			expectError: false,
		},
		{
			name:        "content with tabs",
			content:     "Content with\ttabs",
			expectError: false,
		},
		{
			name:        "content with carriage returns",
			content:     "Content with\rcarriage returns",
			expectError: false,
		},
		{
			name:        "content with null byte",
			content:     "Content with\x00null byte",
			expectError: true,
			errorText:   "control characters",
		},
		{
			name:        "content with control character",
			content:     "Content with\x01control char",
			expectError: true,
			errorText:   "control characters",
		},
		{
			name:        "content with script tag",
			content:     "Content with <script>alert('xss')</script>",
			expectError: true,
			errorText:   "<script",
		},
		{
			name:        "content with javascript protocol",
			content:     "Content with javascript:alert('xss')",
			expectError: true,
			errorText:   "javascript:",
		},
		{
			name:        "content with vbscript",
			content:     "Content with vbscript:msgbox('xss')",
			expectError: true,
			errorText:   "vbscript:",
		},
		{
			name:        "content with data url",
			content:     "Content with data:text/html,<script>alert(1)</script>",
			expectError: true,
			errorText:   "<script",
		},
		{
			name:        "content with eval",
			content:     "Content with eval(maliciousCode)",
			expectError: true,
			errorText:   "eval(",
		},
		{
			name:        "content with exec",
			content:     "Content with exec(systemCommand)",
			expectError: true,
			errorText:   "exec(",
		},
		{
			name:        "content with onload",
			content:     "Content with onload=alert('xss')",
			expectError: true,
			errorText:   "onload=",
		},
		{
			name:        "case insensitive script detection",
			content:     "Content with <SCRIPT>alert('xss')</SCRIPT>",
			expectError: true,
			errorText:   "<script",
		},
		{
			name:        "mixed case javascript",
			content:     "Content with JavaScript:alert('xss')",
			expectError: true,
			errorText:   "javascript:",
		},
		{
			name:        "empty content",
			content:     "",
			expectError: false,
		},
		{
			name:        "unicode content",
			content:     "Content with unicode 字符 and émojis 🚀",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateContentSecurity(tt.content)
			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateContentSecurity(%q) expected error but got none", tt.content)
				} else if !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("ValidateContentSecurity(%q) error = %v, want error containing %q", tt.content, err, tt.errorText)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateContentSecurity(%q) unexpected error: %v", tt.content, err)
				}
			}
		})
	}
}

// Tests for SanitizeIdentifier

func TestSanitizeIdentifier(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		maxLength   int
		expected    string
		expectError bool
		errorText   string
	}{
		{
			name:        "clean identifier",
			input:       "clean_identifier",
			maxLength:   100,
			expected:    "clean_identifier",
			expectError: false,
		},
		{
			name:        "identifier with spaces",
			input:       "identifier with spaces",
			maxLength:   100,
			expected:    "identifier_with_spaces",
			expectError: false,
		},
		{
			name:        "identifier with special characters",
			input:       "identifier@#$%^&*()with!special",
			maxLength:   100,
			expected:    "identifierwithspecial",
			expectError: false,
		},
		{
			name:        "identifier with multiple spaces",
			input:       "identifier  with   multiple    spaces",
			maxLength:   100,
			expected:    "identifier_with_multiple_spaces",
			expectError: false,
		},
		{
			name:        "identifier with hyphens",
			input:       "identifier-with-hyphens",
			maxLength:   100,
			expected:    "identifier-with-hyphens",
			expectError: false,
		},
		{
			name:        "identifier with underscores",
			input:       "identifier_with_underscores",
			maxLength:   100,
			expected:    "identifier_with_underscores",
			expectError: false,
		},
		{
			name:        "identifier with periods",
			input:       "identifier.with.periods",
			maxLength:   100,
			expected:    "identifier.with.periods",
			expectError: false,
		},
		{
			name:        "identifier with leading/trailing separators",
			input:       "_-identifier-with-separators-_",
			maxLength:   100,
			expected:    "identifier-with-separators",
			expectError: false,
		},
		{
			name:        "empty identifier",
			input:       "",
			maxLength:   100,
			expected:    "",
			expectError: true,
			errorText:   "empty",
		},
		{
			name:        "whitespace only identifier",
			input:       "   \t\n  ",
			maxLength:   100,
			expected:    "",
			expectError: true,
			errorText:   "empty",
		},
		{
			name:        "identifier with only special characters",
			input:       "@#$%^&*()",
			maxLength:   100,
			expected:    "",
			expectError: true,
			errorText:   "empty after sanitization",
		},
		{
			name:        "identifier with consecutive separators",
			input:       "identifier--with__consecutive",
			maxLength:   100,
			expected:    "identifier_with_consecutive",
			expectError: false,
		},
		{
			name:        "very long identifier",
			input:       strings.Repeat("a", 150),
			maxLength:   100,
			expected:    strings.Repeat("a", 100),
			expectError: false,
		},
		{
			name:        "mixed case with numbers",
			input:       "TestIdentifier123WithNumbers",
			maxLength:   100,
			expected:    "TestIdentifier123WithNumbers",
			expectError: false,
		},
		{
			name:        "unicode characters",
			input:       "identifier_with_unicode_字符",
			maxLength:   100,
			expected:    "identifier_with_unicode",
			expectError: false,
		},
		{
			name:        "no length limit",
			input:       "very_long_identifier_name",
			maxLength:   0,
			expected:    "very_long_identifier_name",
			expectError: false,
		},
		{
			name:        "exact length limit",
			input:       "exact",
			maxLength:   5,
			expected:    "exact",
			expectError: false,
		},
		{
			name:        "length limit with truncation",
			input:       "toolongidentifier",
			maxLength:   10,
			expected:    "toolongide",
			expectError: false,
		},
		{
			name:        "script injection attempt",
			input:       "<script>alert('xss')</script>",
			maxLength:   100,
			expected:    "scriptalertxssscript",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SanitizeIdentifier(tt.input, tt.maxLength)
			if tt.expectError {
				if err == nil {
					t.Errorf("SanitizeIdentifier(%q, %d) expected error but got none", tt.input, tt.maxLength)
				} else if !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("SanitizeIdentifier(%q, %d) error = %v, want error containing %q", tt.input, tt.maxLength, err, tt.errorText)
				}
			} else {
				if err != nil {
					t.Errorf("SanitizeIdentifier(%q, %d) unexpected error: %v", tt.input, tt.maxLength, err)
				} else if result != tt.expected {
					t.Errorf("SanitizeIdentifier(%q, %d) = %q, want %q", tt.input, tt.maxLength, result, tt.expected)
				}
			}
		})
	}
}

// Tests for ValidateFileSizeLimit

func TestValidateFileSizeLimit(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "fileops-filesize-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name        string
		content     string
		maxSize     int64
		expectError bool
		errorText   string
		skipFile    bool // For testing non-existent files
	}{
		{
			name:        "file within size limit",
			content:     "Small file content",
			maxSize:     100,
			expectError: false,
		},
		{
			name:        "file exceeds size limit",
			content:     strings.Repeat("Large content ", 100),
			maxSize:     50,
			expectError: true,
			errorText:   "exceeds limit",
		},
		{
			name:        "empty file",
			content:     "",
			maxSize:     100,
			expectError: false,
		},
		{
			name:        "file at exact size limit",
			content:     strings.Repeat("x", 50),
			maxSize:     50,
			expectError: false,
		},
		{
			name:        "file one byte over limit",
			content:     strings.Repeat("x", 51),
			maxSize:     50,
			expectError: true,
			errorText:   "exceeds limit",
		},
		{
			name:        "invalid size limit - zero",
			content:     "Content",
			maxSize:     0,
			expectError: true,
			errorText:   "invalid size limit",
		},
		{
			name:        "invalid size limit - negative",
			content:     "Content",
			maxSize:     -1,
			expectError: true,
			errorText:   "invalid size limit",
		},
		{
			name:        "non-existent file",
			content:     "",
			maxSize:     100,
			expectError: true,
			errorText:   "does not exist",
			skipFile:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var testPath string

			if tt.skipFile {
				// Test with non-existent file
				testPath = filepath.Join(tempDir, "nonexistent-file.txt")
			} else {
				// Create test file
				testPath = filepath.Join(tempDir, "test-"+tt.name+".txt")
				if err := os.WriteFile(testPath, []byte(tt.content), 0644); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			}

			err := ValidateFileSizeLimit(testPath, tt.maxSize)
			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateFileSizeLimit(%q, %d) expected error but got none", testPath, tt.maxSize)
				} else if !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("ValidateFileSizeLimit(%q, %d) error = %v, want error containing %q", testPath, tt.maxSize, err, tt.errorText)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateFileSizeLimit(%q, %d) unexpected error: %v", testPath, tt.maxSize, err)
				}
			}
		})
	}
}

// Test ValidateFileSizeLimit with directory instead of file
func TestValidateFileSizeLimitWithDirectory(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "fileops-dir-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a subdirectory to test with
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	err = ValidateFileSizeLimit(subDir, 100)
	if err == nil {
		t.Error("ValidateFileSizeLimit should fail when given a directory")
	} else if !strings.Contains(err.Error(), "directory, not a file") {
		t.Errorf("ValidateFileSizeLimit error = %v, want error containing 'directory, not a file'", err)
	}
}
