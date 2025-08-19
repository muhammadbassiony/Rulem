package filemanager

import (
	"fmt"
	"os"
	"path/filepath"
	"rulem/internal/logging"
	"strings"
	"testing"
	"time"
)

// Integration Tests - tests for domain-specific logic that uses fileops

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

func TestScanCurrDirectory_Integration(t *testing.T) {
	// Integration test for ScanCurrDirectory - tests the complete workflow
	// including fileops integration and markdown file filtering
	structure := map[string]string{
		"README.md":                   "# Test",
		"docs.markdown":               "## Docs",
		"src/main.go":                 "package main",
		"src/README.md":               "# Source",
		"tests/test.md":               "# Tests",
		"script.js":                   "console.log('hello')",
		"config.json":                 "{}",
		".gitignore":                  "*.log",
		"node_modules/package/doc.md": "# Package docs", // Should be skipped
		"vendor/lib/README.md":        "# Vendor lib",   // Should be skipped
	}

	tempDir := createTempDirStructure(t, structure)
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	cleanup := changeToDir(t, tempDir)
	defer cleanup()

	// Create FileManager for scanning
	logger, _ := logging.NewTestLogger()
	fm, err := NewFileManager(tempDir, logger)
	if err != nil {
		t.Fatalf("NewFileManager failed: %v", err)
	}

	// Scan directory
	files, err := fm.ScanCurrDirectory()
	if err != nil {
		t.Fatalf("ScanCurrDirectory failed: %v", err)
	}

	// Expected files (relative paths) - should exclude node_modules and vendor
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

	// Verify that skipped directories are not included
	for _, file := range files {
		if strings.HasPrefix(file.Path, "node_modules") || strings.HasPrefix(file.Path, "vendor") {
			t.Errorf("Found file in skipped directory: %s", file.Path)
		}
	}
}

// Integration Tests for ScanCurrDirectory workflow

func TestScanCurrDirectory_RealisticProject(t *testing.T) {
	// Integration test with realistic project structure
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
		"node_modules/package/doc.md": "# Package docs", // Should be skipped
		"vendor/lib/README.md":        "# Vendor lib",   // Should be skipped
		".vscode/settings.json":       "{}",
		"build/output.txt":            "build output", // Should be skipped
		"temp/cache.tmp":              "cache",
	}

	tempDir := createTempDirStructure(t, structure)
	defer os.RemoveAll(tempDir)

	cleanup := changeToDir(t, tempDir)
	defer cleanup()

	// Create FileManager for scanning
	logger, _ := logging.NewTestLogger()
	fm, err := NewFileManager(tempDir, logger)
	if err != nil {
		t.Fatalf("NewFileManager failed: %v", err)
	}

	files, err := fm.ScanCurrDirectory()
	if err != nil {
		t.Fatalf("ScanCurrDirectory failed: %v", err)
	}

	// Should find markdown files but skip node_modules/vendor/build directories
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

