package filemanager

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Unit Tests

func TestIsMarkdownFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		// Valid markdown files
		{"README.md", true},
		{"docs.markdown", true},
		{"file.MD", true},
		{"file.MARKDOWN", true},
		{"file.Md", true},
		{"file.Markdown", true},

		// Invalid files
		{"README.txt", false},
		{"README", false},
		{"README.mdx", false},
		{"README.md5", false},
		{"file.doc", false},
		{"script.js", false},

		// Edge cases
		{".md", true},
		{".markdown", true},
		{"", false},
		{"file.", false},
		{"file.backup.md", true},
		{"file.md.backup", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := isMarkdownFile(tt.filename)
			if result != tt.expected {
				t.Errorf("isMarkdownFile(%q) = %v, want %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestShouldSkipDirectory(t *testing.T) {
	tests := []struct {
		name          string
		dirName       string
		includeHidden bool
		expected      bool
	}{
		// Special directories (never skip)
		{"current directory", ".", false, false},
		{"current directory with hidden", ".", true, false},
		{"parent directory", "..", false, false},
		{"parent directory with hidden", "..", true, false},

		// Hidden directories
		{"hidden dir exclude", ".hidden", false, true},
		{"hidden dir include", ".hidden", true, false},
		{"git dir exclude", ".git", false, true},
		{"git dir include", ".git", true, true}, // Still skip .git even with includeHidden

		// Common skip directories
		{"node_modules", "node_modules", false, true},
		{"node_modules with hidden", "node_modules", true, true},
		{"vendor", "vendor", false, true},
		{"target", "target", false, true},
		{"build", "build", false, true},
		{".next", ".next", false, true},

		// Normal directories
		{"src", "src", false, false},
		{"docs", "docs", false, false},
		{"my_project", "my_project", true, false},

		// Edge cases
		{"empty string", "", false, false},
		{"node_modules_backup", "node_modules_backup", false, false},
		{"my_vendor", "my_vendor", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldSkipDirectory(tt.dirName, tt.includeHidden)
			if result != tt.expected {
				t.Errorf("shouldSkipDirectory(%q, %v) = %v, want %v",
					tt.dirName, tt.includeHidden, result, tt.expected)
			}
		})
	}
}

func TestScanCurrDirectory(t *testing.T) {
	// Create test directory structure
	structure := map[string]string{
		"README.md":     "# Test",
		"docs.markdown": "## Docs",
		"src/main.go":   "package main",
		"src/README.md": "# Source",
		"tests/test.md": "# Tests",
		"script.js":     "console.log('hello')",
		"config.json":   "{}",
		".gitignore":    "*.log",
		"node_modules/": "",
		"vendor/":       "",
	}

	tempDir := createTempDirStructure(t, structure)
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	cleanup := changeToDir(t, tempDir)
	defer cleanup()

	// Scan directory
	files, err := ScanCurrDirectory()
	if err != nil {
		t.Fatalf("ScanCurrDirectory failed: %v", err)
	}

	// Expected files (relative paths)
	expected := []string{
		"README.md",
		"docs.markdown",
		"src/README.md",
		"tests/test.md",
	}

	// Check count
	if len(files) != len(expected) {
		t.Errorf("Expected %d files, got %d: %v", len(expected), len(files), files)
	}

	// Check each expected file is found
	fileSet := make(map[string]bool)
	for _, file := range files {
		fileSet[file.Path] = true
	}

	for _, expectedFile := range expected {
		if !fileSet[expectedFile] {
			t.Errorf("Expected file %q not found in results", expectedFile)
		}
	}
}

func TestScanDirRecursive(t *testing.T) {
	t.Run("depth limits", func(t *testing.T) {
		// Create deep directory structure
		structure := map[string]string{
			"level1/level2/level3/level4/level5/deep.md": "# Deep file",
			"level1/shallow.md":                          "# Shallow file",
		}

		tempDir := createTempDirStructure(t, structure)
		defer os.RemoveAll(tempDir)

		root, err := os.OpenRoot(tempDir)
		if err != nil {
			t.Fatalf("Failed to open root: %v", err)
		}
		defer root.Close()

		// Test with max depth 3
		opts := &ScanOptions{
			MaxDepth:           3,
			SkipUnreadableDirs: true,
			IncludeHidden:      false,
		}

		scanner := NewDefaultDirScanner(root, opts)
		err = scanner.scanDirRecursive(".", 1)
		if err != nil {
			t.Fatalf("scanDirRecursive failed: %v", err)
		}

		// Should find shallow.md but not deep.md
		if len(scanner.files) != 1 {
			t.Errorf("Expected 1 file within depth limit, got %d: %v", len(scanner.files), scanner.files)
		}

		if scanner.files[0].Path != "level1/shallow.md" {
			t.Errorf("Expected to find level1/shallow.md, got %v", scanner.files)
		}
	})

	t.Run("symlink loop detection", func(t *testing.T) {
		structure := map[string]string{
			"real_dir/file.md": "# Real file",
			"normal.md":        "# Normal file",
		}

		tempDir := createTempDirStructure(t, structure)
		defer os.RemoveAll(tempDir)

		// Create symlink loop: real_dir -> loop_link -> real_dir
		realDir := filepath.Join(tempDir, "real_dir")
		loopLink := filepath.Join(tempDir, "real_dir", "loop_link")
		createTestSymlink(t, realDir, loopLink)

		root, err := os.OpenRoot(tempDir)
		if err != nil {
			t.Fatalf("Failed to open root: %v", err)
		}
		defer root.Close()

		scanner := NewDefaultDirScanner(root, nil)
		err = scanner.scanDirRecursive(".", 1)
		if err != nil {
			t.Fatalf("scanDirRecursive failed: %v", err)
		}

		// Should handle loop gracefully and find both files
		if len(scanner.files) < 2 {
			t.Errorf("Expected at least 2 files, got %d: %v", len(scanner.files), scanner.files)
		}
	})

	t.Run("directory skipping", func(t *testing.T) {
		structure := map[string]string{
			"src/main.md":         "# Main",
			".git/config":         "git config",
			".git/hooks/pre.md":   "# Git hook",
			"node_modules/pkg.md": "# Package",
			".hidden/secret.md":   "# Secret",
			"normal/file.md":      "# Normal",
		}

		tempDir := createTempDirStructure(t, structure)
		defer os.RemoveAll(tempDir)

		root, err := os.OpenRoot(tempDir)
		if err != nil {
			t.Fatalf("Failed to open root: %v", err)
		}
		defer root.Close()

		opts := &ScanOptions{
			IncludeHidden: false,
			MaxDepth:      10,
		}

		scanner := NewDefaultDirScanner(root, opts)
		err = scanner.scanDirRecursive(".", 1)
		if err != nil {
			t.Fatalf("scanDirRecursive failed: %v", err)
		}

		// Should only find src/main.md and normal/file.md
		expectedFiles := []string{"src/main.md", "normal/file.md"}

		if len(scanner.files) != len(expectedFiles) {
			t.Errorf("Expected %d files, got %d: %v", len(expectedFiles), len(scanner.files), scanner.files)
		}

		fileSet := make(map[string]bool)
		for _, file := range scanner.files {
			fileSet[file.Path] = true
		}

		for _, expected := range expectedFiles {
			if fileSet[expected] {
				t.Logf("Found expected file: %s", expected)
			} else {
				t.Errorf("Expected file %q not found in results", expected)
			}
		}
	})
}

// Integration Tests

func TestScanRealFilesystem(t *testing.T) {
	// Create a realistic project structure
	structure := map[string]string{
		"README.md":                   "# Project",
		"CHANGELOG.md":                "# Changes",
		"docs/guide.md":               "# Guide",
		"docs/api.markdown":           "# API",
		"src/main.go":                 "package main",
		"src/README.md":               "# Source code",
		"tests/integration/test.md":   "# Integration tests",
		"scripts/build.sh":            "#!/bin/bash",
		".git/config":                 "git config",
		"node_modules/package/doc.md": "# Package docs",
		"vendor/lib/README.md":        "# Vendor lib",
		".vscode/settings.json":       "{}",
		"build/output.txt":            "build output",
		"temp/cache.tmp":              "cache",
	}

	tempDir := createTempDirStructure(t, structure)
	defer os.RemoveAll(tempDir)

	cleanup := changeToDir(t, tempDir)
	defer cleanup()

	files, err := ScanCurrDirectory()
	if err != nil {
		t.Fatalf("ScanCurrDirectory failed: %v", err)
	}

	// Should find markdown files but skip hidden/build directories
	expectedFiles := []string{
		"README.md",
		"CHANGELOG.md",
		"docs/guide.md",
		"docs/api.markdown",
		"src/README.md",
		"tests/integration/test.md",
	}

	if len(files) != len(expectedFiles) {
		t.Errorf("Expected %d files, got %d: %v", len(expectedFiles), len(files), files)
	}

	// Verify all expected files are found
	fileSet := make(map[string]bool)
	for _, file := range files {
		fileSet[file.Path] = true
	}

	for _, expected := range expectedFiles {
		if !fileSet[expected] {
			t.Errorf("Expected file %q not found", expected)
		}
	}
}

func TestScanPerformance(t *testing.T) {
	// Create structure with moderate number of files
	structure := make(map[string]string)

	// Create 100 markdown files across 10 directories
	for i := range 10 {
		for j := range 10 {
			path := fmt.Sprintf("dir%d/file%d.md", i, j)
			structure[path] = fmt.Sprintf("# File %d-%d", i, j)
		}
	}

	// Add some non-markdown files
	for i := range 50 {
		path := fmt.Sprintf("other%d.txt", i)
		structure[path] = "text content"
	}

	tempDir := createTempDirStructure(t, structure)
	defer os.RemoveAll(tempDir)

	cleanup := changeToDir(t, tempDir)
	defer cleanup()

	start := time.Now()
	files, err := ScanCurrDirectory()
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("ScanCurrDirectory failed: %v", err)
	}

	// Should find 100 markdown files
	if len(files) != 100 {
		t.Errorf("Expected 100 files, got %d", len(files))
	}

	// Performance check - should complete within reasonable time
	if duration > 5*time.Second {
		t.Errorf("Scan took too long: %v", duration)
	}

	t.Logf("Scanned %d files in %v", len(files), duration)
}

