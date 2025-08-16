package filemanager

import (
	"fmt"
	"os"
	"path/filepath"
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

	// Scan directory
	files, err := ScanCurrDirectory()
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

	files, err := ScanCurrDirectory()
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
