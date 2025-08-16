package filemanager

import (
	"fmt"
	"os"
	"path/filepath"
	"rulem/internal/logging"
	"rulem/pkg/fileops"
	"strings"

	"github.com/adrg/xdg"
)

// GetDefaultStorageDir returns default storage in user's data directory
func GetDefaultStorageDir() string {
	return filepath.Join(xdg.DataHome, "rulem")
}

// CreateSecureStorageRoot creates a storage root confined to user's home directory
func CreateSecureStorageRoot(userPath string) (*os.Root, error) {
	if strings.TrimSpace(userPath) == "" {
		return nil, fmt.Errorf("storage directory cannot be empty")
	}

	// Expand the user path
	expandedPath := fileops.ExpandPath(userPath)

	// SECURITY: Ensure path is within user's home directory
	homeRoot, err := createHomeRoot()
	if err != nil {
		return nil, fmt.Errorf("cannot create home root: %w", err)
	}
	defer homeRoot.Close() // We'll create a new root for the specific path

	// Check if the path is within home by trying to resolve it relative to home
	relPath, err := fileops.ValidatePathInHome(expandedPath)
	if err != nil {
		return nil, fmt.Errorf("path must be within your home directory: %w", err)
	}

	// Check if directory already exists
	if stat, err := homeRoot.Stat(relPath); err == nil {
		// Directory exists, check if it's actually a directory
		if !stat.IsDir() {
			logging.Error("Path exists but is not a directory", "relPath", relPath)
			return nil, fmt.Errorf("path exists but is not a directory: %s", relPath)
		}
		logging.Debug("Directory already exists", "relPath", relPath)
	} else {
		// Directory doesn't exist, create it
		if err := homeRoot.Mkdir(relPath, 0755); err != nil {
			logging.Error("Failed to create directory", "relPath", relPath, "error", err)
			return nil, fmt.Errorf("cannot create directory: %w", err)
		}
	}
	logging.Debug("Directory created successfully", "relPath", relPath)

	// Test write permissions
	testFile := filepath.Join(relPath, ".rulem-test")
	logging.Debug("Testing write permissions", "testFile", testFile)
	if f, err := homeRoot.OpenFile(testFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644); err != nil {
		logging.Error("Failed to create test file for write permissions", "testFile", testFile, "error", err)
		return nil, fmt.Errorf("directory is not writable: %w", err)
	} else {
		f.Write([]byte("test"))
		f.Close()
		logging.Debug("Write permission test successful")
	}
	homeRoot.Remove(testFile) // Cleanup

	// Now create a root specifically for the target directory
	targetRoot, err := os.OpenRoot(expandedPath)
	if err != nil {
		return nil, fmt.Errorf("cannot create secure root: %w", err)
	}

	return targetRoot, nil
}

// createHomeRoot creates a root confined to the user's home directory
func createHomeRoot() (*os.Root, error) {
	homeDir := xdg.Home
	logging.Info("Created home root", "homeDir", homeDir)
	if homeDir == "" {
		return nil, fmt.Errorf("cannot determine home directory")
	}

	return os.OpenRoot(homeDir)
}
