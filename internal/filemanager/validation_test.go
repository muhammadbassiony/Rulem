package filemanager

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Test helper functions for helpers_test.go

// createTempTestDir creates a temporary directory for testing
func createTempTestDir(t *testing.T, prefix string) string {
	t.Helper()
	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	return dir
}

// makeReadOnly makes a directory read-only
func makeReadOnly(t *testing.T, path string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		// Windows: Remove write permission
		if err := os.Chmod(path, 0555); err != nil {
			t.Fatalf("failed to make directory read-only: %v", err)
		}
	} else {
		// Unix: Remove write permission
		if err := os.Chmod(path, 0555); err != nil {
			t.Fatalf("failed to make directory read-only: %v", err)
		}
	}
}

// restoreWritePermissions restores write permissions
func restoreWritePermissions(t *testing.T, path string) {
	t.Helper()
	if err := os.Chmod(path, 0755); err != nil {
		t.Logf("warning: failed to restore write permissions: %v", err)
	}
}

// getHomeDir gets the user's home directory for testing
func getHomeDir(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}
	return home
}

// isWindows returns true if running on Windows
func isWindows() bool {
	return runtime.GOOS == "windows"
}

// TestValidateStorageDir tests the ValidateStorageDir function
func TestValidateStorageDir(t *testing.T) {
	tempDir := createTempTestDir(t, "validate-test-")

	tests := []struct {
		name      string
		input     string
		wantError bool
		errorMsg  string
		setup     func() func() // returns cleanup function
	}{
		{
			name:      "empty string",
			input:     "",
			wantError: true,
			errorMsg:  "cannot be empty",
		},
		{
			name:      "whitespace only",
			input:     "   ",
			wantError: true,
			errorMsg:  "cannot be empty",
		},
		{
			name:      "valid home relative path",
			input:     "~/Documents/rulem",
			wantError: false,
		},
		{
			name:      "valid absolute path in temp",
			input:     filepath.Join(tempDir, "valid"),
			wantError: false,
		},
		{
			name:      "path traversal attack",
			input:     "../../../etc/passwd",
			wantError: true,
			errorMsg:  "path traversal not allowed",
		},
		{
			name:      "path traversal in home",
			input:     "~/../../etc/passwd",
			wantError: true,
			errorMsg:  "path traversal not allowed",
		},
		{
			name:      "relative path not from home",
			input:     "relative/path",
			wantError: true,
			errorMsg:  "must be absolute or relative to home",
		},
		{
			name:      "parent directory doesn't exist",
			input:     filepath.Join(tempDir, "nonexistent", "subdir"),
			wantError: true,
			errorMsg:  "no such file or directory",
		},
		{
			name:      "actual system directory",
			input:     "/usr/bin/rulem",
			wantError: true,
			errorMsg:  "parent directory resolves to reserved location",
		},
		{
			name:      "system temp directory",
			input:     filepath.Join(os.TempDir(), "rulem-test"),
			wantError: false,
		},
	}

	// Add platform-specific tests
	if isWindows() {
		tests = append(tests, []struct {
			name      string
			input     string
			wantError bool
			errorMsg  string
			setup     func() func()
		}{
			{
				name:      "windows system directory",
				input:     "C:\\Windows\\System32\\rulem",
				wantError: true,
				errorMsg:  "system or reserved directories",
			},
			{
				name:      "windows program files",
				input:     "C:\\Program Files\\rulem",
				wantError: true,
				errorMsg:  "system or reserved directories",
			},
		}...)
	} else {
		tests = append(tests, []struct {
			name      string
			input     string
			wantError bool
			errorMsg  string
			setup     func() func()
		}{
			{
				name:      "unix system directory",
				input:     "/etc/rulem",
				wantError: true,
				errorMsg:  "parent directory resolves to reserved location",
			},
			{
				name:      "unix bin directory",
				input:     "/bin/rulem",
				wantError: true,
				errorMsg:  "parent directory resolves to reserved location",
			},
		}...)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				cleanup := tt.setup()
				defer cleanup()
			}

			err := ValidateStorageDir(tt.input)

			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateStorageDir(%q) expected error but got none", tt.input)
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("ValidateStorageDir(%q) error = %v, want error containing %q",
						tt.input, err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateStorageDir(%q) unexpected error: %v", tt.input, err)
				}
			}
		})
	}
}

