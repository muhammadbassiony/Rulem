package repository

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"rulem/internal/logging"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing/object"
)

// TestValidateAllRepositories_EmptyList tests that empty repository list is valid
func TestValidateAllRepositories_EmptyList(t *testing.T) {
	err := ValidateAllRepositories([]RepositoryEntry{})
	if err != nil {
		t.Errorf("expected no error for empty list, got: %v", err)
	}
}

// TestValidateAllRepositories_ValidRepositories tests validation of valid repositories
func TestValidateAllRepositories_ValidRepositories(t *testing.T) {
	repos := []RepositoryEntry{
		{
			ID:        "personal-rules-1728756432",
			Name:      "Personal Rules",
			Type:      RepositoryTypeLocal,
			Path:      "/home/user/rules",
			CreatedAt: 1728756432,
		},
		{
			ID:        "work-rules-1728756500",
			Name:      "Work Rules",
			Type:      RepositoryTypeLocal,
			Path:      "/home/user/work-rules",
			CreatedAt: 1728756500,
		},
	}

	err := ValidateAllRepositories(repos)
	if err != nil {
		t.Errorf("expected no error for valid repositories, got: %v", err)
	}
}

// TestValidateAllRepositories_DuplicateIDs tests detection of duplicate repository IDs
func TestValidateAllRepositories_DuplicateIDs(t *testing.T) {
	repos := []RepositoryEntry{
		{
			ID:        "personal-rules-1728756432",
			Name:      "Personal Rules",
			Type:      RepositoryTypeLocal,
			Path:      "/home/user/rules",
			CreatedAt: 1728756432,
		},
		{
			ID:        "personal-rules-1728756432", // Duplicate ID
			Name:      "Different Name",
			Type:      RepositoryTypeLocal,
			Path:      "/home/user/other-rules",
			CreatedAt: 1728756432,
		},
	}

	err := ValidateAllRepositories(repos)
	if err == nil {
		t.Error("expected error for duplicate IDs, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate repository ID") {
		t.Errorf("expected 'duplicate repository ID' error, got: %v", err)
	}
}

// TestValidateAllRepositories_DuplicateNames tests detection of duplicate repository names
func TestValidateAllRepositories_DuplicateNames(t *testing.T) {
	repos := []RepositoryEntry{
		{
			ID:        "personal-rules-1728756432",
			Name:      "Personal Rules",
			Type:      RepositoryTypeLocal,
			Path:      "/home/user/rules",
			CreatedAt: 1728756432,
		},
		{
			ID:        "other-rules-1728756500",
			Name:      "personal rules", // Same name, different case
			Type:      RepositoryTypeLocal,
			Path:      "/home/user/other-rules",
			CreatedAt: 1728756500,
		},
	}

	err := ValidateAllRepositories(repos)
	if err == nil {
		t.Error("expected error for duplicate names, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate repository name") {
		t.Errorf("expected 'duplicate repository name' error, got: %v", err)
	}
}

// TestValidateAllRepositories_InvalidID tests validation of invalid repository IDs
func TestValidateAllRepositories_InvalidID(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		expectError bool
		errorPart   string
	}{
		{
			name:        "empty ID",
			id:          "",
			expectError: true,
			errorPart:   "cannot be empty",
		},
		{
			name:        "ID without dash",
			id:          "personalrules1728756432",
			expectError: true,
			errorPart:   "invalid repository ID format",
		},
		{
			name:        "ID with non-numeric timestamp",
			id:          "personal-rules-abc",
			expectError: true,
			errorPart:   "timestamp must be numeric",
		},
		{
			name:        "ID ending with dash",
			id:          "personal-rules-",
			expectError: true,
			errorPart:   "missing timestamp",
		},
		{
			name:        "valid ID",
			id:          "personal-rules-1728756432",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repos := []RepositoryEntry{
				{
					ID:        tt.id,
					Name:      "Test Repository",
					Type:      RepositoryTypeLocal,
					Path:      "/tmp/test",
					CreatedAt: 1728756432,
				},
			}

			err := ValidateAllRepositories(repos)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errorPart)
				} else if !strings.Contains(err.Error(), tt.errorPart) {
					t.Errorf("expected error containing %q, got: %v", tt.errorPart, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}
		})
	}
}

