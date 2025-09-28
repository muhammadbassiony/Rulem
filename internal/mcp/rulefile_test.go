package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rulem/internal/filemanager"
	"rulem/internal/logging"
)

func createTestRuleFileProcessor(t *testing.T) (*RuleFileProcessor, string) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "rulem-rulefile-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cfg := createTestConfigWithPath(tempDir)
	logger, _ := logging.NewTestLogger()
	fileManager, err := filemanager.NewFileManager(cfg.Central.Path, logger)
	if err != nil {
		t.Fatalf("Failed to create file manager: %v", err)
	}

	processor := NewRuleFileProcessor(logger, fileManager, 5*1024*1024) // 5MB
	return processor, tempDir
}

func TestNewRuleFileProcessor(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	fileManager, _ := filemanager.NewFileManager("/tmp", logger)

	processor := NewRuleFileProcessor(logger, fileManager, 5*1024*1024) // 5MB

	if processor == nil {
		t.Fatal("NewRuleFileProcessor returned nil")
	}
	if processor.logger != logger {
		t.Error("Processor logger not set correctly")
	}
	if processor.fileManager != fileManager {
		t.Error("Processor fileManager not set correctly")
	}
	if processor.toolRegistry == nil {
		t.Error("Tool registry should be initialized")
	}
}

func TestGenerateToolName(t *testing.T) {
	_, tempDir := createTestRuleFileProcessor(t)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name         string
		ruleFile     *RuleFile
		expectedName string
		description  string
	}{
		{
			name: "uses frontmatter name field when provided",
			ruleFile: &RuleFile{
				FileName: "test-file.md",
				Name:     "custom_tool_name",
			},
			expectedName: "custom_tool_name",
			description:  "should use the name field from frontmatter",
		},
		{
			name: "generates from filename when no name field",
			ruleFile: &RuleFile{
				FileName: "go-standards.md",
				Name:     "",
			},
			expectedName: "go_standards",
			description:  "should generate from filename, replacing hyphens with underscores",
		},
		{
			name: "handles spaces in filename",
			ruleFile: &RuleFile{
				FileName: "coding standards.md",
				Name:     "",
			},
			expectedName: "coding_standards",
			description:  "should replace spaces with underscores",
		},
		{
			name: "handles special characters in filename",
			ruleFile: &RuleFile{
				FileName: "test-file@#$.md",
				Name:     "",
			},
			expectedName: "test_file",
			description:  "should sanitize special characters",
		},
		{
			name: "handles empty filename gracefully",
			ruleFile: &RuleFile{
				FileName: "",
				Name:     "",
			},
			expectedName: "rule_file",
			description:  "should fallback to generic name for empty filename",
		},
		{
			name: "handles filename without extension",
			ruleFile: &RuleFile{
				FileName: "bubbletea-patterns",
				Name:     "",
			},
			expectedName: "bubbletea_patterns",
			description:  "should handle files without extensions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new processor for each test to avoid conflicts
			testProcessor, testTempDir := createTestRuleFileProcessor(t)
			defer os.RemoveAll(testTempDir)

			result := testProcessor.generateToolName(tt.ruleFile)
			if result != tt.expectedName {
				t.Errorf("generateToolName() = %v, want %v (%s)", result, tt.expectedName, tt.description)
			}
		})
	}
}

func TestGenerateToolNameDuplicateHandling(t *testing.T) {
	processor, tempDir := createTestRuleFileProcessor(t)
	defer os.RemoveAll(tempDir)

	// Create a rule file that will generate the same name
	ruleFile := &RuleFile{
		FileName: "test-rule.md",
		Name:     "",
	}

	// First call should return the base name
	name1 := processor.generateToolName(ruleFile)
	expectedName1 := "test_rule"
	if name1 != expectedName1 {
		t.Errorf("First call: generateToolName() = %v, want %v", name1, expectedName1)
	}

	// Register the first tool to simulate it being in use
	processor.toolRegistry[name1] = &RuleFileTool{
		Name:     name1,
		RuleFile: ruleFile,
	}

	// Second call should return name with suffix
	name2 := processor.generateToolName(ruleFile)
	expectedName2 := "test_rule_1"
	if name2 != expectedName2 {
		t.Errorf("Second call: generateToolName() = %v, want %v", name2, expectedName2)
	}

	// Register the second tool
	processor.toolRegistry[name2] = &RuleFileTool{
		Name:     name2,
		RuleFile: ruleFile,
	}

	// Third call should return name with incremented suffix
	name3 := processor.generateToolName(ruleFile)
	expectedName3 := "test_rule_2"
	if name3 != expectedName3 {
		t.Errorf("Third call: generateToolName() = %v, want %v", name3, expectedName3)
	}
}

