package filemanager

import (
	"os"
	"path/filepath"
	"rulem/internal/logging"
	"rulem/pkg/fileops"
	"strings"
	"testing"
	"time"
)

// HELPERS

// createTempStorage creates a temporary directory for testing purposes.
func createTempStorage(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "filemanager_test_")
	if err != nil {
		t.Fatalf("Failed to create temp storage: %v", err)
	}
	return dir
}

// 1. Unit Tests for NewFileManager

func TestNewFileManager(t *testing.T) {
	logger := createTestLogger()

	t.Run("valid storage directory", func(t *testing.T) {
		storageDir := createTempStorage(t)
		defer os.RemoveAll(storageDir)

		fm, err := NewFileManager(storageDir, logger)
		if err != nil {
			t.Fatalf("NewFileManager failed with valid directory: %v", err)
		}

		if fm.storageDir != storageDir {
			t.Errorf("Expected storageDir %s, got %s", storageDir, fm.storageDir)
		}
	})

	t.Run("non-existent storage directory", func(t *testing.T) {
		nonExistentDir := "/definitely/does/not/exist"

		_, err := NewFileManager(nonExistentDir, logger)
		if err == nil {
			t.Error("Expected error for non-existent directory")
		}

		if !strings.Contains(err.Error(), "parent directory does not exist") {
			t.Errorf("Expected 'parent directory does not exist' error, got: %v", err)
		}
	})

	t.Run("invalid storage directory path", func(t *testing.T) {
		// Test with path traversal
		invalidDir := "../../../etc"

		_, err := NewFileManager(invalidDir, logger)
		if err == nil {
			t.Error("Expected error for invalid directory path")
		}
	})
}

// 2.1 Happy Path Tests

func TestHappyPath(t *testing.T) {
	logger := createTestLogger()
	storageDir := createTempStorage(t)
	defer os.RemoveAll(storageDir)

	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("Failed to create FileManager: %v", err)
	}

	tempDir := createTempStorage(t)
	defer os.RemoveAll(tempDir)

	testContent := "# Test markdown file\nThis is test content."

	t.Run("copy with original filename", func(t *testing.T) {
		srcPath := createTestFile(t, tempDir, "test.md", testContent)

		destPath, err := fm.CopyFileToStorage(srcPath, nil, false)
		if err != nil {
			t.Fatalf("CopyFileToStorage failed: %v", err)
		}

		expectedDest := filepath.Join(storageDir, "test.md")
		if destPath != expectedDest {
			t.Errorf("Expected dest path %s, got %s", expectedDest, destPath)
		}

		if !fileExists(destPath) {
			t.Error("Destination file was not created")
		}

		content := readFileContent(t, destPath)
		if content != testContent {
			t.Errorf("Content mismatch. Expected %q, got %q", testContent, content)
		}
	})

	t.Run("copy with new filename", func(t *testing.T) {
		srcPath := createTestFile(t, tempDir, "source.md", testContent)
		newName := "renamed.md"

		destPath, err := fm.CopyFileToStorage(srcPath, &newName, false)
		if err != nil {
			t.Fatalf("CopyFileToStorage failed: %v", err)
		}

		expectedDest := filepath.Join(storageDir, "renamed.md")
		if destPath != expectedDest {
			t.Errorf("Expected dest path %s, got %s", expectedDest, destPath)
		}

		content := readFileContent(t, destPath)
		if content != testContent {
			t.Errorf("Content mismatch. Expected %q, got %q", testContent, content)
		}
	})
}

// 2.2 Source File Validation Tests

func TestSourceValidation(t *testing.T) {
	logger := createTestLogger()
	storageDir := createTempStorage(t)
	defer os.RemoveAll(storageDir)

	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("Failed to create FileManager: %v", err)
	}

	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) string
		expectError string
	}{
		{
			name: "non-existent source file",
			setupFunc: func(t *testing.T) string {
				return "/definitely/does/not/exist.md"
			},
			expectError: "does not exist",
		},
		{
			name: "source is directory",
			setupFunc: func(t *testing.T) string {
				tempDir := createTempStorage(t)
				t.Cleanup(func() { os.RemoveAll(tempDir) })
				return createTestDir(t, tempDir, "testdir")
			},
			expectError: "source is a directory",
		},
		{
			name: "unreadable source file",
			setupFunc: func(t *testing.T) string {
				tempDir := createTempStorage(t)
				t.Cleanup(func() { os.RemoveAll(tempDir) })
				srcPath := createTestFile(t, tempDir, "unreadable.md", "content")
				if err := os.Chmod(srcPath, 0000); err != nil {
					t.Skip("Cannot change file permissions")
				}
				t.Cleanup(func() { os.Chmod(srcPath, 0644) }) // Restore for cleanup
				return srcPath
			},
			expectError: "failed to open source file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcPath := tt.setupFunc(t)

			_, err := fm.CopyFileToStorage(srcPath, nil, false)
			if err == nil {
				t.Error("Expected error but got none")
			}

			expectedErrors := []string{tt.expectError}
			if tt.name == "source is directory" {
				expectedErrors = []string{"source is a directory", "path is a directory, not a file"}
			} else if tt.name == "unreadable source file" {
				expectedErrors = []string{"failed to open source file", "file is not readable"}
			}

			foundExpected := false
			for _, expected := range expectedErrors {
				if strings.Contains(err.Error(), expected) {
					foundExpected = true
					break
				}
			}
			if !foundExpected {
				t.Errorf("Expected error containing one of %v, got: %v", expectedErrors, err)
			}
		})
	}
}

