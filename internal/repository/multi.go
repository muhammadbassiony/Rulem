package repository

import (
	"fmt"
	"strings"

	"rulem/internal/logging"
)

// PrepareAllRepositories prepares all repositories for use by validating and preparing each one.
// It returns a map of repository IDs to their local filesystem paths, along with sync results
// for GitHub repositories. This function is the main entry point for multi-repository setup.
//
// The preparation process:
// 1. Validates all repositories (checks for duplicates, validates structure)
// 2. Prepares each repository (clones if needed, validates paths)
// 3. Syncs all GitHub repositories (fetches updates for clean repos)
//
// Parameters:
//   - repos: List of repository entries to prepare
//   - logger: Logger for structured logging (can be nil)
//
// Returns:
//   - map[string]string: Map of repository ID to local filesystem path
//   - []RepositorySyncResult: Sync results for each repository (empty for local repos)
//   - error: Aggregated preparation errors, nil if all successful
//
// Usage:
//
//	paths, syncResults, err := repository.PrepareAllRepositories(cfg.Repositories, logger)
//	if err != nil {
//	    return fmt.Errorf("repository preparation failed: %w", err)
//	}
//	for id, path := range paths {
//	    fmt.Printf("Repository %s ready at: %s\n", id, path)
//	}
func PrepareAllRepositories(repos []RepositoryEntry, logger *logging.AppLogger) (map[string]string, []RepositorySyncResult, error) {
	if logger != nil {
		logger.Info("Starting multi-repository preparation", "repository_count", len(repos))
	}

	// Step 1: Validate all repositories before attempting preparation
	if err := ValidateAllRepositories(repos); err != nil {
		return nil, nil, fmt.Errorf("repository validation failed: %w", err)
	}

	// Step 2: Prepare each repository and build path map
	pathMap := make(map[string]string, len(repos))
	var preparationErrors []string

	for _, repo := range repos {
		if logger != nil {
			logger.Info("Preparing repository",
				"repository_id", repo.ID,
				"repository_name", repo.Name,
				"is_remote", repo.IsRemote(),
			)
		}

		localPath, syncInfo, err := PrepareRepository(repo, logger)
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

		// Store the local path in the map
		pathMap[repo.ID] = localPath

		if logger != nil && syncInfo.Message != "" {
			logger.Info("Repository prepared",
				"repository_id", repo.ID,
				"repository_name", repo.Name,
				"local_path", localPath,
				"sync_message", syncInfo.Message,
			)
		}
	}

	// If any preparation errors occurred, return them as an aggregated error
	if len(preparationErrors) > 0 {
		return pathMap, nil, fmt.Errorf("failed to prepare %d repositories:\n  - %s",
			len(preparationErrors),
			strings.Join(preparationErrors, "\n  - "),
		)
	}

	// Step 3: Sync all GitHub repositories
	var syncResults []RepositorySyncResult
	if len(repos) > 0 {
		if logger != nil {
			logger.Info("Starting repository synchronization")
		}
		syncResults = SyncAllRepositories(repos, logger)
	}

	if logger != nil {
		logger.Info("Multi-repository preparation completed",
			"total_repositories", len(repos),
			"prepared_successfully", len(pathMap),
		)
	}

	return pathMap, syncResults, nil
}

// ValidateAllRepositories validates a list of repository entries for structural correctness
// and uniqueness constraints. This should be called before attempting to prepare repositories.
//
// Validation checks:
// - No duplicate repository IDs
// - No duplicate repository names (case-insensitive)
// - Each repository entry is structurally valid
//
// Parameters:
//   - repos: List of repository entries to validate
//
// Returns:
//   - error: Detailed validation error, nil if all valid
//
// Usage:
//
//	if err := repository.ValidateAllRepositories(cfg.Repositories); err != nil {
//	    return fmt.Errorf("invalid repository configuration: %w", err)
//	}
func ValidateAllRepositories(repos []RepositoryEntry) error {
	if len(repos) == 0 {
		// Empty repository list is valid (user hasn't configured any yet)
		return nil
	}

	// Check for duplicate IDs
	seenIDs := make(map[string]string, len(repos)) // ID -> Name (for error messages)
	for _, repo := range repos {
		if existingName, exists := seenIDs[repo.ID]; exists {
			return fmt.Errorf("duplicate repository ID %q found in repositories %q and %q",
				repo.ID, existingName, repo.Name)
		}
		seenIDs[repo.ID] = repo.Name
	}

	// Check for duplicate names (case-insensitive)
	seenNames := make(map[string]string, len(repos)) // LowercaseName -> OriginalName
	for _, repo := range repos {
		lowerName := strings.ToLower(strings.TrimSpace(repo.Name))
		if existingName, exists := seenNames[lowerName]; exists {
			return fmt.Errorf("duplicate repository name found: %q and %q (names must be unique, case-insensitive)",
				existingName, repo.Name)
		}
		seenNames[lowerName] = repo.Name
	}

	// Validate each repository entry
	var validationErrors []string
	for i, repo := range repos {
		if err := ValidateRepositoryEntry(repo); err != nil {
			validationErrors = append(validationErrors,
				fmt.Sprintf("repository[%d] (%s): %v", i, repo.Name, err))
		}
	}

	if len(validationErrors) > 0 {
		return fmt.Errorf("repository validation failed:\n  - %s",
			strings.Join(validationErrors, "\n  - "))
	}

	return nil
}
