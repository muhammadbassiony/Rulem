package fileops

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// createTempDirStructure creates a temporary directory with a predefined structure for testing
func createTempDirStructure(t *testing.T) string {
	tempDir := createTempDir(t)

	// Create directory structure
	dirs := []string{
		"src",
		"src/main",
		"src/test",
		"build",
		"node_modules",
		"node_modules/lib",
		".git",
		".hidden",
		"docs",
		"docs/api",
	}

	for _, dir := range dirs {
		err := os.MkdirAll(filepath.Join(tempDir, dir), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create files
	files := map[string]string{
		"README.md":                 "# Project README",
		"src/main.go":               "package main",
		"src/main/app.go":           "package main",
		"src/test/test.go":          "package test",
		"build/output.bin":          "binary content",
		"node_modules/lib/index.js": "console.log('hello')",
		".git/config":               "[core]",
		".hidden/secret.txt":        "secret",
		"docs/guide.md":             "# Guide",
		"docs/api/reference.md":     "# API Reference",
		".gitignore":                "*.log",
		"package.json":              `{"name": "test"}`,
		"large-file.dat":            strings.Repeat("x", 1000),
	}

	for filePath, content := range files {
		fullPath := filepath.Join(tempDir, filePath)
		err := os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", filePath, err)
		}
	}

	return tempDir
}

func TestNewDirectoryScanner(t *testing.T) {
	tempDir := createTempDirStructure(t)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name      string
		scanPath  string
		opts      *DirectoryScanOptions
		wantError bool
		errorText string
	}{
		{
			name:      "valid directory with default options",
			scanPath:  tempDir,
			opts:      nil,
			wantError: false,
		},
		{
			name:     "valid directory with custom options",
			scanPath: tempDir,
			opts: &DirectoryScanOptions{
				MaxDepth:      5,
				IncludeHidden: false,
			},
			wantError: false,
		},
		{
			name:      "empty path",
			scanPath:  "",
			opts:      nil,
			wantError: true,
			errorText: "cannot be empty",
		},
		{
			name:      "non-existent directory",
			scanPath:  filepath.Join(tempDir, "nonexistent"),
			opts:      nil,
			wantError: true,
			errorText: "cannot access scan path",
		},
		{
			name:      "file instead of directory",
			scanPath:  filepath.Join(tempDir, "README.md"),
			opts:      nil,
			wantError: true,
			errorText: "not a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner, err := NewDirectoryScanner(tt.scanPath, tt.opts)

			if tt.wantError {
				if err == nil {
					t.Errorf("NewDirectoryScanner() expected error but got none")
					if scanner != nil {
						scanner.Close()
					}
					return
				}
				if tt.errorText != "" && !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("NewDirectoryScanner() error = %v, want error containing %q",
						err, tt.errorText)
				}
			} else {
				if err != nil {
					t.Errorf("NewDirectoryScanner() unexpected error: %v", err)
					return
				}
				if scanner == nil {
					t.Error("NewDirectoryScanner() returned nil scanner without error")
					return
				}

				// Test that scanner can be closed
				if err := scanner.Close(); err != nil {
					t.Errorf("Failed to close scanner: %v", err)
				}
			}
		})
	}
}