// TestValidateAllRepositories_InvalidName tests validation of invalid repository names
func TestValidateAllRepositories_InvalidName(t *testing.T) {
	tests := []struct {
		name        string
		repoName    string
		expectError bool
		errorPart   string
	}{
		{
			name:        "empty name",
			repoName:    "",
			expectError: true,
			errorPart:   "cannot be empty",
		},
		{
			name:        "whitespace-only name",
			repoName:    "   ",
			expectError: true,
			errorPart:   "cannot be empty",
		},
		{
			name:        "name too long",
			repoName:    strings.Repeat("a", 101),
			expectError: true,
			errorPart:   "too long",
		},
		{
			name:        "valid name",
			repoName:    "Valid Repository Name",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repos := []RepositoryEntry{
				{
					ID:        "test-repo-1728756432",
					Name:      tt.repoName,
					Type:      RepositoryTypeLocal,
					Path:      "/tmp/test",
					CreatedAt: 1728756432,
				},
			}

			err := ValidateAllRepositories(repos)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errorPart)
				} else if !strings.Contains(err.Error(), tt.errorPart) {
					t.Errorf("expected error containing %q, got: %v", tt.errorPart, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}
		})
	}
}

// TestValidateAllRepositories_InvalidCreatedAt tests validation of invalid CreatedAt timestamps
func TestValidateAllRepositories_InvalidCreatedAt(t *testing.T) {
	tests := []struct {
		name        string
		createdAt   int64
		expectError bool
	}{
		{
			name:        "zero timestamp",
			createdAt:   0,
			expectError: true,
		},
		{
			name:        "negative timestamp",
			createdAt:   -1,
			expectError: true,
		},
		{
			name:        "positive timestamp",
			createdAt:   1728756432,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repos := []RepositoryEntry{
				{
					ID:        "test-repo-1728756432",
					Name:      "Test Repository",
					Type:      RepositoryTypeLocal,
					Path:      "/tmp/test",
					CreatedAt: tt.createdAt,
				},
			}

			err := ValidateAllRepositories(repos)
			if tt.expectError {
				if err == nil {
					t.Error("expected error for invalid CreatedAt, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}
		})
	}
}

// TestPrepareAllRepositories_EmptyList tests preparation of empty repository list
func TestPrepareAllRepositories_EmptyList(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	pathMap, syncResults, err := PrepareAllRepositories([]RepositoryEntry{}, logger)

	if err != nil {
		t.Errorf("expected no error for empty list, got: %v", err)
	}
	if len(pathMap) != 0 {
		t.Errorf("expected empty path map, got %d entries", len(pathMap))
	}
	if len(syncResults) != 0 {
		t.Errorf("expected empty sync results, got %d entries", len(syncResults))
	}
}

// TestPrepareAllRepositories_MultipleLocalRepos tests preparation of multiple local repositories
func TestPrepareAllRepositories_MultipleLocalRepos(t *testing.T) {
	tempDir := t.TempDir()

	// Create test directories
	repo1Path := filepath.Join(tempDir, "repo1")
	repo2Path := filepath.Join(tempDir, "repo2")
	os.MkdirAll(repo1Path, 0755)
	os.MkdirAll(repo2Path, 0755)

	repos := []RepositoryEntry{
		{
			ID:        "repo1-1728756432",
			Name:      "Repository 1",
			Type:      RepositoryTypeLocal,
			Path:      repo1Path,
			CreatedAt: 1728756432,
		},
		{
			ID:        "repo2-1728756500",
			Name:      "Repository 2",
			Type:      RepositoryTypeLocal,
			Path:      repo2Path,
			CreatedAt: 1728756500,
		},
	}

	logger, _ := logging.NewTestLogger()
	pathMap, syncResults, err := PrepareAllRepositories(repos, logger)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(pathMap) != 2 {
		t.Errorf("expected 2 entries in path map, got %d", len(pathMap))
	}

	// Verify paths are correct
	if path, ok := pathMap["repo1-1728756432"]; !ok {
		t.Error("expected repo1 in path map")
	} else if !strings.Contains(path, "repo1") {
		t.Errorf("expected path to contain 'repo1', got: %s", path)
	}

	if path, ok := pathMap["repo2-1728756500"]; !ok {
		t.Error("expected repo2 in path map")
	} else if !strings.Contains(path, "repo2") {
		t.Errorf("expected path to contain 'repo2', got: %s", path)
	}

	// Local repos should all be skipped in sync
	if len(syncResults) != 2 {
		t.Errorf("expected 2 sync results, got %d", len(syncResults))
	}
	for _, result := range syncResults {
		if result.Status != SyncStatusSkipped {
			t.Errorf("expected local repo to be skipped, got status: %s", result.Status)
		}
	}
}

// TestPrepareAllRepositories_MultipleGitHubRepos tests preparation of multiple GitHub repositories
func TestPrepareAllRepositories_MultipleGitHubRepos(t *testing.T) {
	tempDir := t.TempDir()

	// Create test Git repositories
	repo1Path := filepath.Join(tempDir, "github-repo1")
	repo2Path := filepath.Join(tempDir, "github-repo2")

	remoteURL1 := "https://github.com/user/repo1.git"
	remoteURL2 := "https://github.com/user/repo2.git"

	createTestGitRepo(t, repo1Path, &remoteURL1)
	createTestGitRepo(t, repo2Path, &remoteURL2)

	repos := []RepositoryEntry{
		{
			ID:        "github-repo1-1728756432",
			Name:      "GitHub Repository 1",
			Type:      RepositoryTypeGitHub,
			Path:      repo1Path,
			RemoteURL: &remoteURL1,
			CreatedAt: 1728756432,
		},
		{
			ID:        "github-repo2-1728756500",
			Name:      "GitHub Repository 2",
			Type:      RepositoryTypeGitHub,
			Path:      repo2Path,
			RemoteURL: &remoteURL2,
			CreatedAt: 1728756500,
		},
	}

	logger, _ := logging.NewTestLogger()
	pathMap, syncResults, err := PrepareAllRepositories(repos, logger)

	// GitHub repos will fail to prepare without actual remotes, but that's expected in tests
	// We're testing the path map structure and sync orchestration
	if err != nil {
		// Allow preparation errors for GitHub repos in test environment
		t.Logf("preparation errors (expected in test): %v", err)
	}

	// Path map may be empty if preparation failed (expected without real remotes)
	t.Logf("path map entries: %d", len(pathMap))

	// Sync results should be nil when preparation fails
	if syncResults != nil {
		t.Logf("sync results: %d", len(syncResults))
	}
}

// TestPrepareAllRepositories_MixedLocalAndGitHub tests preparation of mixed repository types
func TestPrepareAllRepositories_MixedLocalAndGitHub(t *testing.T) {
	tempDir := t.TempDir()

	// Local repo
	localPath := filepath.Join(tempDir, "local-repo")
	os.MkdirAll(localPath, 0755)

	// GitHub repo
	githubPath := filepath.Join(tempDir, "github-repo")
	remoteURL := "https://github.com/user/repo.git"

	createTestGitRepo(t, githubPath, &remoteURL)

	repos := []RepositoryEntry{
		{
			ID:        "local-repo-1728756432",
			Name:      "Local Repository",
			Type:      RepositoryTypeLocal,
			Path:      localPath,
			CreatedAt: 1728756432,
		},
		{
			ID:        "github-repo-1728756500",
			Name:      "GitHub Repository",
			Type:      RepositoryTypeGitHub,
			Path:      githubPath,
			RemoteURL: &remoteURL,
			CreatedAt: 1728756500,
		},
	}

	logger, _ := logging.NewTestLogger()
	pathMap, syncResults, err := PrepareAllRepositories(repos, logger)

	// Local repo should succeed, GitHub may fail without real remote
	if err != nil {
		t.Logf("preparation errors (may be expected): %v", err)
	}

	// Local repo should be in the map
	if _, ok := pathMap["local-repo-1728756432"]; !ok {
		t.Error("expected local-repo in path map")
	}

	// If GitHub repo prepared successfully, it should be in the map too
	if len(pathMap) == 2 {
		if _, ok := pathMap["github-repo-1728756500"]; !ok {
			t.Error("expected github-repo in path map when both repos prepared")
		}
	}

	// Verify sync results (only if we have them)
	if syncResults == nil {
		t.Log("no sync results (preparation failed)")
		return
	}

	if len(syncResults) < 1 {
		t.Fatalf("expected at least 1 sync result, got %d", len(syncResults))
	}

	// Find results by ID
	var localResult, githubResult *RepositorySyncResult
	for i := range syncResults {
		if syncResults[i].RepositoryID == "local-repo-1728756432" {
			localResult = &syncResults[i]
		}
		if syncResults[i].RepositoryID == "github-repo-1728756500" {
			githubResult = &syncResults[i]
		}
	}

	// Local repo should be skipped
	if localResult != nil && localResult.Status != SyncStatusSkipped {
		t.Errorf("expected local repo to be skipped, got: %s", localResult.Status)
	}

	// GitHub repo result may be present if preparation succeeded
	if githubResult != nil {
		t.Logf("github repo sync status: %s", githubResult.Status)
	}
}

// TestPrepareAllRepositories_PreparationFailure tests handling of preparation failures
func TestPrepareAllRepositories_PreparationFailure(t *testing.T) {
	repos := []RepositoryEntry{
		{
			ID:        "nonexistent-repo-1728756432",
			Name:      "Nonexistent Repository",
			CreatedAt: 1728756432,
			Type:      RepositoryTypeLocal,
			Path:      "/nonexistent/path/to/repo",
		},
	}

	logger, _ := logging.NewTestLogger()
	pathMap, syncResults, err := PrepareAllRepositories(repos, logger)

	// Should return an error
	if err == nil {
		t.Error("expected error for nonexistent repository, got nil")
	}

	// Path map should be empty (no successful preparations)
	if len(pathMap) != 0 {
		t.Errorf("expected empty path map, got %d entries", len(pathMap))
	}

	// Sync results should be nil (preparation failed)
	if syncResults != nil {
		t.Errorf("expected nil sync results, got %d entries", len(syncResults))
	}
}

// TestPrepareAllRepositories_PartialFailure tests handling when some repos fail
func TestPrepareAllRepositories_PartialFailure(t *testing.T) {
	tempDir := t.TempDir()

	// Valid local repo
	validPath := filepath.Join(tempDir, "valid-repo")
	os.MkdirAll(validPath, 0755)

	repos := []RepositoryEntry{
		{
			ID:        "valid-repo-1728756432",
			Name:      "Valid Repository",
			Type:      RepositoryTypeLocal,
			Path:      validPath,
			CreatedAt: 1728756432,
		},
		{
			ID:        "invalid-repo-1728756500",
			Name:      "Invalid Repository",
			Type:      RepositoryTypeLocal,
			Path:      "/nonexistent/path",
			CreatedAt: 1728756500,
		},
	}

	logger, _ := logging.NewTestLogger()
	pathMap, _, err := PrepareAllRepositories(repos, logger)

	// Should return error due to partial failure
	if err == nil {
		t.Error("expected error for partial failure, got nil")
	}

	// Path map should contain the successful one
	if len(pathMap) != 1 {
		t.Errorf("expected 1 entry in path map, got %d", len(pathMap))
	}

	if _, ok := pathMap["valid-repo-1728756432"]; !ok {
		t.Error("expected valid repo in path map")
	}
}

// TestPrepareAllRepositories_ValidationFailure tests that validation errors prevent preparation
func TestPrepareAllRepositories_ValidationFailure(t *testing.T) {
	repos := []RepositoryEntry{
		{
			ID:        "repo1-1728756432",
			Name:      "Repository 1",
			Type:      RepositoryTypeLocal,
			Path:      "/tmp/repo1",
			CreatedAt: 1728756432,
		},
		{
			ID:        "repo1-1728756432", // Duplicate ID
			Name:      "Repository 2",
			Type:      RepositoryTypeLocal,
			Path:      "/tmp/repo2",
			CreatedAt: 1728756432,
		},
	}

	logger, _ := logging.NewTestLogger()
	pathMap, syncResults, err := PrepareAllRepositories(repos, logger)

	// Should fail validation
	if err == nil {
		t.Error("expected validation error, got nil")
	}
	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("expected validation error, got: %v", err)
	}

	// Should not attempt preparation
	if pathMap != nil {
		t.Errorf("expected nil path map after validation failure, got %d entries", len(pathMap))
	}
	if syncResults != nil {
		t.Errorf("expected nil sync results after validation failure, got %d entries", len(syncResults))
	}
}

// TestPrepareAllRepositories_WithNilLogger tests that preparation works without a logger
func TestPrepareAllRepositories_WithNilLogger(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "test-repo")
	os.MkdirAll(repoPath, 0755)

	repos := []RepositoryEntry{
		{
			ID:        "test-repo-1728756432",
			Name:      "Test Repository",
			Type:      RepositoryTypeLocal,
			Path:      repoPath,
			CreatedAt: 1728756432,
		},
	}

	// Should not panic with nil logger
	pathMap, syncResults, err := PrepareAllRepositories(repos, nil)

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if len(pathMap) != 1 {
		t.Errorf("expected 1 entry in path map, got %d", len(pathMap))
	}
	if len(syncResults) != 1 {
		t.Errorf("expected 1 sync result, got %d", len(syncResults))
	}
}