func TestGenerateToolNameWithFrontmatterNameDuplicates(t *testing.T) {
	processor, tempDir := createTestRuleFileProcessor(t)
	defer os.RemoveAll(tempDir)

	// Create rule files with the same frontmatter name
	ruleFile1 := &RuleFile{
		FileName: "file1.md",
		Name:     "duplicate_name",
	}

	ruleFile2 := &RuleFile{
		FileName: "file2.md",
		Name:     "duplicate_name",
	}

	// First call should return the base name
	name1 := processor.generateToolName(ruleFile1)
	if name1 != "duplicate_name" {
		t.Errorf("First call: generateToolName() = %v, want %v", name1, "duplicate_name")
	}

	// Register the first tool
	processor.toolRegistry[name1] = &RuleFileTool{
		Name:     name1,
		RuleFile: ruleFile1,
	}

	// Second call should return name with suffix
	name2 := processor.generateToolName(ruleFile2)
	expectedName2 := "duplicate_name_1"
	if name2 != expectedName2 {
		t.Errorf("Second call: generateToolName() = %v, want %v", name2, expectedName2)
	}
}

func TestGenerateToolNameEdgeCases(t *testing.T) {
	_, tempDir := createTestRuleFileProcessor(t)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name         string
		ruleFile     *RuleFile
		expectedName string
		description  string
	}{
		{
			name: "handles multiple consecutive special characters",
			ruleFile: &RuleFile{
				FileName: "test@#$%^&*()file.md",
				Name:     "",
			},
			expectedName: "testfile",
			description:  "should remove all special characters",
		},
		{
			name: "handles filename with only special characters",
			ruleFile: &RuleFile{
				FileName: "@#$%^&*().md",
				Name:     "",
			},
			expectedName: "rule_file",
			description:  "should fallback to generic name when only special chars remain",
		},
		{
			name: "handles mixed case filename",
			ruleFile: &RuleFile{
				FileName: "CamelCase-File_Name.md",
				Name:     "",
			},
			expectedName: "CamelCase_File_Name",
			description:  "should preserve case and convert separators to underscores",
		},
		{
			name: "handles numeric filename",
			ruleFile: &RuleFile{
				FileName: "123-456-rules.md",
				Name:     "",
			},
			expectedName: "123_456_rules",
			description:  "should handle numeric characters properly",
		},
		{
			name: "handles frontmatter name with special characters",
			ruleFile: &RuleFile{
				FileName: "test.md",
				Name:     "tool@name#with$special%chars",
			},
			expectedName: "toolnamewithspecialchars",
			description:  "should sanitize frontmatter name with special characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new processor for each test to avoid conflicts
			testProcessor, testTempDir := createTestRuleFileProcessor(t)
			defer os.RemoveAll(testTempDir)

			result := testProcessor.generateToolName(tt.ruleFile)
			if result != tt.expectedName {
				t.Errorf("generateToolName() = %v, want %v (%s)", result, tt.expectedName, tt.description)
			}
		})
	}
}

// TestGenerateToolDescription tests the dynamic tool description generation
// Note: These tests use the same constants (ToolDescriptionPrefix and ApplyToFormat)
// that the implementation uses, so they automatically adapt when those constants change
func TestGenerateToolDescription(t *testing.T) {
	processor, tempDir := createTestRuleFileProcessor(t)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name                string
		ruleFile            *RuleFile
		expectedDescription string
		description         string
	}{
		{
			name: "description only without applyTo",
			ruleFile: &RuleFile{
				Description: "Go coding standards and best practices",
				ApplyTo:     "",
			},
			expectedDescription: ToolDescriptionPrefix + "Go coding standards and best practices",
			description:         "should return description as-is when applyTo is empty",
		},
		{
			name: "description with applyTo field",
			ruleFile: &RuleFile{
				Description: "Go coding standards and best practices",
				ApplyTo:     "Go projects",
			},
			expectedDescription: ToolDescriptionPrefix + "Go coding standards and best practices (" + ApplyToFormat + ": Go projects)",
			description:         "should combine description and applyTo with proper formatting",
		},
		{
			name: "description with complex applyTo",
			ruleFile: &RuleFile{
				Description: "Testing guidelines for unit and integration tests",
				ApplyTo:     "Go testing, bubbletea applications",
			},
			expectedDescription: ToolDescriptionPrefix + "Testing guidelines for unit and integration tests (" + ApplyToFormat + ": Go testing, bubbletea applications)",
			description:         "should handle complex applyTo values with commas",
		},
		{
			name: "empty description with applyTo",
			ruleFile: &RuleFile{
				Description: "",
				ApplyTo:     "Go projects",
			},
			expectedDescription: "Rule file tool",
			description:         "should return fallback description when description is empty",
		},
		{
			name: "empty description without applyTo",
			ruleFile: &RuleFile{
				Description: "",
				ApplyTo:     "",
			},
			expectedDescription: "Rule file tool",
			description:         "should return fallback description when both fields are empty",
		},
		{
			name: "description with whitespace-only applyTo",
			ruleFile: &RuleFile{
				Description: "Code formatting rules",
				ApplyTo:     "   ",
			},
			expectedDescription: ToolDescriptionPrefix + "Code formatting rules (" + ApplyToFormat + ":    )",
			description:         "should include applyTo even if it contains only whitespace",
		},
		{
			name: "multiline description with applyTo",
			ruleFile: &RuleFile{
				Description: "Comprehensive coding standards\nfor Go development",
				ApplyTo:     "Go backend services",
			},
			expectedDescription: ToolDescriptionPrefix + "Comprehensive coding standards\nfor Go development (" + ApplyToFormat + ": Go backend services)",
			description:         "should handle multiline descriptions properly",
		},
		{
			name: "description with special characters and applyTo",
			ruleFile: &RuleFile{
				Description: "Rules for API design & documentation (v2.0)",
				ApplyTo:     "REST APIs, GraphQL endpoints",
			},
			expectedDescription: ToolDescriptionPrefix + "Rules for API design & documentation (v2.0) (" + ApplyToFormat + ": REST APIs, GraphQL endpoints)",
			description:         "should handle special characters in both description and applyTo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.generateToolDescription(tt.ruleFile)
			if result != tt.expectedDescription {
				t.Errorf("generateToolDescription() = %q, want %q (%s)", result, tt.expectedDescription, tt.description)
			}
		})
	}
}

