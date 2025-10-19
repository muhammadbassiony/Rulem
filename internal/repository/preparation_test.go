package repository

import (
	"rulem/internal/logging"
	"strings"
	"testing"
)

// TestPrepareRepository_LocalSource tests preparing a local repository
func TestPrepareRepository_LocalSource(t *testing.T) {
	tempDir := t.TempDir()
	logger, _ := logging.NewTestLogger()

	repo := RepositoryEntry{
		ID:        "test-repo-123",
		Name:      "Test Repo",
		Type:      RepositoryTypeLocal,
		Path:      tempDir,
		CreatedAt: 1234567890,
	}

	localPath, err := PrepareRepository(repo, logger)
	if err != nil {
		t.Fatalf("PrepareRepository failed: %v", err)
	}

	// Should return the absolute path
	if localPath != tempDir {
		t.Errorf("Expected path '%s', got '%s'", tempDir, localPath)
	}
}

// TestPrepareRepository_InvalidLocalPath tests error handling for invalid paths
func TestPrepareRepository_InvalidLocalPath(t *testing.T) {
	logger, _ := logging.NewTestLogger()

	repo := RepositoryEntry{
		ID:        "test-repo-123",
		Name:      "Test Repo",
		Type:      RepositoryTypeLocal,
		Path:      "/nonexistent/directory/that/should/not/exist",
		CreatedAt: 1234567890,
	}

	_, err := PrepareRepository(repo, logger)
	if err == nil {
		t.Fatal("Expected error for invalid local path")
	}

	// Should get an error about the directory not existing
	if !strings.Contains(strings.ToLower(err.Error()), "does not exist") {
		t.Errorf("Expected 'does not exist' error, got: %v", err)
	}
}

// TestPrepareRepository_WithNilLogger tests operation with nil logger
func TestPrepareRepository_WithNilLogger(t *testing.T) {
	tempDir := t.TempDir()

	repo := RepositoryEntry{
		ID:        "test-repo-123",
		Name:      "Test Repo",
		Type:      RepositoryTypeLocal,
		Path:      tempDir,
		CreatedAt: 1234567890,
	}

	// Should work with nil logger (logging calls are guarded)
	localPath, err := PrepareRepository(repo, nil)
	if err != nil {
		t.Fatalf("PrepareRepository with nil logger failed: %v", err)
	}

	if localPath != tempDir {
		t.Errorf("Expected path '%s', got '%s'", tempDir, localPath)
	}
}

// TestNewLocalSource tests local source creation
func TestNewLocalSource(t *testing.T) {
	path := "/home/user/rules"

	source := NewLocalSource(path)

	if source.Path != path {
		t.Errorf("Expected Path to be %s, got %s", path, source.Path)
	}
}

// TestNewGitSource tests git source creation
func TestNewGitSource(t *testing.T) {
	remoteURL := "https://github.com/user/repo.git"
	branch := "main"
	localPath := "/tmp/cached-repo"

	source := NewGitSource(remoteURL, &branch, localPath)

	if source.RemoteURL != remoteURL {
		t.Errorf("Expected RemoteURL to be %s, got %s", remoteURL, source.RemoteURL)
	}

	if source.Branch == nil || *source.Branch != branch {
		t.Errorf("Expected Branch to be %s, got %v", branch, source.Branch)
	}

	if source.Path != localPath {
		t.Errorf("Expected Path to be %s, got %s", localPath, source.Path)
	}
}

// TestNewGitSource_NilBranch tests git source creation with nil branch
func TestNewGitSource_NilBranch(t *testing.T) {
	remoteURL := "https://github.com/user/repo.git"
	localPath := "/tmp/cached-repo"

	source := NewGitSource(remoteURL, nil, localPath)

	if source.RemoteURL != remoteURL {
		t.Errorf("Expected RemoteURL to be %s, got %s", remoteURL, source.RemoteURL)
	}

	if source.Branch != nil {
		t.Errorf("Expected Branch to be nil, got %v", source.Branch)
	}

	if source.Path != localPath {
		t.Errorf("Expected Path to be %s, got %s", localPath, source.Path)
	}
}

