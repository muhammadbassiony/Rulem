package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"rulem/internal/config"
	"rulem/internal/filemanager"
	"rulem/internal/logging"
	"rulem/internal/repository"

	"github.com/mark3labs/mcp-go/mcp"
)

// Test data for reuse across tests
var (
	validRuleFile1 = `---
description: "First test rule"
name: "test_rule_1"
applyTo: "Go projects"
---
# Test Rule 1
This is the content of the first test rule.`

	validRuleFile2 = `---
description: "Second test rule"
name: "test_rule_2"
---
# Test Rule 2
This is the content of the second test rule.`

	invalidRuleFile = `# Invalid Rule
This file has no frontmatter and should be ignored.`

	duplicateNameRule = `---
description: "Duplicate name test"
name: "test_rule_1"
---
# Duplicate Rule
This rule has the same name as another rule.`

	complexContentRule = `---
description: "Complex content test"
name: "complex_content"
---
# Complex Content Rule

## Section 1
This rule tests content handling with:
- Multiple lines
- **Bold text**
- Code blocks

## Section 2
More content here.`
)

func createTestConfigWithPath(path string) *config.Config {
	return &config.Config{
		Repositories: []repository.RepositoryEntry{
			{
				ID:        "test-repo-123456",
				Name:      "Test Repository",
				Type:      repository.RepositoryTypeLocal,
				CreatedAt: 1234567890,
				Path:      path,
			},
		},
	}
}

// Test helpers
func createTestServer(tb testing.TB) (*Server, string) {
	tempDir, err := os.MkdirTemp("", "rulem-test-*")
	if err != nil {
		tb.Fatalf("Failed to create temp dir: %v", err)
	}

	tb.Cleanup(func() {
		if err := os.RemoveAll(tempDir); err != nil {
			tb.Logf("Warning: failed to cleanup temp dir %s: %v", tempDir, err)
		}
	})

	cfg := createTestConfigWithPath(tempDir)
	logger, _ := logging.NewTestLogger()
	server := NewServer(cfg, logger)

	return server, tempDir
}

func createTestServerWithFiles(tb testing.TB, files map[string]string) (*Server, string) {
	server, tempDir := createTestServer(tb)

	for filename, content := range files {
		filePath := filepath.Join(tempDir, filename)
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			tb.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			tb.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	return server, tempDir
}

func TestServer_Construction(t *testing.T) {
	tests := []struct {
		name        string
		storageDir  string
		description string
	}{
		{
			name:        "valid construction",
			storageDir:  "/tmp/test",
			description: "should create server with valid configuration",
		},
		{
			name:        "empty storage dir",
			storageDir:  "",
			description: "should create server even with empty storage dir",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := createTestConfigWithPath(tt.storageDir)
			logger, _ := logging.NewTestLogger()

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

			if server.mcpServer != nil {
				t.Error("MCP server should not be initialized until Start() is called")
			}
			if server.toolRegistry == nil {
				t.Error("Tool registry should be initialized")
			}
			if len(server.toolRegistry) != 0 {
				t.Error("Tool registry should be empty before Start() is called")
			}
		})
	}
}

func TestServer_ComponentInitialization(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t testing.TB) *Server
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid initialization",
			setupFunc: func(t testing.TB) *Server {
				server, _ := createTestServer(t)
				return server
			},
			wantError: false,
		},
		{
			name: "invalid storage directory",
			setupFunc: func(t testing.TB) *Server {
				cfg := createTestConfigWithPath("/non/existent/directory")
				logger, _ := logging.NewTestLogger()
				return NewServer(cfg, logger)
			},
			wantError: true,
			errorMsg:  "failed to initialize file manager",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupFunc(t)
			err := server.InitializeComponents()

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), "failed to prepare repositories") {
					t.Errorf("Expected error containing \"failed to prepare repositories\", got %q", err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if server.ruleProcessor == nil {
					t.Error("RuleProcessor should be initialized")
				}
				if server.preparedRepositories == nil {
					t.Error("PreparedRepositories should be initialized")
				}
			}
		})
	}
}

