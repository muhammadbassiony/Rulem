package repository

import (
	"fmt"
	"strings"

	"rulem/internal/logging"
)

// PrepareRepository creates the appropriate source and prepares it for use.
// This function examines the repository entry to determine whether to use LocalSource or GitSource,
// then prepares the source and returns the local path.
//
// This is the main entry point for preparing a single repository. It handles:
//   - Source selection based on repository type (local vs. GitHub)
//   - Source preparation (validation for local, clone/sync for Git)
//   - Path resolution to absolute local filesystem path
//   - Error handling with user-friendly messages
//
// For local repositories:
//   - Creates LocalSource with the configured path
//   - Validates the directory exists and is accessible
//   - Performs security checks (no traversal, no system dirs)
//   - Returns the absolute path
//
// For GitHub repositories:
//   - Creates GitSource with remote URL, branch, and local path
//   - Clones if not present (shallow clone for performance)
//   - Updates if already cloned and clean
//   - Handles authentication via credential store
//   - Returns the absolute path to the cloned repository
//
// Parameters:
//   - repo: Repository entry with type and configuration
//   - logger: Logger for structured logging during preparation (can be nil)
//
// Returns:
//   - string: Absolute path to the prepared local repository
//   - error: Preparation errors (validation, network, authentication, etc.)
//
// Usage:
//
//	localPath, err := repository.PrepareRepository(repo, logger)
//	if err != nil {
//	    return fmt.Errorf("repository preparation failed: %w", err)
//	}
//	fm := filemanager.NewFileManager(localPath, logger)
//
// Error handling:
//   - Returns detailed errors with context about what failed
//   - Local repos: Directory not found, permission denied, security violations
//   - GitHub repos: Clone failures, authentication errors, network issues
//   - All errors are suitable for display to end users
func PrepareRepository(repo RepositoryEntry, logger *logging.AppLogger) (string, error) {
	if logger != nil {
		if repo.IsRemote() {
			logger.Info("Preparing Git repository source",
				"repository_id", repo.ID,
				"repository_name", repo.Name,
				"remote_url", repo.GetRemoteURL(),
				"path", repo.Path,
			)
		} else {
			logger.Info("Preparing local repository source",
				"repository_id", repo.ID,
				"repository_name", repo.Name,
				"path", repo.Path,
			)
		}
	}

	// Create the appropriate source based on repository type
	var source Source
	if repo.IsLocal() {
		// Local repository mode - use the configured path directly
		source = NewLocalSource(repo.Path)
	} else {
		// Git repository mode - use GitSource with remote URL and branch
		// GetRemoteURL() and GetBranch() handle nil pointer safety
		source = NewGitSource(repo.GetRemoteURL(), repo.Branch, repo.Path)
	}

	// Prepare the source and get the local path
	localPath, err := source.Prepare(logger)
	if err != nil {
		return "", fmt.Errorf("failed to prepare repository %s (%s): %w",
			repo.ID, repo.Name, err)
	}

	if logger != nil {
		logger.Info("Repository prepared successfully",
			"repository_id", repo.ID,
			"repository_name", repo.Name,
			"local_path", localPath,
		)
	}

	return localPath, nil
}

