package fileops

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// IsSymlink checks if a given path is a symbolic link.
// This function uses lstat to examine the file without following symlinks.
//
// Parameters:
//   - path: File path to check
//
// Returns:
//   - bool: true if the path is a symbolic link, false otherwise
//   - error: File system access errors
//
// Usage example:
//
//	isLink, err := fileops.IsSymlink("/path/to/potential/symlink")
//	if err != nil {
//	    return fmt.Errorf("failed to check symlink: %w", err)
//	}
//	if isLink {
//	    fmt.Println("Path is a symbolic link")
//	}
func IsSymlink(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return false, fmt.Errorf("failed to stat path: %w", err)
	}
	return info.Mode()&os.ModeSymlink != 0, nil
}

// CreateRelativeSymlink creates a symbolic link using a relative path from the link location
// to the target. This approach is more portable than absolute symlinks and is preferred
// for most use cases.
//
// Parameters:
//   - target: Absolute path to the target file/directory
//   - linkPath: Absolute path where the symlink should be created
//
// Returns:
//   - error: Symlink creation errors
//
// The function:
//   - Calculates the relative path from link location to target
//   - Creates the symlink using the relative path
//   - Ensures parent directories exist
//   - Validates that the target exists
//
// Security considerations:
//   - Both target and linkPath should be validated before calling
//   - The function does not perform path traversal validation
//   - Symlinks can be security risks if not properly managed
//
// Usage example:
//
//	err := fileops.CreateRelativeSymlink("/storage/file.txt", "/project/link.txt")
//	if err != nil {
//	    return fmt.Errorf("failed to create symlink: %w", err)
//	}
func CreateRelativeSymlink(target, linkPath string) error {
	// Validate that target exists
	if _, err := os.Stat(target); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("target does not exist: %s", target)
		}
		return fmt.Errorf("cannot access target: %w", err)
	}

	// Ensure the directory for the symlink exists
	linkDir := filepath.Dir(linkPath)
	if err := EnsureDirectoryExists(linkDir); err != nil {
		return fmt.Errorf("failed to create symlink directory: %w", err)
	}

	// Calculate relative path from symlink location to target
	relPath, err := filepath.Rel(linkDir, target)
	if err != nil {
		return fmt.Errorf("cannot calculate relative path: %w", err)
	}

	// Create the symlink
	if err := os.Symlink(relPath, linkPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

// CreateAbsoluteSymlink creates a symbolic link using an absolute path to the target.
// While absolute symlinks are less portable, they can be useful in specific scenarios
// where the relative path relationship might change.
//
// Parameters:
//   - target: Absolute path to the target file/directory
//   - linkPath: Absolute path where the symlink should be created
//
// Returns:
//   - error: Symlink creation errors
//
// Usage example:
//
//	err := fileops.CreateAbsoluteSymlink("/storage/file.txt", "/project/link.txt")
//	if err != nil {
//	    return fmt.Errorf("failed to create absolute symlink: %w", err)
//	}
func CreateAbsoluteSymlink(target, linkPath string) error {
	// Validate that target exists
	if _, err := os.Stat(target); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("target does not exist: %s", target)
		}
		return fmt.Errorf("cannot access target: %w", err)
	}

	// Ensure the directory for the symlink exists
	linkDir := filepath.Dir(linkPath)
	if err := EnsureDirectoryExists(linkDir); err != nil {
		return fmt.Errorf("failed to create symlink directory: %w", err)
	}

	// Get absolute path of target
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("cannot resolve absolute target path: %w", err)
	}

	// Create the symlink
	if err := os.Symlink(absTarget, linkPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

// ResolveSymlink resolves a symbolic link and returns the final target path.
// This function follows symlink chains until it reaches a non-symlink target.
//
// Parameters:
//   - linkPath: Path to the symbolic link
//
// Returns:
//   - string: Absolute path to the final target
//   - error: Resolution errors, including broken symlinks or permission issues
//
// Usage example:
//
//	target, err := fileops.ResolveSymlink("/path/to/symlink")
//	if err != nil {
//	    return fmt.Errorf("failed to resolve symlink: %w", err)
//	}
//	fmt.Printf("Symlink points to: %s\n", target)
func ResolveSymlink(linkPath string) (string, error) {
	resolved, err := filepath.EvalSymlinks(linkPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve symlink: %w", err)
	}
	return resolved, nil
}

// ValidateSymlinkSecurity validates that a symlink and its target are safe.
// This function helps prevent symlink-based attacks by checking where symlinks resolve.
//
// Parameters:
//   - linkPath: Path to the symbolic link to validate
//   - allowedBasePaths: List of base paths that the symlink target must be within
//
// Returns:
//   - error: Security validation errors
//
// The function checks:
//   - Symlink exists and is actually a symlink
//   - Symlink can be resolved (not broken)
//   - Resolved target is within one of the allowed base paths
//   - No symlink loops exist
//
// Usage example:
//
//	allowedPaths := []string{"/safe/storage", "/allowed/directory"}
//	err := fileops.ValidateSymlinkSecurity("/project/link.txt", allowedPaths)
//	if err != nil {
//	    return fmt.Errorf("symlink security check failed: %w", err)
//	}
func ValidateSymlinkSecurity(linkPath string, allowedBasePaths []string) error {
	// Check if the path is actually a symlink
	isLink, err := IsSymlink(linkPath)
	if err != nil {
		return fmt.Errorf("cannot check if path is symlink: %w", err)
	}
	if !isLink {
		return fmt.Errorf("path is not a symbolic link: %s", linkPath)
	}

	// Resolve the symlink
	resolved, err := ResolveSymlink(linkPath)
	if err != nil {
		return fmt.Errorf("symlink resolution failed: %w", err)
	}

	// Check if resolved path is within allowed base paths
	resolvedAbs, err := filepath.Abs(resolved)
	if err != nil {
		return fmt.Errorf("cannot get absolute path of resolved target: %w", err)
	}

	// Resolve symlinks in the resolved path to handle macOS /private paths
	resolvedCanonical, err := filepath.EvalSymlinks(resolvedAbs)
	if err != nil {
		resolvedCanonical = resolvedAbs // Use original if symlink resolution fails
	}

	for _, basePath := range allowedBasePaths {
		baseAbs, err := filepath.Abs(basePath)
		if err != nil {
			continue // Skip invalid base paths
		}

		// Resolve symlinks in the base path too
		baseCanonical, err := filepath.EvalSymlinks(baseAbs)
		if err != nil {
			baseCanonical = baseAbs // Use original if symlink resolution fails
		}

		relPath, err := filepath.Rel(baseCanonical, resolvedCanonical)
		if err != nil {
			continue // Skip if relative path cannot be calculated
		}

		// If relative path doesn't start with "..", target is within base path
		if !strings.HasPrefix(relPath, "..") {
			return nil // Valid: target is within an allowed base path
		}
	}

	return fmt.Errorf("symlink target is not within any allowed base path: %s", resolved)
}

// RemoveSymlink safely removes a symbolic link without affecting its target.
// This function specifically checks that the path is a symlink before removal
// to prevent accidental deletion of regular files.
//
// Parameters:
//   - linkPath: Path to the symbolic link to remove
//
// Returns:
//   - error: Removal errors
//
// Usage example:
//
//	err := fileops.RemoveSymlink("/path/to/symlink")
//	if err != nil {
//	    return fmt.Errorf("failed to remove symlink: %w", err)
//	}
func RemoveSymlink(linkPath string) error {
	// Verify it's actually a symlink before removing
	isLink, err := IsSymlink(linkPath)
	if err != nil {
		return fmt.Errorf("cannot verify symlink: %w", err)
	}
	if !isLink {
		return fmt.Errorf("path is not a symbolic link, will not remove: %s", linkPath)
	}

	// Remove the symlink
	if err := os.Remove(linkPath); err != nil {
		return fmt.Errorf("failed to remove symlink: %w", err)
	}

	return nil
}

// GetSymlinkTarget returns the immediate target of a symbolic link without resolving
// the full chain. This is useful when you need to know what a symlink directly points to.
//
// Parameters:
//   - linkPath: Path to the symbolic link
//
// Returns:
//   - string: Direct target of the symlink (may be relative)
//   - error: Read errors or if path is not a symlink
//
// Usage example:
//
//	target, err := fileops.GetSymlinkTarget("/path/to/symlink")
//	if err != nil {
//	    return fmt.Errorf("failed to read symlink target: %w", err)
//	}
//	fmt.Printf("Symlink directly points to: %s\n", target)
func GetSymlinkTarget(linkPath string) (string, error) {
	// Verify it's actually a symlink
	isLink, err := IsSymlink(linkPath)
	if err != nil {
		return "", fmt.Errorf("cannot verify symlink: %w", err)
	}
	if !isLink {
		return "", fmt.Errorf("path is not a symbolic link: %s", linkPath)
	}

	// Read the symlink target
	target, err := os.Readlink(linkPath)
	if err != nil {
		return "", fmt.Errorf("failed to read symlink: %w", err)
	}

	return target, nil
}
