package filemanager

import (
	"os"
	"path/filepath"
	"rulem/internal/logging"
	"runtime"
	"strings"
	"testing"
)

// File and Directory Operations

// createTempTestDir creates a temporary directory with automatic cleanup
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

// createTestFile creates a test file with specified content
func createTestFile(t *testing.T, dir, filename, content string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file %s: %v", path, err)
	}
	return path
}

// createTestDir creates a test directory
func createTestDir(t *testing.T, dir, dirname string) string {
	t.Helper()
	path := filepath.Join(dir, dirname)
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("Failed to create test directory %s: %v", path, err)
	}
	return path
}

// createTempDirStructure creates a complex directory structure from a map
func createTempDirStructure(t *testing.T, structure map[string]string) string {
	t.Helper()

	tempDir := createTempTestDir(t, "structure_test_")

	for path, content := range structure {
		fullPath := filepath.Join(tempDir, path)

		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create parent dirs for %s: %v", path, err)
		}

		// Create file or directory
		if strings.HasSuffix(path, "/") {
			// It's a directory
			if err := os.MkdirAll(fullPath, 0755); err != nil {
				t.Fatalf("Failed to create directory %s: %v", path, err)
			}
		} else {
			// It's a file
			if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to create file %s: %v", path, err)
			}
		}
	}

	return tempDir
}

// File System Checks

// fileExists checks if a file or directory exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// readFileContent reads and returns file content
func readFileContent(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}
	return string(content)
}

// Permission Management

// Directory Navigation

// changeToDir changes working directory with automatic cleanup
func changeToDir(t *testing.T, dir string) func() {
	t.Helper()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Failed to change to directory %s: %v", dir, err)
	}

	return func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("Failed to restore original directory: %v", err)
		}
	}
}

// Platform and System Utilities

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

// Test Object Creation

// createTestLogger creates a test logger instance
func createTestLogger() *logging.AppLogger {
	logger, _ := logging.NewTestLogger()
	return logger
}

// stringPtr creates a pointer to a string (helper for optional parameters)
func stringPtr(s string) *string {
	return &s
}

// Symlink Operations (moved from helpers.go for consolidation)

// createTestSymlink creates a symbolic link with platform-aware error handling
func createTestSymlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		if isWindows() {
			t.Skipf("symlink creation failed on Windows: %v", err)
		}
		t.Fatalf("failed to create symlink: %v", err)
	}
}