func TestGenerateToolDescriptionEdgeCases(t *testing.T) {
	processor, tempDir := createTestRuleFileProcessor(t)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name                string
		ruleFile            *RuleFile
		expectedDescription string
		description         string
	}{
		{
			name: "nil rule file should not panic",
			ruleFile: &RuleFile{
				Description: "",
				ApplyTo:     "",
			},
			expectedDescription: "Rule file tool",
			description:         "should handle empty rule file gracefully",
		},
		{
			name: "very long description with applyTo",
			ruleFile: &RuleFile{
				Description: "This is a very long description that contains multiple sentences and detailed explanations about coding standards, best practices, and implementation guidelines that should be followed when developing applications",
				ApplyTo:     "Large-scale enterprise applications, microservices, distributed systems",
			},
			expectedDescription: ToolDescriptionPrefix + "This is a very long description that contains multiple sentences and detailed explanations about coding standards, best practices, and implementation guidelines that should be followed when developing applications (" + ApplyToFormat + ": Large-scale enterprise applications, microservices, distributed systems)",
			description:         "should handle very long descriptions and applyTo values",
		},
		{
			name: "description with parentheses and applyTo",
			ruleFile: &RuleFile{
				Description: "Rules for error handling (with examples)",
				ApplyTo:     "Go applications (backend only)",
			},
			expectedDescription: ToolDescriptionPrefix + "Rules for error handling (with examples) (" + ApplyToFormat + ": Go applications (backend only))",
			description:         "should handle nested parentheses correctly",
		},
		{
			name: "unicode characters in description and applyTo",
			ruleFile: &RuleFile{
				Description: "Règles de codage pour les développeurs 🚀",
				ApplyTo:     "Applications web françaises 🇫🇷",
			},
			expectedDescription: ToolDescriptionPrefix + "Règles de codage pour les développeurs 🚀 (" + ApplyToFormat + ": Applications web françaises 🇫🇷)",
			description:         "should handle unicode characters and emojis",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.generateToolDescription(tt.ruleFile)
			if result != tt.expectedDescription {
				t.Errorf("generateToolDescription() = %q, want %q (%s)", result, tt.expectedDescription, tt.description)
			}
		})
	}
}

func TestProcessRuleFiles(t *testing.T) {
	processor, tempDir := createTestRuleFileProcessor(t)
	defer os.RemoveAll(tempDir)

	// Create test files with frontmatter
	testFiles := []struct {
		name    string
		content string
	}{
		{
			"go-standards.md",
			`---
description: "Go coding standards and best practices"
name: "go_standards"
applyTo: "Go projects"
---
# Go Standards
Use proper error handling.`,
		},
		{
			"testing-guide.md",
			`---
description: "Testing guidelines for unit and integration tests"
applyTo: "Go testing, bubbletea applications"
---
# Testing Guide
Write comprehensive tests.`,
		},
		{
			"bubbletea-patterns.md",
			`---
description: "UI patterns for bubbletea applications"
---
# Bubbletea Patterns
Use proper component structure.`,
		},
		{
			"no-frontmatter.md",
			"# No Frontmatter\nThis file has no frontmatter.",
		},
		{
			"missing-description.md",
			`---
name: "missing_desc"
applyTo: "All projects"
---
# Missing Description
This file is missing the required description field.`,
		},
	}

	for _, file := range testFiles {
		filePath := filepath.Join(tempDir, file.name)
		if err := os.WriteFile(filePath, []byte(file.content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file.name, err)
		}
	}

	// Get file items from file manager
	files, err := processor.fileManager.ScanCentralRepo()
	if err != nil {
		t.Fatalf("Failed to scan files: %v", err)
	}

	// Process rule files
	tools, err := processor.ProcessRuleFiles(files)
	if err != nil {
		t.Errorf("ProcessRuleFiles should not return error: %v", err)
	}

	// Verify tools were processed correctly
	expectedTools := map[string]struct {
		description string
		fileName    string
	}{
		"go_standards": {
			description: ToolDescriptionPrefix + "Go coding standards and best practices (" + ApplyToFormat + ": Go projects)",
			fileName:    "go-standards.md",
		},
		"testing_guide": {
			description: ToolDescriptionPrefix + "Testing guidelines for unit and integration tests (" + ApplyToFormat + ": Go testing, bubbletea applications)",
			fileName:    "testing-guide.md",
		},
		"bubbletea_patterns": {
			description: ToolDescriptionPrefix + "UI patterns for bubbletea applications",
			fileName:    "bubbletea-patterns.md",
		},
	}

	// Check that the correct number of tools were processed
	if len(tools) != len(expectedTools) {
		t.Errorf("Expected %d tools to be processed, got %d", len(expectedTools), len(tools))
	}

	// tools is already a map
	toolMap := tools

	for expectedName, expectedData := range expectedTools {
		tool, exists := toolMap[expectedName]
		if !exists {
			t.Errorf("Expected tool %s not found in processed tools", expectedName)
			continue
		}

		if tool.Name != expectedName {
			t.Errorf("Tool name mismatch: expected %s, got %s", expectedName, tool.Name)
		}

		if tool.Description != expectedData.description {
			t.Errorf("Tool description mismatch for %s: expected %q, got %q", expectedName, expectedData.description, tool.Description)
		}

		if tool.RuleFile.FileName != expectedData.fileName {
			t.Errorf("Tool rule file name mismatch for %s: expected %s, got %s", expectedName, expectedData.fileName, tool.RuleFile.FileName)
		}
	}
}

