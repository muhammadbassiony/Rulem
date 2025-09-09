package mcp

import (
	"os"
	"path/filepath"
	"testing"

	"rulem/internal/config"
	"rulem/internal/filemanager"
	"rulem/internal/logging"
)

func TestParseRuleFiles(t *testing.T) {
	server, tempDir := createTestServerWithFrontmatterFiles(t)
	defer os.RemoveAll(tempDir)

	// Initialize the server components
	err := server.initializeComponents()
	if err != nil {
		t.Fatalf("Failed to initialize server components: %v", err)
	}

	// Get all files first
	files, err := server.getRepoFiles()
	if err != nil {
		t.Fatalf("Failed to get repo files: %v", err)
	}

	// Parse frontmatter
	ruleFiles, err := server.parseRuleFiles(files)
	if err != nil {
		t.Errorf("parseRuleFiles should not return error: %v", err)
	}

	// Should find 3 valid rule files (valid-rule-1.md, valid-rule-2.md, valid-rule-3.md)
	expectedCount := 3
	if len(ruleFiles) != expectedCount {
		t.Errorf("Expected %d rule files, got %d", expectedCount, len(ruleFiles))
	}

	// Check specific files
	foundFiles := make(map[string]RuleFile)
	for _, rf := range ruleFiles {
		foundFiles[rf.FileName] = rf
	}

	// Test valid-rule-1.md
	if rule, exists := foundFiles["valid-rule-1.md"]; exists {
		if rule.Description != "Basic TypeScript rules" {
			t.Errorf("Expected description 'Basic TypeScript rules', got '%s'", rule.Description)
		}
		if rule.Name != "TypeScript Guidelines" {
			t.Errorf("Expected name 'TypeScript Guidelines', got '%s'", rule.Name)
		}
		if rule.ApplyTo != "typescript" {
			t.Errorf("Expected applyTo 'typescript', got '%s'", rule.ApplyTo)
		}
		if rule.Content == "" {
			t.Error("Content should not be empty")
		}
	} else {
		t.Error("valid-rule-1.md should be found in parsed rules")
	}

	// Test valid-rule-2.md (only description)
	if rule, exists := foundFiles["valid-rule-2.md"]; exists {
		if rule.Description != "React component guidelines" {
			t.Errorf("Expected description 'React component guidelines', got '%s'", rule.Description)
		}
		if rule.Name != "" {
			t.Errorf("Expected empty name, got '%s'", rule.Name)
		}
		if rule.ApplyTo != "" {
			t.Errorf("Expected empty applyTo, got '%s'", rule.ApplyTo)
		}
	} else {
		t.Error("valid-rule-2.md should be found in parsed rules")
	}

	// Test valid-rule-3.md (description + name only)
	if rule, exists := foundFiles["valid-rule-3.md"]; exists {
		if rule.Description != "Python coding standards" {
			t.Errorf("Expected description 'Python coding standards', got '%s'", rule.Description)
		}
		if rule.Name != "Python Rules" {
			t.Errorf("Expected name 'Python Rules', got '%s'", rule.Name)
		}
		if rule.ApplyTo != "" {
			t.Errorf("Expected empty applyTo, got '%s'", rule.ApplyTo)
		}
	} else {
		t.Error("valid-rule-3.md should be found in parsed rules")
	}

	// Ensure invalid files are not included
	invalidFiles := []string{"no-frontmatter.md", "invalid-yaml.md", "no-description.md", "empty-file.md", "not-markdown.txt"}
	for _, fileName := range invalidFiles {
		if _, exists := foundFiles[fileName]; exists {
			t.Errorf("Invalid file %s should not be included in parsed rules", fileName)
		}
	}
}

func TestParseRuleFilesWithoutFileManager(t *testing.T) {
	cfg := &config.Config{StorageDir: "/tmp/test"}
	logger := logging.NewAppLogger()
	server := NewServer(cfg, logger)

	// Try to parse without initializing fileManager
	_, err := server.parseRuleFiles([]filemanager.FileItem{})
	if err == nil {
		t.Error("parseRuleFiles should return error when fileManager is not initialized")
	}
	if err.Error() != "file manager not initialized" {
		t.Errorf("Expected 'file manager not initialized' error, got: %v", err)
	}
}