func TestServer_FileProcessing(t *testing.T) {
	tests := []struct {
		name          string
		files         map[string]string
		expectedFiles []string
		description   string
	}{
		{
			name: "mixed file types",
			files: map[string]string{
				"rule1.md":       validRuleFile1,
				"rule2.md":       validRuleFile2,
				"invalid.md":     invalidRuleFile,
				"subdir/rule.md": validRuleFile2,
			},
			expectedFiles: []string{"rule1.md", "rule2.md", "invalid.md", "rule.md"},
			description:   "should find all files including those in subdirectories",
		},
		{
			name: "only valid files",
			files: map[string]string{
				"valid1.md": validRuleFile1,
				"valid2.md": validRuleFile2,
			},
			expectedFiles: []string{"valid1.md", "valid2.md"},
			description:   "should find all valid rule files",
		},
		{
			name:          "empty directory",
			files:         map[string]string{},
			expectedFiles: []string{},
			description:   "should handle empty directories gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _ := createTestServerWithFiles(t, tt.files)

			if err := server.InitializeComponents(); err != nil {
				t.Fatalf("Failed to initialize server components: %v", err)
			}

			files, err := server.getRepoFiles()
			if err != nil {
				t.Fatalf("getRepoFiles should not return error: %v", err)
			}

			if len(files) != len(tt.expectedFiles) {
				t.Errorf("Expected %d files, got %d", len(tt.expectedFiles), len(files))
			}

			// Verify all expected files are found
			foundFiles := make(map[string]bool)
			for _, file := range files {
				foundFiles[file.Name] = true
			}

			for _, expectedFile := range tt.expectedFiles {
				if !foundFiles[expectedFile] {
					t.Errorf("Expected file %s not found", expectedFile)
				}
			}
		})
	}
}

func TestServer_ToolRegistration(t *testing.T) {
	tests := []struct {
		name          string
		files         map[string]string
		expectedTools []string
		expectedCount int
		description   string
	}{
		{
			name: "valid and invalid files",
			files: map[string]string{
				"valid1.md":    validRuleFile1,
				"valid2.md":    validRuleFile2,
				"invalid.md":   invalidRuleFile,
				"duplicate.md": duplicateNameRule,
			},
			expectedTools: []string{"test_rule_1", "test_rule_2"},
			expectedCount: 3, // 2 valid + 1 duplicate with suffix
			description:   "should register valid tools and handle duplicates",
		},
		{
			name: "only valid files",
			files: map[string]string{
				"rule1.md": validRuleFile1,
				"rule2.md": validRuleFile2,
			},
			expectedTools: []string{"test_rule_1", "test_rule_2"},
			expectedCount: 2,
			description:   "should register all valid tools",
		},
		{
			name: "no valid files",
			files: map[string]string{
				"invalid1.md": invalidRuleFile,
				"invalid2.md": "# Another invalid\nNo frontmatter here either.",
			},
			expectedTools: []string{},
			expectedCount: 0,
			description:   "should handle files with no valid frontmatter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _ := createTestServerWithFiles(t, tt.files)

			if err := server.InitializeComponents(); err != nil {
				t.Fatalf("Failed to initialize server components: %v", err)
			}

			// Process rule files
			files, err := server.getRepoFiles()
			if err != nil {
				t.Fatalf("Failed to get repository files: %v", err)
			}

			toolsMap, err := server.ruleProcessor.ProcessRuleFiles(files)
			if err != nil {
				t.Fatalf("Failed to process rule files: %v", err)
			}

			server.toolRegistry = toolsMap

			if len(server.toolRegistry) != tt.expectedCount {
				t.Errorf("Expected %d registered tools, got %d", tt.expectedCount, len(server.toolRegistry))
			}

			// Check that expected tools are registered
			for _, expectedTool := range tt.expectedTools {
				if _, exists := server.toolRegistry[expectedTool]; !exists {
					t.Errorf("Expected tool %s not found in registry", expectedTool)
				}
			}

			// Check duplicate handling if applicable
			if tt.expectedCount > len(tt.expectedTools) {
				hasDuplicateWithSuffix := false
				for toolName := range server.toolRegistry {
					if strings.HasSuffix(toolName, "_1") || strings.HasSuffix(toolName, "_2") {
						hasDuplicateWithSuffix = true
						break
					}
				}
				if !hasDuplicateWithSuffix && tt.expectedCount > len(tt.expectedTools) {
					t.Error("Expected duplicate tool with suffix not found")
				}
			}
		})
	}
}