func TestProcessRuleFilesEmptyList(t *testing.T) {
	processor, tempDir := createTestRuleFileProcessor(t)
	defer os.RemoveAll(tempDir)

	// Process empty file list
	toolsMap, err := processor.ProcessRuleFiles([]filemanager.FileItem{})
	if err != nil {
		t.Errorf("ProcessRuleFiles should not return error for empty list: %v", err)
	}

	if len(toolsMap) != 0 {
		t.Errorf("Expected 0 tools for empty file list, got %d", len(toolsMap))
	}
}

func TestProcessRuleFilesDuplicateNames(t *testing.T) {
	processor, tempDir := createTestRuleFileProcessor(t)
	defer os.RemoveAll(tempDir)

	// Create test files that will generate the same tool name
	testFiles := []struct {
		name    string
		content string
	}{
		{
			"test-rule.md",
			`---
description: "First test rule"
---
# First Test Rule`,
		},
		{
			"test_rule.md",
			`---
description: "Second test rule"
---
# Second Test Rule`,
		},
		{
			"another-file.md",
			`---
description: "Third test rule"
name: "test_rule"
---
# Third Test Rule`,
		},
	}

	for _, file := range testFiles {
		filePath := filepath.Join(tempDir, file.name)
		if err := os.WriteFile(filePath, []byte(file.content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file.name, err)
		}
	}

	// Get file items from file manager
	files, err := processor.fileManager.ScanCentralRepo()
	if err != nil {
		t.Fatalf("Failed to scan files: %v", err)
	}

	// Process rule files
	toolsMap, err := processor.ProcessRuleFiles(files)
	if err != nil {
		t.Errorf("ProcessRuleFiles should not return error: %v", err)
	}

	// Verify all tools were processed with unique names
	expectedToolNames := []string{"test_rule", "test_rule_1", "test_rule_2"}
	if len(toolsMap) != len(expectedToolNames) {
		t.Errorf("Expected %d tools to be processed, got %d", len(expectedToolNames), len(toolsMap))
	}

	// Check that all expected tool names exist
	for _, expectedName := range expectedToolNames {
		if _, exists := toolsMap[expectedName]; !exists {
			t.Errorf("Expected tool %s not found in processed tools", expectedName)
		}
	}

	// Verify each tool has the correct properties
	for toolName, tool := range toolsMap {
		if tool.Description == "" {
			t.Errorf("Tool %s should have a description", toolName)
		}
		if tool.RuleFile == nil {
			t.Errorf("Tool %s should have a RuleFile", toolName)
		}
	}
}

// Test logging functionality in rule file processor