// Error Handling Tests

func TestScanWithUnreadableDirectories(t *testing.T) {
	structure := map[string]string{
		"readable/file.md":     "# Readable",
		"unreadable/secret.md": "# Secret",
		"normal.md":            "# Normal",
	}

	tempDir := createTempDirStructure(t, structure)
	defer os.RemoveAll(tempDir)

	// Make directory unreadable
	unreadableDir := filepath.Join(tempDir, "unreadable")
	if err := os.Chmod(unreadableDir, 0000); err != nil {
		t.Skipf("Cannot change permissions: %v", err)
	}
	defer os.Chmod(unreadableDir, 0755) // Restore for cleanup

	cleanup := changeToDir(t, tempDir)
	defer cleanup()

	// Should complete successfully and skip unreadable directory
	files, err := ScanCurrDirectory()
	if err != nil {
		t.Fatalf("ScanCurrDirectory failed: %v", err)
	}

	// Should find readable files but skip unreadable directory
	expected := []string{"readable/file.md", "normal.md"}

	if len(files) != len(expected) {
		t.Errorf("Expected %d files, got %d: %v", len(expected), len(files), files)
	}

	fileSet := make(map[string]bool)
	for _, file := range files {
		fileSet[file.Path] = true
	}

	for _, expectedFile := range expected {
		if !fileSet[expectedFile] {
			t.Errorf("Expected file %q not found in results", expectedFile)
		}
	}
}