// 2.3 Filename Parameter Tests

func TestFilenameValidation(t *testing.T) {
	logger := createTestLogger()
	storageDir := createTempStorage(t)
	defer os.RemoveAll(storageDir)

	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("Failed to create FileManager: %v", err)
	}

	tempDir := createTempStorage(t)
	defer os.RemoveAll(tempDir)
	srcPath := createTestFile(t, tempDir, "test.md", "content")

	tests := []struct {
		name        string
		newFileName *string
		expectError bool
		errorText   string
	}{
		{
			name:        "valid simple filename",
			newFileName: stringPtr("valid.md"),
			expectError: false,
		},
		{
			name:        "valid filename with spaces",
			newFileName: stringPtr("file with spaces.md"),
			expectError: false,
		},
		{
			name:        "path traversal with ../",
			newFileName: stringPtr("../../../etc/passwd"),
			expectError: false,
		},
		{
			name:        "filename with forward slash",
			newFileName: stringPtr("folder/file.md"),
			expectError: false,
		},
		{
			name:        "filename with backslash",
			newFileName: stringPtr("folder\\file.md"),
			expectError: false,
		},
		{
			name:        "empty filename",
			newFileName: stringPtr(""),
			expectError: true,
			errorText:   "invalid filename",
		},
		{
			name:        "just dots",
			newFileName: stringPtr(".."),
			expectError: true,
			errorText:   "invalid filename",
		},
		{
			name:        "single dot",
			newFileName: stringPtr("."),
			expectError: true,
			errorText:   "invalid filename",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := fm.CopyFileToStorage(srcPath, tt.newFileName, false)

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

// 2.4 Overwrite Behavior Tests

func TestOverwriteBehavior(t *testing.T) {
	logger := createTestLogger()
	storageDir := createTempStorage(t)
	defer os.RemoveAll(storageDir)

	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("Failed to create FileManager: %v", err)
	}

	tempDir := createTempStorage(t)
	defer os.RemoveAll(tempDir)

	// Create source files
	srcPath1 := createTestFile(t, tempDir, "source1.md", "content 1")
	srcPath2 := createTestFile(t, tempDir, "source2.md", "content 2")

	t.Run("overwrite=false with existing file should error", func(t *testing.T) {
		// First copy should succeed
		_, err := fm.CopyFileToStorage(srcPath1, nil, false)
		if err != nil {
			t.Fatalf("First copy failed: %v", err)
		}

		// Second copy to same destination should fail
		_, err = fm.CopyFileToStorage(srcPath2, stringPtr("source1.md"), false)
		if err == nil {
			t.Error("Expected error for existing file with overwrite=false")
		}

		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("Expected 'already exists' error, got: %v", err)
		}
	})

	t.Run("overwrite=true with existing file should succeed", func(t *testing.T) {
		fileName := "overwrite-test.md"

		// First copy
		_, err := fm.CopyFileToStorage(srcPath1, &fileName, false)
		if err != nil {
			t.Fatalf("First copy failed: %v", err)
		}

		// Verify initial content
		destPath := filepath.Join(storageDir, fileName)
		content := readFileContent(t, destPath)
		if content != "content 1" {
			t.Errorf("Expected 'content 1', got %q", content)
		}

		// Overwrite with second file
		_, err = fm.CopyFileToStorage(srcPath2, &fileName, true)
		if err != nil {
			t.Fatalf("Overwrite copy failed: %v", err)
		}

		// Verify content was replaced
		content = readFileContent(t, destPath)
		if content != "content 2" {
			t.Errorf("Expected 'content 2', got %q", content)
		}
	})

	t.Run("overwrite=false with no conflict should succeed", func(t *testing.T) {
		uniqueName := "unique-file.md"

		_, err := fm.CopyFileToStorage(srcPath1, &uniqueName, false)
		if err != nil {
			t.Errorf("Copy with no conflict failed: %v", err)
		}

		destPath := filepath.Join(storageDir, uniqueName)
		if !fileExists(destPath) {
			t.Error("File was not created")
		}
	})
}

// 2.5 Security Tests

