package mcp

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"rulem/internal/filemanager"
	"rulem/internal/logging"
	"rulem/pkg/fileops"

	"github.com/adrg/frontmatter"
)

// Constants for configuring tool description generation
// These constants allow easy modification of description formatting without
// requiring changes throughout the codebase or test files
const (
	// ToolDescriptionPrefix is prepended to all tool descriptions
	ToolDescriptionPrefix = ""

	// ApplyToFormat defines how the applyTo field is formatted in descriptions
	ApplyToFormat = "apply to"
)

// RuleFrontmatter represents the YAML frontmatter structure expected in rule files
type RuleFrontmatter struct {
	Description string `yaml:"description"`
	Name        string `yaml:"name,omitempty"`
	ApplyTo     string `yaml:"applyTo,omitempty"`
}

// RuleFile represents a parsed rule file with frontmatter and content
type RuleFile struct {
	// File information
	FileName string
	FilePath string

	// Frontmatter fields
	Description string
	Name        string
	ApplyTo     string

	// File content (without frontmatter)
	Content string
}

// RuleFileTool represents a rule file registered as an MCP tool
type RuleFileTool struct {
	Name        string
	Description string
	RuleFile    *RuleFile
}

// RuleFileProcessor handles rule file operations including parsing, naming, and tool generation
type RuleFileProcessor struct {
	logger       *logging.AppLogger
	fileManager  *filemanager.FileManager
	toolRegistry map[string]*RuleFileTool
	maxFileSize  int64 // Maximum file size in bytes
}

// NewRuleFileProcessor creates a new RuleFileProcessor instance
func NewRuleFileProcessor(logger *logging.AppLogger, fileManager *filemanager.FileManager, maxFileSize int64) *RuleFileProcessor {
	return &RuleFileProcessor{
		logger:       logger,
		fileManager:  fileManager,
		toolRegistry: make(map[string]*RuleFileTool),
		maxFileSize:  maxFileSize,
	}
}

// ParseRuleFiles takes a list of file items and parses them for frontmatter
// Returns only files that have valid YAML frontmatter with at least a 'description' field
func (p *RuleFileProcessor) ParseRuleFiles(files []filemanager.FileItem) ([]RuleFile, error) {
	if p.fileManager == nil {
		return nil, fmt.Errorf("file manager not initialized")
	}

	var ruleFiles []RuleFile
	var skippedCount int

	for _, file := range files {
		ruleFile, err := p.processRuleFile(file)
		if err != nil {
			p.logger.Debug("Skipping file", "name", file.Name, "reason", err)
			skippedCount++
			continue
		}

		ruleFiles = append(ruleFiles, *ruleFile)
	}

	p.logger.Info("Rule file parsing completed",
		"totalFiles", len(files),
		"validRules", len(ruleFiles),
		"skipped", skippedCount)

	return ruleFiles, nil
}