func TestSecureDirectoryScanner_ScanDirectory(t *testing.T) {
	tempDir := createTempDirStructure(t)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name            string
		opts            *DirectoryScanOptions
		expectedFiles   []string // Files we expect to find
		unexpectedFiles []string // Files we expect NOT to find
		minFiles        int      // Minimum number of files expected
		maxFiles        int      // Maximum number of files expected
	}{
		{
			name: "default options",
			opts: nil,
			expectedFiles: []string{
				"README.md",
				"src/main.go",
				".gitignore",
			},
			minFiles: 10,
		},
		{
			name: "exclude hidden files",
			opts: &DirectoryScanOptions{
				IncludeHidden: false,
				MaxDepth:      20,
			},
			expectedFiles: []string{
				"README.md",
				"src/main.go",
			},
			unexpectedFiles: []string{
				".gitignore",
				".hidden/secret.txt",
				".git/config",
			},
		},
		{
			name: "limited depth",
			opts: &DirectoryScanOptions{
				MaxDepth:      1,
				IncludeHidden: true,
			},
			expectedFiles: []string{
				"README.md",
				".gitignore",
			},
			unexpectedFiles: []string{
				"src/main.go",
				"docs/guide.md",
			},
		},
		{
			name: "markdown files only",
			opts: &DirectoryScanOptions{
				MaxDepth:      20,
				IncludeHidden: false,
				FileFilter: func(name string) bool {
					return strings.HasSuffix(name, ".md")
				},
			},
			expectedFiles: []string{
				"README.md",
				"docs/guide.md",
				"docs/api/reference.md",
			},
			unexpectedFiles: []string{
				"src/main.go",
				"package.json",
			},
		},
		{
			name: "custom skip patterns",
			opts: &DirectoryScanOptions{
				MaxDepth:      20,
				IncludeHidden: true,
				SkipPatterns:  []string{"src", "docs"},
			},
			expectedFiles: []string{
				"README.md",
				".gitignore",
			},
			unexpectedFiles: []string{
				"src/main.go",
				"docs/guide.md",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner, err := NewDirectoryScanner(tempDir, tt.opts)
			if err != nil {
				t.Fatalf("Failed to create scanner: %v", err)
			}
			defer scanner.Close()

			files, err := scanner.ScanDirectory()
			if err != nil {
				t.Fatalf("ScanDirectory() failed: %v", err)
			}

			// Check expected files are present
			foundFiles := make(map[string]bool)
			for _, file := range files {
				foundFiles[file.Path] = true
			}

			for _, expected := range tt.expectedFiles {
				if !foundFiles[expected] {
					t.Errorf("Expected file %s not found in results", expected)
				}
			}

			// Check unexpected files are not present
			for _, unexpected := range tt.unexpectedFiles {
				if foundFiles[unexpected] {
					t.Errorf("Unexpected file %s found in results", unexpected)
				}
			}

			// Check file count bounds
			fileCount := len(files)
			if tt.minFiles > 0 && fileCount < tt.minFiles {
				t.Errorf("Expected at least %d files, got %d", tt.minFiles, fileCount)
			}
			if tt.maxFiles > 0 && fileCount > tt.maxFiles {
				t.Errorf("Expected at most %d files, got %d", tt.maxFiles, fileCount)
			}
		})
	}
}

func TestSecureDirectoryScanner_FileInfo(t *testing.T) {
	tempDir := createTempDirStructure(t)
	defer os.RemoveAll(tempDir)

	scanner, err := NewDirectoryScanner(tempDir, &DirectoryScanOptions{
		MaxDepth:      2,
		IncludeHidden: true,
	})
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}
	defer scanner.Close()

	files, err := scanner.ScanDirectory()
	if err != nil {
		t.Fatalf("ScanDirectory() failed: %v", err)
	}

	// Find specific files to test
	var readmeFile, largeFile *FileInfo
	for i, file := range files {
		if file.Path == "README.md" {
			readmeFile = &files[i]
		}
		if file.Path == "large-file.dat" {
			largeFile = &files[i]
		}
	}

	// Test README.md file info
	if readmeFile == nil {
		t.Fatal("README.md not found in scan results")
	}
	if readmeFile.Name != "README.md" {
		t.Errorf("Expected Name 'README.md', got %s", readmeFile.Name)
	}
	if readmeFile.IsDir {
		t.Error("README.md should not be marked as directory")
	}
	if readmeFile.Size <= 0 {
		t.Errorf("Expected positive size for README.md, got %d", readmeFile.Size)
	}
	if readmeFile.ModTime.IsZero() {
		t.Error("Expected non-zero ModTime for README.md")
	}

	// Test large file
	if largeFile == nil {
		t.Fatal("large-file.dat not found in scan results")
	}
	if largeFile.Size != 1000 {
		t.Errorf("Expected size 1000 for large-file.dat, got %d", largeFile.Size)
	}
}