func TestScanCurrDirectory_Performance(t *testing.T) {
	// Integration test for performance with larger file sets
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

	// Create FileManager for scanning
	logger, _ := logging.NewTestLogger()
	fm, err := NewFileManager(tempDir, logger)
	if err != nil {
		t.Fatalf("NewFileManager failed: %v", err)
	}

	start := time.Now()
	files, err := fm.ScanCurrDirectory()
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

func TestScanCurrDirectory_ErrorHandling(t *testing.T) {
	// Integration test for error handling with unreadable directories
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

	// Create FileManager for scanning
	logger, _ := logging.NewTestLogger()
	fm, err := NewFileManager(tempDir, logger)
	if err != nil {
		t.Fatalf("NewFileManager failed: %v", err)
	}

	// Should complete successfully and skip unreadable directory
	files, err := fm.ScanCurrDirectory()
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

func TestScanCurrDirectory_SymlinkHandling(t *testing.T) {
	// Integration test for symlink handling behavior
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

	// Create FileManager for scanning
	logger, _ := logging.NewTestLogger()
	fm, err := NewFileManager(tempDir, logger)
	if err != nil {
		t.Fatalf("NewFileManager failed: %v", err)
	}

	files, err := fm.ScanCurrDirectory()
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

// Tests for ScanCentralRepo

func TestScanCentralRepo_BasicDiscovery(t *testing.T) {
	// Test basic markdown file discovery in storage directory
	structure := map[string]string{
		"README.md": "# Root README",
		"notes.txt": "Not markdown",
		"docs.md":   "# Documentation",
		"script.js": "console.log('hello')",
	}

	storageDir := createTempDirStructure(t, structure)
	defer os.RemoveAll(storageDir)

	logger, _ := logging.NewTestLogger()
	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("NewFileManager failed: %v", err)
	}

	files, err := fm.ScanCentralRepo()
	if err != nil {
		t.Fatalf("ScanCentralRepo failed: %v", err)
	}

	expectedFiles := []string{"README.md", "docs.md"}
	if len(files) != len(expectedFiles) {
		t.Errorf("Expected %d files, got %d: %v", len(expectedFiles), len(files), files)
	}

	fileSet := make(map[string]bool)
	for _, file := range files {
		fileSet[file.Path] = true
	}

	for _, expectedFile := range expectedFiles {
		if !fileSet[expectedFile] {
			t.Errorf("Expected file %q not found in results", expectedFile)
		}
	}
}

func TestScanCentralRepo_RecursiveDiscovery(t *testing.T) {
	// Test nested markdown file discovery
	structure := map[string]string{
		"README.md":                  "# Root",
		"docs/guide.md":              "# Guide",
		"docs/sub/readme.md":         "# Sub readme",
		"docs/sub/deep/nested.md":    "# Deep nested",
		"src/main.go":                "package main",
		"config/settings.yaml":       "key: value",
		"tests/integration/setup.md": "# Test setup",
	}

	storageDir := createTempDirStructure(t, structure)
	defer os.RemoveAll(storageDir)

	logger, _ := logging.NewTestLogger()
	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("NewFileManager failed: %v", err)
	}

	files, err := fm.ScanCentralRepo()
	if err != nil {
		t.Fatalf("ScanCentralRepo failed: %v", err)
	}

	expectedFiles := []string{
		"README.md",
		"docs/guide.md",
		"docs/sub/readme.md",
		"docs/sub/deep/nested.md",
		"tests/integration/setup.md",
	}

	if len(files) != len(expectedFiles) {
		t.Errorf("Expected %d files, got %d: %v", len(expectedFiles), len(files), files)
	}

	fileSet := make(map[string]bool)
	for _, file := range files {
		fileSet[file.Path] = true
	}

	for _, expectedFile := range expectedFiles {
		if !fileSet[expectedFile] {
			t.Errorf("Expected file %q not found in results", expectedFile)
		}
	}
}

func TestScanCentralRepo_SkipCommonDirectories(t *testing.T) {
	// Test that common skip patterns are honored
	structure := map[string]string{
		"root.md":                  "# Root markdown",
		"node_modules/pkg/doc.md":  "# Package docs", // Should be skipped
		"vendor/lib/README.md":     "# Vendor lib",   // Should be skipped
		"build/output.md":          "# Build docs",   // Should be skipped
		".git/hooks/pre-commit.md": "# Git hook",     // Should be skipped
		".vscode/snippets.md":      "# VS Code",      // Should be skipped
		"dist/bundle.md":           "# Bundle docs",  // Should be skipped
		"__pycache__/cache.md":     "# Python cache", // Should be skipped
		"normal/docs.md":           "# Normal docs",
	}

	storageDir := createTempDirStructure(t, structure)
	defer os.RemoveAll(storageDir)

	logger, _ := logging.NewTestLogger()
	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("NewFileManager failed: %v", err)
	}

	files, err := fm.ScanCentralRepo()
	if err != nil {
		t.Fatalf("ScanCentralRepo failed: %v", err)
	}

	expectedFiles := []string{"root.md", "normal/docs.md"}
	if len(files) != len(expectedFiles) {
		t.Errorf("Expected %d files, got %d: %v", len(expectedFiles), len(files), files)
	}

	// Verify no skipped directories are included
	for _, file := range files {
		skippedPrefixes := []string{"node_modules", "vendor", "build", ".git", ".vscode", "dist", "__pycache__"}
		for _, prefix := range skippedPrefixes {
			if strings.HasPrefix(file.Path, prefix) {
				t.Errorf("Found file in skipped directory: %s", file.Path)
			}
		}
	}
}