func TestSecurity(t *testing.T) {
	logger := createTestLogger()
	storageDir := createTempStorage(t)
	defer os.RemoveAll(storageDir)

	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("Failed to create FileManager: %v", err)
	}

	tempDir := createTempStorage(t)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) (string, *string)
		expectError bool
		errorText   string
	}{
		{
			name: "path traversal in filename",
			setupFunc: func(t *testing.T) (string, *string) {
				srcPath := createTestFile(t, tempDir, "test.md", "content")
				maliciousName := "../../../etc/passwd"
				return srcPath, &maliciousName
			},
			expectError: false,
		},
		{
			name: "encoded path traversal",
			setupFunc: func(t *testing.T) (string, *string) {
				srcPath := createTestFile(t, tempDir, "test.md", "content")
				maliciousName := "..%2F..%2F..%2Fetc%2Fpasswd"
				return srcPath, &maliciousName
			},
			expectError: false,
		},
		{
			name: "symlink to restricted file",
			setupFunc: func(t *testing.T) (string, *string) {
				// Create a symlink to /etc/passwd (if it exists)
				linkPath := filepath.Join(tempDir, "malicious_link")
				createTestSymlink(t, "/etc/passwd", linkPath)
				return linkPath, nil
			},
			expectError: true,
			errorText:   "symlink security check failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcPath, newFileName := tt.setupFunc(t)

			_, err := fm.CopyFileToStorage(srcPath, newFileName, false)

			if tt.expectError {
				if err == nil {
					t.Error("Expected security error but got none")
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

// 2.6 Atomic Copy Tests

func TestAtomicCopy(t *testing.T) {
	logger := createTestLogger()
	storageDir := createTempStorage(t)
	defer os.RemoveAll(storageDir)

	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("Failed to create FileManager: %v", err)
	}

	tempDir := createTempStorage(t)
	defer os.RemoveAll(tempDir)

	t.Run("no temp files left after successful copy", func(t *testing.T) {
		srcPath := createTestFile(t, tempDir, "atomic-test.md", "atomic content")

		_, err := fm.CopyFileToStorage(srcPath, nil, false)
		if err != nil {
			t.Fatalf("Copy failed: %v", err)
		}

		// Check for any .tmp files in storage directory
		entries, err := os.ReadDir(storageDir)
		if err != nil {
			t.Fatalf("Failed to read storage directory: %v", err)
		}

		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".tmp") {
				t.Errorf("Found temp file after successful copy: %s", entry.Name())
			}
		}
	})

	t.Run("file appears atomically", func(t *testing.T) {
		srcPath := createTestFile(t, tempDir, "atomic-test2.md", "atomic content 2")
		destName := "atomic-dest.md"

		_, err := fm.CopyFileToStorage(srcPath, &destName, false)
		if err != nil {
			t.Fatalf("Copy failed: %v", err)
		}

		destPath := filepath.Join(storageDir, destName)
		content := readFileContent(t, destPath)
		expectedContent := "atomic content 2"

		if content != expectedContent {
			t.Errorf("Content mismatch. Expected %q, got %q", expectedContent, content)
		}
	})
}

// 3.3 Performance Tests (basic)

func TestPerformance(t *testing.T) {
	logger := createTestLogger()
	storageDir := createTempStorage(t)
	defer os.RemoveAll(storageDir)

	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("Failed to create FileManager: %v", err)
	}

	tempDir := createTempStorage(t)
	defer os.RemoveAll(tempDir)

	t.Run("copy large file in reasonable time", func(t *testing.T) {
		// Create a moderately large file (1MB)
		largeContent := strings.Repeat("This is test content for performance testing.\n", 20000)
		srcPath := createTestFile(t, tempDir, "large-file.md", largeContent)

		start := time.Now()
		_, err := fm.CopyFileToStorage(srcPath, nil, false)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Copy failed: %v", err)
		}

		// Should complete within reasonable time (adjust as needed)
		if duration > 10*time.Second {
			t.Errorf("Copy took too long: %v", duration)
		}

		t.Logf("Copied %d bytes in %v", len(largeContent), duration)
	})

	t.Run("empty file copy", func(t *testing.T) {
		srcPath := createTestFile(t, tempDir, "empty.md", "")

		_, err := fm.CopyFileToStorage(srcPath, nil, false)
		if err != nil {
			t.Fatalf("Empty file copy failed: %v", err)
		}

		destPath := filepath.Join(storageDir, "empty.md")
		content := readFileContent(t, destPath)
		if content != "" {
			t.Errorf("Expected empty content, got %q", content)
		}
	})
}

// 4. Tests for CopyFileFromStorage