// processRuleFile handles the complete processing pipeline for a single rule file
func (p *RuleFileProcessor) processRuleFile(file filemanager.FileItem) (*RuleFile, error) {
	absolutePath := p.fileManager.GetAbsolutePath(file)

	// Comprehensive file validation using fileops functions
	if err := p.validateRuleFileAccess(absolutePath, file.Path); err != nil {
		return nil, fmt.Errorf("file validation failed: %w", err)
	}

	// Read and parse file content
	content, err := os.ReadFile(absolutePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Validate content security for malicious patterns
	if err := fileops.ValidateContentSecurity(string(content)); err != nil {
		return nil, fmt.Errorf("content security validation failed: %w", err)
	}

	// Parse frontmatter
	var matter RuleFrontmatter
	body, err := frontmatter.Parse(bytes.NewReader(content), &matter)
	if err != nil {
		return nil, fmt.Errorf("no valid frontmatter found: %w", err)
	}

	// Validate frontmatter fields
	if err := p.validateFrontmatter(&matter, file.Name); err != nil {
		return nil, fmt.Errorf("invalid frontmatter: %w", err)
	}

	// Create and return RuleFile
	ruleFile := &RuleFile{
		FileName:    file.Name,
		FilePath:    file.Path,
		Description: matter.Description,
		Name:        matter.Name,
		ApplyTo:     matter.ApplyTo,
		Content:     string(body),
	}

	return ruleFile, nil
}

// validateRuleFileAccess performs comprehensive file validation using fileops functions
func (p *RuleFileProcessor) validateRuleFileAccess(absolutePath, relativePath string) error {
	// Basic path security validation
	if err := fileops.ValidatePathSecurity(relativePath); err != nil {
		return fmt.Errorf("path security check failed: %w", err)
	}

	// Validate file size limits (10MB max)
	if err := fileops.ValidateFileSizeLimit(absolutePath, p.maxFileSize); err != nil {
		return fmt.Errorf("file size check failed: %w", err)
	}

	// Validate file access permissions (read-only required)
	if err := fileops.ValidateFileAccess(absolutePath, false); err != nil {
		return fmt.Errorf("file access check failed: %w", err)
	}

	// Get storage directory for containment validation
	// Use the file manager's storage directory
	storageDir := p.fileManager.GetStorageDir()

	// Validate that file is within the storage directory boundary
	if err := fileops.ValidateFileInDirectory(absolutePath, storageDir); err != nil {
		return fmt.Errorf("file containment validation failed: %w", err)
	}

	// If it's a symlink, perform comprehensive symlink security validation
	if isSymlink, err := fileops.IsSymlink(absolutePath); err != nil {
		return fmt.Errorf("symlink check failed: %w", err)
	} else if isSymlink {
		allowedPaths := []string{storageDir}
		if err := fileops.ValidateSymlinkSecurity(absolutePath, allowedPaths); err != nil {
			return fmt.Errorf("symlink security check failed: %w", err)
		}

		// Additional symlink validation: ensure target exists and is accessible
		if target, err := fileops.ResolveSymlink(absolutePath); err != nil {
			return fmt.Errorf("symlink resolution failed: %w", err)
		} else {
			// Validate the resolved target is also within bounds
			if err := fileops.ValidateFileInDirectory(target, storageDir); err != nil {
				return fmt.Errorf("symlink target validation failed: %w", err)
			}
		}
	}

	return nil
}

// generateToolName creates a unique tool name from rule file metadata
// Uses frontmatter name field if provided, otherwise generates from filename
// Handles duplicate names by appending numeric suffixes
func (p *RuleFileProcessor) generateToolName(ruleFile *RuleFile) string {
	var baseName string

	// Use frontmatter name field if provided, but sanitize it for security
	if ruleFile.Name != "" {
		var err error
		baseName, err = fileops.SanitizeIdentifier(ruleFile.Name, 100)
		if err != nil {
			baseName = "rule_file" // Fallback if sanitization fails
		}
	} else {
		// Generate from filename using fileops.SanitizeFilename
		filename := ruleFile.FileName
		if idx := strings.LastIndex(filename, "."); idx != -1 {
			filename = filename[:idx]
		}

		sanitized, err := fileops.SanitizeFilename(filename)
		if err != nil {
			baseName = "rule_file"
		} else {
			// Further sanitize for tool names using SanitizeIdentifier
			baseName, err = fileops.SanitizeIdentifier(sanitized, 100)
			if err != nil {
				baseName = "rule_file"
			} else {
				// Replace remaining hyphens with underscores for consistency with test expectations
				baseName = strings.ReplaceAll(baseName, "-", "_")
			}
		}
	}

	// Ensure the name is not empty
	if baseName == "" {
		baseName = "rule_file"
	}

	// Handle duplicate names by checking registry and appending numeric suffix
	finalName := baseName
	counter := 1

	for {
		if _, exists := p.toolRegistry[finalName]; !exists {
			break
		}
		finalName = fmt.Sprintf("%s_%d", baseName, counter)
		counter++
	}

	return finalName
}

// generateToolDescription creates a comprehensive tool description from rule file metadata
// Combines description and applyTo fields according to the format:
// "{description} (applies to: {applyTo})" when applyTo is present, or just "{description}"
func (p *RuleFileProcessor) generateToolDescription(ruleFile *RuleFile) string {
	if ruleFile.Description == "" {
		return "Rule file tool"
	}

	description := ruleFile.Description

	// Add applyTo context if present
	if ruleFile.ApplyTo != "" {
		description = fmt.Sprintf("%s (%s: %s)", description, ApplyToFormat, ruleFile.ApplyTo)
	}

	description = ToolDescriptionPrefix + description

	return description
}

// ProcessRuleFiles processes a list of file items and converts them to RuleFileTools
// This is the main method that orchestrates parsing, naming, and tool creation
// All file validations are performed here during the parsing phase
func (p *RuleFileProcessor) ProcessRuleFiles(files []filemanager.FileItem) (map[string]*RuleFileTool, error) {
	// Parse rule files with comprehensive validation
	ruleFiles, err := p.ParseRuleFiles(files)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rule files: %w", err)
	}

	// Convert each valid rule file to a tool
	for _, ruleFile := range ruleFiles {
		// Generate unique tool name using fileops sanitization
		toolName := p.generateToolName(&ruleFile)

		// Generate comprehensive tool description
		toolDescription := p.generateToolDescription(&ruleFile)

		// Create RuleFileTool instance
		ruleFileTool := &RuleFileTool{
			Name:        toolName,
			Description: toolDescription,
			RuleFile:    &ruleFile,
		}

		// Add to internal registry for duplicate name tracking
		p.toolRegistry[toolName] = ruleFileTool
	}

	p.logger.Info("Rule file tool processing completed",
		"inputFiles", len(files),
		"processedTools", len(p.toolRegistry))

	return p.toolRegistry, nil
}

// validateFrontmatter validates the frontmatter fields for security and correctness
func (p *RuleFileProcessor) validateFrontmatter(matter *RuleFrontmatter, filename string) error {
	// Check if description field exists (required)
	if strings.TrimSpace(matter.Description) == "" {
		return fmt.Errorf("missing required 'description' field")
	}

	// Validate description length and content
	if len(matter.Description) > 500 {
		return fmt.Errorf("description too long (max 500 characters)")
	}

	// Check for potentially malicious content in description
	if err := fileops.ValidateContentSecurity(matter.Description); err != nil {
		return fmt.Errorf("description contains potentially malicious content: %w", err)
	}

	// Validate name field if provided
	if matter.Name != "" {
		if len(matter.Name) > 100 {
			return fmt.Errorf("name too long (max 100 characters)")
		}

		// Check for control characters or other suspicious content
		if err := fileops.ValidateContentSecurity(matter.Name); err != nil {
			return fmt.Errorf("name contains invalid characters: %w", err)
		}
	}

	// Validate applyTo field if provided
	if matter.ApplyTo != "" {
		if len(matter.ApplyTo) > 200 {
			return fmt.Errorf("applyTo field too long (max 200 characters)")
		}

		if err := fileops.ValidateContentSecurity(matter.ApplyTo); err != nil {
			return fmt.Errorf("applyTo contains potentially malicious content: %w", err)
		}
	}

	return nil
}