func TestScanCentralRepo_HiddenFilesAndMaxDepth(t *testing.T) {
	// Test hidden files inclusion and max depth behavior
	structure := map[string]string{
		".hidden.md":          "# Hidden markdown",
		"normal.md":           "# Normal markdown",
		".config/settings.md": "# Config markdown",
	}

	// Create a very deep nested structure beyond MaxDepth (50)
	deepPath := ""
	for i := range 55 {
		deepPath = filepath.Join(deepPath, fmt.Sprintf("dir%d", i))
	}
	structure[filepath.Join(deepPath, "deep.md")] = "# Very deep"

	storageDir := createTempDirStructure(t, structure)
	defer os.RemoveAll(storageDir)

	logger, _ := logging.NewTestLogger()
	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("NewFileManager failed: %v", err)
	}

	files, err := fm.ScanCentralRepo()
	if err != nil {
		t.Fatalf("ScanCentralRepo failed: %v", err)
	}

	// Should include hidden files but not files beyond max depth
	expectedFiles := []string{".hidden.md", "normal.md", ".config/settings.md"}

	foundFiles := make(map[string]bool)
	for _, file := range files {
		foundFiles[file.Path] = true
	}

	for _, expectedFile := range expectedFiles {
		if !foundFiles[expectedFile] {
			t.Errorf("Expected file %q not found in results", expectedFile)
		}
	}

	// Verify that very deep file is not included (beyond MaxDepth)
	deepFile := filepath.Join(deepPath, "deep.md")
	if foundFiles[deepFile] {
		t.Errorf("File beyond MaxDepth should not be included: %s", deepFile)
	}
}