func TestCopyFileFromStorage(t *testing.T) {
	logger := createTestLogger()
	storageDir := createTempStorage(t)
	defer os.RemoveAll(storageDir)

	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("Failed to create FileManager: %v", err)
	}

	// Create test content in storage
	testContent := "# Test content from storage\nThis is test content."
	storageFilePath := createTestFile(t, storageDir, "storage-test.md", testContent)

	// Create a temporary working directory for CWD operations
	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	tempCwd := createTempStorage(t)
	defer os.RemoveAll(tempCwd)
	defer os.Chdir(originalCwd) // Restore original CWD

	if err := os.Chdir(tempCwd); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	t.Run("copy with original filename", func(t *testing.T) {
		destPath, err := fm.CopyFileFromStorage(storageFilePath, "copied-test.md", false)
		if err != nil {
			t.Fatalf("CopyFileFromStorage failed: %v", err)
		}

		// Check if the file is in the expected location by checking parent directory
		expectedDir, _ := filepath.EvalSymlinks(tempCwd)
		actualDir, _ := filepath.EvalSymlinks(filepath.Dir(destPath))
		expectedFile := filepath.Join(expectedDir, "copied-test.md")
		actualFile := filepath.Join(actualDir, filepath.Base(destPath))
		if actualFile != expectedFile {
			t.Errorf("Expected dest path %s, got %s", expectedFile, actualFile)
		}

		if !fileExists(destPath) {
			t.Error("Destination file was not created")
		}

		content := readFileContent(t, destPath)
		if content != testContent {
			t.Errorf("Content mismatch. Expected %q, got %q", testContent, content)
		}
	})

	t.Run("copy to subdirectory", func(t *testing.T) {
		destPath, err := fm.CopyFileFromStorage(storageFilePath, "subdir/nested-test.md", false)
		if err != nil {
			t.Fatalf("CopyFileFromStorage to subdirectory failed: %v", err)
		}

		// Check if the file is in the expected location
		expectedDir, _ := filepath.EvalSymlinks(filepath.Join(tempCwd, "subdir"))
		actualDir, _ := filepath.EvalSymlinks(filepath.Dir(destPath))
		expectedFile := filepath.Join(expectedDir, "nested-test.md")
		actualFile := filepath.Join(actualDir, filepath.Base(destPath))
		if actualFile != expectedFile {
			t.Errorf("Expected dest path %s, got %s", expectedFile, actualFile)
		}

		content := readFileContent(t, destPath)
		if content != testContent {
			t.Errorf("Content mismatch. Expected %q, got %q", testContent, content)
		}
	})

	t.Run("overwrite behavior", func(t *testing.T) {
		// First copy
		_, err := fm.CopyFileFromStorage(storageFilePath, "overwrite-test.md", false)
		if err != nil {
			t.Fatalf("First copy failed: %v", err)
		}

		// Second copy without overwrite should fail
		_, err = fm.CopyFileFromStorage(storageFilePath, "overwrite-test.md", false)
		if err == nil {
			t.Error("Expected error for existing file with overwrite=false")
		}

		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("Expected 'already exists' error, got: %v", err)
		}

		// Third copy with overwrite should succeed
		_, err = fm.CopyFileFromStorage(storageFilePath, "overwrite-test.md", true)
		if err != nil {
			t.Errorf("Overwrite copy failed: %v", err)
		}
	})
}

