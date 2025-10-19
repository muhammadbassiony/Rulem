package repository

import (
	"fmt"
	"strings"

	"rulem/internal/logging"
)

// Source abstracts different types of central rule repositories.
// Implementations must resolve to a local filesystem path that can be used
// by the FileManager as the storage root.
//
// The Source interface provides a unified abstraction for both local directory
// repositories and remote Git repositories. All implementations must:
//   - Validate their configuration (paths, URLs, credentials)
//   - Perform any necessary setup (Git clone, directory validation)
//   - Return an absolute local filesystem path ready for use
//   - Handle errors gracefully with user-friendly messages
//
// Implementations:
//   - LocalSource: Validates existing local directories (see local.go)
//   - GitSource: Handles Git clone/sync operations (see git.go)
//
// Usage pattern:
//
//	// Create source from repository configuration
//	var source Source
//	if repo.IsLocal() {
//	    source = NewLocalSource(repo.Path)
//	} else {
//	    source = NewGitSource(repo.GetRemoteURL(), repo.Branch, repo.Path)
//	}
//
//	// Prepare the source for use
//	localPath, err := source.Prepare(logger)
//	if err != nil {
//	    return fmt.Errorf("source preparation failed: %w", err)
//	}
//
//	// Use the local path with FileManager
//	fm := filemanager.NewFileManager(localPath, logger)
//
// Security considerations:
//   - All paths are validated through pkg/fileops security functions
//   - Git operations use credential store for secure token management
//   - Path traversal and system directory access are prevented
//   - Symlink security is enforced where applicable
type Source interface {
	// Prepare validates and prepares the source for use, returning the absolute
	// path to the local repository root.
	//
	// For LocalSource:
	//   - Validates the path exists and is a directory
	//   - Performs security checks (no traversal, no system dirs)
	//   - Returns the absolute path
	//
	// For GitSource:
	//   - Clones repository if not present (shallow clone for performance)
	//   - Updates repository if already cloned and clean
	//   - Detects uncommitted changes and skips updates to preserve work
	//   - Handles authentication via credential store
	//   - Returns the absolute path to the cloned repository
	//
	// Parameters:
	//   - logger: Structured logger for operation logging (can be nil)
	//
	// Returns:
	//   - localPath: Absolute filesystem path to the prepared repository
	//   - error: Preparation errors (validation, network, authentication, etc.)
	//
	// Error handling:
	//   - LocalSource: Returns errors for missing directories, permission issues, security violations
	//   - GitSource: Returns errors for clone failures, authentication issues, network problems
	//   - All errors include user-friendly messages with actionable guidance
	Prepare(logger *logging.AppLogger) (localPath string, err error)
}

// RepositoryType represents the type of repository storage backend.
type RepositoryType string

const (
	// RepositoryTypeLocal indicates a local directory repository
	RepositoryTypeLocal RepositoryType = "local"

	// RepositoryTypeGitHub indicates a GitHub-hosted Git repository
	RepositoryTypeGitHub RepositoryType = "github"
)

// String returns the string representation of the repository type.
func (rt RepositoryType) String() string {
	return string(rt)
}

// IsValid checks if the repository type is a valid type.
func (rt RepositoryType) IsValid() bool {
	return rt == RepositoryTypeLocal || rt == RepositoryTypeGitHub
}

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

// IsLocal returns true if this repository is a local directory repository.
func (r RepositoryEntry) IsLocal() bool {
	return r.Type == RepositoryTypeLocal
}

// GetRemoteURL returns the remote URL if this is a GitHub repository.
// Returns empty string for local repositories or if RemoteURL is nil.
func (r RepositoryEntry) GetRemoteURL() string {
	if r.RemoteURL != nil {
		return *r.RemoteURL
	}
	return ""
}

// GetBranch returns the branch name if specified, or empty string for default branch.
func (r RepositoryEntry) GetBranch() string {
	if r.Branch != nil {
		return *r.Branch
	}
	return ""
}

// String returns a string representation of the repository entry for logging.
func (r RepositoryEntry) String() string {
	if r.IsRemote() {
		return fmt.Sprintf("Repository{ID: %s, Name: %s, Type: %s, RemoteURL: %s}",
			r.ID, r.Name, r.Type, r.GetRemoteURL())
	}
	return fmt.Sprintf("Repository{ID: %s, Name: %s, Type: %s, Path: %s}",
		r.ID, r.Name, r.Type, r.Path)
}

// PreparedRepository represents a repository that has been validated and prepared for use.
// It bundles the repository metadata, local filesystem path, and synchronization results
// into a single cohesive structure. This eliminates the need for separate maps and arrays
// when working with prepared repositories.
//
// This structure is returned by PrepareAllRepositories and consumed by file scanning
// and other operations that need access to prepared repository data.
type PreparedRepository struct {
	// Entry contains the original repository configuration and metadata
	Entry RepositoryEntry

	// LocalPath is the absolute local filesystem path where the repository is accessible
	// For local repos: the validated path from Entry.Path
	// For GitHub repos: the path where the repository was cloned
	LocalPath string

	// SyncResult contains synchronization status for GitHub repositories
	// For local repos: Status will be SyncStatusSkipped with appropriate reason
	// For GitHub repos: Contains actual sync operation results
	SyncResult RepositorySyncResult
}

