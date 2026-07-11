package repository

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"rulem/internal/logging"

	git "github.com/go-git/go-git/v6"
	gitconfig "github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing/object"
)

// commitFile writes a file in the given worktree repo, commits it, and returns
// the commit hash.
func commitFile(t *testing.T, repoPath, name, content string) {
	t.Helper()
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, name), []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := wt.Add(name); err != nil {
		t.Fatalf("add: %v", err)
	}
	_, err = wt.Commit("add "+name, &git.CommitOptions{
		Author: &object.Signature{Name: "test", Email: "test@example.com", When: time.Now()},
	})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
}

// pushToOrigin pushes the current branch of the repo at repoPath to origin.
func pushToOrigin(t *testing.T, repoPath string) {
	t.Helper()
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	if err := repo.Push(&git.PushOptions{}); err != nil && err != git.NoErrAlreadyUpToDate {
		t.Fatalf("push: %v", err)
	}
}

// setupOriginAndClone creates a bare origin with one commit, a writer clone
// used to advance the origin, and a reader clone that plays the role of the
// rulem-managed repository. Returns (originPath, writerPath, readerPath).
func setupOriginAndClone(t *testing.T) (string, string, string) {
	t.Helper()
	tempDir := t.TempDir()
	origin := filepath.Join(tempDir, "origin.git")
	writer := filepath.Join(tempDir, "writer")
	reader := filepath.Join(tempDir, "reader")

	if _, err := git.PlainInit(origin, true); err != nil {
		t.Fatalf("init bare: %v", err)
	}
	// An empty bare repo cannot be cloned, so init the writer and wire the
	// remote by hand.
	writerRepo, err := git.PlainInit(writer, false)
	if err != nil {
		t.Fatalf("init writer: %v", err)
	}
	if _, err := writerRepo.CreateRemote(&gitconfig.RemoteConfig{
		Name: "origin",
		URLs: []string{origin},
	}); err != nil {
		t.Fatalf("create remote: %v", err)
	}
	commitFile(t, writer, "README.md", "# hello\n")
	pushToOrigin(t, writer)

	if _, err := git.PlainClone(reader, &git.CloneOptions{URL: origin}); err != nil {
		t.Fatalf("clone reader: %v", err)
	}
	return origin, writer, reader
}

// TestFetchUpdates_UpdatesWorkingTree is the regression test for the sync bug:
// FetchUpdates used to fetch into refs/remotes/origin/* but never move the
// local branch / working tree, so "synced" repositories kept serving stale
// files.
func TestFetchUpdates_UpdatesWorkingTree(t *testing.T) {
	_, writer, reader := setupOriginAndClone(t)
	logger, _ := logging.NewTestLogger()

	// Advance the origin after the reader cloned it.
	commitFile(t, writer, "new-rule.md", "# new rule\n")
	pushToOrigin(t, writer)

	gs := GitSource{Path: reader}
	if err := gs.FetchUpdates(logger); err != nil {
		t.Fatalf("FetchUpdates: %v", err)
	}

	if _, err := os.Stat(filepath.Join(reader, "new-rule.md")); err != nil {
		t.Fatalf("working tree was not updated after fetch: %v", err)
	}
}

// TestFetchUpdates_DirtyTreeIsPreserved documents the skip-on-dirty behavior:
// local modifications are never clobbered, and the fetch reports success.
func TestFetchUpdates_DirtyTreeIsPreserved(t *testing.T) {
	_, writer, reader := setupOriginAndClone(t)
	logger, _ := logging.NewTestLogger()

	// Dirty the reader, then advance the origin.
	if err := os.WriteFile(filepath.Join(reader, "README.md"), []byte("local edit\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	commitFile(t, writer, "upstream.md", "# upstream\n")
	pushToOrigin(t, writer)

	gs := GitSource{Path: reader}
	if err := gs.FetchUpdates(logger); err != nil {
		t.Fatalf("FetchUpdates on dirty tree should not error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(reader, "README.md"))
	if err != nil || string(got) != "local edit\n" {
		t.Fatalf("local changes must be preserved on dirty tree, got %q err %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(reader, "upstream.md")); err == nil {
		t.Fatalf("dirty tree must not be synced")
	}
}

// TestFetchUpdates_AfterShallowClone exercises the exact combination rulem
// uses in production: a depth-1 clone (performClone) followed by the remote
// advancing and a FetchUpdates that must land the new commit in the worktree.
func TestFetchUpdates_AfterShallowClone(t *testing.T) {
	_, writer, _ := setupOriginAndClone(t)
	logger, _ := logging.NewTestLogger()

	// Give the origin some history so depth=1 is a real truncation.
	commitFile(t, writer, "second.md", "# two\n")
	pushToOrigin(t, writer)

	origin := filepath.Join(filepath.Dir(writer), "origin.git")
	shallow := filepath.Join(filepath.Dir(writer), "shallow")

	gs := GitSource{Path: shallow}
	if err := gs.performClone(shallow, origin, nil, logger); err != nil {
		t.Fatalf("shallow clone: %v", err)
	}
	if _, err := os.Stat(filepath.Join(shallow, "second.md")); err != nil {
		t.Fatalf("clone missing content: %v", err)
	}

	// Remote advances after the shallow clone.
	commitFile(t, writer, "third.md", "# three\n")
	pushToOrigin(t, writer)

	if err := gs.FetchUpdates(logger); err != nil {
		t.Fatalf("FetchUpdates after shallow clone: %v", err)
	}
	if _, err := os.Stat(filepath.Join(shallow, "third.md")); err != nil {
		t.Fatalf("working tree missing new commit after shallow fetch: %v", err)
	}
}
