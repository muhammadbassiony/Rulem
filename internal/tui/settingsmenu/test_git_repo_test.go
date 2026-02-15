// Package settingsmenu provides the settings modification flow for the rulem TUI application.
package settingsmenu

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	git "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
)

// createBareRemoteRepo initializes a bare repository that can be used as a local "origin" remote.
// Returns the path to the bare repository.
func createBareRemoteRepo(t *testing.T) string {
	t.Helper()

	remotePath := t.TempDir()
	if _, err := git.PlainInit(remotePath, true); err != nil {
		t.Fatalf("failed to init bare repo: %v", err)
	}

	return remotePath
}

// createLocalRepoWithOrigin creates a local repository with an initial commit on the "main" branch
// and adds the provided bare repo as the "origin" remote.
// Returns the local repository path.
func createLocalRepoWithOrigin(t *testing.T, originPath string) string {
	t.Helper()

	repoPath := t.TempDir()

	repo, err := git.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	if err := worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("main"),
		Create: true,
	}); err != nil {
		t.Fatalf("failed to checkout main branch: %v", err)
	}

	filePath := filepath.Join(repoPath, "README.md")
	if err := os.WriteFile(filePath, []byte("initial\n"), 0o644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}

	if _, err := worktree.Add("README.md"); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	if _, err := worktree.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	}); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	if _, err := repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{originPath},
	}); err != nil {
		t.Fatalf("failed to add origin remote: %v", err)
	}

	return repoPath
}

// pushToOrigin pushes the given branch to the origin remote.
func pushToOrigin(t *testing.T, repoPath string, branch string) {
	t.Helper()

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("failed to open repo: %v", err)
	}

	refSpec := config.RefSpec(
		plumbing.NewBranchReferenceName(branch).String() +
			":" +
			plumbing.NewBranchReferenceName(branch).String(),
	)

	if err := repo.Push(&git.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{refSpec},
	}); err != nil && err != git.NoErrAlreadyUpToDate {
		t.Fatalf("failed to push to origin: %v", err)
	}
}

// createLocalCloneFromOrigin clones from the given origin path into a new temp dir.
// Returns the clone path.
func createLocalCloneFromOrigin(t *testing.T, originPath string) string {
	t.Helper()

	clonePath := t.TempDir()
	if _, err := git.PlainClone(clonePath, &git.CloneOptions{
		URL: originPath,
	}); err != nil {
		t.Fatalf("failed to clone from origin: %v", err)
	}

	return clonePath
}

// createOriginAndClone creates a bare origin, a local repo that pushes to it,
// and returns a fresh clone path from that origin. If branch is provided,
// it will be created and pushed in addition to main.
func createOriginAndClone(t *testing.T, branch string) string {
	t.Helper()

	originPath := createBareRemoteRepo(t)
	localRepoPath := createLocalRepoWithOrigin(t, originPath)

	if branch != "" && branch != "main" {
		repo, err := git.PlainOpen(localRepoPath)
		if err != nil {
			t.Fatalf("failed to open repo: %v", err)
		}

		worktree, err := repo.Worktree()
		if err != nil {
			t.Fatalf("failed to get worktree: %v", err)
		}

		if err := worktree.Checkout(&git.CheckoutOptions{
			Branch: plumbing.NewBranchReferenceName(branch),
			Create: true,
		}); err != nil {
			t.Fatalf("failed to checkout branch: %v", err)
		}

		filePath := filepath.Join(localRepoPath, "branch.txt")
		if err := os.WriteFile(filePath, []byte("branch\n"), 0o644); err != nil {
			t.Fatalf("failed to write branch file: %v", err)
		}

		if _, err := worktree.Add("branch.txt"); err != nil {
			t.Fatalf("failed to add branch file: %v", err)
		}

		if _, err := worktree.Commit("branch commit", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "test",
				Email: "test@example.com",
				When:  time.Now(),
			},
		}); err != nil {
			t.Fatalf("failed to commit branch change: %v", err)
		}
	}

	pushToOrigin(t, localRepoPath, "main")

	if branch != "" && branch != "main" {
		pushToOrigin(t, localRepoPath, branch)
	}

	return createLocalCloneFromOrigin(t, originPath)
}

// addUncommittedChange creates an uncommitted change in the repo to simulate a dirty state.
func addUncommittedChange(t *testing.T, repoPath string) {
	t.Helper()

	filePath := filepath.Join(repoPath, "dirty.txt")
	if err := os.WriteFile(filePath, []byte("dirty\n"), 0o644); err != nil {
		t.Fatalf("failed to write dirty file: %v", err)
	}
}