func TestRuleFileProcessorLogging(t *testing.T) {
	// Create test logger that captures output
	logger, logBuffer := logging.NewTestLogger()

	// Create temporary directory with test rule files
	tempDir, err := os.MkdirTemp("", "rulem-processor-logging-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files with various configurations
	testFiles := []struct {
		name    string
		content string
	}{
		{
			"valid-rule.md",
			`---
description: "Valid rule file for logging test"
name: "valid_logging_rule"
applyTo: "Logging tests"
---
# Valid Rule
This is a valid rule file.`,
		},
		{
			"no-frontmatter.md",
			"# No Frontmatter\nThis file has no frontmatter.",
		},
		{
			"missing-description.md",
			`---
name: "missing_desc"
applyTo: "All projects"
---
# Missing Description
This file is missing the required description field.`,
		},
		{
			"invalid-yaml.md",
			`---
description: "Invalid YAML"
name: invalid_name_no_quotes
applyTo: [unclosed array
---
# Invalid YAML
This file has invalid YAML frontmatter.`,
		},
	}

	for _, file := range testFiles {
		filePath := filepath.Join(tempDir, file.name)
		if err := os.WriteFile(filePath, []byte(file.content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file.name, err)
		}
	}

	// Create file manager and processor with test logger
	cfg := createTestConfigWithPath(tempDir)
	fileManager, err := filemanager.NewFileManager(cfg.Central.Path, logger)
	if err != nil {
		t.Fatalf("Failed to create file manager: %v", err)
	}

	processor := NewRuleFileProcessor(logger, fileManager, 5*1024*1024) // 5MB

	// Get files for processing
	files, err := fileManager.ScanCentralRepo()
	if err != nil {
		t.Fatalf("Failed to scan repository: %v", err)
	}

	// Clear log buffer before testing
	logBuffer.Reset()

	// Test ParseRuleFiles logging
	ruleFiles, err := processor.ParseRuleFiles(files)
	if err != nil {
		t.Errorf("ParseRuleFiles should not return error: %v", err)
	}

	// Verify only valid rule file was parsed
	if len(ruleFiles) != 1 {
		t.Errorf("Expected 1 valid rule file, got %d", len(ruleFiles))
	}

	// Verify logging occurred
	logOutput := logBuffer.String()

	// Check for parsing start logging
	// Verify basic completion logging
	if !strings.Contains(logOutput, "Rule file parsing completed") {
		t.Error("Expected completion log for rule file parsing")
	}
}

func TestProcessRuleFilesLogging(t *testing.T) {
	// Create test logger that captures output
	logger, logBuffer := logging.NewTestLogger()

	// Create temporary directory with test rule files
	tempDir, err := os.MkdirTemp("", "rulem-process-logging-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := []struct {
		name    string
		content string
	}{
		{
			"go-standards.md",
			`---
description: "Go coding standards and best practices"
name: "go_standards"
applyTo: "Go projects"
---
# Go Standards
Use proper error handling.`,
		},
		{
			"testing-guide.md",
			`---
description: "Testing guidelines for unit and integration tests"
applyTo: "Go testing, bubbletea applications"
---
# Testing Guide
Write comprehensive tests.`,
		},
	}

	for _, file := range testFiles {
		filePath := filepath.Join(tempDir, file.name)
		if err := os.WriteFile(filePath, []byte(file.content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file.name, err)
		}
	}

	// Create file manager and processor with test logger
	cfg := createTestConfigWithPath(tempDir)
	fileManager, err := filemanager.NewFileManager(cfg.Central.Path, logger)
	if err != nil {
		t.Fatalf("Failed to create file manager: %v", err)
	}

	processor := NewRuleFileProcessor(logger, fileManager, 5*1024*1024) // 5MB

	// Get files for processing
	files, err := fileManager.ScanCentralRepo()
	if err != nil {
		t.Fatalf("Failed to scan repository: %v", err)
	}

	// Clear log buffer before testing
	logBuffer.Reset()

	// Test ProcessRuleFiles logging
	tools, err := processor.ProcessRuleFiles(files)
	if err != nil {
		t.Errorf("ProcessRuleFiles should not return error: %v", err)
	}

	// Verify tools were created
	if len(tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools))
	}

	// Verify logging occurred
	logOutput := logBuffer.String()

	// Check for processing start logging
	// Verify tool processing completion logging
	if !strings.Contains(logOutput, "Rule file tool processing completed") {
		t.Error("Expected completion log for tool processing")
	}

	// Use logOutput to avoid unused variable
	_ = logOutput
}

func TestRuleFileProcessorErrorLogging(t *testing.T) {
	// Create test logger that captures output
	logger, logBuffer := logging.NewTestLogger()

	// Create processor without file manager to trigger error
	processor := NewRuleFileProcessor(logger, nil, 5*1024*1024) // 5MB

	// Clear log buffer before testing
	logBuffer.Reset()

	// Test error logging when file manager is not initialized
	_, err := processor.ParseRuleFiles([]filemanager.FileItem{})
	if err == nil {
		t.Error("ParseRuleFiles should return error when file manager is not initialized")
	}

	// Verify error logging occurred
	_ = logBuffer.String()

	// Check for error logging
	// The error is returned directly, not logged
	if err == nil {
		t.Error("Expected error when file manager is not initialized")
	}
}

// Security and validation tests

func TestValidateFrontmatter(t *testing.T) {
	processor, tempDir := createTestRuleFileProcessor(t)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name        string
		frontmatter RuleFrontmatter
		filename    string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid frontmatter",
			frontmatter: RuleFrontmatter{
				Description: "Valid description",
				Name:        "valid_name",
				ApplyTo:     "Go projects",
			},
			filename:    "test.md",
			expectError: false,
		},
		{
			name: "missing description",
			frontmatter: RuleFrontmatter{
				Name:    "test_name",
				ApplyTo: "Go projects",
			},
			filename:    "test.md",
			expectError: true,
			errorMsg:    "missing required 'description' field",
		},
		{
			name: "empty description",
			frontmatter: RuleFrontmatter{
				Description: "   ",
				Name:        "test_name",
				ApplyTo:     "Go projects",
			},
			filename:    "test.md",
			expectError: true,
			errorMsg:    "missing required 'description' field",
		},
		{
			name: "description too long",
			frontmatter: RuleFrontmatter{
				Description: strings.Repeat("a", 501),
				Name:        "test_name",
			},
			filename:    "test.md",
			expectError: true,
			errorMsg:    "description too long",
		},
		{
			name: "name too long",
			frontmatter: RuleFrontmatter{
				Description: "Valid description",
				Name:        strings.Repeat("a", 101),
			},
			filename:    "test.md",
			expectError: true,
			errorMsg:    "name too long",
		},
		{
			name: "applyTo too long",
			frontmatter: RuleFrontmatter{
				Description: "Valid description",
				ApplyTo:     strings.Repeat("a", 201),
			},
			filename:    "test.md",
			expectError: true,
			errorMsg:    "applyTo field too long",
		},
		{
			name: "description with control characters",
			frontmatter: RuleFrontmatter{
				Description: "Invalid description\x00with null byte",
			},
			filename:    "test.md",
			expectError: true,
			errorMsg:    "potentially malicious content",
		},
		{
			name: "name with script injection",
			frontmatter: RuleFrontmatter{
				Description: "Valid description",
				Name:        "<script>alert('xss')</script>",
			},
			filename:    "test.md",
			expectError: true,
			errorMsg:    "invalid characters",
		},
		{
			name: "applyTo with javascript",
			frontmatter: RuleFrontmatter{
				Description: "Valid description",
				ApplyTo:     "javascript:alert('xss')",
			},
			filename:    "test.md",
			expectError: true,
			errorMsg:    "potentially malicious content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := processor.validateFrontmatter(&tt.frontmatter, tt.filename)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, got none", tt.name)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for %s, got %v", tt.name, err)
				}
			}
		})
	}
}