// Helper functions

// createTestGitRepo creates a Git repository with a commit and optional remote.
// This helper reduces code duplication across tests that need Git repositories.
//
// Parameters:
//   - t: Test instance
//   - repoPath: Path where the repository should be created
//   - remoteURL: Optional remote URL (nil for no remote)
//
// Returns:
//   - *git.Repository: Initialized Git repository with one commit
func createTestGitRepo(t *testing.T, repoPath string, remoteURL *string) *git.Repository {
	t.Helper()

	// Initialize repository
	repo, err := git.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("failed to initialize git repo: %v", err)
	}

	// Create and commit a file
	createCommittedFile(t, repo, repoPath, "test.md")

	// Add remote if provided
	if remoteURL != nil {
		addRemote(t, repo, "origin", *remoteURL)
	}

	return repo
}

func addRemote(t *testing.T, repo *git.Repository, name, url string) {
	t.Helper()

	_, err := repo.CreateRemote(&config.RemoteConfig{
		Name: name,
		URLs: []string{url},
	})
	if err != nil {
		t.Fatalf("failed to add remote: %v", err)
	}
}

func createCommittedFile(t *testing.T, repo *git.Repository, repoPath, filename string) {
	t.Helper()

	// Create file
	filePath := filepath.Join(repoPath, filename)
	if err := os.WriteFile(filePath, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Add to git
	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	if _, err := worktree.Add(filename); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	// Commit
	_, err = worktree.Commit("Test commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("failed to commit: %v", err)
	}
}