func TestScanCentralRepo_SymlinkHandling(t *testing.T) {
	// Test symlink handling within storage directory
	structure := map[string]string{
		"real/file.md":  "# Real file",
		"target/doc.md": "# Target doc",
		"normal.md":     "# Normal",
	}

	storageDir := createTempDirStructure(t, structure)
	defer os.RemoveAll(storageDir)

	// Create symlink to directory within storage
	realDir := filepath.Join(storageDir, "real")
	linkDir := filepath.Join(storageDir, "link_to_real")
	createTestSymlink(t, realDir, linkDir)

	// Create symlink to file within storage
	targetFile := filepath.Join(storageDir, "target", "doc.md")
	linkFile := filepath.Join(storageDir, "link_to_doc.md")
	createTestSymlink(t, targetFile, linkFile)

	logger, _ := logging.NewTestLogger()
	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("NewFileManager failed: %v", err)
	}

	files, err := fm.ScanCentralRepo()
	if err != nil {
		t.Fatalf("ScanCentralRepo failed: %v", err)
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

func TestScanCentralRepo_UnreadableDirectories(t *testing.T) {
	// Test handling of unreadable directories
	structure := map[string]string{
		"readable/file.md":     "# Readable",
		"unreadable/secret.md": "# Secret",
		"normal.md":            "# Normal",
	}

	storageDir := createTempDirStructure(t, structure)
	defer os.RemoveAll(storageDir)

	// Make directory unreadable
	unreadableDir := filepath.Join(storageDir, "unreadable")
	if err := os.Chmod(unreadableDir, 0000); err != nil {
		t.Skipf("Cannot change permissions: %v", err)
	}
	defer os.Chmod(unreadableDir, 0755) // Restore for cleanup

	logger, _ := logging.NewTestLogger()
	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("NewFileManager failed: %v", err)
	}

	// Should complete successfully and skip unreadable directory
	files, err := fm.ScanCentralRepo()
	if err != nil {
		t.Fatalf("ScanCentralRepo failed: %v", err)
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

func TestScanCentralRepo_NonExistentStorageDir(t *testing.T) {
	// Test error when storage directory doesn't exist
	nonExistentDir := filepath.Join(os.TempDir(), "non-existent-storage-dir")

	logger, _ := logging.NewTestLogger()

	// This should fail during NewFileManager creation due to ValidateStoragePath
	_, err := NewFileManager(nonExistentDir, logger)
	if err == nil {
		t.Fatal("Expected NewFileManager to fail with non-existent directory")
	}

	if !strings.Contains(err.Error(), "storage directory does not exist") {
		t.Errorf("Expected error about non-existent directory, got: %v", err)
	}
}

func TestScanCentralRepo_StorageDirIsFile(t *testing.T) {
	// Test error when storage path points to a file instead of directory
	tempDir := createTempTestDir(t, "scan-test-*")
	storageFile := createTestFile(t, tempDir, "notadirectory.txt", "content")

	logger, _ := logging.NewTestLogger()

	// This might fail during NewFileManager creation or later during directory check
	fm, err := NewFileManager(storageFile, logger)
	if err != nil {
		// If NewFileManager fails, that's expected and acceptable
		if strings.Contains(err.Error(), "not a directory") ||
			strings.Contains(err.Error(), "Invalid storage directory") {
			return // Test passed
		}
		t.Errorf("Expected error about invalid directory, got: %v", err)
		return
	}

	// If NewFileManager succeeded, ScanCentralRepo should fail
	_, err = fm.ScanCentralRepo()
	if err == nil {
		t.Fatal("Expected ScanCentralRepo to fail when storage path is a file")
	}

	if !strings.Contains(err.Error(), "not a directory") &&
		!strings.Contains(err.Error(), "storage path is not a directory") {
		t.Errorf("Expected error about not being a directory, got: %v", err)
	}
}

func TestScanCentralRepo_SymlinkToSystemDirectory(t *testing.T) {
	// Test security: symlink pointing to system directory should fail
	tempDir := createTempTestDir(t, "scan-symlink-test-*")

	// Create a symlink to /etc (or other system directory)
	systemDir := "/etc"
	if isWindows() {
		systemDir = "C:\\Windows\\System32"
	}

	symlinkPath := filepath.Join(tempDir, "storage")
	createTestSymlink(t, systemDir, symlinkPath)

	logger, _ := logging.NewTestLogger()

	// This should fail during NewFileManager creation due to ValidateStoragePath
	_, err := NewFileManager(symlinkPath, logger)
	if err == nil {
		t.Fatal("Expected NewFileManager to fail with symlink to system directory")
	}

	// Accept various security-related error messages
	if !strings.Contains(err.Error(), "security validation") &&
		!strings.Contains(err.Error(), "reserved") &&
		!strings.Contains(err.Error(), "path traversal") &&
		!strings.Contains(err.Error(), "Invalid storage directory") {
		t.Errorf("Expected security-related error, got: %v", err)
	}
}

func TestScanCentralRepo_Performance(t *testing.T) {
	// Test performance with larger file sets
	structure := make(map[string]string)

	// Create 500 markdown files across 20 directories
	for i := range 20 {
		for j := range 25 {
			path := fmt.Sprintf("dir%d/file%d.md", i, j)
			structure[path] = fmt.Sprintf("# File %d-%d", i, j)
		}
	}

	// Add some non-markdown files
	for i := range 100 {
		path := fmt.Sprintf("other%d.txt", i)
		structure[path] = "text content"
	}

	storageDir := createTempDirStructure(t, structure)
	defer os.RemoveAll(storageDir)

	logger, _ := logging.NewTestLogger()
	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("NewFileManager failed: %v", err)
	}

	start := time.Now()
	files, err := fm.ScanCentralRepo()
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("ScanCentralRepo failed: %v", err)
	}

	// Should find 500 markdown files
	if len(files) != 500 {
		t.Errorf("Expected 500 files, got %d", len(files))
	}

	// Performance check - should complete within reasonable time
	if duration > 10*time.Second {
		t.Errorf("Scan took too long: %v", duration)
	}

	t.Logf("Scanned %d files in %v", len(files), duration)
}