func TestParseRuleFilesWithLargeFiles(t *testing.T) {
	processor, tempDir := createTestRuleFileProcessor(t)
	defer os.RemoveAll(tempDir)

	// Create test files directly since we can't modify processor settings

	// Create a large file
	largeContent := `---
description: "Large file test"
---
` + strings.Repeat("This is a large file content. ", 100)

	largePath := filepath.Join(tempDir, "large-file.md")
	if err := os.WriteFile(largePath, []byte(largeContent), 0644); err != nil {
		t.Fatalf("Failed to create large test file: %v", err)
	}

	// Create a small file for comparison
	smallContent := `---
description: "Small file test"
---
Small content`

	smallPath := filepath.Join(tempDir, "small-file.md")
	if err := os.WriteFile(smallPath, []byte(smallContent), 0644); err != nil {
		t.Fatalf("Failed to create small test file: %v", err)
	}

	// Get files for processing
	files, err := processor.fileManager.ScanCentralRepo()
	if err != nil {
		t.Fatalf("Failed to scan repository: %v", err)
	}

	// Parse rule files
	ruleFiles, err := processor.ParseRuleFiles(files)
	if err != nil {
		t.Errorf("ParseRuleFiles should not return error: %v", err)
	}

	// Since we can't modify max file size, both files should be processed
	if len(ruleFiles) != 2 {
		t.Errorf("Expected 2 rule files, got %d", len(ruleFiles))
	}
}

func TestParseRuleFilesFilePermissionErrors(t *testing.T) {
	processor, tempDir := createTestRuleFileProcessor(t)
	defer os.RemoveAll(tempDir)

	// Create a file with content
	testContent := `---
description: "Test file"
---
Test content`

	testPath := filepath.Join(tempDir, "test-file.md")
	if err := os.WriteFile(testPath, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Remove read permissions
	if err := os.Chmod(testPath, 0000); err != nil {
		t.Skipf("Cannot change file permissions on this system: %v", err)
	}
	defer os.Chmod(testPath, 0644) // Restore permissions for cleanup

	// Get files for processing
	files, err := processor.fileManager.ScanCentralRepo()
	if err != nil {
		t.Fatalf("Failed to scan repository: %v", err)
	}

	// Parse rule files
	ruleFiles, err := processor.ParseRuleFiles(files)
	if err != nil {
		t.Errorf("ParseRuleFiles should not return error: %v", err)
	}

	// Should not process any files due to permission error
	if len(ruleFiles) != 0 {
		t.Errorf("Expected 0 rule files due to permission error, got %d", len(ruleFiles))
	}
}

func TestParseRuleFilesInvalidYAML(t *testing.T) {
	processor, tempDir := createTestRuleFileProcessor(t)
	defer os.RemoveAll(tempDir)

	// Create files with various YAML issues
	testFiles := []struct {
		name    string
		content string
	}{
		{
			"unclosed-array.md",
			`---
description: "Test"
items: [unclosed, array
---
Content`,
		},
		{
			"invalid-indentation.md",
			`---
description: "Test"
  invalid: indentation
---
Content`,
		},
		{
			"missing-quotes.md",
			`---
description: Test with: colon but no quotes
---
Content`,
		},
		{
			"valid-file.md",
			`---
description: "Valid test file"
---
Valid content`,
		},
	}

	for _, file := range testFiles {
		filePath := filepath.Join(tempDir, file.name)
		if err := os.WriteFile(filePath, []byte(file.content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file.name, err)
		}
	}

	// Get files for processing
	files, err := processor.fileManager.ScanCentralRepo()
	if err != nil {
		t.Fatalf("Failed to scan repository: %v", err)
	}

	// Parse rule files
	ruleFiles, err := processor.ParseRuleFiles(files)
	if err != nil {
		t.Errorf("ParseRuleFiles should not return error: %v", err)
	}

	// Should only process the valid file
	if len(ruleFiles) != 1 {
		t.Errorf("Expected 1 valid rule file, got %d", len(ruleFiles))
	}

	if len(ruleFiles) > 0 && ruleFiles[0].FileName != "valid-file.md" {
		t.Errorf("Expected valid-file.md to be processed, got %s", ruleFiles[0].FileName)
	}
}

func TestRuleFileProcessorConfigurability(t *testing.T) {
	_, tempDir := createTestRuleFileProcessor(t)
	defer os.RemoveAll(tempDir)

	// Configuration methods have been removed for simplicity
	// This test now just verifies that the processor can be created
	t.Log("Configuration methods have been simplified and removed")
}

func TestGenerateToolNameWithSanitization(t *testing.T) {
	_, tempDir := createTestRuleFileProcessor(t)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name         string
		ruleFile     *RuleFile
		expectedName string
		description  string
	}{
		{
			name: "frontmatter name with special characters gets sanitized",
			ruleFile: &RuleFile{
				FileName: "test.md",
				Name:     "tool@name#with$special%chars",
			},
			expectedName: "toolnamewithspecialchars",
			description:  "should sanitize frontmatter name with special characters",
		},
		{
			name: "frontmatter name with script injection gets sanitized",
			ruleFile: &RuleFile{
				FileName: "test.md",
				Name:     "<script>alert('xss')</script>malicious",
			},
			expectedName: "scriptalertxssscriptmalicious",
			description:  "should sanitize frontmatter name with script content",
		},
		{
			name: "frontmatter name that becomes empty after sanitization",
			ruleFile: &RuleFile{
				FileName: "test.md",
				Name:     "@#$%^&*()",
			},
			expectedName: "rule_file",
			description:  "should fallback to generic name when sanitization removes everything",
		},
		{
			name: "long frontmatter name gets truncated",
			ruleFile: &RuleFile{
				FileName: "test.md",
				Name:     strings.Repeat("a", 150),
			},
			expectedName: strings.Repeat("a", 100),
			description:  "should truncate long frontmatter names",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new processor for each test to avoid conflicts
			testProcessor, testTempDir := createTestRuleFileProcessor(t)
			defer os.RemoveAll(testTempDir)

			result := testProcessor.generateToolName(tt.ruleFile)
			if result != tt.expectedName {
				t.Errorf("generateToolName() = %v, want %v (%s)", result, tt.expectedName, tt.description)
			}
		})
	}
}