func TestSecureDirectoryScanner_GetScanStats(t *testing.T) {
	tempDir := createTempDirStructure(t)
	defer os.RemoveAll(tempDir)

	scanner, err := NewDirectoryScanner(tempDir, &DirectoryScanOptions{
		MaxDepth:      20,
		IncludeHidden: true,
	})
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}
	defer scanner.Close()

	_, err = scanner.ScanDirectory()
	if err != nil {
		t.Fatalf("ScanDirectory() failed: %v", err)
	}

	stats := scanner.GetScanStats()

	if stats.TotalFiles <= 0 {
		t.Error("Expected positive number of total files")
	}
	if stats.TotalSize <= 0 {
		t.Error("Expected positive total size")
	}
	if stats.LargestFile < 1000 {
		t.Errorf("Expected largest file to be at least 1000 bytes, got %d", stats.LargestFile)
	}
}

func TestSecureDirectoryScanner_SymlinkProtection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Symlink test not supported on Windows")
	}

	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Create a safe directory and file
	safeDir := filepath.Join(tempDir, "safe")
	err := os.MkdirAll(safeDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create safe directory: %v", err)
	}

	safeFile := filepath.Join(safeDir, "safe.txt")
	err = os.WriteFile(safeFile, []byte("safe content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create safe file: %v", err)
	}

	// Create a symlink pointing outside the scan area
	outsideDir := filepath.Join(tempDir, "outside")
	err = os.MkdirAll(outsideDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create outside directory: %v", err)
	}

	outsideFile := filepath.Join(outsideDir, "outside.txt")
	err = os.WriteFile(outsideFile, []byte("outside content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create outside file: %v", err)
	}

	// Create symlink in safe directory pointing to outside file
	symlinkPath := filepath.Join(safeDir, "bad_link.txt")
	err = os.Symlink(outsideFile, symlinkPath)
	if err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Scan only the safe directory
	scanner, err := NewDirectoryScanner(safeDir, nil)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}
	defer scanner.Close()

	files, err := scanner.ScanDirectory()
	if err != nil {
		t.Fatalf("ScanDirectory() failed: %v", err)
	}

	// Verify that only files within the scan root are found
	// The symlink itself should be found, but it cannot access content outside the root
	expectedFiles := []string{"safe.txt", "bad_link.txt"}
	if len(files) < 1 || len(files) > 2 {
		t.Errorf("Expected 1-2 files, got %d: %v", len(files), files)
	}

	for _, file := range files {
		found := false
		if slices.Contains(expectedFiles, file.Path) {
			found = true
			break
		}
		if !found {
			t.Errorf("Found unexpected file: %s", file.Path)
		}
	}
}

func TestSecureDirectoryScanner_LoopDetection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Symlink test not supported on Windows")
	}

	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Create directory structure with symlink loop
	dir1 := filepath.Join(tempDir, "dir1")
	dir2 := filepath.Join(tempDir, "dir1", "dir2")
	err := os.MkdirAll(dir2, 0755)
	if err != nil {
		t.Fatalf("Failed to create directories: %v", err)
	}

	// Create symlink loop: dir2/back_to_dir1 -> dir1
	loopLink := filepath.Join(dir2, "back_to_dir1")
	err = os.Symlink(dir1, loopLink)
	if err != nil {
		t.Fatalf("Failed to create loop symlink: %v", err)
	}

	// Add a file to make sure scanning works
	testFile := filepath.Join(dir1, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	scanner, err := NewDirectoryScanner(tempDir, &DirectoryScanOptions{
		MaxDepth: 50, // High depth to test loop detection, not depth limiting
	})
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}
	defer scanner.Close()

	// This should complete without infinite recursion
	files, err := scanner.ScanDirectory()
	if err != nil {
		t.Fatalf("ScanDirectory() failed: %v", err)
	}

	// Should find at least the test file
	found := false
	for _, file := range files {
		if file.Name == "test.txt" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find test.txt in results")
	}
}

