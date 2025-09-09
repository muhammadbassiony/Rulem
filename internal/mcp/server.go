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
	"fmt"

	"rulem/internal/config"
	"rulem/internal/filemanager"
	"rulem/internal/logging"

	"github.com/mark3labs/mcp-go/server"
)

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

	files, err := s.getRepoFiles()
	if err != nil {
		return fmt.Errorf("failed to get repository files: %w", err)
	}

	s.logger.Info("Repository files loaded", "fileCount", len(files))
	if len(files) > 0 {
		for _, f := range files {
			s.logger.Info("File", "name", f.Name, "path", f.Path)
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