func TestCopyFileFromStorageValidation(t *testing.T) {
	logger := createTestLogger()
	storageDir := createTempStorage(t)
	defer os.RemoveAll(storageDir)

	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("Failed to create FileManager: %v", err)
	}

	// Create test file in storage
	validStorageFile := createTestFile(t, storageDir, "valid-file.md", "content")

	// Change to temp CWD
	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	tempCwd := createTempStorage(t)
	defer os.RemoveAll(tempCwd)
	defer os.Chdir(originalCwd)
	if err := os.Chdir(tempCwd); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	tests := []struct {
		name        string
		storagePath string
		destPath    string
		expectError bool
		errorText   string
	}{
		{
			name:        "non-existent storage file",
			storagePath: filepath.Join(storageDir, "does-not-exist.md"),
			destPath:    "dest.md",
			expectError: true,
			errorText:   "file does not exist",
		},
		{
			name:        "empty storage path",
			storagePath: "",
			destPath:    "dest.md",
			expectError: true,
			errorText:   "path is a directory, not a file",
		},
		{
			name:        "path outside storage directory",
			storagePath: "/etc/passwd",
			destPath:    "dest.md",
			expectError: true,
			errorText:   "file is not within base directory",
		},
		{
			name:        "empty destination path",
			storagePath: validStorageFile,
			destPath:    "",
			expectError: true,
			errorText:   "destination path cannot be empty",
		},
		{
			name:        "absolute destination path",
			storagePath: validStorageFile,
			destPath:    "/tmp/absolute.md",
			expectError: true,
			errorText:   "must be relative to current working directory",
		},
		{
			name:        "path traversal in destination",
			storagePath: validStorageFile,
			destPath:    "../escape.md",
			expectError: true,
			errorText:   "path traversal not allowed",
		},
		{
			name:        "valid relative subdirectory",
			storagePath: validStorageFile,
			destPath:    "sub/dir/file.md",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := fm.CopyFileFromStorage(tt.storagePath, tt.destPath, false)

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

// 5. Tests for CreateSymlinkFromStorage

func TestCreateSymlinkFromStorage(t *testing.T) {
	logger := createTestLogger()
	storageDir := createTempStorage(t)
	defer os.RemoveAll(storageDir)

	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("Failed to create FileManager: %v", err)
	}

	// Create test content in storage
	testContent := "# Symlink test content\nThis content should be accessible via symlink."
	storageFilePath := createTestFile(t, storageDir, "symlink-source.md", testContent)

	// Create a temporary working directory for CWD operations
	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	tempCwd := createTempStorage(t)
	defer os.RemoveAll(tempCwd)
	defer os.Chdir(originalCwd)

	if err := os.Chdir(tempCwd); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	t.Run("create symlink in CWD", func(t *testing.T) {
		linkPath, err := fm.CreateSymlinkFromStorage(storageFilePath, "linked-file.md", false)
		if err != nil {
			t.Fatalf("CreateSymlinkFromStorage failed: %v", err)
		}

		// Check if the symlink is in the expected location
		expectedDir, _ := filepath.EvalSymlinks(tempCwd)
		actualDir, _ := filepath.EvalSymlinks(filepath.Dir(linkPath))
		expectedFile := filepath.Join(expectedDir, "linked-file.md")
		actualFile := filepath.Join(actualDir, filepath.Base(linkPath))
		if actualFile != expectedFile {
			t.Errorf("Expected link path %s, got %s", expectedFile, actualFile)
		}

		// Verify symlink exists
		if !fileExists(linkPath) {
			t.Error("Symlink was not created")
		}

		// Verify it's actually a symlink
		isLink, err := fileops.IsSymlink(linkPath)
		if err != nil {
			t.Fatalf("Failed to check if path is symlink: %v", err)
		}
		if !isLink {
			t.Error("Created file is not a symlink")
		}

		// Verify content is accessible through symlink
		content := readFileContent(t, linkPath)
		if content != testContent {
			t.Errorf("Content mismatch through symlink. Expected %q, got %q", testContent, content)
		}

		// Verify symlink target is relative
		target, err := os.Readlink(linkPath)
		if err != nil {
			t.Fatalf("Failed to read symlink target: %v", err)
		}
		if filepath.IsAbs(target) {
			t.Errorf("Expected relative symlink target, got absolute: %s", target)
		}
	})

	t.Run("create symlink in subdirectory", func(t *testing.T) {
		linkPath, err := fm.CreateSymlinkFromStorage(storageFilePath, "subdir/deep/linked-file.md", false)
		if err != nil {
			t.Fatalf("CreateSymlinkFromStorage in subdirectory failed: %v", err)
		}

		// Check if the symlink is in the expected location
		expectedDir, _ := filepath.EvalSymlinks(filepath.Join(tempCwd, "subdir", "deep"))
		actualDir, _ := filepath.EvalSymlinks(filepath.Dir(linkPath))
		expectedFile := filepath.Join(expectedDir, "linked-file.md")
		actualFile := filepath.Join(actualDir, filepath.Base(linkPath))
		if actualFile != expectedFile {
			t.Errorf("Expected link path %s, got %s", expectedFile, actualFile)
		}

		// Verify content accessibility
		content := readFileContent(t, linkPath)
		if content != testContent {
			t.Errorf("Content mismatch through nested symlink. Expected %q, got %q", testContent, content)
		}

		// Resolve the symlink to verify it points to correct file
		resolvedTarget, err := filepath.EvalSymlinks(linkPath)
		if err != nil {
			t.Fatalf("Failed to resolve symlink: %v", err)
		}

		expectedTarget, _ := filepath.EvalSymlinks(storageFilePath)
		if resolvedTarget != expectedTarget {
			t.Errorf("Symlink doesn't resolve to storage file. Expected %s, got %s", expectedTarget, resolvedTarget)
		}
	})

	t.Run("overwrite behavior", func(t *testing.T) {
		// Create initial file
		initialFile := filepath.Join(tempCwd, "overwrite-target.md")
		if err := os.WriteFile(initialFile, []byte("initial content"), 0644); err != nil {
			t.Fatalf("Failed to create initial file: %v", err)
		}

		// Try to create symlink without overwrite - should fail
		_, err := fm.CreateSymlinkFromStorage(storageFilePath, "overwrite-target.md", false)
		if err == nil {
			t.Error("Expected error when creating symlink over existing file without overwrite")
		}

		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("Expected 'already exists' error, got: %v", err)
		}

		// Create symlink with overwrite - should succeed
		linkPath, err := fm.CreateSymlinkFromStorage(storageFilePath, "overwrite-target.md", true)
		if err != nil {
			t.Fatalf("CreateSymlinkFromStorage with overwrite failed: %v", err)
		}

		// Verify it's now a symlink
		isLink, err := fileops.IsSymlink(linkPath)
		if err != nil {
			t.Fatalf("Failed to check if path is symlink: %v", err)
		}
		if !isLink {
			t.Error("File was not replaced with symlink")
		}

		// Verify content comes from storage
		content := readFileContent(t, linkPath)
		if content != testContent {
			t.Errorf("Expected storage content, got: %q", content)
		}
	})

	t.Run("bidirectional sync test", func(t *testing.T) {
		// Create symlink
		linkPath, err := fm.CreateSymlinkFromStorage(storageFilePath, "bidirectional-test.md", false)
		if err != nil {
			t.Fatalf("CreateSymlinkFromStorage failed: %v", err)
		}

		// Modify content through symlink
		newContent := "# Modified through symlink\nThis was edited via the symlink."
		if err := os.WriteFile(linkPath, []byte(newContent), 0644); err != nil {
			t.Fatalf("Failed to write through symlink: %v", err)
		}

		// Verify storage file was modified
		storageContent := readFileContent(t, storageFilePath)
		if storageContent != newContent {
			t.Errorf("Storage file not updated. Expected %q, got %q", newContent, storageContent)
		}

		// Verify reading through symlink shows updated content
		linkContent := readFileContent(t, linkPath)
		if linkContent != newContent {
			t.Errorf("Symlink doesn't show updated content. Expected %q, got %q", newContent, linkContent)
		}
	})
}

