package repostatusmenu

import (
	"path/filepath"
	"strings"
	"testing"

	"rulem/internal/repository"
)

func strPtr(s string) *string { return &s }

func TestBuildStatusRows(t *testing.T) {
	localDir := t.TempDir()
	missingClone := filepath.Join(t.TempDir(), "gone")
	remote := "https://github.com/example/rules"

	repos := []repository.RepositoryEntry{
		{ID: "l1", Name: "Local Rules", Type: repository.RepositoryTypeLocal, Path: localDir},
		{ID: "g1", Name: "Team Rules", Type: repository.RepositoryTypeGitHub, Path: missingClone, RemoteURL: &remote, Branch: strPtr("main")},
	}

	rows := buildStatusRows(repos, map[string]string{"g1": "Synced successfully"})
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	if rows[0].Kind != "local" || !strings.Contains(rows[0].Status, "not synced") {
		t.Errorf("local repo row wrong: %+v", rows[0])
	}

	if rows[1].Kind != "github • main" {
		t.Errorf("github row kind wrong: %+v", rows[1])
	}
	if !strings.Contains(rows[1].Status, "clone missing") {
		t.Errorf("missing clone must be reported: %+v", rows[1])
	}
	if !strings.Contains(rows[1].Status, "Synced successfully") {
		t.Errorf("last refresh outcome must be shown: %+v", rows[1])
	}
}

func TestBuildStatusRows_DefaultBranch(t *testing.T) {
	remote := "https://github.com/example/rules"
	repos := []repository.RepositoryEntry{
		{ID: "g1", Name: "R", Type: repository.RepositoryTypeGitHub, Path: filepath.Join(t.TempDir(), "x"), RemoteURL: &remote},
	}
	rows := buildStatusRows(repos, nil)
	if rows[0].Kind != "github • default branch" {
		t.Errorf("expected default branch marker, got %q", rows[0].Kind)
	}
}