// ID returns the repository ID for convenience.
func (pr PreparedRepository) ID() string {
	return pr.Entry.ID
}

// Name returns the repository name for convenience.
func (pr PreparedRepository) Name() string {
	return pr.Entry.Name
}

// Type returns the repository type for convenience.
func (pr PreparedRepository) Type() RepositoryType {
	return pr.Entry.Type
}

// IsRemote returns true if this is a remote repository.
func (pr PreparedRepository) IsRemote() bool {
	return pr.Entry.IsRemote()
}

// IsLocal returns true if this is a local repository.
func (pr PreparedRepository) IsLocal() bool {
	return pr.Entry.IsLocal()
}

// WasSynced returns true if the repository was successfully synchronized.
func (pr PreparedRepository) WasSynced() bool {
	return pr.SyncResult.Status == SyncStatusSuccess
}

// HasError returns true if the repository preparation or sync failed.
func (pr PreparedRepository) HasError() bool {
	return pr.SyncResult.Status == SyncStatusFailed
}

// WasSkipped returns true if the repository sync was skipped.
func (pr PreparedRepository) WasSkipped() bool {
	return pr.SyncResult.Status == SyncStatusSkipped
}

// GetStatusMessage returns a user-friendly status message for this repository.
func (pr PreparedRepository) GetStatusMessage() string {
	return pr.SyncResult.GetMessage()
}

// String returns a string representation of the prepared repository for logging.
func (pr PreparedRepository) String() string {
	return fmt.Sprintf("PreparedRepository{ID: %s, Name: %s, LocalPath: %s, Status: %s}",
		pr.ID(), pr.Name(), pr.LocalPath, pr.SyncResult.Status.String())
}

// ValidateBasicFields validates the basic fields of a repository entry.
// This is a helper for ValidateRepositoryEntry that checks non-type-specific fields.
func (r RepositoryEntry) ValidateBasicFields() error {
	// Validate ID format: must match pattern "sanitized-name-timestamp"
	if r.ID == "" {
		return fmt.Errorf("repository ID cannot be empty")
	}

	// ID should contain at least one dash and end with digits
	parts := strings.Split(r.ID, "-")
	if len(parts) < 2 {
		return fmt.Errorf("invalid repository ID format %q (expected: name-timestamp)", r.ID)
	}

	// Last part should be numeric (timestamp)
	lastPart := parts[len(parts)-1]
	if len(lastPart) == 0 {
		return fmt.Errorf("invalid repository ID format %q (missing timestamp)", r.ID)
	}
	for _, ch := range lastPart {
		if ch < '0' || ch > '9' {
			return fmt.Errorf("invalid repository ID format %q (timestamp must be numeric)", r.ID)
		}
	}

	// Validate name
	trimmedName := strings.TrimSpace(r.Name)
	if trimmedName == "" {
		return fmt.Errorf("repository name cannot be empty")
	}
	if len(trimmedName) > 100 {
		return fmt.Errorf("repository name too long (%d characters, maximum 100)", len(trimmedName))
	}

	// Validate type
	if !r.Type.IsValid() {
		return fmt.Errorf("invalid repository type %q (must be %q or %q)",
			r.Type, RepositoryTypeLocal, RepositoryTypeGitHub)
	}

	// Validate CreatedAt timestamp
	if r.CreatedAt <= 0 {
		return fmt.Errorf("invalid created_at timestamp: %d (must be positive Unix timestamp)", r.CreatedAt)
	}

	// Validate path (required for all repository types)
	trimmedPath := strings.TrimSpace(r.Path)
	if trimmedPath == "" {
		return fmt.Errorf("repository path cannot be empty")
	}

	return nil
}

// ValidateTypeSpecificFields validates fields specific to the repository type.
// This is a helper for ValidateRepositoryEntry that checks type-specific constraints.
func (r RepositoryEntry) ValidateTypeSpecificFields() error {
	if r.Type == RepositoryTypeGitHub {
		// GitHub repositories must have a RemoteURL
		if r.RemoteURL == nil || strings.TrimSpace(*r.RemoteURL) == "" {
			return fmt.Errorf("github repository must have a remote URL")
		}

		// Branch, if provided, must be non-empty
		if r.Branch != nil && strings.TrimSpace(*r.Branch) == "" {
			return fmt.Errorf("branch cannot be empty string (use nil for default branch)")
		}

		// LastSyncTime, if provided, must be positive
		if r.LastSyncTime != nil && *r.LastSyncTime <= 0 {
			return fmt.Errorf("last_sync_time must be positive Unix timestamp, got: %d", *r.LastSyncTime)
		}
	} else if r.Type == RepositoryTypeLocal {
		// Local repositories should not have GitHub-specific fields
		if r.RemoteURL != nil && *r.RemoteURL != "" {
			return fmt.Errorf("local repository should not have a remote URL")
		}
		if r.Branch != nil && *r.Branch != "" {
			return fmt.Errorf("local repository should not have a branch")
		}
		if r.LastSyncTime != nil {
			return fmt.Errorf("local repository should not have a last_sync_time")
		}
	}

	return nil
}