func TestServer_ToolHandlers(t *testing.T) {
	// Setup server with test files
	testFiles := map[string]string{
		"handler-test.md": validRuleFile1,
		"another-rule.md": validRuleFile2,
		"complex.md":      complexContentRule,
	}

	server, _ := createTestServerWithFiles(t, testFiles)

	if err := server.InitializeComponents(); err != nil {
		t.Fatalf("Failed to initialize server components: %v", err)
	}

	// Process rule files
	files, err := server.getRepoFiles()
	if err != nil {
		t.Fatalf("Failed to get repository files: %v", err)
	}

	toolsMap, err := server.ruleProcessor.ProcessRuleFiles(files)
	if err != nil {
		t.Fatalf("Failed to process rule files: %v", err)
	}

	server.toolRegistry = toolsMap

	tests := []struct {
		name          string
		toolName      string
		wantError     bool
		errorSubstr   string
		wantContent   string
		setupContext  func() (context.Context, context.CancelFunc)
		validateExtra func(t *testing.T, result *mcp.CallToolResult)
	}{
		{
			name:        "valid tool handler",
			toolName:    "test_rule_1",
			wantError:   false,
			wantContent: "# Test Rule 1",
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 5*time.Second)
			},
		},
		{
			name:        "another valid tool",
			toolName:    "test_rule_2",
			wantError:   false,
			wantContent: "# Test Rule 2",
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 5*time.Second)
			},
		},
		{
			name:        "complex content tool",
			toolName:    "complex_content",
			wantError:   false,
			wantContent: "# Complex Content Rule",
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 5*time.Second)
			},
		},
		{
			name:        "non-existent tool",
			toolName:    "nonexistent",
			wantError:   true,
			errorSubstr: "not found in registry",
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 5*time.Second)
			},
		},
		{
			name:        "empty tool name",
			toolName:    "",
			wantError:   true,
			errorSubstr: "not found in registry",
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 5*time.Second)
			},
		},
		{
			name:      "cancelled context",
			toolName:  "test_rule_1",
			wantError: true,
			setupContext: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately
				return ctx, cancel
			},
			validateExtra: func(t *testing.T, result *mcp.CallToolResult) {
				if result != nil {
					t.Error("Expected nil result when context is cancelled")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := tt.setupContext()
			defer cancel()

			handler, err := server.getRulefileToolHandler(tt.toolName)

			if tt.wantError && err != nil {
				// Expected error during handler creation
				if tt.errorSubstr != "" && !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("Expected error containing %q, got %q", tt.errorSubstr, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error creating handler, got %v", err)
				return
			}

			if handler == nil {
				t.Error("Expected valid handler, got nil")
				return
			}

			// Test the handler function
			req := mcp.CallToolRequest{}
			result, err := handler(ctx, req)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if tt.validateExtra != nil {
					tt.validateExtra(t, result)
				}
			} else {
				if err != nil {
					t.Errorf("Handler should not return error: %v", err)
					return
				}

				if result == nil {
					t.Error("Handler should return a result")
					return
				}

				if len(result.Content) == 0 {
					t.Error("Result should contain content")
					return
				}

				// Check if result contains expected content
				if textContent, ok := mcp.AsTextContent(result.Content[0]); ok {
					if tt.wantContent != "" && !strings.Contains(textContent.Text, tt.wantContent) {
						t.Errorf("Expected content to contain %q, got %q", tt.wantContent, textContent.Text)
					}
				} else {
					t.Error("Result content should be text content")
				}
			}
		})
	}
}

func TestServer_ConcurrentAccess(t *testing.T) {
	testFiles := map[string]string{
		"concurrent-test.md": validRuleFile1,
	}

	server, _ := createTestServerWithFiles(t, testFiles)

	if err := server.InitializeComponents(); err != nil {
		t.Fatalf("Failed to initialize server components: %v", err)
	}

	// Process rule files
	files, err := server.getRepoFiles()
	if err != nil {
		t.Fatalf("Failed to get repository files: %v", err)
	}

	toolsMap, err := server.ruleProcessor.ProcessRuleFiles(files)
	if err != nil {
		t.Fatalf("Failed to process rule files: %v", err)
	}

	server.toolRegistry = toolsMap

	handler, err := server.getRulefileToolHandler("test_rule_1")
	if err != nil {
		t.Fatalf("Failed to get handler: %v", err)
	}

	// Test concurrent access
	const numGoroutines = 10
	const numCalls = 5

	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines*numCalls)

	for i := range numGoroutines {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for range numCalls {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				req := mcp.CallToolRequest{}

				result, err := handler(ctx, req)
				cancel()

				if err != nil {
					errChan <- err
					return
				}

				if result == nil || len(result.Content) == 0 {
					errChan <- err
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			t.Errorf("Concurrent handler call failed: %v", err)
		}
	}
}