func TestScanCentralRepo_NilFileManager(t *testing.T) {
	// Test error handling with nil FileManager
	var fm *FileManager = nil

	_, err := fm.ScanCentralRepo()
	if err == nil {
		t.Fatal("Expected error with nil FileManager")
	}

	if !strings.Contains(err.Error(), "filemanager is nil") {
		t.Errorf("Expected 'filemanager is nil' error, got: %v", err)
	}
}

func TestScanCentralRepo_EmptyStorageDir(t *testing.T) {
	// Test error handling with empty storage directory configuration
	logger, _ := logging.NewTestLogger()

	// Create FileManager with empty storage directory by directly setting the field
	// Note: This shouldn't normally happen as NewFileManager validates, but test the method's robustness
	fm := &FileManager{
		logger:     logger,
		storageDir: "", // Empty storage dir
	}

	_, err := fm.ScanCentralRepo()
	if err == nil {
		t.Fatal("Expected error with empty storage directory")
	}

	if !strings.Contains(err.Error(), "storage directory is not configured") {
		t.Errorf("Expected 'storage directory is not configured' error, got: %v", err)
	}
}

func TestScanCentralRepo_ValidStorageSymlink(t *testing.T) {
	// Test valid symlink within allowed paths (positive test for symlink security)
	// Create actual storage directory within user's home or temp directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("Cannot get home directory: %v", err)
	}

	// Create temp dir within home for valid symlink test
	tempDir, err := os.MkdirTemp(homeDir, "scan-valid-symlink-test-*")
	if err != nil {
		t.Skipf("Cannot create temp dir in home: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create actual storage directory
	realStorageDir := createTestDir(t, tempDir, "real_storage")
	createTestFile(t, realStorageDir, "test.md", "# Test markdown")

	// Create symlink to the real storage directory
	symlinkStorageDir := filepath.Join(tempDir, "symlink_storage")
	createTestSymlink(t, realStorageDir, symlinkStorageDir)

	logger, _ := logging.NewTestLogger()
	fm, err := NewFileManager(symlinkStorageDir, logger)
	if err != nil {
		t.Fatalf("NewFileManager failed with valid symlink: %v", err)
	}

	files, err := fm.ScanCentralRepo()
	if err != nil {
		t.Fatalf("ScanCentralRepo failed with valid symlink: %v", err)
	}

	// Should find the test markdown file
	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d: %v", len(files), files)
	}

	if len(files) > 0 && files[0].Name != "test.md" {
		t.Errorf("Expected file 'test.md', got %q", files[0].Name)
	}
}

// Tests for ConvertToAbsolutePaths

func TestConvertToAbsolutePaths(t *testing.T) {
	storageDir := createTempTestDir(t, "convert-paths-test-*")

	// Create test files in storage
	structure := map[string]string{
		"root.md":          "# Root file",
		"docs/guide.md":    "# Guide",
		"sub/deep/file.md": "# Deep file",
	}

	for path, content := range structure {
		fullPath := filepath.Join(storageDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		createTestFile(t, filepath.Dir(fullPath), filepath.Base(fullPath), content)
	}

	logger, _ := logging.NewTestLogger()
	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("NewFileManager failed: %v", err)
	}

	// Get files with relative paths
	relativeFiles, err := fm.ScanCentralRepo()
	if err != nil {
		t.Fatalf("ScanCentralRepo failed: %v", err)
	}

	// Convert to absolute paths
	absoluteFiles := fm.ConvertToAbsolutePaths(relativeFiles)

	t.Run("same length", func(t *testing.T) {
		if len(absoluteFiles) != len(relativeFiles) {
			t.Errorf("Expected same length, got relative: %d, absolute: %d", len(relativeFiles), len(absoluteFiles))
		}
	})

	t.Run("paths are absolute", func(t *testing.T) {
		for _, file := range absoluteFiles {
			if !filepath.IsAbs(file.Path) {
				t.Errorf("Expected absolute path, got: %s", file.Path)
			}
		}
	})

	t.Run("paths point to storage directory", func(t *testing.T) {
		for _, file := range absoluteFiles {
			if !strings.HasPrefix(file.Path, storageDir) {
				t.Errorf("Expected path to be under storage dir %s, got: %s", storageDir, file.Path)
			}
		}
	})

	t.Run("names are preserved", func(t *testing.T) {
		for i, file := range absoluteFiles {
			if file.Name != relativeFiles[i].Name {
				t.Errorf("Expected name %s, got %s", relativeFiles[i].Name, file.Name)
			}
		}
	})

	t.Run("files are readable", func(t *testing.T) {
		for _, file := range absoluteFiles {
			content, err := os.ReadFile(file.Path)
			if err != nil {
				t.Errorf("Failed to read file %s: %v", file.Path, err)
			}
			if len(content) == 0 {
				t.Errorf("File %s is empty", file.Path)
			}
		}
	})
}