// TestTestStorageDir tests the TestStorageDir function
func TestTestStorageDir(t *testing.T) {
	tempDir := createTempTestDir(t, "test-storage-")

	tests := []struct {
		name      string
		path      string
		setup     func(string) func() // setup function with path, returns cleanup
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid writable directory",
			path: filepath.Join(tempDir, "writable"),
			setup: func(path string) func() {
				return func() {}
			},
			wantError: false,
		},
		{
			name: "create nested directories",
			path: filepath.Join(tempDir, "nested", "deep", "structure"),
			setup: func(path string) func() {
				return func() {}
			},
			wantError: false,
		},
		{
			name: "read-only parent directory",
			path: filepath.Join(tempDir, "readonly", "subdir"),
			setup: func(path string) func() {
				parentDir := filepath.Dir(path)
				os.MkdirAll(parentDir, 0755)
				makeReadOnly(t, parentDir)
				return func() {
					restoreWritePermissions(t, parentDir)
				}
			},
			wantError: true,
			errorMsg:  "cannot create directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setup(tt.path)
			defer cleanup()

			err := TestWriteToStorageDir(tt.path)

			if tt.wantError {
				if err == nil {
					t.Errorf("TestStorageDir(%q) expected error but got none", tt.path)
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("TestStorageDir(%q) error = %v, want error containing %q",
						tt.path, err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("TestStorageDir(%q) unexpected error: %v", tt.path, err)
				}
				// Verify directory was created
				if _, err := os.Stat(tt.path); err != nil {
					t.Errorf("TestStorageDir(%q) didn't create directory: %v", tt.path, err)
				}
			}
		})
	}
}

// TestCreateStorageDir tests the CreateStorageDir function
func TestCreateStorageDir(t *testing.T) {
	tempDir := createTempTestDir(t, "create-storage-")

	tests := []struct {
		name      string
		input     string
		setup     func() func()
		wantError bool
		errorMsg  string
		validate  func(t *testing.T, path string)
	}{
		{
			name:  "valid directory creation",
			input: filepath.Join(tempDir, "valid-storage"),
			setup: func() func() {
				return func() {}
			},
			wantError: false,
			validate: func(t *testing.T, path string) {
				expanded := ExpandPath(path)
				if _, err := os.Stat(expanded); err != nil {
					t.Errorf("Directory not created: %v", err)
				}
			},
		},
		{
			name:  "empty input",
			input: "",
			setup: func() func() {
				return func() {}
			},
			wantError: true,
			errorMsg:  "invalid storage directory",
		},
		{
			name:  "path traversal attempt",
			input: "../../../tmp/malicious",
			setup: func() func() {
				return func() {}
			},
			wantError: true,
			errorMsg:  "invalid storage directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setup()
			defer cleanup()

			err := CreateStorageDir(tt.input)

			if tt.wantError {
				if err == nil {
					t.Errorf("CreateStorageDir(%q) expected error but got none", tt.input)
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("CreateStorageDir(%q) error = %v, want error containing %q",
						tt.input, err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("CreateStorageDir(%q) unexpected error: %v", tt.input, err)
				}
				if tt.validate != nil {
					tt.validate(t, tt.input)
				}
			}
		})
	}
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
			result := isReservedDirectory(tt.path)
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

// TestPathTraversalSecurity tests security aspects of helper functions
func TestPathTraversalSecurity(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		function string // which function to test
	}{
		{
			name:     "basic path traversal",
			input:    "../../../etc/passwd",
			function: "validate",
		},
		{
			name:     "windows path traversal",
			input:    "..\\..\\..\\Windows\\System32",
			function: "validate",
		},
		{
			name:     "mixed separators",
			input:    "../..\\../etc/passwd",
			function: "validate",
		},
		{
			name:     "url encoded traversal",
			input:    "..%2F..%2F..%2Fetc%2Fpasswd",
			function: "validate",
		},
		{
			name:     "double encoded",
			input:    "..%252F..%252F..%252Fetc%252Fpasswd",
			function: "validate",
		},
		{
			name:     "unicode traversal",
			input:    "..∕..∕..∕etc∕passwd",
			function: "validate",
		},
		{
			name:     "home directory traversal",
			input:    "~/../../etc/shadow",
			function: "validate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error

			switch tt.function {
			case "validate":
				err = ValidateStorageDir(tt.input)
			case "create":
				err = CreateStorageDir(tt.input)
			default:
				t.Fatalf("unknown function: %s", tt.function)
			}

			if err == nil {
				t.Errorf("Expected path traversal attack to be rejected for input: %s", tt.input)
			}

			// Ensure error message mentions path traversal or similar security concern
			errMsg := strings.ToLower(err.Error())
			if !strings.Contains(errMsg, "traversal") &&
				!strings.Contains(errMsg, "invalid") &&
				!strings.Contains(errMsg, "not allowed") {
				t.Errorf("Error message should indicate security issue, got: %s", err.Error())
			}
		})
	}
}