func TestServer_ErrorConditions(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t testing.TB) *Server
		operation   string
		expectError bool
		errorMsg    string
		description string
	}{
		{
			name: "missing storage directory",
			setupFunc: func(t testing.TB) *Server {
				cfg := createTestConfigWithPath("/this/path/does/not/exist")
				logger, _ := logging.NewTestLogger()
				return NewServer(cfg, logger)
			},
			operation:   "InitializeComponents",
			expectError: true,
			errorMsg:    "failed to initialize file manager",
			description: "should error when storage directory doesn't exist",
		},
		{
			name: "handler for non-existent tool",
			setupFunc: func(t testing.TB) *Server {
				server, _ := createTestServer(t)
				return server
			},
			operation:   "getRulefileToolHandler",
			expectError: true,
			errorMsg:    "not found in registry",
			description: "should error when requesting handler for non-existent tool",
		},
		{
			name: "empty tool registry",
			setupFunc: func(t testing.TB) *Server {
				server, _ := createTestServer(t)
				return server
			},
			operation:   "checkEmptyRegistry",
			expectError: false,
			description: "should handle empty tool registry gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupFunc(t)

			switch tt.operation {
			case "InitializeComponents":
				err := server.InitializeComponents()
				if tt.expectError {
					if err == nil {
						t.Errorf("Expected error containing \"failed to prepare repositories\", got no error")
					} else if !strings.Contains(err.Error(), "failed to prepare repositories") {
						t.Errorf("Expected error containing \"failed to prepare repositories\", got %q", err.Error())
					}
				} else if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}

			case "getRulefileToolHandler":
				if err := server.InitializeComponents(); err != nil {
					t.Fatalf("Failed to initialize: %v", err)
				}
				_, err := server.getRulefileToolHandler("nonexistent_tool")
				if tt.expectError {
					if err == nil {
						t.Error("Expected error but got none")
					} else if !strings.Contains(err.Error(), tt.errorMsg) {
						t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
					}
				} else if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}

			case "checkEmptyRegistry":
				if err := server.InitializeComponents(); err != nil {
					t.Fatalf("Failed to initialize: %v", err)
				}
				files, err := server.getRepoFiles()
				if err != nil {
					t.Fatalf("Failed to get files: %v", err)
				}
				toolsMap, err := server.ruleProcessor.ProcessRuleFiles(files)
				if err != nil {
					t.Fatalf("Failed to process files: %v", err)
				}
				server.toolRegistry = toolsMap

				if len(server.toolRegistry) != 0 {
					t.Errorf("Expected empty registry, got %d tools", len(server.toolRegistry))
				}
			}
		})
	}
}

func BenchmarkServer_Performance(b *testing.B) {
	testFiles := map[string]string{
		"perf-test.md": validRuleFile1,
	}

	server, _ := createTestServerWithFiles(b, testFiles)

	if err := server.InitializeComponents(); err != nil {
		b.Fatalf("Failed to initialize server components: %v", err)
	}

	// Process rule files
	files, err := server.getRepoFiles()
	if err != nil {
		b.Fatalf("Failed to get repository files: %v", err)
	}

	toolsMap, err := server.ruleProcessor.ProcessRuleFiles(files)
	if err != nil {
		b.Fatalf("Failed to process rule files: %v", err)
	}

	server.toolRegistry = toolsMap

	handler, err := server.getRulefileToolHandler("test_rule_1")
	if err != nil {
		b.Fatalf("Failed to get handler: %v", err)
	}

	// Reset timer to exclude setup time
	b.ResetTimer()

	ctx := context.Background()
	req := mcp.CallToolRequest{}

	// Benchmark loop - b.N is determined by the testing framework
	for i := 0; i < b.N; i++ {
		result, err := handler(ctx, req)
		if err != nil {
			b.Errorf("Handler call %d failed: %v", i, err)
		}
		if result == nil {
			b.Errorf("Handler call %d returned nil result", i)
		}
	}
}

