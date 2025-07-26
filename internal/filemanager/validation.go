package filemanager

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"rulem/internal/logging"
)

// ValidateStorageDir checks if the provided storage directory path is valid
// without creating it. This is pure validation with no side effects.
func ValidateStorageDir(input string) error {
	path := strings.TrimSpace(input)
	if path == "" {
		return fmt.Errorf("storage directory cannot be empty")
	}

	// FIRST: Check for path traversal in raw input (before expansion)
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal not allowed")
	}

	expandedPath := ExpandPath(path)

	// SECOND: Check for path traversal in expanded path (after cleaning)
	if strings.Contains(filepath.Clean(expandedPath), "..") {
		return fmt.Errorf("path traversal not allowed")
	}

	// THIRD: Check if it's an absolute path or relative to home
	if !filepath.IsAbs(expandedPath) && !strings.HasPrefix(path, "~/") {
		return fmt.Errorf("path must be absolute or relative to home directory (~)")
	}

	// FOURTH: Check for symlink security (before reserved directory checks)
	if err := validateSymlinkSecurity(expandedPath); err != nil {
		return err
	}

	// FIFTH: Check for reserved directories (after symlink checks)
	if isReservedDirectory(expandedPath) {
		return fmt.Errorf("cannot use system or reserved directories")
	}

	// FIFTH: Check if parent directory exists and is accessible
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

// TestWriteToStorageDir tests if we can write to the storage directory.
// This function has side effects and should be called after ValidateStorageDir.
func TestWriteToStorageDir(path string) error {
	expandedPath := ExpandPath(strings.TrimSpace(path))

	// Create directory if it doesn't exist
	if err := os.MkdirAll(expandedPath, 0755); err != nil {
		return fmt.Errorf("cannot create directory: %w", err)
	}

	// Test write permissions
	testFile := filepath.Join(expandedPath, ".rulem-test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("no write permission in directory: %w", err)
	}

	// Clean up test file
	if err := os.Remove(testFile); err != nil {
		// Log but don't fail - the directory is usable
		logging.Warn("Warning: could not remove test file", "error", err)
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
	if err := TestWriteToStorageDir(expandedPath); err != nil {
		return fmt.Errorf("storage directory is not writable: %w", err)
	}

	if err := os.MkdirAll(expandedPath, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	return nil
}

//Reserved directories helpers

// isReservedDirectory checks if the path is a system or reserved directory
func isReservedDirectory(path string) bool {
	// Convert to absolute path for comparison
	absPath, err := filepath.Abs(path)
	if err != nil {
		return true // If we can't resolve it, treat as reserved
	}
	absPath = filepath.Clean(absPath)

	// Resolve any symlinks in the path for comparison
	resolvedPath, err := filepath.EvalSymlinks(absPath)
	if err == nil {
		absPath = resolvedPath // Use resolved path if available
	}

	// Always treat root as reserved
	if absPath == "/" || absPath == "\\" || absPath == "C:\\" {
		return true
	}

	absPath = filepath.Clean(absPath)
	reservedDirs := getReservedDirectories()

	for _, reserved := range reservedDirs {
		// Canonicalize the reserved directory
		reservedAbs, err := filepath.Abs(reserved)
		if err != nil {
			continue
		}
		resolvedReserved, err := filepath.EvalSymlinks(reservedAbs)
		if err == nil {
			reservedAbs = filepath.Clean(resolvedReserved)
		} else {
			reservedAbs = filepath.Clean(reservedAbs)
		}

		// Exact match
		if strings.EqualFold(absPath, reservedAbs) {
			return true
		}

		// Child directory match - but with exceptions
		reservedPrefix := strings.ToLower(reserved) + string(os.PathSeparator)
		pathLower := strings.ToLower(absPath)

		if strings.HasPrefix(pathLower, reservedPrefix) {
			// Exception: Allow user temp directories
			if isUserTempDirectory(absPath) {
				continue
			}
			return true
		}
	}

	return false
}

// getReservedDirectories returns platform-specific reserved directories
func getReservedDirectories() []string {
	var reservedDirs []string

	switch runtime.GOOS {
	case "windows":
		reservedDirs = []string{
			"C:\\Windows",
			"C:\\Program Files",
			"C:\\Program Files (x86)",
			"C:\\System32",
			"C:\\ProgramData\\Microsoft", // More specific
		}

	case "darwin": // macOS
		reservedDirs = []string{
			"/System",
			"/usr/bin",
			"/usr/sbin",
			"/bin",
			"/sbin",
			"/etc",
			"/var/log",  // More specific - not all of /var
			"/var/db",   // More specific
			"/var/root", // More specific
			"/Library/System",
			"/Applications", // System apps
			"/private/etc",
		}

	default: // Linux and other Unix
		reservedDirs = []string{
			"/bin",
			"/sbin",
			"/usr/bin",
			"/usr/sbin",
			"/etc",
			"/boot",
			"/dev",
			"/proc",
			"/sys",
			"/var/log",   // More specific
			"/var/lib",   // More specific
			"/var/cache", // More specific
			"/root",
		}
	}

	// Add current user's system directories to avoid
	if home, err := os.UserHomeDir(); err == nil {
		// Avoid critical user directories
		systemUserDirs := []string{
			filepath.Join(home, ".ssh"),
			filepath.Join(home, ".gnupg"),
		}
		reservedDirs = append(reservedDirs, systemUserDirs...)
	}

	return reservedDirs
}

// helper function to detect legitimate user temp directories
func isUserTempDirectory(path string) bool {
	// macOS: /var/folders/xx/yyyy/T/ are user temp dirs
	if runtime.GOOS == "darwin" {
		if strings.Contains(path, "/var/folders/") {
			return true
		}
	}

	// Linux: /tmp is usually safe, /var/tmp may be safe
	if runtime.GOOS == "linux" {
		if strings.HasPrefix(path, "/tmp/") || path == "/tmp" {
			return true
		}
	}

	// Windows: temp directories under user profile
	if runtime.GOOS == "windows" {
		if strings.Contains(strings.ToLower(path), "\\temp\\") ||
			strings.Contains(strings.ToLower(path), "\\tmp\\") {
			return true
		}
	}

	// Check if path is under system temp directory
	systemTemp := os.TempDir()
	cleanSystemTemp := filepath.Clean(systemTemp)
	cleanPath := filepath.Clean(path)

	if strings.HasPrefix(cleanPath, cleanSystemTemp) {
		return true
	}

	return false
}

//SymLink Validation helpers

// validateSymlinkSecurity checks if a path is a symlink and validates its security
func validateSymlinkSecurity(path string) error {
	// try to resolve the target path itself
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		// Path exists (might be a symlink), check where it resolves to
		if isReservedDirectory(resolved) {
			return fmt.Errorf("path resolves to reserved directory: %s", resolved)
		}
		return nil
	}

	// Path doesn't exist, check if parent has dangerous symlinks
	parentDir := filepath.Dir(path)
	resolved, err = filepath.EvalSymlinks(parentDir)
	if err != nil {
		return fmt.Errorf("cannot resolve parent directory: %w", err)
	}

	if isReservedDirectory(resolved) {
		return fmt.Errorf("parent directory resolves to reserved location: %s", resolved)
	}

	return nil
}

// // isSymlink checks if a path is a symbolic link
// func isSymlink(path string) (bool, error) {
// 	info, err := os.Lstat(path)
// 	if err != nil {
// 		return false, err
// 	}
// 	return info.Mode()&os.ModeSymlink != 0, nil
// }
