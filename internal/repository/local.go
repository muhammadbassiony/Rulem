package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"rulem/internal/logging"
	"rulem/pkg/fileops"
)

// LocalSource represents a local directory used directly as the central repository.
// No network operations are performed; it only validates the configured path.
//
// LocalSource is used when a repository is configured with Type == RepositoryTypeLocal.
// It validates that the specified directory exists, is accessible, and meets security
// requirements before being used as a rule repository.
type LocalSource struct {
	// Path is the local directory for the central repository.
	// It should be an absolute path or home-relative (~/...).
	Path string
}

// NewLocalSource creates a new LocalSource instance with the specified path.
// This constructor provides a consistent way to create LocalSource instances
// and can be extended in the future for validation or default setting.
//
// Parameters:
//   - path: Local directory path for the repository
//
// Returns:
//   - LocalSource: Configured local source instance
//
// Usage:
//
//	source := repository.NewLocalSource("/home/user/rules")
//	localPath, err := source.Prepare(logger)
//	if err != nil {
//	    return fmt.Errorf("local source preparation failed: %w", err)
//	}
func NewLocalSource(path string) LocalSource {
	return LocalSource{
		Path: path,
	}
}

// Prepare validates the local path and returns it for use as the storage root.
// This method implements the Source interface for local directory repositories.
//
// Validation performed:
//   - Non-empty path
//   - Expand "~/" to the user's home directory
//   - Security validation (prevents traversal, reserved/system dirs) via fileops.ValidateStoragePath
//   - Directory must exist and be a directory
//   - Returns absolute path for consistent usage
//
// No network operations are performed. The directory must already exist - this method
// does not create directories. For directory creation during config setup, use
// EnsureLocalStorageDirectory instead.
//
// Parameters:
//   - logger: Logger for structured logging (can be nil for silent operation)
//
// Returns:
//   - string: Absolute path to the validated local repository
//   - error: Validation errors (empty path, security violations, missing directory, etc.)
//
// Usage:
//
//	ls := repository.NewLocalSource("~/my-rules")
//	localPath, err := ls.Prepare(logger)
//	if err != nil {
//	    return fmt.Errorf("failed to prepare local repository: %w", err)
//	}
//	// localPath is now ready for use with FileManager
func (ls LocalSource) Prepare(logger *logging.AppLogger) (string, error) {
	if logger != nil {
		logger.Info("Preparing local repository source", "path", ls.Path)
	}

	// Validate path is not empty
	trimmed := strings.TrimSpace(ls.Path)
	if trimmed == "" {
		return "", fmt.Errorf("local source path cannot be empty")
	}

	// Expand "~/" and clean the path
	expanded := fileops.ExpandPath(trimmed)
	clean := filepath.Clean(expanded)

	// Security and structural validation
	// This checks for:
	// - Path traversal attempts (../)
	// - System/reserved directories (/etc, /sys, /proc, etc.)
	// - Absolute path requirement
	// - Symlink security
	if err := fileops.ValidateStoragePath(clean); err != nil {
		return "", fmt.Errorf("invalid local source path: %w", err)
	}

	// Verify the directory exists
	info, err := os.Stat(clean)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("local source directory does not exist: %s", clean)
		}
		return "", fmt.Errorf("cannot access local source directory: %w", err)
	}

	// Verify it's actually a directory (not a file)
	if !info.IsDir() {
		return "", fmt.Errorf("local source path is not a directory: %s", clean)
	}

	// Convert to absolute path for consistency
	// ValidateStoragePath enforces absolute or "~/", and we expanded "~"
	abs, err := filepath.Abs(clean)
	if err != nil {
		// If Abs fails for some reason, use the cleaned path
		// (which should already be absolute after expansion and validation)
		abs = clean
	}

	if logger != nil {
		logger.Debug("Local repository source validated", "resolved_path", abs)
	}

	return abs, nil
}

// ValidatePath performs validation on the configured path without accessing the filesystem.
// This is useful for pre-flight checks before attempting preparation.
//
// Returns:
//   - error: Validation error if path is invalid (empty, contains null bytes, etc.)
func (ls LocalSource) ValidatePath() error {
	return ValidateRepositoryPath(ls.Path)
}

// String returns a string representation of the LocalSource for logging and debugging.
func (ls LocalSource) String() string {
	return fmt.Sprintf("LocalSource{Path: %s}", ls.Path)
}
