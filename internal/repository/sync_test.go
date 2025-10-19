package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"rulem/internal/logging"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
)

// TestSyncStatus_String tests the String() method for all SyncStatus values
func TestSyncStatus_String(t *testing.T) {
	tests := []struct {
		name     string
		status   SyncStatus
		expected string
	}{
		{
			name:     "success status",
			status:   SyncStatusSuccess,
			expected: "Success",
		},
		{
			name:     "failed status",
			status:   SyncStatusFailed,
			expected: "Failed",
		},
		{
			name:     "skipped status",
			status:   SyncStatusSkipped,
			expected: "Skipped",
		},
		{
			name:     "unknown status",
			status:   SyncStatus(999),
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.status.String()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestRepositorySyncResult_GetMessage tests the GetMessage() method for various result states
func TestRepositorySyncResult_GetMessage(t *testing.T) {
	tests := []struct {
		name     string
		result   RepositorySyncResult
		expected string
	}{
		{
			name: "success with duration",
			result: RepositorySyncResult{
				Status:   SyncStatusSuccess,
				Duration: 1234 * time.Millisecond,
			},
			expected: "Synced successfully in 1.2s",
		},
		{
			name: "failed with error",
			result: RepositorySyncResult{
				Status: SyncStatusFailed,
				Error:  fmt.Errorf("network timeout"),
			},
			expected: "Sync failed: network timeout",
		},
		{
			name: "failed without error",
			result: RepositorySyncResult{
				Status: SyncStatusFailed,
			},
			expected: "Sync failed: unknown error",
		},
		{
			name: "skipped with reason",
			result: RepositorySyncResult{
				Status:     SyncStatusSkipped,
				SkipReason: "uncommitted changes",
			},
			expected: "Skipped: uncommitted changes",
		},
		{
			name: "skipped without reason",
			result: RepositorySyncResult{
				Status: SyncStatusSkipped,
			},
			expected: "Skipped",
		},
		{
			name: "unknown status",
			result: RepositorySyncResult{
				Status: SyncStatus(999),
			},
			expected: "Unknown status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := tt.result.GetMessage()
			if message != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, message)
			}
		})
	}
}

// TestSyncAllRepositories_EmptyList tests syncing an empty repository list
func TestSyncAllRepositories_EmptyList(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	results := SyncAllRepositories([]RepositoryEntry{}, logger)

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

// TestSyncAllRepositories_LocalRepositorySkipped tests that local repositories are skipped
func TestSyncAllRepositories_LocalRepositorySkipped(t *testing.T) {
	logger, _ := logging.NewTestLogger()

	repos := []RepositoryEntry{
		{
			ID:        "local-repo-123",
			Name:      "Local Repository",
			CreatedAt: time.Now().Unix(),
			Type:      RepositoryTypeLocal,
			Path:      "/tmp/local-repo",
		},
	}

	results := SyncAllRepositories(repos, logger)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	result := results[0]
	if result.Status != SyncStatusSkipped {
		t.Errorf("expected status Skipped, got %s", result.Status)
	}
	if result.SkipReason != "not a GitHub repository" {
		t.Errorf("expected skip reason 'not a GitHub repository', got %q", result.SkipReason)
	}
	if result.RepositoryID != "local-repo-123" {
		t.Errorf("expected repository ID 'local-repo-123', got %q", result.RepositoryID)
	}
	if result.RepositoryName != "Local Repository" {
		t.Errorf("expected repository name 'Local Repository', got %q", result.RepositoryName)
	}
}

// TestSyncAllRepositories_DirtyRepositorySkipped tests that dirty repositories are skipped
func TestSyncAllRepositories_DirtyRepositorySkipped(t *testing.T) {
	// Create a temporary directory for the test repository
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "test-repo")

	// Initialize a Git repository
	repo, err := git.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("failed to initialize git repo: %v", err)
	}

	// Create a test file to make the repository dirty
	testFile := filepath.Join(repoPath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Add the file to the index (but don't commit, making it dirty)
	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}
	if _, err := worktree.Add("test.txt"); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	// Create repository entry with GitHub remote
	remoteURL := "https://github.com/test/test-repo.git"
	repos := []RepositoryEntry{
		{
			ID:        "github-repo-123",
			Name:      "GitHub Repository",
			CreatedAt: time.Now().Unix(),
			Type:      RepositoryTypeGitHub,
			Path:      repoPath,
			RemoteURL: &remoteURL,
		},
	}

	logger, _ := logging.NewTestLogger()
	results := SyncAllRepositories(repos, logger)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	result := results[0]
	if result.Status != SyncStatusSkipped {
		t.Errorf("expected status Skipped, got %s", result.Status)
	}
	if result.SkipReason != "uncommitted changes" {
		t.Errorf("expected skip reason 'uncommitted changes', got %q", result.SkipReason)
	}
}

