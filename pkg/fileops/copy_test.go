package fileops

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
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

// TestAtomicCopyTempSymlinkAttack verifies that AtomicCopy does not follow a
// pre-created "<dest>.tmp" symlink and truncate/overwrite the victim it points
// at. Against the old implementation (which opened exactly destPath+".tmp" with
// O_CREATE|O_WRONLY|O_TRUNC, no O_EXCL) the open followed the symlink and
// clobbered the victim file.
func TestAtomicCopyTempSymlinkAttack(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Symlink test not supported on Windows")
	}

	srcDir := createTempDir(t)
	defer os.RemoveAll(srcDir)
	destDir := createTempDir(t)
	defer os.RemoveAll(destDir)
	victimDir := createTempDir(t)
	defer os.RemoveAll(victimDir)

	const victimContent = "VICTIM - must not be modified"
	victimPath := createTestFile(t, victimDir, "victim.txt", victimContent)

	srcPath := createTestFile(t, srcDir, "source.txt", "new copied content")
	destPath := filepath.Join(destDir, "dest.txt")

	// Attacker pre-creates the predictable temp path as a symlink to the victim.
	attackLink := destPath + ".tmp"
	if err := os.Symlink(victimPath, attackLink); err != nil {
		t.Fatalf("Failed to create attack symlink: %v", err)
	}

	if err := AtomicCopy(srcPath, destPath); err != nil {
		t.Fatalf("AtomicCopy failed: %v", err)
	}

	// The victim file must be completely untouched.
	if got := readFileContent(t, victimPath); got != victimContent {
		t.Errorf("Victim file was modified via temp symlink: got %q, want %q", got, victimContent)
	}

	// The copy itself must have succeeded to the real destination.
	if got := readFileContent(t, destPath); got != "new copied content" {
		t.Errorf("Destination content = %q, want %q", got, "new copied content")
	}
}

// TestAtomicCopyConcurrentSameDest verifies that many concurrent AtomicCopy
// calls targeting the same destination file complete without error or content
// corruption. The old implementation shared a single "<dest>.tmp" path across
// all calls, so concurrent writers clobbered each other's temp file and racing
// renames could fail.
func TestAtomicCopyConcurrentSameDest(t *testing.T) {
	srcDir := createTempDir(t)
	defer os.RemoveAll(srcDir)
	destDir := createTempDir(t)
	defer os.RemoveAll(destDir)

	const n = 12
	sources := make([]string, n)
	contents := make(map[string]bool, n)
	for i := 0; i < n; i++ {
		content := fmt.Sprintf("content-from-source-%02d", i)
		sources[i] = createTestFile(t, srcDir, fmt.Sprintf("src_%02d.txt", i), content)
		contents[content] = true
	}

	destPath := filepath.Join(destDir, "shared_dest.txt")

	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(src string) {
			defer wg.Done()
			if err := AtomicCopy(src, destPath); err != nil {
				errs <- err
			}
		}(sources[i])
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("AtomicCopy failed under concurrency: %v", err)
	}

	// Final destination must equal exactly one full source's content - never a
	// truncated or interleaved mix from a shared temp file.
	final := readFileContent(t, destPath)
	if !contents[final] {
		t.Errorf("Destination content is corrupted/interleaved: %q", final)
	}

	// No stray temp files should remain in the destination directory.
	entries, err := os.ReadDir(destDir)
	if err != nil {
		t.Fatalf("Failed to read destination directory: %v", err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".tmp") {
			t.Errorf("Found leftover temp file after concurrent copies: %s", entry.Name())
		}
	}
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