func TestServer_ContentHandling(t *testing.T) {
	tests := []struct {
		name            string
		content         string
		expectedElement []string
		description     string
	}{
		{
			name:    "simple content",
			content: validRuleFile1,
			expectedElement: []string{
				"# Test Rule 1",
				"This is the content",
			},
			description: "should handle simple rule content",
		},
		{
			name:    "complex content",
			content: complexContentRule,
			expectedElement: []string{
				"# Complex Content Rule",
				"## Section 1",
				"Multiple lines",
				"**Bold text**",
				"## Section 2",
			},
			description: "should handle complex content with multiple sections",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFiles := map[string]string{
				"content-test.md": tt.content,
			}

			server, _ := createTestServerWithFiles(t, testFiles)

			if err := server.InitializeComponents(); err != nil {
				t.Fatalf("Failed to initialize server components: %v", err)
			}

			// Process rule files
			files, err := server.getRepoFiles()
			if err != nil {
				t.Fatalf("Failed to get repository files: %v", err)
			}

			toolsMap, err := server.ruleProcessor.ProcessRuleFiles(files)
			if err != nil {
				t.Fatalf("Failed to process rule files: %v", err)
			}

			server.toolRegistry = toolsMap

			// Get the first tool (should be our test content)
			var toolName string
			for name := range server.toolRegistry {
				toolName = name
				break
			}

			if toolName == "" {
				t.Fatal("No tools registered")
			}

			handler, err := server.getRulefileToolHandler(toolName)
			if err != nil {
				t.Fatalf("Failed to get handler: %v", err)
			}

			ctx := context.Background()
			req := mcp.CallToolRequest{}
			result, err := handler(ctx, req)

			if err != nil {
				t.Errorf("Handler should not return error: %v", err)
				return
			}

			if result == nil || len(result.Content) == 0 {
				t.Fatal("Handler should return content")
			}

			textContent, ok := mcp.AsTextContent(result.Content[0])
			if !ok {
				t.Fatal("Result content should be text")
			}

			// Verify the content includes expected elements
			for _, element := range tt.expectedElement {
				if !strings.Contains(textContent.Text, element) {
					t.Errorf("Expected content to contain %q, but it was missing", element)
				}
			}
		})
	}
}

// TestServer_MultiRepositoryInitialization tests server initialization with multiple repositories
func TestServer_MultiRepositoryInitialization(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(t *testing.T) *config.Config
		expectError   bool
		errorContains string
		validateFunc  func(t *testing.T, s *Server)
	}{
		{
			name: "single repository",
			setupFunc: func(t *testing.T) *config.Config {
				tempDir := t.TempDir()
				return createTestConfigWithPath(tempDir)
			},
			expectError: false,
			validateFunc: func(t *testing.T, s *Server) {
				if len(s.preparedRepositories) != 1 {
					t.Errorf("Expected 1 prepared repository, got %d", len(s.preparedRepositories))
				}
			},
		},
		{
			name: "multiple local repositories",
			setupFunc: func(t *testing.T) *config.Config {
				tempDir1 := t.TempDir()
				tempDir2 := t.TempDir()
				tempDir3 := t.TempDir()

				return &config.Config{
					Repositories: []repository.RepositoryEntry{
						{
							ID:        "repo1-123456",
							Name:      "Repository 1",
							Type:      repository.RepositoryTypeLocal,
							Path:      tempDir1,
							CreatedAt: 1234567890,
						},
						{
							ID:        "repo2-123457",
							Name:      "Repository 2",
							Type:      repository.RepositoryTypeLocal,
							Path:      tempDir2,
							CreatedAt: 1234567891,
						},
						{
							ID:        "repo3-123458",
							Name:      "Repository 3",
							Type:      repository.RepositoryTypeLocal,
							Path:      tempDir3,
							CreatedAt: 1234567892,
						},
					},
				}
			},
			expectError: false,
			validateFunc: func(t *testing.T, s *Server) {
				if len(s.preparedRepositories) != 3 {
					t.Errorf("Expected 3 prepared repositories, got %d", len(s.preparedRepositories))
				}
				expectedIDs := []string{"repo1-123456", "repo2-123457", "repo3-123458"}
				foundIDs := make(map[string]bool)
				for _, prep := range s.preparedRepositories {
					foundIDs[prep.ID()] = true
				}
				for _, id := range expectedIDs {
					if !foundIDs[id] {
						t.Errorf("Expected prepared repository for ID %s", id)
					}
				}
			},
		},
		{
			name: "empty repository list",
			setupFunc: func(t *testing.T) *config.Config {
				return &config.Config{
					Repositories: []repository.RepositoryEntry{},
				}
			},
			expectError: false,
			validateFunc: func(t *testing.T, s *Server) {
				if len(s.preparedRepositories) != 0 {
					t.Errorf("Expected 0 prepared repositories, got %d", len(s.preparedRepositories))
				}
			},
		},
		{
			name: "one invalid repository among valid ones",
			setupFunc: func(t *testing.T) *config.Config {
				tempDir1 := t.TempDir()
				tempDir2 := t.TempDir()

				return &config.Config{
					Repositories: []repository.RepositoryEntry{
						{
							ID:        "repo1-123456",
							Name:      "Valid Repo",
							Type:      repository.RepositoryTypeLocal,
							Path:      tempDir1,
							CreatedAt: 1234567890,
						},
						{
							ID:        "repo2-123457",
							Name:      "Invalid Repo",
							Type:      repository.RepositoryTypeLocal,
							Path:      "/nonexistent/invalid/path",
							CreatedAt: 1234567891,
						},
						{
							ID:        "repo3-123458",
							Name:      "Another Valid Repo",
							Type:      repository.RepositoryTypeLocal,
							Path:      tempDir2,
							CreatedAt: 1234567892,
						},
					},
				}
			},
			expectError:   true,
			errorContains: "failed to prepare",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.setupFunc(t)
			logger, _ := logging.NewTestLogger()
			server := NewServer(cfg, logger)

			err := server.InitializeComponents()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if tt.validateFunc != nil {
					tt.validateFunc(t, server)
				}
			}
		})
	}
}