func TestCreateSymlinkFromStorageValidation(t *testing.T) {
	logger := createTestLogger()
	storageDir := createTempStorage(t)
	defer os.RemoveAll(storageDir)

	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("Failed to create FileManager: %v", err)
	}

	// Create test file in storage
	validStorageFile := createTestFile(t, storageDir, "valid-source.md", "content")

	// Change to temp CWD
	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	tempCwd := createTempStorage(t)
	defer os.RemoveAll(tempCwd)
	defer os.Chdir(originalCwd)
	if err := os.Chdir(tempCwd); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	tests := []struct {
		name        string
		storagePath string
		destPath    string
		expectError bool
		errorText   string
	}{
		{
			name:        "non-existent storage file",
			storagePath: filepath.Join(storageDir, "missing-file.md"),
			destPath:    "link.md",
			expectError: true,
			errorText:   "file does not exist",
		},
		{
			name:        "empty storage path",
			storagePath: "",
			destPath:    "link.md",
			expectError: true,
			errorText:   "path is a directory, not a file",
		},
		{
			name:        "path outside storage directory",
			storagePath: "/etc/passwd",
			destPath:    "link.md",
			expectError: true,
			errorText:   "file is not within base directory",
		},
		{
			name:        "empty destination path",
			storagePath: validStorageFile,
			destPath:    "",
			expectError: true,
			errorText:   "destination path cannot be empty",
		},
		{
			name:        "absolute destination path",
			storagePath: validStorageFile,
			destPath:    "/tmp/absolute-link.md",
			expectError: true,
			errorText:   "must be relative to current working directory",
		},
		{
			name:        "path traversal in destination",
			storagePath: validStorageFile,
			destPath:    "../escape-link.md",
			expectError: true,
			errorText:   "path traversal not allowed",
		},
		{
			name:        "valid nested destination",
			storagePath: validStorageFile,
			destPath:    "deep/nested/link.md",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := fm.CreateSymlinkFromStorage(tt.storagePath, tt.destPath, false)

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

// Test for broken symlink detection in CopyFileFromStorage
func TestCopyFileFromStorage_BrokenSymlinkDetection(t *testing.T) {
	logger := createTestLogger()
	storageDir := createTempStorage(t)
	defer os.RemoveAll(storageDir)

	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("Failed to create FileManager: %v", err)
	}

	// Create test content in storage
	testContent := "# Test content for broken symlink copy test"
	storageFilePath := createTestFile(t, storageDir, "copy-source.md", testContent)

	// Create a temporary working directory
	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	tempCwd := createTempStorage(t)
	defer os.RemoveAll(tempCwd)
	defer os.Chdir(originalCwd)

	if err := os.Chdir(tempCwd); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	t.Run("detects broken symlink without overwrite", func(t *testing.T) {
		// Create a broken symlink (pointing to non-existent target)
		brokenLinkPath := filepath.Join(tempCwd, "broken-copy.md")
		if err := os.Symlink("nonexistent-copy-target.md", brokenLinkPath); err != nil {
			t.Fatalf("Failed to create broken symlink: %v", err)
		}

		// Verify the symlink exists but points to nothing
		if _, err := os.Lstat(brokenLinkPath); err != nil {
			t.Fatalf("Broken symlink should exist as a symlink: %v", err)
		}
		if _, err := os.Stat(brokenLinkPath); err == nil {
			t.Fatal("Broken symlink should not resolve to a valid target")
		}

		// Try to copy file without overwrite - should fail
		_, err := fm.CopyFileFromStorage(storageFilePath, "broken-copy.md", false)
		if err == nil {
			t.Error("Expected error when copying over broken symlink without overwrite")
		}

		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("Expected 'already exists' error, got: %v", err)
		}
	})

	t.Run("overwrites broken symlink with overwrite=true", func(t *testing.T) {
		// Create another broken symlink
		brokenLinkPath := filepath.Join(tempCwd, "broken-copy2.md")
		if err := os.Symlink("another-nonexistent-copy-target.md", brokenLinkPath); err != nil {
			t.Fatalf("Failed to create broken symlink: %v", err)
		}

		// Copy file with overwrite - should succeed
		resultPath, err := fm.CopyFileFromStorage(storageFilePath, "broken-copy2.md", true)
		if err != nil {
			t.Fatalf("CopyFileFromStorage with overwrite should succeed: %v", err)
		}

		// Verify new file content
		content := readFileContent(t, resultPath)
		if content != testContent {
			t.Errorf("Content mismatch. Expected %q, got %q", testContent, content)
		}

		// Verify it's no longer a symlink
		isLink, err := fileops.IsSymlink(resultPath)
		if err != nil {
			t.Fatalf("Failed to check if path is symlink: %v", err)
		}
		if isLink {
			t.Error("Result should not be a symlink after copy")
		}
	})
}

