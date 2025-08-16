package fileops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Test helpers

func createTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "fileops_test_")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	return dir
}

func createTestFile(t *testing.T, dir, filename, content string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file %s: %v", path, err)
	}
	return path
}

func readFileContent(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}
	return string(content)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Tests for AtomicCopy

func TestAtomicCopy(t *testing.T) {
	srcDir := createTempDir(t)
	defer os.RemoveAll(srcDir)
	destDir := createTempDir(t)
	defer os.RemoveAll(destDir)

	t.Run("basic copy operation", func(t *testing.T) {
		content := "Hello, atomic copy world!"
		srcPath := createTestFile(t, srcDir, "source.txt", content)
		destPath := filepath.Join(destDir, "destination.txt")

		err := AtomicCopy(srcPath, destPath)
		if err != nil {
			t.Fatalf("AtomicCopy failed: %v", err)
		}

		if !fileExists(destPath) {
			t.Error("Destination file was not created")
		}

		copiedContent := readFileContent(t, destPath)
		if copiedContent != content {
			t.Errorf("Content mismatch. Expected %q, got %q", content, copiedContent)
		}
	})

	t.Run("overwrite existing file", func(t *testing.T) {
		originalContent := "Original content"
		newContent := "New content"

		srcPath := createTestFile(t, srcDir, "new_source.txt", newContent)
		destPath := createTestFile(t, destDir, "existing.txt", originalContent)

		err := AtomicCopy(srcPath, destPath)
		if err != nil {
			t.Fatalf("AtomicCopy failed: %v", err)
		}

		copiedContent := readFileContent(t, destPath)
		if copiedContent != newContent {
			t.Errorf("Content not overwritten. Expected %q, got %q", newContent, copiedContent)
		}
	})

	t.Run("large file copy", func(t *testing.T) {
		largeContent := strings.Repeat("Large file content line.\n", 10000)
		srcPath := createTestFile(t, srcDir, "large.txt", largeContent)
		destPath := filepath.Join(destDir, "large_copy.txt")

		start := time.Now()
		err := AtomicCopy(srcPath, destPath)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("AtomicCopy failed: %v", err)
		}

		copiedContent := readFileContent(t, destPath)
		if copiedContent != largeContent {
			t.Error("Large file content mismatch")
		}

		t.Logf("Copied %d bytes in %v", len(largeContent), duration)
	})

	t.Run("empty file copy", func(t *testing.T) {
		srcPath := createTestFile(t, srcDir, "empty.txt", "")
		destPath := filepath.Join(destDir, "empty_copy.txt")

		err := AtomicCopy(srcPath, destPath)
		if err != nil {
			t.Fatalf("AtomicCopy failed: %v", err)
		}

		copiedContent := readFileContent(t, destPath)
		if copiedContent != "" {
			t.Errorf("Expected empty content, got %q", copiedContent)
		}
	})
}

func TestAtomicCopyErrors(t *testing.T) {
	srcDir := createTempDir(t)
	defer os.RemoveAll(srcDir)
	destDir := createTempDir(t)
	defer os.RemoveAll(destDir)

	t.Run("non-existent source file", func(t *testing.T) {
		srcPath := filepath.Join(srcDir, "nonexistent.txt")
		destPath := filepath.Join(destDir, "dest.txt")

		err := AtomicCopy(srcPath, destPath)
		if err == nil {
			t.Error("Expected error for non-existent source file")
		}

		if !strings.Contains(err.Error(), "failed to open source file") {
			t.Errorf("Expected 'failed to open source file' error, got: %v", err)
		}
	})

	t.Run("non-existent destination directory", func(t *testing.T) {
		srcPath := createTestFile(t, srcDir, "source.txt", "content")
		destPath := filepath.Join(destDir, "nonexistent", "dest.txt")

		err := AtomicCopy(srcPath, destPath)
		if err == nil {
			t.Error("Expected error for non-existent destination directory")
		}

		if !strings.Contains(err.Error(), "failed to create temporary file") {
			t.Errorf("Expected 'failed to create temporary file' error, got: %v", err)
		}
	})

	t.Run("source is directory", func(t *testing.T) {
		srcPath := createTempDir(t)
		defer os.RemoveAll(srcPath)
		destPath := filepath.Join(destDir, "dest.txt")

		err := AtomicCopy(srcPath, destPath)
		if err == nil {
			t.Error("Expected error when source is directory")
		}
	})
}