// TestServer_MultiRepositoryFileDiscovery tests file discovery from multiple repositories
func TestServer_MultiRepositoryFileDiscovery(t *testing.T) {
	// Create test directories with rule files
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()
	tempDir3 := t.TempDir()

	// Create rule files in each repository
	ruleFile1 := `---
description: "Rule from repository 1"
name: "repo1_rule"
---
# Repository 1 Rule
Content from repo 1`

	ruleFile2 := `---
description: "Rule from repository 2"
name: "repo2_rule"
---
# Repository 2 Rule
Content from repo 2`

	ruleFile3 := `---
description: "Rule from repository 3"
name: "repo3_rule"
---
# Repository 3 Rule
Content from repo 3`

	// Write files to repositories
	if err := os.WriteFile(filepath.Join(tempDir1, "rule1.md"), []byte(ruleFile1), 0644); err != nil {
		t.Fatalf("Failed to write rule file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir2, "rule2.md"), []byte(ruleFile2), 0644); err != nil {
		t.Fatalf("Failed to write rule file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir3, "rule3.md"), []byte(ruleFile3), 0644); err != nil {
		t.Fatalf("Failed to write rule file: %v", err)
	}

	// Create config with multiple repositories
	cfg := &config.Config{
		Repositories: []repository.RepositoryEntry{
			{
				ID:        "repo1-123456",
				Name:      "Repository 1",
				Type:      repository.RepositoryTypeLocal,
				Path:      tempDir1,
				CreatedAt: 1234567890,
			},
			{
				ID:        "repo2-123457",
				Name:      "Repository 2",
				Type:      repository.RepositoryTypeLocal,
				Path:      tempDir2,
				CreatedAt: 1234567891,
			},
			{
				ID:        "repo3-123458",
				Name:      "Repository 3",
				Type:      repository.RepositoryTypeLocal,
				Path:      tempDir3,
				CreatedAt: 1234567892,
			},
		},
	}

	logger, _ := logging.NewTestLogger()

	// Prepare repositories directly
	prepared, err := repository.PrepareAllRepositories(cfg.Repositories, logger)
	if err != nil {
		t.Fatalf("Failed to prepare repositories: %v", err)
	}

	// Scan files from all repositories
	files, err := filemanager.ScanAllRepositories(prepared, logger)
	if err != nil {
		t.Fatalf("Failed to scan repositories: %v", err)
	}

	// Verify all files are discovered
	if len(files) != 3 {
		t.Errorf("Expected 3 files from 3 repositories, got %d", len(files))
	}

	// Verify files have correct repository metadata
	repoIDCount := make(map[string]int)
	for _, file := range files {
		repoIDCount[file.RepositoryID]++
	}

	expectedRepos := []string{"repo1-123456", "repo2-123457", "repo3-123458"}
	for _, repoID := range expectedRepos {
		if count := repoIDCount[repoID]; count != 1 {
			t.Errorf("Expected 1 file from repository %s, got %d", repoID, count)
		}
	}
}

// TestServer_MultiRepositoryFilesWithSameNames tests handling of files with same names from different repos
func TestServer_MultiRepositoryFilesWithSameNames(t *testing.T) {
	// Create test directories
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	// Create rule files with same filename but different content
	ruleFile1 := `---
description: "Standards from team A"
name: "coding_standards"
---
# Team A Standards`

	ruleFile2 := `---
description: "Standards from team B"
name: "coding_standards"
---
# Team B Standards`

	// Write files to repositories
	if err := os.WriteFile(filepath.Join(tempDir1, "standards.md"), []byte(ruleFile1), 0644); err != nil {
		t.Fatalf("Failed to write rule file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir2, "standards.md"), []byte(ruleFile2), 0644); err != nil {
		t.Fatalf("Failed to write rule file: %v", err)
	}

	// Create config with multiple repositories
	cfg := &config.Config{
		Repositories: []repository.RepositoryEntry{
			{
				ID:        "teamA-123456",
				Name:      "Team A Repo",
				Type:      repository.RepositoryTypeLocal,
				Path:      tempDir1,
				CreatedAt: 1234567890,
			},
			{
				ID:        "teamB-123457",
				Name:      "Team B Repo",
				Type:      repository.RepositoryTypeLocal,
				Path:      tempDir2,
				CreatedAt: 1234567891,
			},
		},
	}

	logger, _ := logging.NewTestLogger()

	// Prepare repositories and scan files
	prepared, err := repository.PrepareAllRepositories(cfg.Repositories, logger)
	if err != nil {
		t.Fatalf("Failed to prepare repositories: %v", err)
	}

	files, err := filemanager.ScanAllRepositories(prepared, logger)
	if err != nil {
		t.Fatalf("Failed to scan repositories: %v", err)
	}

	// Both files should be discovered
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}

	// Verify files have correct repository IDs
	repoIDs := make(map[string]bool)
	for _, file := range files {
		repoIDs[file.RepositoryID] = true
		if file.Name != "standards.md" {
			t.Errorf("Expected filename 'standards.md', got %q", file.Name)
		}
	}

	if len(repoIDs) != 2 {
		t.Errorf("Expected files from 2 different repositories, got %d", len(repoIDs))
	}
}

// TestServer_MultiRepositoryWithMixedContent tests repositories with valid and invalid files
func TestServer_MultiRepositoryWithMixedContent(t *testing.T) {
	// Create test directories
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	// Repository 1: All valid files
	validFile1 := `---
description: "Valid rule 1"
---
Content 1`
	validFile2 := `---
description: "Valid rule 2"
---
Content 2`

	// Repository 2: Mix of valid and invalid files
	validFile3 := `---
description: "Valid rule 3"
---
Content 3`
	invalidFile := `No frontmatter here
Just plain content`

	// Write files
	if err := os.WriteFile(filepath.Join(tempDir1, "valid1.md"), []byte(validFile1), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir1, "valid2.md"), []byte(validFile2), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir2, "valid3.md"), []byte(validFile3), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir2, "invalid.md"), []byte(invalidFile), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Create config
	cfg := &config.Config{
		Repositories: []repository.RepositoryEntry{
			{
				ID:        "repo1-123456",
				Name:      "All Valid Repo",
				Type:      repository.RepositoryTypeLocal,
				Path:      tempDir1,
				CreatedAt: 1234567890,
			},
			{
				ID:        "repo2-123457",
				Name:      "Mixed Content Repo",
				Type:      repository.RepositoryTypeLocal,
				Path:      tempDir2,
				CreatedAt: 1234567891,
			},
		},
	}

	logger, _ := logging.NewTestLogger()

	// Prepare and scan repositories
	prepared, err := repository.PrepareAllRepositories(cfg.Repositories, logger)
	if err != nil {
		t.Fatalf("Failed to prepare repositories: %v", err)
	}

	files, err := filemanager.ScanAllRepositories(prepared, logger)
	if err != nil {
		t.Fatalf("Failed to scan repositories: %v", err)
	}

	// Should discover all 4 files (scanner doesn't validate frontmatter)
	if len(files) != 4 {
		t.Errorf("Expected 4 files discovered, got %d", len(files))
	}

	// Build paths map for processor
	pathsMap := make(map[string]string)
	for _, prep := range prepared {
		pathsMap[prep.ID()] = prep.LocalPath
	}

	// Test that processor would skip invalid file during parsing
	processor := NewRuleFileProcessor(logger, pathsMap, 5*1024*1024)
	ruleFiles, err := processor.ParseRuleFiles(files)
	if err != nil {
		t.Fatalf("ParseRuleFiles failed: %v", err)
	}

	// Should only parse 3 valid files (invalid file should be skipped)
	if len(ruleFiles) != 3 {
		t.Errorf("Expected 3 valid rule files, got %d", len(ruleFiles))
	}
}

