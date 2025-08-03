package filemanager

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"rulem/internal/logging"
	"strings"
)

type FileManager struct {
	logger     *logging.AppLogger
	storageDir string
}

// NewFileManager initializes a new FileManager with the given logger,
// and sets the storage directory to the configured value.
func NewFileManager(storageDir string, logger *logging.AppLogger) (*FileManager, error) {
	// Validate the storage directory path
	if err := ValidateStorageDir(storageDir); err != nil {
		return nil, fmt.Errorf("Invalid storage directory: %w", err)
	}

	// Verify storage directory exists (don't create it)
	if _, err := os.Stat(storageDir); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("storage directory does not exist: %s", storageDir)
		}
		return nil, fmt.Errorf("cannot access storage directory: %w", err)
	}

	return &FileManager{
		logger:     logger,
		storageDir: storageDir,
	}, nil
}

// CopyFileToStorage copies a file from the source path to the storage directory.
// It returns the destination path of the copied file or an error if the operation fails.
// If newFileName is provided, it will be used as the name of the copied file.
// If overwrite is true, existing files will be replaced; otherwise returns error if file exists.
func (fm *FileManager) CopyFileToStorage(srcPath string, newFileName *string, overwrite bool) (string, error) {
	// Validate and resolve source path
	absPath, err := filepath.Abs(srcPath)
	if err != nil {
		return "", fmt.Errorf("invalid source path: %w", err)
	}

	// Check if source file exists and is accessible
	srcInfo, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("source file does not exist: %s", srcPath)
		}
		return "", fmt.Errorf("cannot access source file: %w", err)
	}

	// Ensure source is a regular file, not a directory
	if srcInfo.IsDir() {
		return "", fmt.Errorf("source is a directory, not a file: %s", srcPath)
	}

	// Security: validate symlinks
	if err := validateSymlinkSecurity(absPath); err != nil {
		return "", fmt.Errorf("symlink security check failed: %w", err)
	}

	// Determine safe destination filename
	var fileName string
	if newFileName != nil {
		// Security: prevent path traversal in filename
		cleanName := filepath.Base(*newFileName)
		if strings.Contains(*newFileName, "..") ||
			strings.ContainsAny(*newFileName, `/\`) ||
			cleanName != *newFileName {
			return "", fmt.Errorf("invalid filename: contains path separators or traversal attempts")
		}
		if cleanName == "" || cleanName == "." || cleanName == ".." {
			return "", fmt.Errorf("invalid filename: %q", *newFileName)
		}
		fileName = cleanName
	} else {
		fileName = filepath.Base(srcPath)
	}

	// Construct destination path
	destPath := filepath.Join(fm.storageDir, fileName)

	// Check if destination exists
	if _, err := os.Stat(destPath); err == nil {
		if !overwrite {
			return "", fmt.Errorf("destination file already exists: %s (use overwrite=true to replace)", fileName)
		}
		fm.logger.Debug("Overwriting existing file", "dest", destPath)
	}

	// Verify we can write to storage directory
	if err := TestWriteToStorageDir(fm.storageDir); err != nil {
		return "", fmt.Errorf("storage directory is not writable: %w", err)
	}

	// Perform atomic copy
	if err := fm.atomicCopy(absPath, destPath); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	fm.logger.Info("File copied successfully", "src", srcPath, "dest", destPath)
	return destPath, nil

}

// atomicCopy performs an atomic file copy operation.
// It copies to a temporary file first, then renames to the final destination.
// This ensures we don't corrupt existing files if the copy fails midway.
func (fm *FileManager) atomicCopy(srcPath, destPath string) error {
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

// GetStorageDir returns the storage directory path
func (fm *FileManager) GetStorageDir() string {
	return fm.storageDir
}