func TestSecureDirectoryScanner_Close(t *testing.T) {
	tempDir := createTempDirStructure(t)
	defer os.RemoveAll(tempDir)

	scanner, err := NewDirectoryScanner(tempDir, nil)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	// Close scanner
	err = scanner.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Trying to scan after close should fail
	_, err = scanner.ScanDirectory()
	if err == nil {
		t.Error("Expected error when scanning after close")
	}
	if !strings.Contains(err.Error(), "closed") {
		t.Errorf("Expected 'closed' error, got: %v", err)
	}

	// Multiple closes should be safe
	err = scanner.Close()
	if err != nil {
		t.Errorf("Second Close() failed: %v", err)
	}
}

func TestScanWithFilter(t *testing.T) {
	tempDir := createTempDirStructure(t)
	defer os.RemoveAll(tempDir)

	// Test scanning for Go files
	goFiles, err := ScanWithFilter(tempDir, func(name string) bool {
		return strings.HasSuffix(name, ".go")
	}, 10)
	if err != nil {
		t.Fatalf("ScanWithFilter() failed: %v", err)
	}

	if len(goFiles) == 0 {
		t.Error("Expected to find at least one .go file")
	}

	for _, file := range goFiles {
		if !strings.HasSuffix(file.Name, ".go") {
			t.Errorf("Found non-Go file: %s", file.Name)
		}
	}

	// Test scanning for markdown files
	mdFiles, err := ScanWithFilter(tempDir, func(name string) bool {
		return strings.HasSuffix(name, ".md")
	}, 5)
	if err != nil {
		t.Fatalf("ScanWithFilter() failed: %v", err)
	}

	expectedMdFiles := []string{"README.md", "guide.md", "reference.md"}
	if len(mdFiles) < len(expectedMdFiles) {
		t.Errorf("Expected at least %d markdown files, got %d", len(expectedMdFiles), len(mdFiles))
	}
}

func TestDirectoryScanOptions_DefaultValues(t *testing.T) {
	opts := getDefaultScanOptions()

	if !opts.SkipUnreadableDirs {
		t.Error("Expected SkipUnreadableDirs to be true by default")
	}
	if opts.MaxDepth != 20 {
		t.Errorf("Expected MaxDepth to be 20, got %d", opts.MaxDepth)
	}
	if !opts.IncludeHidden {
		t.Error("Expected IncludeHidden to be true by default")
	}
	if len(opts.SkipPatterns) == 0 {
		t.Error("Expected default skip patterns to be non-empty")
	}

	// Check that common directories are in skip patterns
	expectedSkips := []string{"node_modules", ".git", "vendor"}
	for _, expected := range expectedSkips {
		found := false
		for _, pattern := range opts.SkipPatterns {
			if pattern == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected skip pattern %s not found in defaults", expected)
		}
	}
}

func BenchmarkDirectoryScanning(b *testing.B) {
	tempDir := createBenchTempDirStructure(b)
	defer os.RemoveAll(tempDir)

	b.ResetTimer()
	for range b.N {
		scanner, err := NewDirectoryScanner(tempDir, nil)
		if err != nil {
			b.Fatalf("Failed to create scanner: %v", err)
		}

		_, err = scanner.ScanDirectory()
		if err != nil {
			b.Fatalf("ScanDirectory() failed: %v", err)
		}

		scanner.Close()
	}
}

func BenchmarkScanWithFilter(b *testing.B) {
	tempDir := createBenchTempDirStructure(b)
	defer os.RemoveAll(tempDir)

	filter := func(name string) bool {
		return strings.HasSuffix(name, ".go") || strings.HasSuffix(name, ".md")
	}

	b.ResetTimer()
	for range b.N {
		_, err := ScanWithFilter(tempDir, filter, 10)
		if err != nil {
			b.Fatalf("ScanWithFilter() failed: %v", err)
		}
	}
}

