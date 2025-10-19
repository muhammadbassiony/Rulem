package repository

import (
	"fmt"
	"time"

	"rulem/internal/logging"
)

// SyncStatus represents the outcome of a repository synchronization operation.
// It categorizes sync results into three states for proper error handling and UI display.
type SyncStatus int

const (
	// SyncStatusSuccess indicates the repository was successfully synchronized
	SyncStatusSuccess SyncStatus = iota

	// SyncStatusFailed indicates the synchronization failed due to an error
	// (network issues, authentication failures, etc.)
	SyncStatusFailed

	// SyncStatusSkipped indicates the synchronization was intentionally skipped
	// (e.g., dirty working tree, non-GitHub repository)
	SyncStatusSkipped
)

// String returns a human-readable representation of the sync status.
func (s SyncStatus) String() string {
	switch s {
	case SyncStatusSuccess:
		return "Success"
	case SyncStatusFailed:
		return "Failed"
	case SyncStatusSkipped:
		return "Skipped"
	default:
		return "Unknown"
	}
}

// RepositorySyncResult contains the outcome of synchronizing a single repository.
// It provides detailed status information for UI display and error reporting.
type RepositorySyncResult struct {
	// RepositoryID is the unique identifier of the repository (e.g., "personal-rules-1728756432")
	RepositoryID string

	// RepositoryName is the user-friendly display name (e.g., "Personal Rules")
	RepositoryName string

	// Status indicates the outcome of the sync operation
	Status SyncStatus

	// Error contains the error message if Status is SyncStatusFailed
	// This field is nil when Status is SyncStatusSuccess or SyncStatusSkipped
	Error error

	// SkipReason contains the reason for skipping if Status is SyncStatusSkipped
	// Common reasons include "uncommitted changes", "not a GitHub repository"
	SkipReason string

	// Duration is the time taken for the sync operation
	Duration time.Duration
}

// GetMessage returns a UI-friendly message describing the sync result.
// The message format varies based on the sync status:
// - Success: "Synced successfully in 1.2s"
// - Failed: "Sync failed: network timeout"
// - Skipped: "Skipped: uncommitted changes"
func (r *RepositorySyncResult) GetMessage() string {
	switch r.Status {
	case SyncStatusSuccess:
		return fmt.Sprintf("Synced successfully in %s", r.Duration.Round(100*time.Millisecond))
	case SyncStatusFailed:
		if r.Error != nil {
			return fmt.Sprintf("Sync failed: %v", r.Error)
		}
		return "Sync failed: unknown error"
	case SyncStatusSkipped:
		if r.SkipReason != "" {
			return fmt.Sprintf("Skipped: %s", r.SkipReason)
		}
		return "Skipped"
	default:
		return "Unknown status"
	}
}

// SyncAllRepositories synchronizes all GitHub repositories in the provided list.
// It handles each repository independently, ensuring that failures in one repository
// don't prevent others from being synced. Local repositories are skipped.
//
// The function performs the following for each repository:
// 1. Check if it's a GitHub repository (skip if local)
// 2. Check for uncommitted changes (skip if dirty)
// 3. Fetch updates from the remote (fail on error)
// 4. Track duration and status for each operation
//
// Parameters:
//   - repos: List of repository entries to synchronize
//   - logger: Logger for structured logging (can be nil)
//
// Returns:
//   - []RepositorySyncResult: Results for all repositories, in the same order as input
//
// Usage:
//
//	results := repository.SyncAllRepositories(cfg.Repositories, logger)
//	for _, result := range results {
//	    fmt.Printf("%s: %s\n", result.RepositoryName, result.GetMessage())
//	}
func SyncAllRepositories(repos []RepositoryEntry, logger *logging.AppLogger) []RepositorySyncResult {
	if logger != nil {
		logger.Info("Starting multi-repository sync", "repository_count", len(repos))
	}

	results := make([]RepositorySyncResult, 0, len(repos))

	for _, repo := range repos {
		result := syncSingleRepository(repo, logger)
		results = append(results, result)

		if logger != nil {
			logger.Info("Repository sync completed",
				"repository_id", result.RepositoryID,
				"repository_name", result.RepositoryName,
				"status", result.Status.String(),
				"duration", result.Duration,
			)
		}
	}

	if logger != nil {
		successCount := 0
		failedCount := 0
		skippedCount := 0
		for _, r := range results {
			switch r.Status {
			case SyncStatusSuccess:
				successCount++
			case SyncStatusFailed:
				failedCount++
			case SyncStatusSkipped:
				skippedCount++
			}
		}
		logger.Info("Multi-repository sync completed",
			"total", len(results),
			"success", successCount,
			"failed", failedCount,
			"skipped", skippedCount,
		)
	}

	return results
}

// syncSingleRepository synchronizes a single repository and returns the result.
// This is an internal helper function used by SyncAllRepositories.
func syncSingleRepository(repo RepositoryEntry, logger *logging.AppLogger) RepositorySyncResult {
	startTime := time.Now()

	result := RepositorySyncResult{
		RepositoryID:   repo.ID,
		RepositoryName: repo.Name,
	}

	// Skip non-GitHub repositories
	if !repo.IsRemote() {
		result.Status = SyncStatusSkipped
		result.SkipReason = "not a GitHub repository"
		result.Duration = time.Since(startTime)
		return result
	}

	// Check for uncommitted changes
	isDirty, err := CheckGithubRepositoryStatus(repo.Path)
	if err != nil {
		result.Status = SyncStatusFailed
		result.Error = fmt.Errorf("failed to check repository status: %w", err)
		result.Duration = time.Since(startTime)
		return result
	}

	if isDirty {
		result.Status = SyncStatusSkipped
		result.SkipReason = "uncommitted changes"
		result.Duration = time.Since(startTime)
		return result
	}

	// Perform sync operation
	gitSource := NewGitSource(*repo.RemoteURL, repo.Branch, repo.Path)
	_, err = gitSource.FetchUpdates(logger)
	if err != nil {
		result.Status = SyncStatusFailed
		result.Error = fmt.Errorf("fetch updates failed: %w", err)
		result.Duration = time.Since(startTime)
		return result
	}

	// Success
	result.Status = SyncStatusSuccess
	result.Duration = time.Since(startTime)
	return result
}
