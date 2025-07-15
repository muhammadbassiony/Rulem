package filemanager

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// GetDefaultStorageDir returns the default storage directory based on the platform
// and environment variables.
// It uses XDG standards for Linux and macOS, and APPDATA for Windows.
// This is used to store rulemig's data files.
func GetDefaultStorageDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home is unavailable
		home, _ = os.Getwd()
	}

	var path string
	switch runtime.GOOS {
	case "darwin": // macOS
		if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" && isValidPath(xdgData) {
			return filepath.Join(xdgData, "rulemig")
		}
		return filepath.Join(home, ".local", "share", "rulemig")

	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" && isValidPath(appData) {
			return filepath.Join(appData, "rulemig", "data")
		}
		return filepath.Join(home, "AppData", "Roaming", "rulemig", "data")

	case "linux":
		if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" && isValidPath(xdgData) {
			return filepath.Join(xdgData, "rulemig")
		}
		return filepath.Join(home, ".local", "share", "rulemig")

	default:
		// Fallback for other Unix-like systems
		return filepath.Join(home, ".local", "share", "rulemig")
	}

	return path
}

// ExpandPath is a utility function that expands a path that starts with "~/" to the user's home directory.
func ExpandPath(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path // Return original path if home directory unavailable
	}

	return filepath.Join(home, path[2:])
}

// ValidateStorageDir checks if the provided storage directory path is valid
// without creating it. This is pure validation with no side effects.
func ValidateStorageDir(input string) error {
	path := strings.TrimSpace(input)
	if path == "" {
		return fmt.Errorf("storage directory cannot be empty")
	}

	expandedPath := ExpandPath(path)

	// Security check: prevent path traversal
	if strings.Contains(expandedPath, "..") {
		return fmt.Errorf("path traversal not allowed")
	}

	// Check if it's an absolute path or relative to home
	if !filepath.IsAbs(expandedPath) && !strings.HasPrefix(path, "~/") {
		return fmt.Errorf("path must be absolute or relative to home directory (~)")
	}

	// Check for reserved directories (basic check)
	if isReservedDirectory(expandedPath) {
		return fmt.Errorf("cannot use system or reserved directories")
	}

	// Check if parent directory exists and is accessible
	parentDir := filepath.Dir(expandedPath)
	if parentDir != "." {
		if _, err := os.Stat(parentDir); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("parent directory does not exist: %s", parentDir)
			}
			return fmt.Errorf("cannot access parent directory: %w", err)
		}
	}

	return nil
}

// TestStorageDir tests if we can write to the storage directory.
// This function has side effects and should be called after ValidateStorageDir.
func TestStorageDir(path string) error {
	expandedPath := ExpandPath(strings.TrimSpace(path))

	// Create directory if it doesn't exist
	if err := os.MkdirAll(expandedPath, 0755); err != nil {
		return fmt.Errorf("cannot create directory: %w", err)
	}

	// Test write permissions
	testFile := filepath.Join(expandedPath, ".rulemig-test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("no write permission in directory: %w", err)
	}

	// Clean up test file
	if err := os.Remove(testFile); err != nil {
		// Log but don't fail - the directory is usable
		fmt.Printf("Warning: could not remove test file: %v\n", err)
	}

	return nil
}

// CreateStorageDir creates the storage directory if it doesn't exist.
func CreateStorageDir(storageDir string) error {
	expandedPath := ExpandPath(strings.TrimSpace(storageDir))

	// Validate the path first
	if err := ValidateStorageDir(expandedPath); err != nil {
		return fmt.Errorf("invalid storage directory: %w", err)
	}

	// test if we can write to the directory
	if err := TestStorageDir(expandedPath); err != nil {
		return fmt.Errorf("storage directory is not writable: %w", err)
	}

	if err := os.MkdirAll(expandedPath, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	return nil
}

// Helper functions

// isValidPath performs basic validation on environment variable paths
func isValidPath(path string) bool {
	if path == "" {
		return false
	}

	// Check for obvious invalid characters (basic check)
	if strings.ContainsAny(path, "<>|*?\"") {
		return false
	}

	// Must be absolute path
	return filepath.IsAbs(path)
}

// isReservedDirectory checks if the path is a system or reserved directory
func isReservedDirectory(path string) bool {
	// Convert to absolute path for comparison
	absPath, err := filepath.Abs(path)
	if err != nil {
		return true // If we can't resolve it, treat as reserved
	}

	// Common system directories to avoid
	reservedDirs := []string{
		"/",
		"/bin",
		"/boot",
		"/dev",
		"/etc",
		"/lib",
		"/proc",
		"/root",
		"/sbin",
		"/sys",
		"/usr",
		"/var",
	}

	// Windows system directories
	if runtime.GOOS == "windows" {
		windowsReserved := []string{
			"C:\\Windows",
			"C:\\Program Files",
			"C:\\Program Files (x86)",
			"C:\\System32",
		}
		reservedDirs = append(reservedDirs, windowsReserved...)
	}

	for _, reserved := range reservedDirs {
		if strings.EqualFold(absPath, reserved) || strings.HasPrefix(strings.ToLower(absPath), strings.ToLower(reserved)+string(os.PathSeparator)) {
			return true
		}
	}

	return false
}
