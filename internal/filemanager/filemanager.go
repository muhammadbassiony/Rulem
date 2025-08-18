package filemanager

import (
	"fmt"
	"os"
	"path/filepath"
	"rulem/internal/logging"
	"rulem/pkg/fileops"
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
	if err := fileops.ValidateStoragePath(storageDir); err != nil {
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

	// Validate source file access using fileops
	if err := fileops.ValidateFileAccess(absPath, false); err != nil {
		return "", fmt.Errorf("source file validation failed: %w", err)
	}

	// Security: validate symlinks using allowlist approach
	// Only validate if the path is actually a symlink
	if isLink, err := fileops.IsSymlink(absPath); err == nil && isLink {
		// Create allowlist of safe source locations
		allowedPaths := []string{fm.storageDir}
		if cwd, err := os.Getwd(); err == nil {
			allowedPaths = append(allowedPaths, cwd)
		}

		if err := fileops.ValidateSymlinkSecurity(absPath, allowedPaths); err != nil {
			return "", fmt.Errorf("symlink security check failed: %w", err)
		}
	}

	// Determine safe destination filename
	var fileName string
	if newFileName != nil {
		// Security: sanitize filename using fileops
		cleanName, err := fileops.SanitizeFilename(*newFileName)
		if err != nil {
			return "", fmt.Errorf("invalid filename: %w", err)
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
	if err := fileops.ValidateDirectoryWritable(fm.storageDir); err != nil {
		return "", fmt.Errorf("storage directory is not writable: %w", err)
	}

	// Perform atomic copy
	if err := fileops.AtomicCopy(absPath, destPath); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	fm.logger.Info("File copied successfully", "src", srcPath, "dest", destPath)
	return destPath, nil
}

// CopyFileFromStorage copies a file from the storage directory to the current working directory.
// Performs atomic copy operation to ensure data integrity.
//
// Parameters:
//   - storagePath: Path to the file in storage directory (can be absolute or relative)
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
	fm.logger.Debug("Copying file from storage", "src", storagePath, "dest", destPath, "overwrite", overwrite)

	// Validate destination path
	if err := fileops.ValidateCWDPath(destPath); err != nil {
		return "", fmt.Errorf("invalid destination path: %w", err)
	}

	// Handle both absolute and relative storage paths intelligently
	var absStoragePath string
	if filepath.IsAbs(storagePath) {
		// If path is absolute, use it directly but validate it's within storage
		absStoragePath = storagePath
		fm.logger.Debug("Using absolute storage path", "absolute", absStoragePath)
	} else {
		// If path is relative, join with storage directory
		absStoragePath = filepath.Join(fm.storageDir, storagePath)
		fm.logger.Debug("Converted relative to absolute storage path", "relative", storagePath, "absolute", absStoragePath)
	}

	// Validate that source file exists and is within storage directory
	if err := fileops.ValidateFileInDirectory(absStoragePath, fm.storageDir); err != nil {
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
	if err := fileops.EnsureDirectoryExists(destDir); err != nil {
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
	if err := fileops.AtomicCopy(absStoragePath, absDestPath); err != nil {
		return "", fmt.Errorf("failed to copy file from storage: %w", err)
	}

	fm.logger.Info("File copied from storage successfully", "src", absStoragePath, "dest", absDestPath)
	return absDestPath, nil
}

// CreateSymlinkFromStorage creates a symbolic link in the current working directory
// that points to a file in the storage directory using relative paths.
//
// Parameters:
//   - storagePath: Path to the file in storage directory (can be absolute or relative)
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
	if err := fileops.ValidateCWDPath(destPath); err != nil {
		return "", fmt.Errorf("invalid destination path: %w", err)
	}

	// Handle both absolute and relative storage paths intelligently
	var absStoragePath string
	if filepath.IsAbs(storagePath) {
		// If path is absolute, use it directly but validate it's within storage
		absStoragePath = storagePath
		fm.logger.Debug("Using absolute storage path", "absolute", absStoragePath)
	} else {
		// If path is relative, join with storage directory
		absStoragePath = filepath.Join(fm.storageDir, storagePath)
		fm.logger.Debug("Converted relative to absolute storage path", "relative", storagePath, "absolute", absStoragePath)
	}

	// Validate that source file exists and is within storage directory
	if err := fileops.ValidateFileInDirectory(absStoragePath, fm.storageDir); err != nil {
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
	if err := fileops.EnsureDirectoryExists(destDir); err != nil {
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

	// Create the relative symlink
	if err := fileops.CreateRelativeSymlink(absStoragePath, absDestPath); err != nil {
		return "", fmt.Errorf("failed to create symlink: %w", err)
	}

	fm.logger.Info("Symlink created successfully", "target", absStoragePath, "link", absDestPath)
	return absDestPath, nil
}

// GetStorageDir returns the storage directory path.
//
// Returns:
//   - string: Absolute path to the configured storage directory
func (fm *FileManager) GetStorageDir() string {
	return fm.storageDir
}
