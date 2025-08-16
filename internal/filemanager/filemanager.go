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

// NewFileManager initializes a new FileManager with the given logger and storage directory.
// The storage directory must exist and be accessible. No directory creation is performed.
//
// Parameters:
//   - storageDir: Absolute path to the storage directory
//   - logger: Application logger instance
//
// Returns:
//   - *FileManager: Configured FileManager instance
//   - error: Validation or access errors
//
// Security: Validates storage directory path to prevent path traversal and access to system directories.
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
// Performs atomic copy operation to ensure data integrity.
//
// Parameters:
//   - srcPath: Source file path (can be relative or absolute)
//   - newFileName: Optional new filename in storage (nil to keep original name)
//   - overwrite: Whether to replace existing files
//
// Returns:
//   - string: Destination path of the copied file
//   - error: Copy operation errors
//
// Security:
//   - Validates source file accessibility and type
//   - Prevents path traversal in filename parameter
//   - Validates symlink security
//   - Uses atomic copy to prevent corruption
//
// The operation is atomic - either the file is fully copied or no changes are made.
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

// CopyFileFromStorage copies a file from the storage directory to the current working directory.
// Performs atomic copy operation to ensure data integrity.
//
// Parameters:
//   - storagePath: Path to the file in storage directory
//   - destPath: Destination path relative to current working directory
//   - overwrite: Whether to replace existing files
//
// Returns:
//   - string: Absolute destination path of the copied file
//   - error: Copy operation errors
//
// Security:
//   - Validates that source file exists within storage directory
//   - Prevents path traversal in both parameters
//   - Validates destination is within current working directory tree
//   - Uses atomic copy to prevent corruption
//
// The destPath must be relative to CWD and cannot escape to parent directories.
func (fm *FileManager) CopyFileFromStorage(storagePath string, destPath string, overwrite bool) (string, error) {
	// Validate destination path
	if err := validateCWDPath(destPath); err != nil {
		return "", fmt.Errorf("invalid destination path: %w", err)
	}

	// Validate that source file exists and is within storage directory
	if err := validateFileInStorage(storagePath, fm.storageDir); err != nil {
		return "", fmt.Errorf("source file validation failed: %w", err)
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot get current working directory: %w", err)
	}

	// Construct absolute destination path
	absDestPath := filepath.Join(cwd, destPath)

	// Ensure destination directory exists
	destDir := filepath.Dir(absDestPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("cannot create destination directory: %w", err)
	}

	// Check if destination exists
	if _, err := os.Stat(absDestPath); err == nil {
		if !overwrite {
			return "", fmt.Errorf("destination file already exists: %s (use overwrite=true to replace)", destPath)
		}
		fm.logger.Debug("Overwriting existing file", "dest", absDestPath)
	}

	// Perform atomic copy
	if err := fm.atomicCopy(storagePath, absDestPath); err != nil {
		return "", fmt.Errorf("failed to copy file from storage: %w", err)
	}

	fm.logger.Info("File copied from storage successfully", "src", storagePath, "dest", absDestPath)
	return absDestPath, nil
}