// TestSyncAllRepositories_NonExistentRepositoryFailed tests error handling for non-existent repositories
func TestSyncAllRepositories_NonExistentRepositoryFailed(t *testing.T) {
	logger, _ := logging.NewTestLogger()

	// Create repository entry pointing to non-existent path
	remoteURL := "https://github.com/test/test-repo.git"
	repos := []RepositoryEntry{
		{
			ID:        "github-repo-123",
			Name:      "GitHub Repository",
			CreatedAt: time.Now().Unix(),
			Type:      RepositoryTypeGitHub,
			Path:      "/nonexistent/path/to/repo",
			RemoteURL: &remoteURL,
		},
	}

	results := SyncAllRepositories(repos, logger)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	result := results[0]
	if result.Status != SyncStatusFailed {
		t.Errorf("expected status Failed, got %s", result.Status)
	}
	if result.Error == nil {
		t.Error("expected error to be set, got nil")
	}
}

// TestSyncAllRepositories_MixedResults tests syncing multiple repositories with different outcomes
func TestSyncAllRepositories_MixedResults(t *testing.T) {
	// Create a temporary directory for test repositories
	tempDir := t.TempDir()

	// Repository 1: Local repository (should be skipped)
	repos := []RepositoryEntry{
		{
			ID:        "local-repo-123",
			Name:      "Local Repository",
			CreatedAt: time.Now().Unix(),
			Type:      RepositoryTypeLocal,
			Path:      filepath.Join(tempDir, "local-repo"),
		},
	}

	// Repository 2: GitHub repository with uncommitted changes (should be skipped)
	dirtyRepoPath := filepath.Join(tempDir, "dirty-repo")
	dirtyRepo, err := git.PlainInit(dirtyRepoPath, false)
	if err != nil {
		t.Fatalf("failed to initialize dirty repo: %v", err)
	}

	// Make it dirty by adding an uncommitted file
	testFile := filepath.Join(dirtyRepoPath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	dirtyWorktree, _ := dirtyRepo.Worktree()
	dirtyWorktree.Add("test.txt")

	dirtyRemoteURL := "https://github.com/test/dirty-repo.git"
	repos = append(repos, RepositoryEntry{
		ID:        "dirty-repo-456",
		Name:      "Dirty Repository",
		CreatedAt: time.Now().Unix(),
		Type:      RepositoryTypeGitHub,
		Path:      dirtyRepoPath,
		RemoteURL: &dirtyRemoteURL,
	})

	// Repository 3: Non-existent GitHub repository (should fail)
	failedRemoteURL := "https://github.com/test/failed-repo.git"
	repos = append(repos, RepositoryEntry{
		ID:        "failed-repo-789",
		Name:      "Failed Repository",
		CreatedAt: time.Now().Unix(),
		Type:      RepositoryTypeGitHub,
		Path:      "/nonexistent/path",
		RemoteURL: &failedRemoteURL,
	})

	logger, _ := logging.NewTestLogger()
	results := SyncAllRepositories(repos, logger)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Verify first result (local repo - skipped)
	if results[0].Status != SyncStatusSkipped {
		t.Errorf("result[0]: expected Skipped, got %s", results[0].Status)
	}
	if results[0].SkipReason != "not a GitHub repository" {
		t.Errorf("result[0]: expected skip reason 'not a GitHub repository', got %q", results[0].SkipReason)
	}

	// Verify second result (dirty repo - skipped)
	if results[1].Status != SyncStatusSkipped {
		t.Errorf("result[1]: expected Skipped, got %s", results[1].Status)
	}
	if results[1].SkipReason != "uncommitted changes" {
		t.Errorf("result[1]: expected skip reason 'uncommitted changes', got %q", results[1].SkipReason)
	}

	// Verify third result (non-existent - failed)
	if results[2].Status != SyncStatusFailed {
		t.Errorf("result[2]: expected Failed, got %s", results[2].Status)
	}
	if results[2].Error == nil {
		t.Error("result[2]: expected error to be set")
	}
}

// TestSyncAllRepositories_DurationTracking tests that duration is properly tracked
func TestSyncAllRepositories_DurationTracking(t *testing.T) {
	logger, _ := logging.NewTestLogger()

	repos := []RepositoryEntry{
		{
			ID:        "local-repo-123",
			Name:      "Local Repository",
			CreatedAt: time.Now().Unix(),
			Type:      RepositoryTypeLocal,
			Path:      "/tmp/local-repo",
		},
	}

	results := SyncAllRepositories(repos, logger)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Duration <= 0 {
		t.Errorf("expected positive duration, got %v", results[0].Duration)
	}
}

// TestSyncAllRepositories_IntegrationWithCleanRepo tests syncing a clean Git repository
func TestSyncAllRepositories_IntegrationWithCleanRepo(t *testing.T) {
	// This is an integration test that would require a real Git repository
	// For now, we'll test the clean repository path without actual network operations

	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "clean-repo")

	// Initialize a Git repository
	repo, err := git.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("failed to initialize git repo: %v", err)
	}

	// Create and commit a file to make it clean
	testFile := filepath.Join(repoPath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	if _, err := worktree.Add("test.txt"); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	// Commit the file
	_, err = worktree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Add a remote (but don't fetch - this will fail in the sync)
	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{"https://github.com/test/test-repo.git"},
	})
	if err != nil {
		t.Fatalf("failed to create remote: %v", err)
	}

	// Verify the repository is clean
	status, err := worktree.Status()
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}
	if !status.IsClean() {
		t.Fatal("expected repository to be clean")
	}

	// Create repository entry
	remoteURL := "https://github.com/test/test-repo.git"
	repos := []RepositoryEntry{
		{
			ID:        "clean-repo-123",
			Name:      "Clean Repository",
			CreatedAt: time.Now().Unix(),
			Type:      RepositoryTypeGitHub,
			Path:      repoPath,
			RemoteURL: &remoteURL,
		},
	}

	logger, _ := logging.NewTestLogger()
	results := SyncAllRepositories(repos, logger)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	result := results[0]

	// The sync will fail because we can't actually fetch from the remote
	// but it should get past the dirty check
	if result.Status == SyncStatusSkipped && result.SkipReason == "uncommitted changes" {
		t.Error("clean repository should not be skipped due to uncommitted changes")
	}

	// Verify repository metadata is preserved
	if result.RepositoryID != "clean-repo-123" {
		t.Errorf("expected repository ID 'clean-repo-123', got %q", result.RepositoryID)
	}
	if result.RepositoryName != "Clean Repository" {
		t.Errorf("expected repository name 'Clean Repository', got %q", result.RepositoryName)
	}
}

