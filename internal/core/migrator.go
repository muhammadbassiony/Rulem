package core

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CreateSymlinks creates symlinks for the given files from srcDir to dstDir.
// If force is true, existing files are overwritten.
// Returns a map of file name to error (nil if successful).
func CreateSymlinks(files []string, srcDir, dstDir string, force bool) map[string]error {
	results := make(map[string]error)
	for _, name := range files {
		src := filepath.Join(srcDir, name)
		dst := filepath.Join(dstDir, name)

		// Check if destination file exists
		if _, err := os.Lstat(dst); err == nil {
			if !force {
				results[name] = fmt.Errorf("destination file already exists: %s", dst)
				continue
			}
			// Remove existing file or symlink
			if err := os.Remove(dst); err != nil {
				results[name] = fmt.Errorf("failed to remove existing file: %w", err)
				continue
			}
		}

		if err := os.Symlink(src, dst); err != nil {
			results[name] = fmt.Errorf("failed to create symlink: %w", err)
			continue
		}
		results[name] = nil
	}
	return results
}

// CopyFiles copies the given files from srcDir to dstDir.
// If force is true, existing files are overwritten.
// Returns a map of file name to error (nil if successful).
func CopyFiles(files []string, srcDir, dstDir string, force bool) map[string]error {
	results := make(map[string]error)
	for _, name := range files {
		src := filepath.Join(srcDir, name)
		dst := filepath.Join(dstDir, name)

		// Check if destination file exists
		if _, err := os.Stat(dst); err == nil {
			if !force {
				results[name] = fmt.Errorf("destination file already exists: %s", dst)
				continue
			}
			// Remove existing file
			if err := os.Remove(dst); err != nil {
				results[name] = fmt.Errorf("failed to remove existing file: %w", err)
				continue
			}
		}

		srcF, err := os.Open(src)
		if err != nil {
			results[name] = fmt.Errorf("failed to open source: %w", err)
			continue
		}
		defer srcF.Close()

		dstF, err := os.Create(dst)
		if err != nil {
			results[name] = fmt.Errorf("failed to create destination: %w", err)
			srcF.Close()
			continue
		}
		_, err = io.Copy(dstF, srcF)
		dstF.Close()
		srcF.Close()
		if err != nil {
			results[name] = fmt.Errorf("copy failed: %w", err)
			continue
		}
		results[name] = nil
	}
	return results
}