func TestConvertToAbsolutePaths_EmptySlice(t *testing.T) {
	storageDir := createTempTestDir(t, "convert-empty-test-*")

	logger, _ := logging.NewTestLogger()
	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("NewFileManager failed: %v", err)
	}

	// Test with empty slice
	emptyFiles := []FileItem{}
	result := fm.ConvertToAbsolutePaths(emptyFiles)

	if len(result) != 0 {
		t.Errorf("Expected empty slice, got length: %d", len(result))
	}
}

func TestConvertToAbsolutePaths_NilSlice(t *testing.T) {
	storageDir := createTempTestDir(t, "convert-nil-test-*")

	logger, _ := logging.NewTestLogger()
	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("NewFileManager failed: %v", err)
	}

	// Test with nil slice
	var nilFiles []FileItem
	result := fm.ConvertToAbsolutePaths(nilFiles)

	if result == nil {
		t.Error("Expected non-nil slice, got nil")
	}
	if len(result) != 0 {
		t.Errorf("Expected empty slice, got length: %d", len(result))
	}
}

func TestConvertToAbsolutePaths_DoesNotModifyOriginal(t *testing.T) {
	storageDir := createTempTestDir(t, "convert-immutable-test-*")

	// Create test file
	createTestFile(t, storageDir, "test.md", "# Test")

	logger, _ := logging.NewTestLogger()
	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("NewFileManager failed: %v", err)
	}

	// Get original files
	originalFiles, err := fm.ScanCentralRepo()
	if err != nil {
		t.Fatalf("ScanCentralRepo failed: %v", err)
	}

	// Store original paths for comparison
	originalPaths := make([]string, len(originalFiles))
	for i, file := range originalFiles {
		originalPaths[i] = file.Path
	}

	// Convert to absolute paths
	_ = fm.ConvertToAbsolutePaths(originalFiles)

	// Verify original slice was not modified
	for i, file := range originalFiles {
		if file.Path != originalPaths[i] {
			t.Errorf("Original slice was modified at index %d: expected %s, got %s", i, originalPaths[i], file.Path)
		}
	}
}

// Tests for ConvertToAbsolutePathsCWD