// TestSyncAllRepositories_PreservesRepositoryOrder tests that results maintain input order
func TestSyncAllRepositories_PreservesRepositoryOrder(t *testing.T) {
	logger, _ := logging.NewTestLogger()

	repos := []RepositoryEntry{
		{
			ID:        "repo-1",
			Name:      "First Repository",
			CreatedAt: time.Now().Unix(),
			Type:      RepositoryTypeLocal,
			Path:      "/tmp/repo1",
		},
		{
			ID:        "repo-2",
			Name:      "Second Repository",
			CreatedAt: time.Now().Unix(),
			Type:      RepositoryTypeLocal,
			Path:      "/tmp/repo2",
		},
		{
			ID:        "repo-3",
			Name:      "Third Repository",
			CreatedAt: time.Now().Unix(),
			Type:      RepositoryTypeLocal,
			Path:      "/tmp/repo3",
		},
	}

	results := SyncAllRepositories(repos, logger)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Verify order is preserved
	expectedIDs := []string{"repo-1", "repo-2", "repo-3"}
	for i, result := range results {
		if result.RepositoryID != expectedIDs[i] {
			t.Errorf("result[%d]: expected ID %q, got %q", i, expectedIDs[i], result.RepositoryID)
		}
	}
}

// TestSyncAllRepositories_WithNilLogger tests that sync works without a logger
func TestSyncAllRepositories_WithNilLogger(t *testing.T) {
	repos := []RepositoryEntry{
		{
			ID:        "local-repo-123",
			Name:      "Local Repository",
			CreatedAt: time.Now().Unix(),
			Type:      RepositoryTypeLocal,
			Path:      "/tmp/local-repo",
		},
	}

	// Should not panic with nil logger
	results := SyncAllRepositories(repos, nil)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Status != SyncStatusSkipped {
		t.Errorf("expected status Skipped, got %s", results[0].Status)
	}
}

// TestSyncAllRepositories_WithOptionalBranch tests syncing with optional branch configuration
func TestSyncAllRepositories_WithOptionalBranch(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "test-repo")

	// Initialize a Git repository
	repo, err := git.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("failed to initialize git repo: %v", err)
	}

	// Create and commit a file
	testFile := filepath.Join(repoPath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	if _, err := worktree.Add("test.txt"); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	_, err = worktree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Create a custom branch
	headRef, err := repo.Head()
	if err != nil {
		t.Fatalf("failed to get HEAD: %v", err)
	}

	customBranch := "custom-branch"
	branchRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName(customBranch), headRef.Hash())
	if err := repo.Storer.SetReference(branchRef); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	// Test with explicit branch configuration
	remoteURL := "https://github.com/test/test-repo.git"
	repos := []RepositoryEntry{
		{
			ID:        "branch-repo-123",
			Name:      "Repository with Branch",
			CreatedAt: time.Now().Unix(),
			Type:      RepositoryTypeGitHub,
			Path:      repoPath,
			RemoteURL: &remoteURL,
			Branch:    &customBranch,
		},
	}

	logger, _ := logging.NewTestLogger()
	results := SyncAllRepositories(repos, logger)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// The sync will attempt to use the custom branch
	// Since we can't actually fetch, it will fail, but it should respect the branch setting
	result := results[0]
	if result.RepositoryID != "branch-repo-123" {
		t.Errorf("expected repository ID 'branch-repo-123', got %q", result.RepositoryID)
	}
}
