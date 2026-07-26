package mcp

import (
	"testing"

	"rulem/internal/config"
	"rulem/internal/logging"
	"rulem/internal/repository"
)

// The MCP server can be started against a configuration the user has emptied.
// It should come up with no tools rather than failing to start.

func TestServer_NoRepositoriesConfigured(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	cfg := &config.Config{Repositories: []repository.RepositoryEntry{}}

	server := NewServer(cfg, logger)
	if server == nil {
		t.Fatal("expected a server for an empty configuration")
	}

	prepared, err := repository.PrepareAllRepositories(t.Context(), cfg.Repositories, logger)
	if err != nil {
		t.Fatalf("preparing zero repositories should succeed, got: %v", err)
	}
	if len(prepared) != 0 {
		t.Fatalf("expected 0 prepared repositories, got %d", len(prepared))
	}

	// Mirror what Start() wires up before registering tools.
	server.preparedRepositories = prepared
	repositoryPaths := make(map[string]string, len(prepared))
	for _, prep := range prepared {
		repositoryPaths[prep.ID()] = prep.LocalPath
	}
	server.ruleProcessor = NewRuleFileProcessor(logger, repositoryPaths, 5*1024*1024)

	files, err := server.getRepoFiles()
	if err != nil {
		t.Fatalf("scanning zero repositories should succeed, got: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected 0 files, got %d", len(files))
	}

	if err := server.RegisterRuleFileTools(); err != nil {
		t.Fatalf("registering tools with no repositories should succeed, got: %v", err)
	}
	if len(server.toolRegistry) != 0 {
		t.Fatalf("expected no registered tools, got %d", len(server.toolRegistry))
	}
}