func TestConvertToAbsolutePathsCWD(t *testing.T) {
	// Create a temporary directory structure for CWD testing
	tempCwd := createTempTestDir(t, "convert-cwd-test-*")

	// Create test files in temp CWD
	structure := map[string]string{
		"root.md":          "# Root file",
		"docs/guide.md":    "# Guide",
		"sub/deep/file.md": "# Deep file",
	}

	for path, content := range structure {
		fullPath := filepath.Join(tempCwd, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		createTestFile(t, filepath.Dir(fullPath), filepath.Base(fullPath), content)
	}

	// Change to temp CWD
	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalCwd)

	if err := os.Chdir(tempCwd); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create FileManager (storage dir doesn't matter for this test)
	storageDir := createTempTestDir(t, "storage-*")
	logger, _ := logging.NewTestLogger()
	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("NewFileManager failed: %v", err)
	}

	// Get files with relative paths from CWD
	relativeFiles, err := fm.ScanCurrDirectory()
	if err != nil {
		t.Fatalf("ScanCurrDirectory failed: %v", err)
	}

	// Convert to absolute paths
	absoluteFiles := fm.ConvertToAbsolutePathsCWD(relativeFiles)

	t.Run("same length", func(t *testing.T) {
		if len(absoluteFiles) != len(relativeFiles) {
			t.Errorf("Expected same length, got relative: %d, absolute: %d", len(relativeFiles), len(absoluteFiles))
		}
	})

	t.Run("paths are absolute", func(t *testing.T) {
		for _, file := range absoluteFiles {
			if !filepath.IsAbs(file.Path) {
				t.Errorf("Expected absolute path, got: %s", file.Path)
			}
		}
	})

	t.Run("paths point to CWD", func(t *testing.T) {
		// Resolve symlinks for both paths to handle macOS /var vs /private/var differences
		resolvedTempCwd, err := filepath.EvalSymlinks(tempCwd)
		if err != nil {
			resolvedTempCwd = tempCwd // fallback to original if symlink resolution fails
		}

		for _, file := range absoluteFiles {
			resolvedFilePath, err := filepath.EvalSymlinks(file.Path)
			if err != nil {
				resolvedFilePath = file.Path // fallback to original if symlink resolution fails
			}

			if !strings.HasPrefix(resolvedFilePath, resolvedTempCwd) {
				t.Errorf("Expected path to be under CWD %s, got: %s", resolvedTempCwd, resolvedFilePath)
			}
		}
	})

	t.Run("names are preserved", func(t *testing.T) {
		for i, file := range absoluteFiles {
			if file.Name != relativeFiles[i].Name {
				t.Errorf("Expected name %s, got %s", relativeFiles[i].Name, file.Name)
			}
		}
	})

	t.Run("files are readable", func(t *testing.T) {
		for _, file := range absoluteFiles {
			content, err := os.ReadFile(file.Path)
			if err != nil {
				t.Errorf("Failed to read file %s: %v", file.Path, err)
			}
			if len(content) == 0 {
				t.Errorf("File %s is empty", file.Path)
			}
		}
	})
}

func TestConvertToAbsolutePathsCWD_EmptySlice(t *testing.T) {
	storageDir := createTempTestDir(t, "convert-cwd-empty-test-*")

	logger, _ := logging.NewTestLogger()
	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("NewFileManager failed: %v", err)
	}

	// Test with empty slice
	emptyFiles := []FileItem{}
	result := fm.ConvertToAbsolutePathsCWD(emptyFiles)

	if len(result) != 0 {
		t.Errorf("Expected empty slice, got length: %d", len(result))
	}
}

func TestConvertToAbsolutePathsCWD_NilSlice(t *testing.T) {
	storageDir := createTempTestDir(t, "convert-cwd-nil-test-*")

	logger, _ := logging.NewTestLogger()
	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("NewFileManager failed: %v", err)
	}

	// Test with nil slice
	var nilFiles []FileItem
	result := fm.ConvertToAbsolutePathsCWD(nilFiles)

	if result == nil {
		t.Error("Expected non-nil slice, got nil")
	}
	if len(result) != 0 {
		t.Errorf("Expected empty slice, got length: %d", len(result))
	}
}

func TestConvertToAbsolutePathsCWD_DoesNotModifyOriginal(t *testing.T) {
	// Create temporary CWD with test file
	tempCwd := createTempTestDir(t, "convert-cwd-immutable-test-*")
	createTestFile(t, tempCwd, "test.md", "# Test")

	// Change to temp CWD
	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalCwd)

	if err := os.Chdir(tempCwd); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create FileManager
	storageDir := createTempTestDir(t, "storage-*")
	logger, _ := logging.NewTestLogger()
	fm, err := NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("NewFileManager failed: %v", err)
	}

	// Get original files
	originalFiles, err := fm.ScanCurrDirectory()
	if err != nil {
		t.Fatalf("ScanCurrDirectory failed: %v", err)
	}

	// Store original paths for comparison
	originalPaths := make([]string, len(originalFiles))
	for i, file := range originalFiles {
		originalPaths[i] = file.Path
	}

	// Convert to absolute paths
	_ = fm.ConvertToAbsolutePathsCWD(originalFiles)

	// Verify original slice was not modified
	for i, file := range originalFiles {
		if file.Path != originalPaths[i] {
			t.Errorf("Original slice was modified at index %d: expected %s, got %s", i, originalPaths[i], file.Path)
		}
	}
}
