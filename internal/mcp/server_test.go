package mcp

import (
	"os"
	"path/filepath"
	"testing"

	"rulem/internal/config"
	"rulem/internal/filemanager"
	"rulem/internal/logging"
)

func TestNewServer(t *testing.T) {
	cfg := &config.Config{StorageDir: "/tmp/test"}
	logger := logging.NewAppLogger()

	server := NewServer(cfg, logger)

	if server == nil {
		t.Fatal("NewServer returned nil")
	}
	if server.config != cfg {
		t.Error("Server config not set correctly")
	}
	if server.logger != logger {
		t.Error("Server logger not set correctly")
	}
	if server.fileManager != nil {
		t.Error("FileManager should not be initialized until Start() is called")
	}
	if server.mcpServer != nil {
		t.Error("MCP server should not be initialized until Start() is called")
	}
}

func TestGetRepoFiles(t *testing.T) {
	server, tempDir := createTestServerWithFiles(t)
	defer os.RemoveAll(tempDir)

	// Initialize the server components (normally done in Start())
	err := server.initializeComponents()
	if err != nil {
		t.Fatalf("Failed to initialize server components: %v", err)
	}

	files, err := server.getRepoFiles()
	if err != nil {
		t.Errorf("getRepoFiles should not return error: %v", err)
	}

	if len(files) == 0 {
		t.Error("getRepoFiles should find at least one test file")
	}

	// Check that the test file is found
	found := false
	for _, file := range files {
		if file.Name == "test-rule.md" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Test file should be found in repository scan")
	}
}

func TestGetRepoFilesWithoutFileManager(t *testing.T) {
	cfg := &config.Config{StorageDir: "/tmp/test"}
	logger := logging.NewAppLogger()
	server := NewServer(cfg, logger)

	// Try to call getRepoFiles without initializing fileManager
	_, err := server.getRepoFiles()
	if err == nil {
		t.Error("getRepoFiles should return error when fileManager is not initialized")
	}
	if err.Error() != "file manager not initialized" {
		t.Errorf("Expected 'file manager not initialized' error, got: %v", err)
	}
}

func TestGetRepoFilesWithInvalidStorageDir(t *testing.T) {
	// Use a non-existent directory
	cfg := &config.Config{StorageDir: "/non/existent/directory"}
	logger := logging.NewAppLogger()
	server := NewServer(cfg, logger)

	err := server.initializeComponents()
	if err == nil {
		t.Error("initializeComponents should fail with non-existent storage directory")
	}
}

func TestStop(t *testing.T) {
	server := createTestServer(t)

	err := server.Stop()
	if err != nil {
		t.Errorf("Stop should not return error: %v", err)
	}
}

func TestGetRepoFilesEmptyDirectory(t *testing.T) {
	// Create an empty temporary directory
	tempDir, err := os.MkdirTemp("", "rulem-empty-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		StorageDir: tempDir,
	}
	logger := logging.NewAppLogger()
	server := NewServer(cfg, logger)

	// Initialize the server components
	err = server.initializeComponents()
	if err != nil {
		t.Fatalf("Failed to initialize server components: %v", err)
	}

	files, err := server.getRepoFiles()
	if err != nil {
		t.Errorf("getRepoFiles should not return error for empty directory: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("Expected 0 files in empty directory, got %d", len(files))
	}
}

func TestGetRepoFilesMultipleFiles(t *testing.T) {
	// Create temporary directory with multiple markdown files
	tempDir, err := os.MkdirTemp("", "rulem-multi-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := []struct {
		name    string
		content string
	}{
		{"copilot-rules.md", "# Copilot Rules\nUse TypeScript"},
		{"cursor-rules.md", "# Cursor Rules\nUse React hooks"},
		{"claude-rules.md", "# Claude Rules\nWrite clean code"},
		{"not-markdown.txt", "This should be ignored"},
	}

	for _, file := range testFiles {
		filePath := filepath.Join(tempDir, file.name)
		if err := os.WriteFile(filePath, []byte(file.content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file.name, err)
		}
	}

	cfg := &config.Config{
		StorageDir: tempDir,
	}
	logger := logging.NewAppLogger()
	server := NewServer(cfg, logger)

	// Initialize the server components
	err = server.initializeComponents()
	if err != nil {
		t.Fatalf("Failed to initialize server components: %v", err)
	}

	files, err := server.getRepoFiles()
	if err != nil {
		t.Errorf("getRepoFiles should not return error: %v", err)
	}

	// Should find 3 markdown files (excluding .txt file)
	if len(files) != 3 {
		t.Errorf("Expected 3 markdown files, got %d", len(files))
	}

	// Check that all markdown files are found
	expectedFiles := []string{"copilot-rules.md", "cursor-rules.md", "claude-rules.md"}
	foundFiles := make(map[string]bool)
	for _, file := range files {
		foundFiles[file.Name] = true
	}

	for _, expected := range expectedFiles {
		if !foundFiles[expected] {
			t.Errorf("Expected file %s not found in scan results", expected)
		}
	}

	// Ensure .txt file is not included
	if foundFiles["not-markdown.txt"] {
		t.Error("Non-markdown file should not be included in scan results")
	}
}

// Helper functions

func createTestServer(t *testing.T) *Server {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "rulem-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	cfg := &config.Config{
		StorageDir: tempDir,
	}
	logger := logging.NewAppLogger()

	return NewServer(cfg, logger)
}

func createTestServerWithFiles(t *testing.T) (*Server, string) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "rulem-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create a test rule file
	testContent := "# Test Rule\nThis is a test rule file."
	testFile := filepath.Join(tempDir, "test-rule.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cfg := &config.Config{
		StorageDir: tempDir,
	}
	logger := logging.NewAppLogger()

	return NewServer(cfg, logger), tempDir
}

// initializeComponents is a helper method for testing that initializes
// the server components without starting the full MCP server
func (s *Server) initializeComponents() error {
	var err error
	s.fileManager, err = s.initFileManager()
	return err
}

// initFileManager is extracted for testing purposes
func (s *Server) initFileManager() (*filemanager.FileManager, error) {
	return filemanager.NewFileManager(s.config.StorageDir, s.logger)
}
