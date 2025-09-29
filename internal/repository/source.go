package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"rulem/internal/logging"
	"rulem/pkg/fileops"
)

// Source abstracts different types of central rule repositories.
// Implementations must resolve to a local filesystem path that can be used
// by the FileManager as the storage root.
type Source interface {
	// Prepare validates and prepares the source for use, returning:
	// - localPath: absolute path to the local repository root
	// - info: sync details/messages for UI consumption
	// - err: non-nil if resolution fails
	Prepare(logger *logging.AppLogger) (localPath string, info SyncInfo, err error)
}

// SyncInfo contains details about repository synchronization for UI messaging.
type SyncInfo struct {
	Cloned  bool   // true if a clone occurred during Prepare()
	Updated bool   // true if a fetch/pull occurred during Prepare()
	Dirty   bool   // true if local working tree has uncommitted changes
	Message string // UI-friendly message about sync status/results
}

// LocalSource represents a local directory used directly as the central repository.
// No network operations are performed; it only validates the configured path.
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
//	localPath, syncInfo, err := source.Prepare(logger)
func NewLocalSource(path string) LocalSource {
	return LocalSource{
		Path: path,
	}
}

// Prepare validates the local path and returns it for use as the storage root.
// Validation performed:
// - Non-empty path
// - Expand "~/" to the user's home
// - Security validation (prevents traversal, reserved/system dirs) via fileops.ValidateStoragePath
// - Directory must exist and be a directory
func (ls LocalSource) Prepare(logger *logging.AppLogger) (string, SyncInfo, error) {
	if logger != nil {
		logger.Info("Preparing local repository source", "path", ls.Path)
	}

	trimmed := strings.TrimSpace(ls.Path)
	if trimmed == "" {
		return "", SyncInfo{}, fmt.Errorf("local source path cannot be empty")
	}

	// Expand "~/" and clean the path
	expanded := fileops.ExpandPath(trimmed)
	clean := filepath.Clean(expanded)

	// Security and structural validation
	if err := fileops.ValidateStoragePath(clean); err != nil {
		return "", SyncInfo{}, fmt.Errorf("invalid local source path: %w", err)
	}

	// Must exist and be a directory
	info, err := os.Stat(clean)
	if err != nil {
		if os.IsNotExist(err) {
			return "", SyncInfo{}, fmt.Errorf("local source directory does not exist: %s", clean)
		}
		return "", SyncInfo{}, fmt.Errorf("cannot access local source directory: %w", err)
	}
	if !info.IsDir() {
		return "", SyncInfo{}, fmt.Errorf("local source path is not a directory: %s", clean)
	}

	// Ensure absolute path (ValidateStoragePath enforces absolute or "~/", and we expanded "~")
	abs, err := filepath.Abs(clean)
	if err != nil {
		// Fallback to the cleaned path if Abs resolution fails
		abs = clean
	}

	if logger != nil {
		logger.Debug("Local repository source validated", "resolved_path", abs)
	}

	// No syncing is performed for LocalSource; return informational message only
	return abs, SyncInfo{
		Cloned:  false,
		Updated: false,
		Dirty:   false,
		Message: "Using local repository",
	}, nil
}
