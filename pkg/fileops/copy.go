// Package fileops provides secure, atomic file operations for Go applications.
// This package implements file operations with security-first design principles,
// including atomic operations, path validation, and comprehensive error handling.
package fileops

import (
	"fmt"
	"io"
	"os"
)

// AtomicCopy performs an atomic file copy operation from source to destination.
// The operation is atomic at the filesystem level - the destination file either
// appears fully copied or not at all.
//
// The function uses a temporary file approach:
//  1. Creates a temporary file in the destination directory
//  2. Copies all data to the temporary file
//  3. Syncs data to disk to ensure durability
//  4. Atomically renames the temporary file to the final destination
//
// Parameters:
//   - srcPath: Absolute path to the source file
//   - destPath: Absolute path to the destination file
//
// Returns:
//   - error: Copy operation errors, including source access, destination creation,
//     or filesystem errors
//
// Security considerations:
//   - Both paths should be validated before calling this function
//   - The function does not perform path traversal validation
//   - Temporary files are cleaned up on any failure
//   - File permissions are set to 0644 (readable by owner and group, writable by owner)
//
// Usage example:
//
//	if err := fileops.AtomicCopy("/path/to/source.txt", "/path/to/dest.txt"); err != nil {
//	    log.Fatalf("Copy failed: %v", err)
//	}
//
// Note: This function requires write permissions in the destination directory
// and will overwrite existing files without warning.
func AtomicCopy(srcPath, destPath string) error {
	// Open source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Create temporary file in same directory as destination
	tempPath := destPath + ".tmp"
	tempFile, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	// Ensure cleanup of temp file if anything goes wrong
	var copySuccess bool
	defer func() {
		tempFile.Close()
		if !copySuccess {
			os.Remove(tempPath) // Clean up on failure
		}
	}()

	// Copy file contents
	if _, err := io.Copy(tempFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Sync to ensure data is written to disk
	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	// Close temp file before rename
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Atomic rename - this is the atomic operation
	if err := os.Rename(tempPath, destPath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	copySuccess = true
	return nil
}

// EnsureDirectoryExists creates a directory and all necessary parent directories.
// This is equivalent to `mkdir -p` and is safe to call multiple times.
//
// Parameters:
//   - path: Directory path to create
//
// Returns:
//   - error: Directory creation errors
//
// The function sets directory permissions to 0755 (readable and executable by all,
// writable by owner only).
//
// Usage example:
//
//	if err := fileops.EnsureDirectoryExists("/path/to/nested/directory"); err != nil {
//	    log.Fatalf("Failed to create directory: %v", err)
//	}
func EnsureDirectoryExists(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	return nil
}
