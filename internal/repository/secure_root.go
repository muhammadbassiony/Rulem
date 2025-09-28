package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"rulem/internal/logging"
	"rulem/pkg/fileops"
	"strings"

	"github.com/adrg/xdg"
)

// EnsureLocalStorageDirectory creates and validates a local storage directory for rulem's central repository.
// This function is specifically designed for local repository setup scenarios (not Git repositories).
//
// The function performs comprehensive validation and setup:
//   - Validates path is within user's home directory (security boundary)
//   - Creates directory structure if it doesn't exist (mkdir -p behavior)
//   - Verifies the path is actually a directory (not a file)
//   - Tests write permissions by creating a temporary test file
//   - Returns a secure os.Root confined to the validated directory
//
// Security guarantees:
//   - All paths must be within user's home directory (prevents system directory access)
//   - Uses os.Root for confined filesystem operations
//   - Validates directory permissions before use
//   - Prevents path traversal attacks through home directory validation
//
// Usage scenarios:
//   - Initial config setup (CreateNewConfig)
//   - User updates central repository path (UpdateCentralPath)
//   - Local repository preparation (not Git cloning)
//
// Parameters:
//   - userPath: Path to the desired storage directory (can be relative to home with ~/)
//
// Returns:
//   - *os.Root: Secure root confined to the validated directory
//   - error: Validation, creation, or permission errors
//
// Example:
//
//	root, err := EnsureLocalStorageDirectory("~/Documents/rulem-rules")
//	if err != nil {
//	    return fmt.Errorf("failed to setup storage: %w", err)
//	}
//	defer root.Close()
func EnsureLocalStorageDirectory(userPath string) (*os.Root, error) {
	if strings.TrimSpace(userPath) == "" {
		return nil, fmt.Errorf("local storage directory path cannot be empty")
	}

	// Expand ~/ and other user path shortcuts
	expandedPath := fileops.ExpandPath(userPath)

	// SECURITY: Ensure path is within user's home directory to prevent system access
	homeRoot, err := createSecureHomeRoot()
	if err != nil {
		return nil, fmt.Errorf("cannot establish secure home boundary: %w", err)
	}
	defer homeRoot.Close() // Will create a new root for the specific validated path

	// Validate the expanded path is within home directory bounds
	relPath, err := fileops.ValidatePathInHome(expandedPath)
	if err != nil {
		return nil, fmt.Errorf("storage path must be within your home directory for security: %w", err)
	}

	// Check if target directory already exists
	if stat, err := homeRoot.Stat(relPath); err == nil {
		// Path exists - verify it's actually a directory
		if !stat.IsDir() {
			logging.Error("Storage path exists but is not a directory", "relPath", relPath)
			return nil, fmt.Errorf("storage path exists but is not a directory: %s", relPath)
		}
		logging.Debug("Local storage directory already exists", "relPath", relPath)
	} else {
		// Directory doesn't exist - create it with proper permissions
		if err := homeRoot.Mkdir(relPath, 0755); err != nil {
			logging.Error("Failed to create local storage directory", "relPath", relPath, "error", err)
			return nil, fmt.Errorf("cannot create local storage directory: %w", err)
		}
		logging.Info("Created local storage directory", "relPath", relPath)
	}

	// Test write permissions by creating a temporary test file
	testFile := filepath.Join(relPath, ".rulem-write-test")
	logging.Debug("Testing write permissions for storage directory", "testFile", testFile)
	if f, err := homeRoot.OpenFile(testFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644); err != nil {
		logging.Error("Storage directory is not writable", "testFile", testFile, "error", err)
		return nil, fmt.Errorf("local storage directory is not writable: %w", err)
	} else {
		f.Write([]byte("rulem write permission test"))
		f.Close()
		logging.Debug("Write permission test successful for storage directory")
	}
	homeRoot.Remove(testFile) // Cleanup

	// Create a secure root confined to the validated storage directory
	targetRoot, err := os.OpenRoot(expandedPath)
	if err != nil {
		return nil, fmt.Errorf("cannot create secure root for storage directory: %w", err)
	}

	logging.Info("Local storage directory ready", "path", expandedPath)

	return targetRoot, nil
}

// createSecureHomeRoot creates a secure root confined to the user's home directory.
// This establishes the security boundary for all storage directory operations.
//
// Returns:
//   - *os.Root: Secure root confined to user's home directory
//   - error: Home directory resolution or security setup errors
func createSecureHomeRoot() (*os.Root, error) {
	homeDir := xdg.Home
	if homeDir == "" {
		return nil, fmt.Errorf("cannot determine user home directory")
	}

	logging.Debug("Establishing secure home directory boundary", "homeDir", homeDir)
	return os.OpenRoot(homeDir)
}