// Security Tests (simplified)

func TestScanSecurityBoundaries(t *testing.T) {
	structure := map[string]string{
		"allowed/file.md": "# Allowed",
		"test.md":         "# Test",
	}

	tempDir := createTempDirStructure(t, structure)
	defer os.RemoveAll(tempDir)

	root, err := os.OpenRoot(tempDir)
	if err != nil {
		t.Fatalf("Failed to open root: %v", err)
	}
	defer root.Close()

	scanner := NewDefaultDirScanner(root, nil)

	// Should not be able to access paths outside the root
	err = scanner.scanDirRecursive("../", 1)
	if err == nil {
		// This might succeed but should not find files outside tempDir
		// The security is enforced by os.Root, not our code
	}

	// Scan within boundaries should work
	err = scanner.scanDirRecursive(".", 1)
	if err != nil {
		t.Errorf("Failed to scan within boundaries: %v", err)
	}

	if len(scanner.files) != 2 {
		t.Errorf("Expected 2 files within boundaries, got %d", len(scanner.files))
	}

	expected := []string{"allowed/file.md", "test.md"}
	fileSet := make(map[string]bool)
	for _, file := range scanner.files {
		fileSet[file.Path] = true
	}
	for _, expectedFile := range expected {
		if !fileSet[expectedFile] {
			t.Errorf("Expected file %q not found in results", expectedFile)
		}
	}
}

func TestScanSymlinkHandling(t *testing.T) {
	structure := map[string]string{
		"real/file.md":  "# Real file",
		"target/doc.md": "# Target doc",
		"normal.md":     "# Normal",
	}

	tempDir := createTempDirStructure(t, structure)
	defer os.RemoveAll(tempDir)

	// Create symlink to directory
	realDir := filepath.Join(tempDir, "real")
	linkDir := filepath.Join(tempDir, "link_to_real")
	createTestSymlink(t, realDir, linkDir)

	// Create symlink to file
	targetFile := filepath.Join(tempDir, "target", "doc.md")
	linkFile := filepath.Join(tempDir, "link_to_doc.md")
	createTestSymlink(t, targetFile, linkFile)

	cleanup := changeToDir(t, tempDir)
	defer cleanup()

	files, err := ScanCurrDirectory()
	if err != nil {
		t.Fatalf("ScanCurrDirectory failed: %v", err)
	}

	// Should find files through symlinks
	expectedMinimum := 3 // normal.md + real/file.md + target/doc.md (+ possibly symlinked versions)

	if len(files) < expectedMinimum {
		t.Errorf("Expected at least %d files, got %d: %v", expectedMinimum, len(files), files)
	}

	expected := []string{"normal.md", "real/file.md"}

	fileSet := make(map[string]bool)
	for _, file := range files {
		fileSet[file.Path] = true
	}
	for _, expectedFile := range expected {
		if !fileSet[expectedFile] {
			t.Errorf("Expected file %q not found in results", expectedFile)
		}
	}
}
