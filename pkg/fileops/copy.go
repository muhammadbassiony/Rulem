// Package fileops provides secure, atomic file operations for Go applications.
// This package implements file operations with security-first design principles,
// including atomic operations, path validation, and comprehensive error handling.
package fileops

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// tempNameAttempts bounds how many random temporary names are tried before
// giving up. Collisions are astronomically unlikely, so more than one attempt
// is only ever needed if the directory is under adversarial pressure.
const tempNameAttempts = 10

// AtomicCopy performs an atomic file copy operation from source to destination.
// The operation is atomic at the filesystem level - the destination file either
// appears fully copied or not at all.
//
// The function uses a temporary file approach:
//  1. Opens the destination directory as an os.Root
//  2. Creates a randomly named temporary file inside it with O_EXCL
//  3. Copies all data to the temporary file
//  4. Syncs data to disk to ensure durability
//  5. Atomically renames the temporary file to the final destination
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
//   - Every write happens through an os.Root scoped to the destination directory,
//     so a symlink planted inside that directory cannot redirect the write outside it
//   - The temporary name is random and created with O_EXCL, so an attacker cannot
//     pre-create it to have the copy follow a link or clobber an unrelated file
//   - Concurrent copies to the same destination use distinct temporary files
//   - Temporary files are cleaned up on any failure
//   - The destination inherits the source file's permission bits
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
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}
	if srcInfo.IsDir() {
		return fmt.Errorf("source is a directory, not a file: %s", srcPath)
	}

	destDir, destName := filepath.Split(destPath)
	if destName == "" {
		return fmt.Errorf("destination path has no file name: %s", destPath)
	}
	if destDir == "" {
		destDir = "."
	}

	// Scope everything that follows to the destination directory. os.Root
	// resolves each name against an open directory handle rather than against
	// a string, so the checks cannot go stale between validation and use.
	root, err := os.OpenRoot(destDir)
	if err != nil {
		return fmt.Errorf("failed to open destination directory: %w", err)
	}
	defer func() { _ = root.Close() }()

	tempName, tempFile, err := createTempFile(root)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	var renamed bool
	defer func() {
		tempFile.Close()
		if !renamed {
			_ = root.Remove(tempName) // Clean up on failure
		}
	}()

	// Match the source permissions. Chmod on the open handle is used because
	// the O_CREATE mode above is masked by umask.
	if err := tempFile.Chmod(srcInfo.Mode().Perm()); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

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
	if err := root.Rename(tempName, destName); err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	renamed = true
	return nil
}

// createTempFile creates a uniquely named, empty file directly inside root and
// returns its name along with the open handle. The caller owns both: it must
// close the file and remove the name if it does not rename it into place.
//
// The name is unpredictable and the file is created with O_EXCL, so an existing
// entry - including a symlink planted by another user - makes the open fail
// rather than be followed.
func createTempFile(root *os.Root) (string, *os.File, error) {
	for range tempNameAttempts {
		var suffix [8]byte
		if _, err := rand.Read(suffix[:]); err != nil {
			return "", nil, fmt.Errorf("cannot generate temporary file name: %w", err)
		}
		name := ".fileops-" + hex.EncodeToString(suffix[:]) + ".tmp"

		file, err := root.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
		if err == nil {
			return name, file, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return "", nil, err
		}
	}
	return "", nil, fmt.Errorf("no unique temporary name available in %s", root.Name())
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