// TestServer_MultiRepositoryMetadata tests that repository metadata is preserved in file items
func TestServer_MultiRepositoryMetadata(t *testing.T) {
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	// Create rule files
	ruleFile := `---
description: "Test rule"
---
Content`

	if err := os.WriteFile(filepath.Join(tempDir1, "rule1.md"), []byte(ruleFile), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir2, "rule2.md"), []byte(ruleFile), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	cfg := &config.Config{
		Repositories: []repository.RepositoryEntry{
			{
				ID:        "local-repo-123456",
				Name:      "Local Repository",
				Type:      repository.RepositoryTypeLocal,
				Path:      tempDir1,
				CreatedAt: 1234567890,
			},
			{
				ID:        "second-repo-123457",
				Name:      "Second Local Repository",
				Type:      repository.RepositoryTypeLocal,
				Path:      tempDir2,
				CreatedAt: 1234567891,
			},
		},
	}

	logger, _ := logging.NewTestLogger()
	server := NewServer(cfg, logger)

	if err := server.InitializeComponents(); err != nil {
		t.Fatalf("Failed to initialize components: %v", err)
	}

	// Get files from all repositories
	files, err := filemanager.ScanAllRepositories(server.preparedRepositories, logger)
	if err != nil {
		t.Fatalf("Failed to scan repositories: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(files))
	}

	// Verify metadata is set on file items
	for _, file := range files {
		if file.RepositoryID == "" {
			t.Error("File missing RepositoryID")
		}
		if file.RepositoryName == "" {
			t.Error("File missing RepositoryName")
		}
		if file.RepositoryType == "" {
			t.Error("File missing RepositoryType")
		}

		// Verify correct metadata based on repository
		if file.RepositoryID == "local-repo-123456" {
			if file.RepositoryName != "Local Repository" {
				t.Errorf("Expected repository name 'Local Repository', got %q", file.RepositoryName)
			}
			if file.RepositoryType != "local" {
				t.Errorf("Expected repository type 'local', got %q", file.RepositoryType)
			}
		}
		if file.RepositoryID == "second-repo-123457" {
			if file.RepositoryName != "Second Local Repository" {
				t.Errorf("Expected repository name 'Second Local Repository', got %q", file.RepositoryName)
			}
			if file.RepositoryType != "local" {
				t.Errorf("Expected repository type 'local', got %q", file.RepositoryType)
			}
		}
	}
}

// TestServer_SyncResultsLogging tests that sync results are properly logged
func TestServer_SyncResultsLogging(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		Repositories: []repository.RepositoryEntry{
			{
				ID:        "test-repo-123456",
				Name:      "Test Repository",
				Type:      repository.RepositoryTypeLocal,
				Path:      tempDir,
				CreatedAt: 1234567890,
			},
		},
	}

	logger, logBuffer := logging.NewTestLogger()
	server := NewServer(cfg, logger)

	if err := server.InitializeComponents(); err != nil {
		t.Fatalf("Failed to initialize components: %v", err)
	}

	// Check that prepared repositories are stored
	if server.preparedRepositories == nil {
		t.Error("Expected preparedRepositories to be initialized")
	}

	// Verify we have prepared repositories
	if len(server.preparedRepositories) != 1 {
		t.Errorf("Expected 1 prepared repository, got %d", len(server.preparedRepositories))
	}

	// Verify logging occurred
	logOutput := logBuffer.String()
	if !strings.Contains(logOutput, "Multi-repository preparation") {
		t.Error("Expected log message about multi-repository preparation")
	}
}

// Helper function for creating string pointers in tests
func StringPtr(s string) *string {
	return &s
}