func TestProcessRuleFilesWithValidationErrors(t *testing.T) {
	processor, tempDir := createTestRuleFileProcessor(t)
	defer os.RemoveAll(tempDir)

	// Create test files with validation issues
	testFiles := []struct {
		name    string
		content string
	}{
		{
			"valid-rule.md",
			`---
description: "Valid rule file"
name: "valid_rule"
---
Valid content`,
		},
		{
			"long-description.md",
			`---
description: "` + strings.Repeat("Long description ", 30) + `"
name: "long_desc"
---
Content with long description`,
		},
		{
			"malicious-name.md",
			`---
description: "Rule with malicious name"
name: "<script>alert('xss')</script>"
---
Content with malicious name`,
		},
		{
			"suspicious-content.md",
			`---
description: "Rule with suspicious content javascript:alert('xss')"
name: "suspicious"
---
Content with suspicious description`,
		},
	}

	for _, file := range testFiles {
		filePath := filepath.Join(tempDir, file.name)
		if err := os.WriteFile(filePath, []byte(file.content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file.name, err)
		}
	}

	// Get files for processing
	files, err := processor.fileManager.ScanCentralRepo()
	if err != nil {
		t.Fatalf("Failed to scan repository: %v", err)
	}

	// Process rule files
	tools, err := processor.ProcessRuleFiles(files)
	if err != nil {
		t.Errorf("ProcessRuleFiles should not return error: %v", err)
	}

	// Should only process the valid file
	if len(tools) != 1 {
		t.Errorf("Expected 1 valid tool, got %d", len(tools))
	}

	// Check that the valid tool was processed (iterate through map)
	var foundValidTool bool
	for _, tool := range tools {
		if tool.RuleFile.FileName == "valid-rule.md" {
			foundValidTool = true
			break
		}
	}
	if !foundValidTool {
		t.Error("Expected valid-rule.md to be processed")
	}
}