// TestPrepareAllRepositories_EmptyList tests preparation of empty repository list
func TestPrepareAllRepositories_EmptyList(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	prepared, err := PrepareAllRepositories([]RepositoryEntry{}, logger)

	if err != nil {
		t.Errorf("expected no error for empty list, got: %v", err)
	}
	if len(prepared) != 0 {
		t.Errorf("expected empty prepared list, got %d entries", len(prepared))
	}
}

// TestPrepareAllRepositories_SingleLocalRepo tests preparation of a single local repository
func TestPrepareAllRepositories_SingleLocalRepo(t *testing.T) {
	tempDir := t.TempDir()
	logger, _ := logging.NewTestLogger()

	repos := []RepositoryEntry{
		{
			ID:        "test-repo-1234567890",
			Name:      "Test Repo",
			Type:      RepositoryTypeLocal,
			Path:      tempDir,
			CreatedAt: 1234567890,
		},
	}

	prepared, err := PrepareAllRepositories(repos, logger)
	if err != nil {
		t.Fatalf("PrepareAllRepositories failed: %v", err)
	}

	if len(prepared) != 1 {
		t.Fatalf("expected 1 prepared repo, got %d", len(prepared))
	}

	prep := prepared[0]
	if prep.ID() != "test-repo-1234567890" {
		t.Errorf("expected ID 'test-repo-1234567890', got '%s'", prep.ID())
	}
	if prep.LocalPath != tempDir {
		t.Errorf("expected path '%s', got '%s'", tempDir, prep.LocalPath)
	}
	if prep.WasSynced() {
		t.Error("local repo should not be synced")
	}
	if prep.SyncResult.Status != SyncStatusSkipped {
		t.Errorf("expected skipped status for local repo, got %s", prep.SyncResult.Status)
	}
}

// TestPrepareAllRepositories_MultipleLocalRepos tests preparation of multiple local repositories
func TestPrepareAllRepositories_MultipleLocalRepos(t *testing.T) {
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()
	logger, _ := logging.NewTestLogger()

	repos := []RepositoryEntry{
		{
			ID:        "repo1-1234567890",
			Name:      "Repo 1",
			Type:      RepositoryTypeLocal,
			Path:      tempDir1,
			CreatedAt: 1234567890,
		},
		{
			ID:        "repo2-1234567891",
			Name:      "Repo 2",
			Type:      RepositoryTypeLocal,
			Path:      tempDir2,
			CreatedAt: 1234567891,
		},
	}

	prepared, err := PrepareAllRepositories(repos, logger)
	if err != nil {
		t.Fatalf("PrepareAllRepositories failed: %v", err)
	}

	if len(prepared) != 2 {
		t.Fatalf("expected 2 prepared repos, got %d", len(prepared))
	}

	// Check first repo
	if prepared[0].ID() != "repo1-1234567890" {
		t.Errorf("expected first repo ID 'repo1-1234567890', got '%s'", prepared[0].ID())
	}
	if prepared[0].LocalPath != tempDir1 {
		t.Errorf("expected first repo path '%s', got '%s'", tempDir1, prepared[0].LocalPath)
	}

	// Check second repo
	if prepared[1].ID() != "repo2-1234567891" {
		t.Errorf("expected second repo ID 'repo2-1234567891', got '%s'", prepared[1].ID())
	}
	if prepared[1].LocalPath != tempDir2 {
		t.Errorf("expected second repo path '%s', got '%s'", tempDir2, prepared[1].LocalPath)
	}
}

// TestPrepareAllRepositories_PreparationFailure tests handling of preparation failures
func TestPrepareAllRepositories_PreparationFailure(t *testing.T) {
	logger, _ := logging.NewTestLogger()

	repos := []RepositoryEntry{
		{
			ID:        "invalid-repo-1234567890",
			Name:      "Invalid Repo",
			Type:      RepositoryTypeLocal,
			Path:      "/nonexistent/directory/that/should/not/exist",
			CreatedAt: 1234567890,
		},
	}

	prepared, err := PrepareAllRepositories(repos, logger)
	if err == nil {
		t.Fatal("expected error for invalid repository")
	}

	// Should still return partial results
	if len(prepared) != 0 {
		t.Errorf("expected 0 prepared repos for complete failure, got %d", len(prepared))
	}

	// Error should mention the failure
	if !strings.Contains(err.Error(), "failed to prepare") {
		t.Errorf("expected error to mention preparation failure, got: %v", err)
	}
}

