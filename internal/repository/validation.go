package repository

import (
	"fmt"
	"strings"
)

// ValidateRepositoryEntry validates a single repository entry for structural correctness.
// This function checks both basic fields (ID, name, type, timestamps) and type-specific
// requirements (e.g., GitHub repos must have RemoteURL).
//
// Validation checks:
//   - ID format matches pattern "sanitized-name-timestamp"
//   - Name is non-empty and within length limits (1-100 characters)
//   - Type is valid ("local" or "github")
//   - CreatedAt is a positive Unix timestamp
//   - Path is non-empty
//   - Type-specific fields are validated (e.g., GitHub repos must have RemoteURL)
//
// Parameters:
//   - repo: Repository entry to validate
//
// Returns:
//   - error: Validation error with detailed description, nil if valid
//
// Usage:
//
//	if err := repository.ValidateRepositoryEntry(repo); err != nil {
//	    return fmt.Errorf("invalid repository: %w", err)
//	}
func ValidateRepositoryEntry(repo RepositoryEntry) error {
	// Validate basic fields (ID, name, type, timestamps, path)
	if err := repo.ValidateBasicFields(); err != nil {
		return err
	}

	// Validate type-specific fields (RemoteURL for GitHub, no Git fields for local)
	if err := repo.ValidateTypeSpecificFields(); err != nil {
		return err
	}

	return nil
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

// ValidateRepositoryName validates a repository name for use in configuration.
// This is a convenience function for UI validation before creating repository entries.
//
// Validation rules:
//   - Non-empty after trimming whitespace
//   - Maximum 100 characters
//   - No control characters or special characters that could cause issues
//
// Parameters:
//   - name: Repository name to validate
//
// Returns:
//   - error: Validation error, nil if valid
func ValidateRepositoryName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("repository name cannot be empty")
	}
	if len(trimmed) > 100 {
		return fmt.Errorf("repository name too long (%d characters, maximum 100)", len(trimmed))
	}

	// Check for control characters
	for _, ch := range trimmed {
		if ch < 32 || ch == 127 {
			return fmt.Errorf("repository name contains invalid control characters")
		}
	}

	return nil
}

// ValidateRepositoryPath validates a repository path for basic correctness.
// This is a lightweight check that doesn't access the filesystem.
// For full validation including filesystem checks, use LocalSource.Prepare() or PrepareRepository().
//
// Validation rules:
//   - Non-empty after trimming whitespace
//   - No null bytes (filesystem security)
//
// Parameters:
//   - path: Repository path to validate
//
// Returns:
//   - error: Validation error, nil if valid
func ValidateRepositoryPath(path string) error {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return fmt.Errorf("repository path cannot be empty")
	}

	// Check for null bytes (filesystem security)
	if strings.Contains(trimmed, "\x00") {
		return fmt.Errorf("repository path contains null bytes")
	}

	return nil
}
