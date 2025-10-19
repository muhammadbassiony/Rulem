package repository

import (
	"fmt"
	"path/filepath"
	"strings"

	"rulem/internal/logging"

	"github.com/adrg/xdg"
)

// RepositoryType represents the type of repository storage backend.
type RepositoryType string

const (
	// RepositoryTypeLocal indicates a local directory repository
	RepositoryTypeLocal RepositoryType = "local"

	// RepositoryTypeGitHub indicates a GitHub-hosted Git repository
	RepositoryTypeGitHub RepositoryType = "github"
)

// RepositoryEntry represents a single configured repository.
// This is the domain entity for repositories - it belongs in the repository package
// as it represents repository concepts, not configuration persistence concerns.
//
// Fields:
//   - ID: Unique identifier in format "sanitized-name-timestamp" (e.g., "personal-rules-1728756432")
//   - Name: User-provided display name for UI (e.g., "Personal Rules")
//   - Type: Repository type ("local" or "github")
//   - CreatedAt: Unix timestamp when repository was added (used for ordering and ID generation)
//   - Path: Local filesystem path (for local repos) or clone path (for GitHub repos)
//   - RemoteURL: GitHub repository URL (only for Type == RepositoryTypeGitHub)
//   - Branch: Git branch name (optional, only for GitHub repos)
//   - LastSyncTime: Unix timestamp of last sync (only for GitHub repos)
type RepositoryEntry struct {
	// Identity fields
	ID        string         `yaml:"id"`         // Unique identifier (e.g., "personal-rules-1728756432")
	Name      string         `yaml:"name"`       // User-provided display name
	Type      RepositoryType `yaml:"type"`       // Repository type ("local" or "github")
	CreatedAt int64          `yaml:"created_at"` // Unix timestamp (for ordering and ID generation)

	// Location
	Path string `yaml:"path"` // Local path for local repos, clone path for GitHub repos

	// Git-specific fields (only used when Type == RepositoryTypeGitHub)
	RemoteURL    *string `yaml:"remote_url,omitempty"`     // GitHub repository URL
	Branch       *string `yaml:"branch,omitempty"`         // Git branch (optional)
	LastSyncTime *int64  `yaml:"last_sync_time,omitempty"` // Last sync timestamp
}

// IsRemote returns true if this repository is a remote Git repository.
func (r RepositoryEntry) IsRemote() bool {
	return r.Type == RepositoryTypeGitHub
}

// GetDefaultStorageDir returns the default storage directory in the user's data directory.
// This function was moved from internal/filemanager/storagedir.go to avoid circular dependencies
// when the repository package needs to reference the default storage location.
//
// Returns:
//   - string: Absolute path to the default storage directory (e.g., ~/.local/share/rulem)
func GetDefaultStorageDir() string {
	return filepath.Join(xdg.DataHome, "rulem")
}

// PrepareRepository creates the appropriate source and prepares it for use.
// This function examines the repository entry to determine whether to use LocalSource or GitSource,
// then prepares the source and returns the local path and sync information.
//
// Parameters:
//   - repo: Repository entry with type and configuration
//   - logger: Logger for structured logging during preparation
//
// Returns:
//   - string: Absolute path to the prepared local repository
//   - SyncInfo: Information about sync operations for UI display
//   - error: Preparation errors (validation, network, authentication, etc.)
//
// Usage:
//
//	localPath, syncInfo, err := repository.PrepareRepository(repo, logger)
//	if err != nil {
//	    return fmt.Errorf("repository preparation failed: %w", err)
//	}
//	if syncInfo.Message != "" {
//	    logger.Info("Repository status", "message", syncInfo.Message)
//	}
//	fm := filemanager.NewFileManager(localPath, logger)
func PrepareRepository(repo RepositoryEntry, logger *logging.AppLogger) (string, SyncInfo, error) {
	if logger != nil {
		if repo.IsRemote() {
			logger.Info("Preparing Git repository source", "remoteURL", *repo.RemoteURL, "path", repo.Path)
		} else {
			logger.Info("Preparing local repository source", "path", repo.Path)
		}
	}

	var source Source
	if !repo.IsRemote() {
		// Local repository mode - use the configured path directly
		source = NewLocalSource(repo.Path)
	} else {
		// Git repository mode - use GitSource with PAT authentication
		source = NewGitSource(*repo.RemoteURL, repo.Branch, repo.Path)
	}

	return source.Prepare(logger)
}

// ValidateRepositoryEntry validates a single repository entry for structural correctness.
// This function is used by ValidateAllRepositories to check individual repository entries.
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
func ValidateRepositoryEntry(repo RepositoryEntry) error {
	// Validate ID format: must match pattern "sanitized-name-timestamp"
	if repo.ID == "" {
		return fmt.Errorf("repository ID cannot be empty")
	}

	// ID should contain at least one dash and end with digits
	parts := strings.Split(repo.ID, "-")
	if len(parts) < 2 {
		return fmt.Errorf("invalid repository ID format %q (expected: name-timestamp)", repo.ID)
	}

	// Last part should be numeric (timestamp)
	lastPart := parts[len(parts)-1]
	if len(lastPart) == 0 {
		return fmt.Errorf("invalid repository ID format %q (missing timestamp)", repo.ID)
	}
	for _, ch := range lastPart {
		if ch < '0' || ch > '9' {
			return fmt.Errorf("invalid repository ID format %q (timestamp must be numeric)", repo.ID)
		}
	}

	// Validate name
	trimmedName := strings.TrimSpace(repo.Name)
	if trimmedName == "" {
		return fmt.Errorf("repository name cannot be empty")
	}
	if len(trimmedName) > 100 {
		return fmt.Errorf("repository name too long (%d characters, maximum 100)", len(trimmedName))
	}

	// Validate type
	if repo.Type != RepositoryTypeLocal && repo.Type != RepositoryTypeGitHub {
		return fmt.Errorf("invalid repository type %q (must be %q or %q)", repo.Type, RepositoryTypeLocal, RepositoryTypeGitHub)
	}

	// Validate CreatedAt timestamp
	if repo.CreatedAt <= 0 {
		return fmt.Errorf("invalid created_at timestamp: %d (must be positive Unix timestamp)", repo.CreatedAt)
	}

	// Validate path (required for all repository types)
	trimmedPath := strings.TrimSpace(repo.Path)
	if trimmedPath == "" {
		return fmt.Errorf("repository path cannot be empty")
	}

	// Type-specific validation
	if repo.Type == RepositoryTypeGitHub {
		// GitHub repositories must have a RemoteURL
		if repo.RemoteURL == nil || strings.TrimSpace(*repo.RemoteURL) == "" {
			return fmt.Errorf("github repository must have a remote URL")
		}

		// Branch, if provided, must be non-empty
		if repo.Branch != nil && strings.TrimSpace(*repo.Branch) == "" {
			return fmt.Errorf("branch cannot be empty string (use nil for default branch)")
		}

		// LastSyncTime, if provided, must be positive
		if repo.LastSyncTime != nil && *repo.LastSyncTime <= 0 {
			return fmt.Errorf("last_sync_time must be positive Unix timestamp, got: %d", *repo.LastSyncTime)
		}
	} else if repo.Type == RepositoryTypeLocal {
		// Local repositories should not have GitHub-specific fields
		if repo.RemoteURL != nil && *repo.RemoteURL != "" {
			return fmt.Errorf("local repository should not have a remote URL")
		}
		if repo.Branch != nil && *repo.Branch != "" {
			return fmt.Errorf("local repository should not have a branch")
		}
	}

	return nil
}