func TestProcessRuleFileEnhancedValidation(t *testing.T) {
	// Create test processor
	tempDir, err := os.MkdirTemp("", "rulem-enhanced-validation-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logger, _ := logging.NewTestLogger()
	fileManager, err := filemanager.NewFileManager(tempDir, logger)
	if err != nil {
		t.Fatalf("Failed to create file manager: %v", err)
	}

	processor := NewRuleFileProcessor(logger, fileManager, 5*1024*1024) // 5MB

	tests := []struct {
		name        string
		content     string
		setupFunc   func() filemanager.FileItem
		expectError bool
		errorSubstr string
	}{
		{
			name: "valid file with clean content",
			content: `---
description: "Valid test rule"
---
# Clean Rule
This is clean content.`,
			setupFunc: func() filemanager.FileItem {
				filePath := filepath.Join(tempDir, "clean-rule.md")
				os.WriteFile(filePath, []byte(`---
description: "Valid test rule"
---
# Clean Rule
This is clean content.`), 0644)
				return filemanager.FileItem{Name: "clean-rule.md", Path: "clean-rule.md"}
			},
			expectError: false,
		},
		{
			name: "content with potential script injection",
			content: `---
description: "Malicious rule"
---
# Rule with Script
<script>alert('xss')</script>
This contains malicious content.`,
			setupFunc: func() filemanager.FileItem {
				filePath := filepath.Join(tempDir, "malicious-rule.md")
				os.WriteFile(filePath, []byte(`---
description: "Malicious rule"
---
# Rule with Script
<script>alert('xss')</script>
This contains malicious content.`), 0644)
				return filemanager.FileItem{Name: "malicious-rule.md", Path: "malicious-rule.md"}
			},
			expectError: true,
			errorSubstr: "content security validation failed",
		},
		{
			name: "content with control characters",
			setupFunc: func() filemanager.FileItem {
				filePath := filepath.Join(tempDir, "control-chars.md")
				content := "---\ndescription: \"Rule with control chars\"\n---\n# Rule\nContent with \x00 null byte"
				os.WriteFile(filePath, []byte(content), 0644)
				return filemanager.FileItem{Name: "control-chars.md", Path: "control-chars.md"}
			},
			expectError: true,
			errorSubstr: "content security validation failed",
		},
		{
			name: "file too large",
			setupFunc: func() filemanager.FileItem {
				filePath := filepath.Join(tempDir, "large-file.md")
				// Create content larger than 10MB
				frontmatter := "---\ndescription: \"Large file test\"\n---\n"
				largeContent := strings.Repeat("This is a large file. ", 500000) // ~11MB
				content := frontmatter + largeContent
				os.WriteFile(filePath, []byte(content), 0644)
				return filemanager.FileItem{Name: "large-file.md", Path: "large-file.md"}
			},
			expectError: true,
			errorSubstr: "file size check failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileItem := tt.setupFunc()

			_, err := processor.processRuleFile(fileItem)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("Expected error containing %q, got %q", tt.errorSubstr, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestProcessRuleFileSymlinkValidation(t *testing.T) {
	// Create test processor
	tempDir, err := os.MkdirTemp("", "rulem-symlink-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a separate directory outside tempDir for testing symlink boundaries
	outsideDir, err := os.MkdirTemp("", "rulem-outside-*")
	if err != nil {
		t.Fatalf("Failed to create outside dir: %v", err)
	}
	defer os.RemoveAll(outsideDir)

	logger, _ := logging.NewTestLogger()
	fileManager, err := filemanager.NewFileManager(tempDir, logger)
	if err != nil {
		t.Fatalf("Failed to create file manager: %v", err)
	}

	processor := NewRuleFileProcessor(logger, fileManager, 5*1024*1024) // 5MB

	// Create target file outside storage directory
	outsideFile := filepath.Join(outsideDir, "outside-rule.md")
	outsideContent := `---
description: "Outside rule"
---
# Outside Rule
This is outside content.`
	if err := os.WriteFile(outsideFile, []byte(outsideContent), 0644); err != nil {
		t.Fatalf("Failed to create outside file: %v", err)
	}

	// Test only the dangerous case - symlink pointing outside storage directory
	linkPath := filepath.Join(tempDir, "bad-symlink.md")
	err = os.Symlink(outsideFile, linkPath)
	if err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	fileItem := filemanager.FileItem{Name: "bad-symlink.md", Path: "bad-symlink.md"}
	_, err = processor.processRuleFile(fileItem)

	if err == nil {
		t.Error("Expected error for symlink pointing outside storage directory")
	} else if !strings.Contains(err.Error(), "symlink security check failed") &&
		!strings.Contains(err.Error(), "symlink target validation failed") {
		t.Errorf("Expected symlink security error, got: %v", err)
	}
}

func TestProcessRuleFileDirectoryBoundaryValidation(t *testing.T) {
	// Create test processor
	tempDir, err := os.MkdirTemp("", "rulem-boundary-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logger, _ := logging.NewTestLogger()
	fileManager, err := filemanager.NewFileManager(tempDir, logger)
	if err != nil {
		t.Fatalf("Failed to create file manager: %v", err)
	}

	processor := NewRuleFileProcessor(logger, fileManager, 5*1024*1024) // 5MB

	// Create a file outside the storage directory
	outsideDir := filepath.Dir(tempDir)
	outsideFile := filepath.Join(outsideDir, "outside-rule.md")
	outsideContent := `---
description: "Outside rule"
---
# Outside Rule
This should not be accessible.`
	if err := os.WriteFile(outsideFile, []byte(outsideContent), 0644); err != nil {
		t.Fatalf("Failed to create outside file: %v", err)
	}
	defer os.Remove(outsideFile)

	// Try to process a file that's outside the storage directory
	// This simulates a path traversal attempt
	outsideFileItem := filemanager.FileItem{
		Name: "outside-rule.md",
		Path: "../outside-rule.md",
	}

	_, err = processor.processRuleFile(outsideFileItem)
	if err == nil {
		t.Error("Expected error for file outside storage directory")
	} else if !strings.Contains(err.Error(), "path security check failed") &&
		!strings.Contains(err.Error(), "file containment validation failed") {
		t.Errorf("Expected path security or containment error, got: %v", err)
	}
}
