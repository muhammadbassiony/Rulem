package fileops

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
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

		if !strings.Contains(err.Error(), "failed to open destination directory") {
			t.Errorf("Expected 'failed to open destination directory' error, got: %v", err)
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

	t.Run("destination path has no file name", func(t *testing.T) {
		srcPath := createTestFile(t, srcDir, "named_source.txt", "content")

		err := AtomicCopy(srcPath, destDir+string(os.PathSeparator))
		if err == nil {
			t.Fatal("Expected error for destination path without a file name")
		}
		if !strings.Contains(err.Error(), "no file name") {
			t.Errorf("Expected 'no file name' error, got: %v", err)
		}
	})
}

// TestAtomicCopyPermissions verifies the destination inherits the source's
// permission bits instead of being forced to a hardcoded, world-readable mode.
func TestAtomicCopyPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits are not meaningful on Windows")
	}

	srcDir := createTempDir(t)
	defer os.RemoveAll(srcDir)
	destDir := createTempDir(t)
	defer os.RemoveAll(destDir)

	for _, mode := range []os.FileMode{0600, 0640, 0644} {
		t.Run(mode.String(), func(t *testing.T) {
			srcPath := filepath.Join(srcDir, "perm_"+mode.String()+".txt")
			if err := os.WriteFile(srcPath, []byte("secret"), mode); err != nil {
				t.Fatalf("Failed to create source file: %v", err)
			}
			// WriteFile is subject to umask, so assert against what landed on disk.
			srcInfo, err := os.Stat(srcPath)
			if err != nil {
				t.Fatalf("Failed to stat source file: %v", err)
			}

			destPath := filepath.Join(destDir, "perm_"+mode.String()+".txt")
			if err := AtomicCopy(srcPath, destPath); err != nil {
				t.Fatalf("AtomicCopy failed: %v", err)
			}

			destInfo, err := os.Stat(destPath)
			if err != nil {
				t.Fatalf("Failed to stat destination file: %v", err)
			}
			if destInfo.Mode().Perm() != srcInfo.Mode().Perm() {
				t.Errorf("Permissions not preserved. Source %v, destination %v",
					srcInfo.Mode().Perm(), destInfo.Mode().Perm())
			}
		})
	}
}

// TestAtomicCopySymlinkedTempPath covers the attack the fixed temporary-name
// implementation allowed: an attacker who can write in the destination
// directory pre-creates the predictable temp path as a symlink to a file
// outside it, and the copy follows the link and truncates the victim.
func TestAtomicCopySymlinkedTempPath(t *testing.T) {
	srcDir := createTempDir(t)
	defer os.RemoveAll(srcDir)
	destDir := createTempDir(t)
	defer os.RemoveAll(destDir)
	victimDir := createTempDir(t)
	defer os.RemoveAll(victimDir)

	victimContent := "important victim data"
	victimPath := createTestFile(t, victimDir, "victim.txt", victimContent)

	srcPath := createTestFile(t, srcDir, "attacker.txt", "attacker controlled content")
	destPath := filepath.Join(destDir, "dest.txt")

	// The pre-1.0 implementation used exactly this path, without O_EXCL.
	if err := os.Symlink(victimPath, destPath+".tmp"); err != nil {
		t.Fatalf("Failed to plant symlink: %v", err)
	}

	if err := AtomicCopy(srcPath, destPath); err != nil {
		t.Fatalf("AtomicCopy failed: %v", err)
	}

	if got := readFileContent(t, victimPath); got != victimContent {
		t.Errorf("Victim file outside the destination directory was overwritten: %q", got)
	}
}

// TestAtomicCopyConcurrent verifies that concurrent copies to the same
// destination no longer share one temporary path and corrupt each other.
func TestAtomicCopyConcurrent(t *testing.T) {
	srcDir := createTempDir(t)
	defer os.RemoveAll(srcDir)
	destDir := createTempDir(t)
	defer os.RemoveAll(destDir)

	const copies = 8
	contents := make([]string, copies)
	sources := make([]string, copies)
	for i := range copies {
		contents[i] = strings.Repeat("content from source "+strconv.Itoa(i)+"\n", 2000)
		sources[i] = createTestFile(t, srcDir, "src_"+strconv.Itoa(i)+".txt", contents[i])
	}

	destPath := filepath.Join(destDir, "contended.txt")

	var wg sync.WaitGroup
	errs := make([]error, copies)
	for i := range copies {
		wg.Go(func() {
			errs[i] = AtomicCopy(sources[i], destPath)
		})
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("Concurrent AtomicCopy %d failed: %v", i, err)
		}
	}

	// Whichever copy won the race, the destination must be one intact source,
	// never an interleaving of several.
	got := readFileContent(t, destPath)
	if !slices.Contains(contents, got) {
		t.Errorf("Destination is not an intact copy of any source (%d bytes)", len(got))
	}

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
