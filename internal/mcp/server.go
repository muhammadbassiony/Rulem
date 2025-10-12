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
	"context"
	"fmt"
	"rulem/internal/config"
	"rulem/internal/filemanager"
	"rulem/internal/logging"
	"rulem/internal/repository"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Server struct {
	config        *config.Config
	logger        *logging.AppLogger
	fileManager   *filemanager.FileManager
	mcpServer     *server.MCPServer
	toolRegistry  map[string]*RuleFileTool // Maps tool names to their RuleFileTool instances
	ruleProcessor *RuleFileProcessor       // Handles rule file parsing and processing
}

// NewServer creates a new MCP server instance
func NewServer(cfg *config.Config, logger *logging.AppLogger) *Server {
	return &Server{
		config:       cfg,
		logger:       logger,
		toolRegistry: make(map[string]*RuleFileTool),
	}
}

// Start initializes and starts the MCP server
func (s *Server) Start() error {
	s.logger.Info("Initializing MCP server")

	// Create MCP server instance
	s.mcpServer = server.NewMCPServer("rulem", "1.0.0", server.WithToolCapabilities(true))

	// Prepare repository and initialize file manager for scanning
	localPath, syncInfo, err := repository.PrepareRepository(s.config.Central, s.logger)
	if err != nil {
		// Log the original low-level error for diagnostics
		s.logger.Error("Repository preparation failed", "error", err)

		// Return a clearer, user-facing message to avoid misleading low-level details
		return fmt.Errorf(
			"Cannot prepare the central repository at '%s'. The location may be read-only or not writable. Choose a writable directory and try again. Suggested location: %s",
			s.config.Central.Path,
			repository.GetDefaultStorageDir(),
		)
	}

	// Surface sync information to user if available
	if syncInfo.Message != "" {
		s.logger.Info("Repository status", "message", syncInfo.Message)
	}

	s.fileManager, err = filemanager.NewFileManager(localPath, s.logger)
	if err != nil {
		return fmt.Errorf("failed to initialize file manager: %w", err)
	}

	// Initialize rule file processor
	maxFileSize := int64(5 * 1024 * 1024) // 5 MB
	s.ruleProcessor = NewRuleFileProcessor(s.logger, s.fileManager, maxFileSize)

	// Register rule files as MCP tools
	err = s.RegisterRuleFileTools()
	if err != nil {
		s.logger.Error("Failed to register rule file tools", "error", err)
		return err
	}

	s.logger.Info("Successfully registered rule file tools", "toolCount", len(s.toolRegistry))

	s.logger.Info("MCP server setup complete")

	// Start the stdio server
	s.logger.Info("Starting MCP stdio server")
	if err := server.ServeStdio(s.mcpServer); err != nil {
		s.logger.Error("MCP server error", "error", err)
		return fmt.Errorf("MCP server failed: %w", err)
	}

	s.logger.Info("MCP server stopped")
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

	files, err := s.fileManager.ScanCentralRepo()
	if err != nil {
		s.logger.Error("Failed to scan central repository", "error", err)
		return nil, fmt.Errorf("failed to scan central repository: %w", err)
	}

	return files, nil
}

// RegisterRuleFileTools registers all valid rule files as MCP tools
// This method scans rule files with frontmatter and registers them as callable MCP tools
func (s *Server) RegisterRuleFileTools() error {
	// Get all files from repository
	files, err := s.getRepoFiles()
	if err != nil {
		return fmt.Errorf("failed to get repository files: %w", err)
	}

	// Process rule files using the rule processor
	toolsMap, err := s.ruleProcessor.ProcessRuleFiles(files)
	if err != nil {
		return fmt.Errorf("failed to process rule files: %w", err)
	}

	// Set the server's registry to the processed tools
	s.toolRegistry = toolsMap

	// Loop through tools and register them with the MCP server
	for toolName, tool := range toolsMap {
		s.logger.Debug("Registering MCP tool", "name", toolName, "description", tool.Description)
		// create new MCP tool and its handler
		mcpTool := mcp.NewTool(toolName, mcp.WithDescription(tool.Description))
		handler, err := s.getRulefileToolHandler(toolName)
		if err != nil {
			s.logger.Error("Failed to get tool handler", "tool", toolName, "error", err)
			continue
		}
		s.mcpServer.AddTool(mcpTool, handler)
	}

	return nil
}

// getRulefileToolHandler creates an MCP tool handler function for a specific rule file tool.
// This function returns a handler that can be registered with the MCP server to handle
// tool invocation requests. The handler will return the pre-processed content of the rule file.
//
// The function performs tool validation at handler creation time rather than at each invocation
// for better performance. The returned handler is thread-safe and can be called concurrently.
//
// Parameters:
//   - toolName: The name of the tool to create a handler for (must exist in toolRegistry)
//
// Returns:
//   - server.ToolHandlerFunc: Handler function that can process MCP tool requests
//   - error: Validation error if the tool doesn't exist in the registry
//
// Usage:
//
//	handler, err := server.getRulefileToolHandler("my_tool")
//	if err != nil {
//	    return fmt.Errorf("failed to create handler: %w", err)
//	}
//	mcpServer.AddTool(tool, handler)
func (s *Server) getRulefileToolHandler(toolName string) (server.ToolHandlerFunc, error) {
	// Validate tool exists in registry at handler creation time
	tool, exists := s.toolRegistry[toolName]
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found in registry", toolName)
	}

	// Capture the content at handler creation time for better performance
	content := tool.RuleFile.Content

	// Return the handler function that will be called for each tool invocation
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Log the tool invocation for debugging/monitoring purposes
		s.logger.Debug("Processing rule file tool request",
			"tool", toolName,
			"contentLength", len(content))

		// Check if context was cancelled
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Return the pre-processed rule file content
		return mcp.NewToolResultText(content), nil
	}, nil
}

// InitializeComponents initializes the server components (file manager and rule processor)
// without starting the full MCP server. This method is specifically designed for testing
// scenarios where you need to access server functionality without the MCP server lifecycle.
//
// This method initializes:
//   - FileManager for repository scanning and file operations
//   - RuleFileProcessor for parsing and processing rule files
//
// Use this method in tests when you need to call methods like RegisterRuleFileTools,
// getRepoFiles, or other server methods that depend on these components being initialized.
//
// For production use, prefer Start() which initializes components and starts the MCP server.
//
// Returns:
//   - error: Initialization error if file manager creation fails
func (s *Server) InitializeComponents() error {
	// Prepare repository and initialize file manager for scanning
	localPath, syncInfo, err := repository.PrepareRepository(s.config.Central, s.logger)
	if err != nil {
		// Log the original low-level error for diagnostics
		s.logger.Error("Repository preparation failed", "error", err)

		// Return a clearer, user-facing message to avoid misleading low-level details
		return fmt.Errorf(
			"Cannot prepare the central repository at '%s'. The location may be read-only or not writable. Choose a writable directory and try again. Suggested location: %s",
			s.config.Central.Path,
			repository.GetDefaultStorageDir(),
		)
	}

	// Surface sync information to user if available
	if syncInfo.Message != "" {
		s.logger.Info("Repository status", "message", syncInfo.Message)
	}

	s.fileManager, err = filemanager.NewFileManager(localPath, s.logger)
	if err != nil {
		return fmt.Errorf("failed to initialize file manager: %w", err)
	}

	// Initialize rule file processor
	maxFileSize := int64(5 * 1024 * 1024) // 5 MB
	s.ruleProcessor = NewRuleFileProcessor(s.logger, s.fileManager, maxFileSize)

	return nil
}