// CreateSymlinkFromStorage creates a symbolic link in the current working directory
// that points to a file in the storage directory using relative paths.
//
// Parameters:
//   - storagePath: Path to the file in storage directory
//   - destPath: Destination path for symlink relative to current working directory
//   - overwrite: Whether to replace existing files/symlinks
//
// Returns:
//   - string: Absolute path of the created symlink
//   - error: Symlink creation errors
//
// Security:
//   - Validates that target file exists within storage directory
//   - Prevents path traversal in both parameters
//   - Validates destination is within current working directory tree
//   - Creates relative symlinks for portability
//
// The symlink will be bidirectional - editing the symlinked file modifies the storage file.
// This provides real-time synchronization but requires careful permission management.
func (fm *FileManager) CreateSymlinkFromStorage(storagePath string, destPath string, overwrite bool) (string, error) {
	// Validate destination path
	if err := validateCWDPath(destPath); err != nil {
		return "", fmt.Errorf("invalid destination path: %w", err)
	}

	// Validate that source file exists and is within storage directory
	if err := validateFileInStorage(storagePath, fm.storageDir); err != nil {
		return "", fmt.Errorf("source file validation failed: %w", err)
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot get current working directory: %w", err)
	}

	// Construct absolute destination path
	absDestPath := filepath.Join(cwd, destPath)

	// Ensure destination directory exists
	destDir := filepath.Dir(absDestPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("cannot create destination directory: %w", err)
	}

	// Check if destination exists
	if _, err := os.Stat(absDestPath); err == nil {
		if !overwrite {
			return "", fmt.Errorf("destination file already exists: %s (use overwrite=true to replace)", destPath)
		}
		if err := os.Remove(absDestPath); err != nil {
			return "", fmt.Errorf("cannot remove existing destination: %w", err)
		}
		fm.logger.Debug("Removed existing file for symlink", "dest", absDestPath)
	}

	// Calculate relative path from symlink location to storage file
	relPath, err := filepath.Rel(destDir, storagePath)
	if err != nil {
		return "", fmt.Errorf("cannot calculate relative path: %w", err)
	}

	// Create the symlink
	if err := os.Symlink(relPath, absDestPath); err != nil {
		return "", fmt.Errorf("failed to create symlink: %w", err)
	}

	fm.logger.Info("Symlink created successfully", "target", storagePath, "link", absDestPath, "relative", relPath)
	return absDestPath, nil
}

// atomicCopy performs an atomic file copy operation.
// It copies to a temporary file first, then renames to the final destination.
// This ensures we don't corrupt existing files if the copy fails midway.
//
// Parameters:
//   - srcPath: Absolute path to source file
//   - destPath: Absolute path to destination file
//
// Returns:
//   - error: Copy operation errors
//
// The operation is atomic at the filesystem level - the destination file
// either appears fully copied or not at all.
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

// GetStorageDir returns the storage directory path.
//
// Returns:
//   - string: Absolute path to the configured storage directory
func (fm *FileManager) GetStorageDir() string {
	return fm.storageDir
}

// validateCWDPath validates that a destination path is safe relative to current working directory.
// Prevents path traversal outside of CWD and ensures path is relative.
func validateCWDPath(destPath string) error {
	if destPath == "" {
		return fmt.Errorf("destination path cannot be empty")
	}

	// Path must be relative
	if filepath.IsAbs(destPath) {
		return fmt.Errorf("destination path must be relative to current working directory")
	}

	// Check for path traversal
	if strings.Contains(destPath, "..") {
		return fmt.Errorf("path traversal not allowed in destination path")
	}

	// Clean and validate the path
	cleanPath := filepath.Clean(destPath)
	if strings.HasPrefix(cleanPath, "..") || strings.Contains(cleanPath, "..") {
		return fmt.Errorf("destination path attempts to escape current working directory")
	}

	return nil
}

// validateFileInStorage validates that a file path is within the storage directory
// and that the file exists and is accessible.
func validateFileInStorage(filePath, storageDir string) error {
	// Resolve absolute paths
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("cannot resolve file path: %w", err)
	}

	absStorageDir, err := filepath.Abs(storageDir)
	if err != nil {
		return fmt.Errorf("cannot resolve storage directory: %w", err)
	}

	// Check if file is within storage directory
	relPath, err := filepath.Rel(absStorageDir, absFilePath)
	if err != nil {
		return fmt.Errorf("cannot determine relative path: %w", err)
	}

	if strings.HasPrefix(relPath, "..") {
		return fmt.Errorf("file is not within storage directory")
	}

	// Check if file exists and is accessible
	fileInfo, err := os.Stat(absFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist in storage: %s", filepath.Base(filePath))
		}
		return fmt.Errorf("cannot access file: %w", err)
	}

	// Ensure it's a regular file
	if fileInfo.IsDir() {
		return fmt.Errorf("path is a directory, not a file")
	}

	// Security: validate symlinks
	if err := validateSymlinkSecurity(absFilePath); err != nil {
		return fmt.Errorf("symlink security check failed: %w", err)
	}

	return nil
}