// Tests for GetAbsolutePath and GetAbsolutePathCWD methods

func TestGetAbsolutePath(t *testing.T) {
	tempDir := createTempTestDir(t, "test-get-absolute-*")
	defer os.RemoveAll(tempDir)

	logger, _ := logging.NewTestLogger()
	fm, err := NewFileManager(tempDir, logger)
	if err != nil {
		t.Fatalf("NewFileManager failed: %v", err)
	}

	// Test with simple filename
	item := FileItem{
		Name: "test.md",
		Path: "test.md",
	}

	expected := filepath.Join(tempDir, "test.md")
	actual := fm.GetAbsolutePath(item)

	if actual != expected {
		t.Errorf("GetAbsolutePath() = %q, want %q", actual, expected)
	}

	// Test with nested path
	nestedItem := FileItem{
		Name: "nested.md",
		Path: "docs/nested.md",
	}

	expectedNested := filepath.Join(tempDir, "docs/nested.md")
	actualNested := fm.GetAbsolutePath(nestedItem)

	if actualNested != expectedNested {
		t.Errorf("GetAbsolutePath() nested = %q, want %q", actualNested, expectedNested)
	}
}

func TestGetAbsolutePathCWD(t *testing.T) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	// Create FileManager (storage dir doesn't matter for this test)
	storageDir := createTempTestDir(t, "storage-*")
	defer os.RemoveAll(storageDir)

	logger, _ := logging.NewTestLogger()
	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("NewFileManager failed: %v", err)
	}

	// Test with simple filename
	item := FileItem{
		Name: "test.md",
		Path: "test.md",
	}

	expected := filepath.Join(cwd, "test.md")
	actual := fm.GetAbsolutePathCWD(item)

	if actual != expected {
		t.Errorf("GetAbsolutePathCWD() = %q, want %q", actual, expected)
	}

	// Test with nested path
	nestedItem := FileItem{
		Name: "nested.md",
		Path: "docs/nested.md",
	}

	expectedNested := filepath.Join(cwd, "docs/nested.md")
	actualNested := fm.GetAbsolutePathCWD(nestedItem)

	if actualNested != expectedNested {
		t.Errorf("GetAbsolutePathCWD() nested = %q, want %q", actualNested, expectedNested)
	}
}

// Test for broken symlink detection and overwrite behavior
func TestCreateSymlinkFromStorage_BrokenSymlinkDetection(t *testing.T) {
	logger := createTestLogger()
	storageDir := createTempStorage(t)
	defer os.RemoveAll(storageDir)

	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("Failed to create FileManager: %v", err)
	}

	// Create test content in storage
	testContent := "# Test content for broken symlink test"
	storageFilePath := createTestFile(t, storageDir, "test-source.md", testContent)

	// Create a temporary working directory
	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	tempCwd := createTempStorage(t)
	defer os.RemoveAll(tempCwd)
	defer os.Chdir(originalCwd)

	if err := os.Chdir(tempCwd); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	t.Run("detects broken symlink without overwrite", func(t *testing.T) {
		// Create a broken symlink (pointing to non-existent target)
		brokenLinkPath := filepath.Join(tempCwd, "broken-link.md")
		if err := os.Symlink("nonexistent-target.md", brokenLinkPath); err != nil {
			t.Fatalf("Failed to create broken symlink: %v", err)
		}

		// Verify the symlink exists but points to nothing
		if _, err := os.Lstat(brokenLinkPath); err != nil {
			t.Fatalf("Broken symlink should exist as a symlink: %v", err)
		}
		if _, err := os.Stat(brokenLinkPath); err == nil {
			t.Fatal("Broken symlink should not resolve to a valid target")
		}

		// Try to create symlink without overwrite - should fail
		_, err := fm.CreateSymlinkFromStorage(storageFilePath, "broken-link.md", false)
		if err == nil {
			t.Error("Expected error when creating symlink over broken symlink without overwrite")
		}

		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("Expected 'already exists' error, got: %v", err)
		}
	})

	t.Run("overwrites broken symlink with overwrite=true", func(t *testing.T) {
		// Create another broken symlink
		brokenLinkPath := filepath.Join(tempCwd, "broken-link2.md")
		if err := os.Symlink("another-nonexistent-target.md", brokenLinkPath); err != nil {
			t.Fatalf("Failed to create broken symlink: %v", err)
		}

		// Create symlink with overwrite - should succeed
		resultPath, err := fm.CreateSymlinkFromStorage(storageFilePath, "broken-link2.md", true)
		if err != nil {
			t.Fatalf("CreateSymlinkFromStorage with overwrite should succeed: %v", err)
		}

		// Verify new symlink works
		content := readFileContent(t, resultPath)
		if content != testContent {
			t.Errorf("Content mismatch. Expected %q, got %q", testContent, content)
		}

		// Verify it's actually a symlink
		isLink, err := fileops.IsSymlink(resultPath)
		if err != nil {
			t.Fatalf("Failed to check if path is symlink: %v", err)
		}
		if !isLink {
			t.Error("Result should be a symlink")
		}
	})
}

