// Package mcp implements a Model Context Protocol (MCP) server for rulem using the mcp-go library.
//
// This package provides MCP server functionality that allows AI assistants to interact
// with rulem's rule management capabilities through a standardized protocol. The server
// exposes tools and resources for accessing organized instruction files.
//
// The implementation uses the mcp-go library for protocol handling and communicates
// via stdin/stdout using JSON-RPC 2.0 as specified by the MCP standard.
package mcp

import (
	"bytes"
	"fmt"
	"os"

	"rulem/internal/config"
	"rulem/internal/filemanager"
	"rulem/internal/logging"

	"github.com/adrg/frontmatter"
	"github.com/mark3labs/mcp-go/server"
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

// Server represents an MCP server instance using mcp-go
type Server struct {
	config      *config.Config
	logger      *logging.AppLogger
	fileManager *filemanager.FileManager
	mcpServer   *server.MCPServer
}

// NewServer creates a new MCP server instance
func NewServer(cfg *config.Config, logger *logging.AppLogger) *Server {
	return &Server{
		config: cfg,
		logger: logger,
	}
}

// Start initializes and starts the MCP server
func (s *Server) Start() error {
	s.logger.Info("Initializing MCP server")

	// Initialize file manager for repository scanning
	var err error
	s.fileManager, err = filemanager.NewFileManager(s.config.StorageDir, s.logger)
	if err != nil {
		return fmt.Errorf("failed to initialize file manager: %w", err)
	}

	ruleFiles, err := s.getRuleFilesWithFrontmatter()
	if err != nil {
		return fmt.Errorf("failed to get rule files with frontmatter: %w", err)
	}

	s.logger.Info("Rule files with frontmatter loaded", "ruleCount", len(ruleFiles))
	if len(ruleFiles) > 0 {
		for _, rf := range ruleFiles {
			s.logger.Info("Rule",
				"name", rf.FileName,
				"description", rf.Description,
				"ruleName", rf.Name,
				"applyTo", rf.ApplyTo,
			)
		}
	}

	// MCP server setup and start logic would go here
	// For now, we log that the server would start
	s.logger.Info("MCP server setup complete (server start logic not implemented)")

	// // Create MCP server with basic capabilities
	// s.mcpServer = server.NewMCPServer("rulem", "1.0.0")

	// s.logger.Info("MCP server created, starting stdio communication")

	// // Start the server using stdio transport
	// if err := server.ServeStdio(s.mcpServer); err != nil {
	// 	return fmt.Errorf("MCP server failed: %w", err)
	// }

	return nil
}

// Stop gracefully shuts down the MCP server
func (s *Server) Stop() error {
	s.logger.Info("Stopping MCP server")
	// The mcp-go server will handle cleanup when context is cancelled
	return nil
}

// getRepoFiles scans the central repository and returns the list of files
// This helper function uses the filemanager to call ScanCentralRepo()
func (s *Server) getRepoFiles() ([]filemanager.FileItem, error) {
	if s.fileManager == nil {
		return nil, fmt.Errorf("file manager not initialized")
	}

	s.logger.Debug("Scanning central repository for files")

	files, err := s.fileManager.ScanCentralRepo()
	if err != nil {
		s.logger.Error("Failed to scan central repository", "error", err)
		return nil, fmt.Errorf("failed to scan central repository: %w", err)
	}

	s.logger.Debug("Successfully scanned central repository", "fileCount", len(files))
	return files, nil
}

// parseRuleFiles takes a list of file items and parses them for frontmatter
// Returns only files that have valid YAML frontmatter with at least a 'description' field
func (s *Server) parseRuleFiles(files []filemanager.FileItem) ([]RuleFile, error) {
	if s.fileManager == nil {
		return nil, fmt.Errorf("file manager not initialized")
	}

	var ruleFiles []RuleFile
	var skippedCount int

	s.logger.Debug("Starting frontmatter parsing", "totalFiles", len(files))

	for _, file := range files {
		// Get absolute path for reading
		absolutePath := s.fileManager.GetAbsolutePath(file)

		s.logger.Debug("Parsing file", "name", file.Name, "path", file.Path)

		// Read file content
		content, err := os.ReadFile(absolutePath)
		if err != nil {
			s.logger.Debug("Failed to read file, skipping", "file", file.Name, "error", err)
			skippedCount++
			continue
		}

		// Parse frontmatter
		var matter RuleFrontmatter
		body, err := frontmatter.Parse(bytes.NewReader(content), &matter)
		if err != nil {
			s.logger.Debug("No valid frontmatter found, skipping", "file", file.Name, "error", err)
			skippedCount++
			continue
		}

		// Check if description field exists (required)
		if matter.Description == "" {
			s.logger.Debug("Missing required 'description' field in frontmatter, skipping", "file", file.Name)
			skippedCount++
			continue
		}

		// Create RuleFile with parsed data
		ruleFile := RuleFile{
			FileName:    file.Name,
			FilePath:    file.Path,
			Description: matter.Description,
			Name:        matter.Name,
			ApplyTo:     matter.ApplyTo,
			Content:     string(body),
		}

		ruleFiles = append(ruleFiles, ruleFile)
		s.logger.Debug("Successfully parsed rule file",
			"file", file.Name,
			"description", matter.Description,
			"name", matter.Name,
			"applyTo", matter.ApplyTo,
		)
	}

	s.logger.Info("Frontmatter parsing completed",
		"validRules", len(ruleFiles),
		"skippedFiles", skippedCount,
		"totalFiles", len(files),
	)

	return ruleFiles, nil
}

// getRuleFilesWithFrontmatter scans the repository and returns parsed rule files with frontmatter
func (s *Server) getRuleFilesWithFrontmatter() ([]RuleFile, error) {
	// First get all files from repository
	files, err := s.getRepoFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to get repository files: %w", err)
	}

	// Parse frontmatter from files
	ruleFiles, err := s.parseRuleFiles(files)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rule files: %w", err)
	}

	return ruleFiles, nil
}

// InitializeComponents is a public method for testing and demo purposes
// It initializes the file manager without starting the full MCP server
func (s *Server) InitializeComponents() error {
	var err error
	s.fileManager, err = filemanager.NewFileManager(s.config.StorageDir, s.logger)
	return err
}

// GetRuleFilesWithFrontmatter is a public method for testing and demo purposes
// It returns parsed rule files with frontmatter
func (s *Server) GetRuleFilesWithFrontmatter() ([]RuleFile, error) {
	return s.getRuleFilesWithFrontmatter()
}