// createBenchTempDirStructure creates temp directory structure for benchmarks
func createBenchTempDirStructure(b *testing.B) string {
	tempDir, err := os.MkdirTemp("", "dirscan-bench-")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create directory structure
	dirs := []string{
		"src",
		"src/main",
		"src/test",
		"build",
		"node_modules",
		"node_modules/lib",
		".git",
		".hidden",
		"docs",
		"docs/api",
	}

	for _, dir := range dirs {
		err := os.MkdirAll(filepath.Join(tempDir, dir), 0755)
		if err != nil {
			b.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create files
	files := map[string]string{
		"README.md":                 "# Project README",
		"src/main.go":               "package main",
		"src/main/app.go":           "package main",
		"src/test/test.go":          "package test",
		"build/output.bin":          "binary content",
		"node_modules/lib/index.js": "console.log('hello')",
		".git/config":               "[core]",
		".hidden/secret.txt":        "secret",
		"docs/guide.md":             "# Guide",
		"docs/api/reference.md":     "# API Reference",
		".gitignore":                "*.log",
		"package.json":              `{"name": "test"}`,
		"large-file.dat":            strings.Repeat("x", 1000),
	}

	for filePath, content := range files {
		fullPath := filepath.Join(tempDir, filePath)
		err := os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			b.Fatalf("Failed to create file %s: %v", filePath, err)
		}
	}

	return tempDir
}

// Tests for new security validation features

func TestDirectoryScanOptions_SecurityFeatures(t *testing.T) {
	tempDir := createTempDirStructure(t)
	defer os.RemoveAll(tempDir)

	t.Run("built-in reserved directory blocking", func(t *testing.T) {
		// Test scanning actual system directory (should always fail due to built-in security)
		systemDir := "/etc"
		if runtime.GOOS == "windows" {
			systemDir = "C:\\Windows\\System32"
		}

		// Should always fail with built-in security (reserved directories always blocked)
		_, err := NewDirectoryScanner(systemDir, nil)
		if err == nil {
			t.Error("Expected error when scanning reserved directory - security should be built-in")
		}
		// The error could be from path security validation or reserved directory check
		if !strings.Contains(err.Error(), "reserved") && !strings.Contains(err.Error(), "path traversal") {
			t.Errorf("Expected reserved directory or path security error, got: %v", err)
		}

		// Even with custom options, reserved directories should be blocked
		opts := &DirectoryScanOptions{
			MaxDepth: 1,
		}
		_, err = NewDirectoryScanner(systemDir, opts)
		if err == nil {
			t.Error("Expected error when scanning reserved directory even with custom options")
		}
	})

	t.Run("ValidateFileAccess option", func(t *testing.T) {
		opts := &DirectoryScanOptions{
			ValidateFileAccess: true,
			MaxDepth:           2,
		}

		scanner, err := NewDirectoryScanner(tempDir, opts)
		if err != nil {
			t.Fatalf("Failed to create scanner: %v", err)
		}
		defer scanner.Close()

		files, err := scanner.ScanDirectory()
		if err != nil {
			t.Fatalf("ScanDirectory with ValidateFileAccess failed: %v", err)
		}

		// Should find files (access validation should pass for readable files)
		if len(files) == 0 {
			t.Error("Expected to find files with ValidateFileAccess enabled")
		}
	})
}

func TestNewDirectoryScanner_SecurityValidation(t *testing.T) {
	t.Run("home directory path validation", func(t *testing.T) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Skipf("Cannot get home directory: %v", err)
		}

		// Test with tilde expansion
		tildePathOpts := &DirectoryScanOptions{}
		scanner, err := NewDirectoryScanner("~/", tildePathOpts)
		if err != nil {
			t.Errorf("Failed to create scanner for ~/: %v", err)
		} else {
			scanner.Close()
		}

		// Test explicit home directory path
		explicitHomeOpts := &DirectoryScanOptions{}
		scanner, err = NewDirectoryScanner(homeDir, explicitHomeOpts)
		if err != nil {
			t.Errorf("Failed to create scanner for home directory: %v", err)
		} else {
			scanner.Close()
		}
	})

	t.Run("path security validation", func(t *testing.T) {
		// Test path traversal attempts
		maliciousPaths := []string{
			"../../../etc",
			"..\\..\\..\\Windows",
			"/etc/../etc/../etc",
		}

		for _, maliciousPath := range maliciousPaths {
			_, err := NewDirectoryScanner(maliciousPath, nil)
			if err == nil {
				t.Errorf("Expected security validation to reject path: %s", maliciousPath)
			}
		}
	})
}

