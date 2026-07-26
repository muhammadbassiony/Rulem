package fileops

import (
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// Tests for the directory walk.
//
// These were written against SecureDirectoryScanner, which is gone: the walk is
// now reached through Dir.Scan, and the boundary is opened by OpenExistingDir
// rather than by a scanner constructor. Every property below is the one the
// scanner test asserted; only the two lines that obtain the handle changed.
//
//	Original test                                 Now
//	--------------------------------------------  ------------------------------
//	TestNewDirectoryScanner                       TestOpenExistingDir (opendir_test.go)
//	TestSecureDirectoryScanner_ScanDirectory      TestDirScanOptions
//	TestSecureDirectoryScanner_FileInfo           TestDirScanFileInfo
//	TestSecureDirectoryScanner_SymlinkProtection  TestDirScanSymlinkProtection
//	TestSecureDirectoryScanner_SymlinkClassification TestDirScanSymlinkClassification
//	TestSecureDirectoryScanner_LoopDetection      TestDirScanSymlinkLoop
//	TestSecureDirectoryScanner_Close              TestDirAfterClose (dir_test.go)
//	TestNewDirectoryScanner_SecurityValidation    TestScanRootSecurityValidation
//	TestDirectoryScanOptions_SecurityFeatures     TestDirScanSecurityFeatures
//	TestValidateSymlinks_Integration              TestDirScanSymlinkWithinRoot
//	TestSecurityValidationErrors                  TestDirScanUnreadableFile

// openScanDir opens a Dir on an existing directory for the walk tests and
// closes it on cleanup.
func openScanDir(t *testing.T, path string) *Dir {
	t.Helper()

	dir, err := OpenExistingDir(path)
	if err != nil {
		t.Fatalf("OpenExistingDir(%q) failed: %v", path, err)
	}
	t.Cleanup(func() { _ = dir.Close() })

	return dir
}

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

func TestDirScanOptions(t *testing.T) {
	tempDir := createTempDirStructure(t)
	defer os.RemoveAll(tempDir)

	dir := openScanDir(t, tempDir)

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
			files, err := dir.Scan(tt.opts)
			if err != nil {
				t.Fatalf("Scan() failed: %v", err)
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

func TestDirScanFileInfo(t *testing.T) {
	tempDir := createTempDirStructure(t)
	defer os.RemoveAll(tempDir)

	dir := openScanDir(t, tempDir)

	files, err := dir.Scan(&DirectoryScanOptions{
		MaxDepth:      2,
		IncludeHidden: true,
	})
	if err != nil {
		t.Fatalf("Scan() failed: %v", err)
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

func TestDirScanSymlinkProtection(t *testing.T) {
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
	files, err := openScanDir(t, safeDir).Scan(nil)
	if err != nil {
		t.Fatalf("Scan() failed: %v", err)
	}

	// The symlink escapes the scan root, so it must not be reported at all.
	// Only the genuine file inside the root may appear.
	var names []string
	for _, file := range files {
		names = append(names, file.Path)
	}
	if !slices.Equal(names, []string{"safe.txt"}) {
		t.Errorf("Expected exactly [safe.txt], got %v", names)
	}
}

// TestDirScanSymlinkClassification covers what a DirEntry actually reports for
// a symlink: IsDir() is false and Type() is ModeSymlink. Classifying on IsDir()
// alone emitted symlinked directories as bogus files and left the scanner's own
// symlink guard unreachable.
func TestDirScanSymlinkClassification(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Symlink test not supported on Windows")
	}

	tempDir := createTempDir(t)
	defer func() { _ = os.RemoveAll(tempDir) }()

	root := filepath.Join(tempDir, "root")
	realDir := filepath.Join(root, "real_dir")
	if err := os.MkdirAll(realDir, 0755); err != nil {
		t.Fatalf("Failed to create directories: %v", err)
	}
	if err := os.WriteFile(filepath.Join(realDir, "nested.txt"), []byte("nested"), 0644); err != nil {
		t.Fatalf("Failed to create nested file: %v", err)
	}

	realFileContent := "real file content"
	if err := os.WriteFile(filepath.Join(root, "real.txt"), []byte(realFileContent), 0644); err != nil {
		t.Fatalf("Failed to create real file: %v", err)
	}

	// Relative link to a file inside the root: should be reported as a file,
	// carrying the target's size rather than the link's.
	if err := os.Symlink("real.txt", filepath.Join(root, "link_to_file.txt")); err != nil {
		t.Fatalf("Failed to create file symlink: %v", err)
	}

	// Relative link to a directory inside the root: must not be reported as a
	// file, and must not be traversed.
	if err := os.Symlink("real_dir", filepath.Join(root, "link_to_dir")); err != nil {
		t.Fatalf("Failed to create directory symlink: %v", err)
	}

	// Link that dangles.
	if err := os.Symlink("does_not_exist.txt", filepath.Join(root, "broken_link.txt")); err != nil {
		t.Fatalf("Failed to create broken symlink: %v", err)
	}

	files, err := openScanDir(t, root).Scan(nil)
	if err != nil {
		t.Fatalf("Scan() failed: %v", err)
	}

	byPath := make(map[string]FileInfo, len(files))
	for _, file := range files {
		byPath[file.Path] = file
	}

	// A symlinked directory must not surface as a file entry.
	if _, ok := byPath["link_to_dir"]; ok {
		t.Error("Symlink to a directory was emitted as a file")
	}
	// ...and must not be traversed either, so its contents appear only once,
	// under the real directory.
	if _, ok := byPath[filepath.Join("link_to_dir", "nested.txt")]; ok {
		t.Error("Scan traversed a symlinked directory")
	}
	if _, ok := byPath[filepath.Join("real_dir", "nested.txt")]; !ok {
		t.Errorf("Expected to find real_dir/nested.txt, got %v", slices.Sorted(maps.Keys(byPath)))
	}

	if _, ok := byPath["broken_link.txt"]; ok {
		t.Error("Dangling symlink was reported")
	}

	link, ok := byPath["link_to_file.txt"]
	if !ok {
		t.Fatalf("Expected symlink to a file inside the root to be reported, got %v", slices.Sorted(maps.Keys(byPath)))
	}
	if link.IsDir {
		t.Error("Symlink to a file was reported as a directory")
	}
	if link.Size != int64(len(realFileContent)) {
		t.Errorf("Symlink reports the link's size %d, want the target's size %d",
			link.Size, len(realFileContent))
	}
	if link.Mode&os.ModeSymlink != 0 {
		t.Errorf("Symlink reports the link's mode %v, want the target's mode", link.Mode)
	}
}

func TestDirScanSymlinkLoop(t *testing.T) {
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

	// This should complete without infinite recursion. MaxDepth is high on
	// purpose: fs.WalkDir never descends a symlink, so the loop is structurally
	// unreachable rather than merely bounded.
	files, err := openScanDir(t, tempDir).Scan(&DirectoryScanOptions{MaxDepth: 50})
	if err != nil {
		t.Fatalf("Scan() failed: %v", err)
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
		dir, err := OpenExistingDir(tempDir)
		if err != nil {
			b.Fatalf("OpenExistingDir failed: %v", err)
		}

		_, err = dir.Scan(nil)
		if err != nil {
			b.Fatalf("Scan() failed: %v", err)
		}

		_ = dir.Close()
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

// Tests for security validation features

func TestDirScanSecurityFeatures(t *testing.T) {
	tempDir := createTempDirStructure(t)
	defer os.RemoveAll(tempDir)

	t.Run("built-in reserved directory blocking", func(t *testing.T) {
		// Scanning a system directory must be refused, and the refusal now
		// happens where the boundary is opened rather than in a scanner
		// constructor - one policy, one place.
		systemDir := "/etc"
		if runtime.GOOS == "windows" {
			systemDir = "C:\\Windows\\System32"
		}

		_, err := OpenExistingDir(systemDir)
		if err == nil {
			t.Error("Expected error when opening a reserved directory - security should be built-in")
		}
		// The error could be from path security validation or reserved directory check
		if !strings.Contains(err.Error(), "reserved") && !strings.Contains(err.Error(), "path traversal") {
			t.Errorf("Expected reserved directory or path security error, got: %v", err)
		}
	})

	t.Run("ValidateFileAccess option", func(t *testing.T) {
		opts := &DirectoryScanOptions{
			ValidateFileAccess: true,
			MaxDepth:           2,
		}

		files, err := openScanDir(t, tempDir).Scan(opts)
		if err != nil {
			t.Fatalf("Scan with ValidateFileAccess failed: %v", err)
		}

		// Should find files (access validation should pass for readable files)
		if len(files) == 0 {
			t.Error("Expected to find files with ValidateFileAccess enabled")
		}
	})
}

func TestScanRootSecurityValidation(t *testing.T) {
	t.Run("home directory path validation", func(t *testing.T) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Skipf("Cannot get home directory: %v", err)
		}

		// Test with tilde expansion
		dir, err := OpenExistingDir("~/")
		if err != nil {
			t.Errorf("Failed to open ~/ : %v", err)
		} else {
			_ = dir.Close()
		}

		// Test explicit home directory path
		dir, err = OpenExistingDir(homeDir)
		if err != nil {
			t.Errorf("Failed to open home directory: %v", err)
		} else {
			_ = dir.Close()
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
			if _, err := OpenExistingDir(maliciousPath); err == nil {
				t.Errorf("Expected security validation to reject path: %s", maliciousPath)
			}
		}
	})
}

func TestDirScanSymlinkWithinRoot(t *testing.T) {
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
		// Scan the whole temp dir so both directories are inside the boundary.
		files, err := openScanDir(t, tempDir).Scan(&DirectoryScanOptions{MaxDepth: 3})
		if err != nil {
			t.Fatalf("Scan with built-in symlink validation failed: %v", err)
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

func TestDirScanUnreadableFile(t *testing.T) {
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

		// Should complete (skipping unreadable files)
		files, err := openScanDir(t, tempDir).Scan(opts)
		if err != nil {
			t.Fatalf("Scan failed: %v", err)
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
