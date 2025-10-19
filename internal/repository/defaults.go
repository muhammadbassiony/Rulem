package repository

import (
	"path/filepath"

	"github.com/adrg/xdg"
)

// GetDefaultStorageDir returns the default storage directory in the user's data directory.
// This function provides a consistent default location for rule repositories across all
// platforms, following XDG Base Directory specification on Linux and equivalent conventions
// on other operating systems.
//
// The default directory is typically:
//   - Linux: ~/.local/share/rulem
//   - macOS: ~/Library/Application Support/rulem
//   - Windows: %LOCALAPPDATA%\rulem
//
// This function was moved from internal/filemanager/storagedir.go to avoid circular dependencies
// when the repository package needs to reference the default storage location.
//
// Returns:
//   - string: Absolute path to the default storage directory
//
// Usage:
//
//	defaultDir := repository.GetDefaultStorageDir()
//	fmt.Printf("Default storage: %s\n", defaultDir)
//
// Note: This function only returns the path - it does not create the directory.
// Use EnsureLocalStorageDirectory() if you need to create and validate the directory.
func GetDefaultStorageDir() string {
	return filepath.Join(xdg.DataHome, "rulem")
}

// GetDefaultGitClonePath returns the default path for cloning a Git repository.
// This generates a path within the default storage directory based on the repository name.
//
// The path format is: <DefaultStorageDir>/<repoName>
// For example:
//   - https://github.com/user/rules.git -> ~/.local/share/rulem/rules
//   - git@github.com:org/policies.git -> ~/.local/share/rulem/policies
//
// Parameters:
//   - repoName: Name of the repository (extracted from URL)
//
// Returns:
//   - string: Absolute path where the repository should be cloned
//
// Usage:
//
//	clonePath := repository.GetDefaultGitClonePath("my-rules")
//	// Returns: ~/.local/share/rulem/my-rules (on Linux)
func GetDefaultGitClonePath(repoName string) string {
	return filepath.Join(GetDefaultStorageDir(), repoName)
}