// TestSymlinkSecurity tests symlink-related security
func TestSymlinkSecurity(t *testing.T) {
	if isWindows() {
		t.Skip("Symlink security tests require Unix-like system")
	}

	tempDir := createTempTestDir(t, "symlink-test-")

	tests := []struct {
		name  string
		setup func() string // returns path to test
	}{
		{
			name: "symlink to system directory",
			setup: func() string {
				linkPath := filepath.Join(tempDir, "evil-link")
				CreateSymlink(t, "/etc", linkPath)
				return linkPath
			},
		},
		{
			name: "symlink chain escape",
			setup: func() string {
				// Create chain: link1 -> link2 -> /etc
				link1 := filepath.Join(tempDir, "link1")
				link2 := filepath.Join(tempDir, "link2")
				CreateSymlink(t, "/etc", link2)
				CreateSymlink(t, link2, link1)
				return link1
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testPath := tt.setup()

			// Test that operations on symlinked paths are handled securely
			err := ValidateStorageDir(testPath)
			if err == nil {
				t.Errorf("Expected symlink to be rejected or handled securely")
			}
		})
	}
}

// TestErrorHandlingAndRecovery tests error handling scenarios
func TestErrorHandlingAndRecovery(t *testing.T) {
	tempDir := createTempTestDir(t, "error-test-")

	tests := []struct {
		name     string
		setup    func() (string, func()) // returns test path and cleanup
		testFunc func(string) error
		wantErr  bool
	}{
		{
			name: "test file cleanup failure",
			setup: func() (string, func()) {
				testPath := filepath.Join(tempDir, "cleanup-test")
				os.MkdirAll(testPath, 0755)
				return testPath, func() {}
			},
			testFunc: func(path string) error {
				// This should succeed but may have cleanup warnings
				return TestWriteToStorageDir(path)
			},
			wantErr: false,
		},
		{
			name: "partial directory creation",
			setup: func() (string, func()) {
				parentPath := filepath.Join(tempDir, "partial-test")
				testPath := filepath.Join(parentPath, "subdir")
				os.MkdirAll(parentPath, 0755)
				makeReadOnly(t, parentPath)
				return testPath, func() {
					restoreWritePermissions(t, parentPath)
				}
			},
			testFunc: func(path string) error {
				return TestWriteToStorageDir(path)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testPath, cleanup := tt.setup()
			defer cleanup()

			err := tt.testFunc(testPath)

			if tt.wantErr && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestPerformanceAndConcurrency tests performance aspects
func TestPerformanceAndConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance tests in short mode")
	}

	tempDir := createTempTestDir(t, "perf-test-")

	t.Run("deep directory creation", func(t *testing.T) {
		// Test creating deeply nested directory structure
		parts := make([]string, 20) // 20 levels deep
		for i := range parts {
			parts[i] = fmt.Sprintf("level-%d", i)
		}
		deepPath := filepath.Join(append([]string{tempDir}, parts...)...)

		start := time.Now()
		err := TestWriteToStorageDir(deepPath)
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Deep directory creation failed: %v", err)
		}

		// Should complete in reasonable time (adjust threshold as needed)
		if duration > time.Second*2 {
			t.Errorf("Deep directory creation too slow: %v", duration)
		}
	})

	t.Run("concurrent directory operations", func(t *testing.T) {
		const numGoroutines = 10
		results := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				path := filepath.Join(tempDir, fmt.Sprintf("concurrent-%d", id))
				results <- TestWriteToStorageDir(path)
			}(i)
		}

		for i := 0; i < numGoroutines; i++ {
			err := <-results
			if err != nil {
				t.Errorf("Concurrent operation %d failed: %v", i, err)
			}
		}
	})
}

// Benchmark tests for performance monitoring

func BenchmarkValidateStorageDir(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "benchmark-")
	defer os.RemoveAll(tempDir)

	testPath := filepath.Join(tempDir, "benchmark-test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateStorageDir(testPath)
	}
}

func BenchmarkTestStorageDir(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "benchmark-")
	defer os.RemoveAll(tempDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testPath := filepath.Join(tempDir, fmt.Sprintf("benchmark-%d", i))
		TestWriteToStorageDir(testPath)
	}
}

func BenchmarkIsReservedDirectory(b *testing.B) {
	testPaths := []string{
		"/etc",
		"/tmp",
		"C:\\Windows",
		"C:\\Users",
		"/home/user/documents",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range testPaths {
			isReservedDirectory(path)
		}
	}
}

func BenchmarkGetReservedDirectories(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getReservedDirectories()
	}
}