func TestValidateSymlinks_Integration(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Symlink test not supported on Windows")
	}

	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Create directory structure with symlinks
	mainDir := filepath.Join(tempDir, "main")
	targetDir := filepath.Join(tempDir, "target")
	err := os.MkdirAll(mainDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create main directory: %v", err)
	}
	err = os.MkdirAll(targetDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create target directory: %v", err)
	}

	// Create target file
	targetFile := filepath.Join(targetDir, "target.txt")
	err = os.WriteFile(targetFile, []byte("target content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	// Create symlink in main directory pointing to target file (within same hierarchy)
	symlinkPath := filepath.Join(mainDir, "link_to_target.txt")
	err = os.Symlink(targetFile, symlinkPath)
	if err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	t.Run("built-in symlink validation", func(t *testing.T) {
		opts := &DirectoryScanOptions{
			MaxDepth: 3,
		}

		scanner, err := NewDirectoryScanner(tempDir, opts) // Scan entire temp dir to include both directories
		if err != nil {
			t.Fatalf("Failed to create scanner: %v", err)
		}
		defer scanner.Close()

		// Should successfully scan with built-in symlink validation (symlinks within scan area are safe)
		files, err := scanner.ScanDirectory()
		if err != nil {
			t.Fatalf("ScanDirectory with built-in symlink validation failed: %v", err)
		}

		// Should find files (including the symlinked file)
		foundSymlink := false
		foundTarget := false
		for _, file := range files {
			if file.Name == "link_to_target.txt" {
				foundSymlink = true
			}
			if file.Name == "target.txt" {
				foundTarget = true
			}
		}
		if !foundSymlink && !foundTarget {
			t.Error("Expected to find either symlink or target file")
		}
	})

}

func TestSecurityValidationErrors(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	t.Run("unreadable files with validation", func(t *testing.T) {
		// Create a file and then remove read permissions
		testFile := filepath.Join(tempDir, "unreadable.txt")
		err := os.WriteFile(testFile, []byte("content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Remove read permissions (if possible)
		err = os.Chmod(testFile, 0000)
		if err != nil {
			t.Skipf("Cannot change file permissions: %v", err)
		}
		defer func() {
			if err := os.Chmod(testFile, 0644); err != nil {
				t.Logf("warning: failed to restore permissions: %v", err)
			}
		}()

		opts := &DirectoryScanOptions{
			ValidateFileAccess: true,
			SkipUnreadableDirs: true, // Skip unreadable files
			MaxDepth:           1,
		}

		scanner, err := NewDirectoryScanner(tempDir, opts)
		if err != nil {
			t.Fatalf("Failed to create scanner: %v", err)
		}
		defer scanner.Close()

		// Should complete (skipping unreadable files)
		files, err := scanner.ScanDirectory()
		if err != nil {
			t.Fatalf("ScanDirectory failed: %v", err)
		}

		// Should not find the unreadable file
		for _, file := range files {
			if file.Name == "unreadable.txt" {
				t.Error("Should not have found unreadable file")
			}
		}
	})
}

func TestGetDefaultScanOptions_SecurityDefaults(t *testing.T) {
	opts := getDefaultScanOptions()

	// Verify security defaults
	if opts.ValidateFileAccess {
		t.Error("Expected ValidateFileAccess to be false by default (performance)")
	}
}
