// Package mcp provides Model Context Protocol (MCP) server implementation for rulem using mcp-go.
//
// This package implements an MCP server that allows AI assistants to interact with
// rulem's rule management capabilities through a standardized protocol. The server
// provides the rules in the central rule files repo as tools.
// It only adds rule files that have frontmatter with a description field.
//
// # Implementation
//
// The package uses the mcp-go library (github.com/mark3labs/mcp-go).
//
// # File Management Integration
//
// The server integrates with rulem's filemanager package to provide secure access
// to organized instruction files. It uses the existing scanning and validation
// infrastructure to ensure safe file operations.
//
// # Security
//
// Security is handled through the underlying filemanager and fileops packages:
//   - Path validation to prevent directory traversal
//   - Access restricted to configured storage directory
//   - Secure file scanning with symlink protection
//   - Read-only access to rule files
//
// # Usage
//
// The MCP server is typically started as a subprocess by AI assistants that support
// MCP integration. It can also be started manually for testing:
//
//	rulem mcp
//
// The server will read JSON-RPC requests from stdin and write responses to stdout
// until it receives EOF or is terminated.
//
// # Architecture
//
// The Server struct contains:
//   - config: Application configuration with storage directory
//   - logger: Application logger for debugging and audit
//   - fileManager: File management interface for repository operations
//   - mcpServer: The underlying mcp-go server instance
//
// # Example Integration
//
// AI assistants can integrate with this MCP server to access rulem's organized
// instruction files, enabling context-aware assistance based on the user's
// configured rules and guidelines.
//
// # References
//
// - MCP Specification: https://modelcontextprotocol.io/specification
// - mcp-go Library: https://github.com/mark3labs/mcp-go
// - rulem Project: See project documentation for integration details
package mcp