// 6. Cross-function Integration Tests

func TestFileManagerIntegration(t *testing.T) {
	logger := createTestLogger()
	storageDir := createTempStorage(t)
	defer os.RemoveAll(storageDir)

	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("Failed to create FileManager: %v", err)
	}

	// Create temp source directory and CWD
	tempSourceDir := createTempStorage(t)
	defer os.RemoveAll(tempSourceDir)

	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	tempCwd := createTempStorage(t)
	defer os.RemoveAll(tempCwd)
	defer os.Chdir(originalCwd)
	if err := os.Chdir(tempCwd); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	t.Run("full workflow: copy to storage, copy from storage, create symlink", func(t *testing.T) {
		originalContent := "# Integration Test\nThis tests the full workflow."

		// Step 1: Create source file and copy to storage
		srcPath := createTestFile(t, tempSourceDir, "workflow-test.md", originalContent)
		storagePath, err := fm.CopyFileToStorage(srcPath, nil, false)
		if err != nil {
			t.Fatalf("CopyFileToStorage failed: %v", err)
		}

		// Step 2: Copy from storage to CWD
		copyPath, err := fm.CopyFileFromStorage(storagePath, "copied-from-storage.md", false)
		if err != nil {
			t.Fatalf("CopyFileFromStorage failed: %v", err)
		}

		// Verify copy content
		copyContent := readFileContent(t, copyPath)
		if copyContent != originalContent {
			t.Errorf("Copy content mismatch. Expected %q, got %q", originalContent, copyContent)
		}

		// Step 3: Create symlink from storage to CWD
		linkPath, err := fm.CreateSymlinkFromStorage(storagePath, "symlinked-from-storage.md", false)
		if err != nil {
			t.Fatalf("CreateSymlinkFromStorage failed: %v", err)
		}

		// Verify symlink content
		linkContent := readFileContent(t, linkPath)
		if linkContent != originalContent {
			t.Errorf("Symlink content mismatch. Expected %q, got %q", originalContent, linkContent)
		}

		// Step 4: Modify through symlink and verify storage is updated
		modifiedContent := "# Modified Content\nThis was changed through the symlink."
		if err := os.WriteFile(linkPath, []byte(modifiedContent), 0644); err != nil {
			t.Fatalf("Failed to modify through symlink: %v", err)
		}

		// Verify storage file is updated
		storageContent := readFileContent(t, storagePath)
		if storageContent != modifiedContent {
			t.Errorf("Storage not updated through symlink. Expected %q, got %q", modifiedContent, storageContent)
		}

		// Verify the copy is NOT updated (it's independent)
		independentCopyContent := readFileContent(t, copyPath)
		if independentCopyContent != originalContent {
			t.Errorf("Independent copy should not change. Expected %q, got %q", originalContent, independentCopyContent)
		}
	})

	t.Run("overwrite consistency across functions", func(t *testing.T) {
		content1 := "First version"
		content2 := "Second version"

		// Create initial files in temp source
		srcPath1 := createTestFile(t, tempSourceDir, "overwrite1.md", content1)
		srcPath2 := createTestFile(t, tempSourceDir, "overwrite2.md", content2)

		fileName := "overwrite-test.md"

		// Copy first file to storage
		_, err := fm.CopyFileToStorage(srcPath1, &fileName, false)
		if err != nil {
			t.Fatalf("Initial copy to storage failed: %v", err)
		}

		// Copy from storage to CWD
		storagePath := filepath.Join(fm.storageDir, fileName)
		cwdPath, err := fm.CopyFileFromStorage(storagePath, "cwd-file.md", false)
		if err != nil {
			t.Fatalf("Copy from storage failed: %v", err)
		}

		// Create symlink
		linkPath, err := fm.CreateSymlinkFromStorage(storagePath, "linked-file.md", false)
		if err != nil {
			t.Fatalf("Create symlink failed: %v", err)
		}

		// Verify all have first content
		cwdContent := readFileContent(t, cwdPath)
		linkContent := readFileContent(t, linkPath)

		if cwdContent != content1 || linkContent != content1 {
			t.Errorf("Initial content mismatch. CWD: %q, Link: %q", cwdContent, linkContent)
		}

		// Overwrite in storage
		_, err = fm.CopyFileToStorage(srcPath2, &fileName, true)
		if err != nil {
			t.Fatalf("Overwrite in storage failed: %v", err)
		}

		// CWD copy should still have old content
		cwdContent = readFileContent(t, cwdPath)
		if cwdContent != content1 {
			t.Errorf("CWD copy should be unchanged. Expected %q, got %q", content1, cwdContent)
		}

		// Symlink should have new content
		linkContent = readFileContent(t, linkPath)
		if linkContent != content2 {
			t.Errorf("Symlink should show new content. Expected %q, got %q", content2, linkContent)
		}
	})
}