// TestPrepareAllRepositories_PartialFailure tests handling when some repos fail
func TestPrepareAllRepositories_PartialFailure(t *testing.T) {
	tempDir := t.TempDir()
	logger, _ := logging.NewTestLogger()

	repos := []RepositoryEntry{
		{
			ID:        "valid-repo-1234567890",
			Name:      "Valid Repo",
			Type:      RepositoryTypeLocal,
			Path:      tempDir,
			CreatedAt: 1234567890,
		},
		{
			ID:        "invalid-repo-1234567891",
			Name:      "Invalid Repo",
			Type:      RepositoryTypeLocal,
			Path:      "/nonexistent/directory",
			CreatedAt: 1234567891,
		},
	}

	prepared, err := PrepareAllRepositories(repos, logger)

	// Should return error for partial failure
	if err == nil {
		t.Fatal("expected error for partial failure")
	}

	// Should still prepare the valid repo
	if len(prepared) != 1 {
		t.Fatalf("expected 1 prepared repo (valid one), got %d", len(prepared))
	}

	if prepared[0].ID() != "valid-repo-1234567890" {
		t.Errorf("expected valid repo to be prepared, got ID '%s'", prepared[0].ID())
	}
}

// TestPrepareAllRepositories_ValidationFailure tests that validation errors prevent preparation
func TestPrepareAllRepositories_ValidationFailure(t *testing.T) {
	logger, _ := logging.NewTestLogger()

	repos := []RepositoryEntry{
		{
			ID:        "valid-1234567890",
			Name:      "Valid Repo",
			Type:      RepositoryTypeLocal,
			Path:      "/tmp/test",
			CreatedAt: 1234567890,
		},
		{
			ID:        "", // Invalid: empty ID
			Name:      "Invalid Repo",
			Type:      RepositoryTypeLocal,
			Path:      "/tmp/test2",
			CreatedAt: 1234567891,
		},
	}

	prepared, err := PrepareAllRepositories(repos, logger)
	if err == nil {
		t.Fatal("expected validation error")
	}

	// No repos should be prepared if validation fails
	if len(prepared) != 0 {
		t.Errorf("expected 0 prepared repos after validation failure, got %d", len(prepared))
	}

	// Error should mention validation
	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("expected validation error, got: %v", err)
	}
}

// TestPrepareAllRepositories_WithNilLogger tests operation with nil logger
func TestPrepareAllRepositories_WithNilLogger(t *testing.T) {
	tempDir := t.TempDir()

	repos := []RepositoryEntry{
		{
			ID:        "test-repo-1234567890",
			Name:      "Test Repo",
			Type:      RepositoryTypeLocal,
			Path:      tempDir,
			CreatedAt: 1234567890,
		},
	}

	// Should work with nil logger (all logging calls are guarded)
	prepared, err := PrepareAllRepositories(repos, nil)
	if err != nil {
		t.Fatalf("PrepareAllRepositories with nil logger failed: %v", err)
	}

	if len(prepared) != 1 {
		t.Fatalf("expected 1 prepared repo, got %d", len(prepared))
	}

	if prepared[0].LocalPath != tempDir {
		t.Errorf("expected path '%s', got '%s'", tempDir, prepared[0].LocalPath)
	}
}

// TestPrepareAllRepositories_DuplicateNames tests that duplicate names are caught
func TestPrepareAllRepositories_DuplicateNames(t *testing.T) {
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()
	logger, _ := logging.NewTestLogger()

	repos := []RepositoryEntry{
		{
			ID:        "repo1-1234567890",
			Name:      "Same Name",
			Type:      RepositoryTypeLocal,
			Path:      tempDir1,
			CreatedAt: 1234567890,
		},
		{
			ID:        "repo2-1234567891",
			Name:      "Same Name", // Duplicate name
			Type:      RepositoryTypeLocal,
			Path:      tempDir2,
			CreatedAt: 1234567891,
		},
	}

	prepared, err := PrepareAllRepositories(repos, logger)
	if err == nil {
		t.Fatal("expected error for duplicate names")
	}

	if len(prepared) != 0 {
		t.Errorf("expected 0 prepared repos after validation failure, got %d", len(prepared))
	}

	if !strings.Contains(err.Error(), "duplicate repository name") {
		t.Errorf("expected duplicate name error, got: %v", err)
	}
}