func TestParseRuleFilesEmptyList(t *testing.T) {
	server := createTestServer(t)

	// Initialize the server components
	err := server.initializeComponents()
	if err != nil {
		t.Fatalf("Failed to initialize server components: %v", err)
	}

	// Parse empty file list
	ruleFiles, err := server.parseRuleFiles([]filemanager.FileItem{})
	if err != nil {
		t.Errorf("parseRuleFiles should not return error for empty list: %v", err)
	}

	if len(ruleFiles) != 0 {
		t.Errorf("Expected 0 rule files for empty input, got %d", len(ruleFiles))
	}
}

func TestGetRuleFilesWithFrontmatter(t *testing.T) {
	server, tempDir := createTestServerWithFrontmatterFiles(t)
	defer os.RemoveAll(tempDir)

	// Initialize the server components
	err := server.initializeComponents()
	if err != nil {
		t.Fatalf("Failed to initialize server components: %v", err)
	}

	// Get rule files with frontmatter (end-to-end test)
	ruleFiles, err := server.getRuleFilesWithFrontmatter()
	if err != nil {
		t.Errorf("getRuleFilesWithFrontmatter should not return error: %v", err)
	}

	// Should find 3 valid rule files
	expectedCount := 3
	if len(ruleFiles) != expectedCount {
		t.Errorf("Expected %d rule files, got %d", expectedCount, len(ruleFiles))
	}

	// Verify all have required description field
	for _, rule := range ruleFiles {
		if rule.Description == "" {
			t.Errorf("Rule file %s should have description", rule.FileName)
		}
		if rule.FileName == "" {
			t.Error("Rule file should have filename")
		}
		if rule.FilePath == "" {
			t.Error("Rule file should have file path")
		}
	}
}

func TestParseRuleFilesUnreadableFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "rulem-unreadable-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a file and make it unreadable
	unreadableFile := filepath.Join(tempDir, "unreadable.md")
	if err := os.WriteFile(unreadableFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create unreadable file: %v", err)
	}
	if err := os.Chmod(unreadableFile, 0000); err != nil {
		t.Fatalf("Failed to make file unreadable: %v", err)
	}
	// Restore permissions for cleanup
	defer os.Chmod(unreadableFile, 0644)

	cfg := &config.Config{StorageDir: tempDir}
	logger := logging.NewAppLogger()
	server := NewServer(cfg, logger)

	// Initialize the server components
	err = server.initializeComponents()
	if err != nil {
		t.Fatalf("Failed to initialize server components: %v", err)
	}

	// Create file item for unreadable file
	files := []filemanager.FileItem{
		{Name: "unreadable.md", Path: "unreadable.md"},
	}

	// Parse should handle unreadable file gracefully
	ruleFiles, err := server.parseRuleFiles(files)
	if err != nil {
		t.Errorf("parseRuleFiles should handle unreadable files gracefully: %v", err)
	}

	// Should return empty list since file is unreadable
	if len(ruleFiles) != 0 {
		t.Errorf("Expected 0 rule files for unreadable file, got %d", len(ruleFiles))
	}
}

// Helper functions

func createTestServerWithFrontmatterFiles(t *testing.T) (*Server, string) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "rulem-frontmatter-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create test files with various frontmatter scenarios
	testFiles := []struct {
		name    string
		content string
	}{
		{
			"valid-rule-1.md",
			`---
description: "Basic TypeScript rules"
name: "TypeScript Guidelines"
applyTo: "typescript"
---

# TypeScript Rules

Use strict types and interfaces.`,
		},
		{
			"valid-rule-2.md",
			`---
description: "React component guidelines"
---

# React Components

Always use functional components with hooks.`,
		},
		{
			"valid-rule-3.md",
			`---
description: "Python coding standards"
name: "Python Rules"
---

# Python Guidelines

Follow PEP 8 standards.`,
		},
		{
			"no-frontmatter.md",
			`# Regular Markdown

This file has no frontmatter at all.`,
		},
		{
			"invalid-yaml.md",
			`---
invalid yaml: [
description: "This will fail
---

# Invalid YAML

Content here.`,
		},
		{
			"no-description.md",
			`---
name: "Missing Description"
applyTo: "javascript"
---

# No Description

This frontmatter is missing the required description field.`,
		},
		{
			"empty-file.md",
			"",
		},
		{
			"not-markdown.txt",
			`---
description: "This is not a markdown file"
---

Text content.`,
		},
	}

	for _, file := range testFiles {
		filePath := filepath.Join(tempDir, file.name)
		if err := os.WriteFile(filePath, []byte(file.content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file.name, err)
		}
	}

	cfg := &config.Config{StorageDir: tempDir}
	logger := logging.NewAppLogger()

	return NewServer(cfg, logger), tempDir
}