func TestAtomicCopyAtomicity(t *testing.T) {
	srcDir := createTempDir(t)
	defer os.RemoveAll(srcDir)
	destDir := createTempDir(t)
	defer os.RemoveAll(destDir)

	t.Run("no temp files left after successful copy", func(t *testing.T) {
		content := "Test content for atomicity"
		srcPath := createTestFile(t, srcDir, "atomic_source.txt", content)
		destPath := filepath.Join(destDir, "atomic_dest.txt")

		err := AtomicCopy(srcPath, destPath)
		if err != nil {
			t.Fatalf("AtomicCopy failed: %v", err)
		}

		// Check for any .tmp files in destination directory
		entries, err := os.ReadDir(destDir)
		if err != nil {
			t.Fatalf("Failed to read destination directory: %v", err)
		}

		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".tmp") {
				t.Errorf("Found temp file after successful copy: %s", entry.Name())
			}
		}
	})
}

// Tests for EnsureDirectoryExists

func TestEnsureDirectoryExists(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	t.Run("create single directory", func(t *testing.T) {
		dirPath := filepath.Join(tempDir, "single_dir")

		err := EnsureDirectoryExists(dirPath)
		if err != nil {
			t.Fatalf("EnsureDirectoryExists failed: %v", err)
		}

		info, err := os.Stat(dirPath)
		if err != nil {
			t.Fatalf("Directory was not created: %v", err)
		}

		if !info.IsDir() {
			t.Error("Created path is not a directory")
		}
	})

	t.Run("create nested directories", func(t *testing.T) {
		dirPath := filepath.Join(tempDir, "nested", "deep", "directory")

		err := EnsureDirectoryExists(dirPath)
		if err != nil {
			t.Fatalf("EnsureDirectoryExists failed: %v", err)
		}

		info, err := os.Stat(dirPath)
		if err != nil {
			t.Fatalf("Nested directory was not created: %v", err)
		}

		if !info.IsDir() {
			t.Error("Created nested path is not a directory")
		}
	})

	t.Run("directory already exists", func(t *testing.T) {
		dirPath := filepath.Join(tempDir, "existing_dir")

		// Create directory first
		if err := os.Mkdir(dirPath, 0755); err != nil {
			t.Fatalf("Failed to create initial directory: %v", err)
		}

		// Should not error when directory exists
		err := EnsureDirectoryExists(dirPath)
		if err != nil {
			t.Errorf("EnsureDirectoryExists failed on existing directory: %v", err)
		}
	})

	t.Run("check directory permissions", func(t *testing.T) {
		dirPath := filepath.Join(tempDir, "perm_dir")

		err := EnsureDirectoryExists(dirPath)
		if err != nil {
			t.Fatalf("EnsureDirectoryExists failed: %v", err)
		}

		info, err := os.Stat(dirPath)
		if err != nil {
			t.Fatalf("Directory was not created: %v", err)
		}

		expectedPerm := os.FileMode(0755)
		if info.Mode().Perm() != expectedPerm {
			t.Errorf("Directory permissions incorrect. Expected %v, got %v", expectedPerm, info.Mode().Perm())
		}
	})
}

func TestEnsureDirectoryExistsErrors(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	t.Run("file exists with same name", func(t *testing.T) {
		filePath := createTestFile(t, tempDir, "file_blocking_dir", "content")

		err := EnsureDirectoryExists(filePath)
		if err == nil {
			t.Error("Expected error when file exists with same name as directory")
		}
	})
}