// PrepareAllRepositories prepares all repositories for use by validating and preparing each one.
// It returns a slice of PreparedRepository which bundles the repository entry, local path,
// and sync results into a unified structure. This function is the main entry point for multi-repository setup.
//
// The preparation process:
// 1. Validates all repositories (checks for duplicates, validates structure)
// 2. Prepares each repository (clones if needed, validates paths)
// 3. Syncs all GitHub repositories (fetches updates for clean repos)
// 4. Logs sync results for each repository (success, failed, skipped)
//
// Parameters:
//   - repos: List of repository entries to prepare
//   - logger: Logger for structured logging (can be nil)
//
// Returns:
//   - []PreparedRepository: Slice of prepared repositories with paths and sync status
//   - error: Aggregated preparation errors, nil if all successful
//
// Usage:
//
//	prepared, err := repository.PrepareAllRepositories(cfg.Repositories, logger)
//	if err != nil {
//	    return fmt.Errorf("repository preparation failed: %w", err)
//	}
//	for _, prep := range prepared {
//	    fmt.Printf("Repository %s ready at: %s\n", prep.ID(), prep.LocalPath)
//	}
func PrepareAllRepositories(repos []RepositoryEntry, logger *logging.AppLogger) ([]PreparedRepository, error) {
	if logger != nil {
		logger.Info("Starting multi-repository preparation", "repository_count", len(repos))
	}

	// Step 1: Validate all repositories before attempting preparation
	if err := ValidateAllRepositories(repos); err != nil {
		return nil, fmt.Errorf("repository validation failed: %w", err)
	}

	// Step 2: Prepare each repository and build prepared repository list
	prepared := make([]PreparedRepository, 0, len(repos))
	var preparationErrors []string

	for _, repo := range repos {
		if logger != nil {
			logger.Info("Preparing repository",
				"repository_id", repo.ID,
				"repository_name", repo.Name,
				"is_remote", repo.IsRemote(),
			)
		}

		localPath, err := PrepareRepository(repo, logger)
		if err != nil {
			errorMsg := fmt.Sprintf("repository %s (%s): %v", repo.ID, repo.Name, err)
			preparationErrors = append(preparationErrors, errorMsg)
			if logger != nil {
				logger.Error("Repository preparation failed",
					"repository_id", repo.ID,
					"repository_name", repo.Name,
					"error", err,
				)
			}
			// Continue with other repositories instead of failing fast
			continue
		}

		// Create prepared repository with initial sync result (will be updated during sync)
		preparedRepo := PreparedRepository{
			Entry:     repo,
			LocalPath: localPath,
			SyncResult: RepositorySyncResult{
				RepositoryID:   repo.ID,
				RepositoryName: repo.Name,
				Status:         SyncStatusSkipped,
				SkipReason:     "Not synced yet",
			},
		}

		prepared = append(prepared, preparedRepo)

		if logger != nil {
			logger.Info("Repository prepared",
				"repository_id", repo.ID,
				"repository_name", repo.Name,
				"local_path", localPath,
			)
		}
	}

	// If any preparation errors occurred, return them as an aggregated error
	if len(preparationErrors) > 0 {
		return prepared, fmt.Errorf("failed to prepare %d repositories:\n  - %s",
			len(preparationErrors),
			strings.Join(preparationErrors, "\n  - "),
		)
	}

	// Step 3: Sync all GitHub repositories and update sync results
	if len(prepared) > 0 {
		if logger != nil {
			logger.Info("Starting repository synchronization")
		}

		// Get the original repository entries for syncing
		repoEntries := make([]RepositoryEntry, len(prepared))
		for i, p := range prepared {
			repoEntries[i] = p.Entry
		}

		syncResults := SyncAllRepositories(repoEntries, logger)

		// Update prepared repositories with sync results
		syncResultMap := make(map[string]RepositorySyncResult)
		for _, result := range syncResults {
			syncResultMap[result.RepositoryID] = result
		}

		for i := range prepared {
			if result, exists := syncResultMap[prepared[i].Entry.ID]; exists {
				prepared[i].SyncResult = result

				// Log sync results
				if logger != nil {
					switch result.Status {
					case SyncStatusFailed:
						logger.Warn("Repository sync failed",
							"repository_id", result.RepositoryID,
							"repository_name", result.RepositoryName,
							"error", result.Error,
							"message", result.GetMessage())
					case SyncStatusSkipped:
						logger.Info("Repository sync skipped",
							"repository_id", result.RepositoryID,
							"repository_name", result.RepositoryName,
							"reason", result.SkipReason,
							"message", result.GetMessage())
					case SyncStatusSuccess:
						logger.Debug("Repository sync succeeded",
							"repository_id", result.RepositoryID,
							"repository_name", result.RepositoryName,
							"duration", result.Duration,
							"message", result.GetMessage())
					}
				}
			}
		}
	}

	if logger != nil {
		logger.Info("Multi-repository preparation completed",
			"total_repositories", len(repos),
			"prepared_successfully", len(prepared),
		)
	}

	return prepared, nil
}
