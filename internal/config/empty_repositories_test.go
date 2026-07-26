package config

import (
	"os"
	"path/filepath"
	"testing"

	"rulem/internal/repository"
)

// A configuration with no repositories is a valid, reachable state: it is what
// remains after the user deletes every repository from the settings menu.

func TestHasRepositories(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want bool
	}{
		{"nil config", nil, false},
		{"nil slice", &Config{}, false},
		{"empty slice", &Config{Repositories: []repository.RepositoryEntry{}}, false},
		{"one repository", &Config{Repositories: []repository.RepositoryEntry{{ID: "a"}}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.HasRepositories(); got != tt.want {
				t.Errorf("HasRepositories() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSaveLoad_EmptyRepositoriesRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")

	// Start from a populated config, as a real user would.
	cfg := &Config{
		Version: "1.0",
		Repositories: []repository.RepositoryEntry{
			{ID: "only-repo", Name: "Only", Type: repository.RepositoryTypeLocal, Path: t.TempDir()},
		},
	}
	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("failed to save populated config: %v", err)
	}

	// Delete the last repository and persist the empty result.
	cfg.Repositories = cfg.Repositories[:0]
	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("saving an empty repository list must succeed: %v", err)
	}

	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("failed to load the emptied config: %v", err)
	}
	if len(loaded.Repositories) != 0 {
		t.Fatalf("expected 0 repositories after reload, got %d", len(loaded.Repositories))
	}
	if loaded.HasRepositories() {
		t.Error("reloaded config should report no repositories")
	}

	// The emptied config must not read as a first run: the file still exists,
	// so the user keeps their settings menu instead of being sent to setup.
	if _, err := os.Stat(path); err != nil {
		t.Errorf("config file should still exist after emptying it: %v", err)
	}
}

func TestFindRepository_NilConfigDoesNotPanic(t *testing.T) {
	var cfg *Config

	if _, err := cfg.FindRepositoryByID("anything"); err == nil {
		t.Error("expected an error looking up an ID on a nil config")
	}
	if _, err := cfg.FindRepositoryByName("anything"); err == nil {
		t.Error("expected an error looking up a name on a nil config")
	}
}

func TestFindRepository_EmptyConfigReturnsError(t *testing.T) {
	cfg := &Config{Repositories: []repository.RepositoryEntry{}}

	if _, err := cfg.FindRepositoryByID("gone"); err == nil {
		t.Error("expected an error looking up an ID with no repositories")
	}
	if _, err := cfg.FindRepositoryByName("gone"); err == nil {
		t.Error("expected an error looking up a name with no repositories")
	}
}
