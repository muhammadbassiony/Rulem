package filemanager

import (
	"os"
	"path/filepath"
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

		if !strings.Contains(err.Error(), "cannot resolve parent directory") {
			t.Errorf("Expected 'does not exist' error, got: %v", err)
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

			if !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("Expected error containing %q, got: %v", tt.expectError, err)
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
			expectError: true,
			errorText:   "invalid filename",
		},
		{
			name:        "filename with forward slash",
			newFileName: stringPtr("folder/file.md"),
			expectError: true,
			errorText:   "invalid filename",
		},
		{
			name:        "filename with backslash",
			newFileName: stringPtr("folder\\file.md"),
			expectError: true,
			errorText:   "invalid filename",
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
			expectError: true,
			errorText:   "invalid filename",
		},
		{
			name: "encoded path traversal",
			setupFunc: func(t *testing.T) (string, *string) {
				srcPath := createTestFile(t, tempDir, "test.md", "content")
				maliciousName := "..%2F..%2F..%2Fetc%2Fpasswd"
				return srcPath, &maliciousName
			},
			expectError: true,
			errorText:   "invalid filename",
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
