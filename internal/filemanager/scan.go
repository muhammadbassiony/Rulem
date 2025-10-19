package filemanager

import (
	"fmt"
	"os"
	"path/filepath"
	"rulem/internal/logging"
	"rulem/internal/repository"
	"rulem/pkg/fileops"
	"slices"
	"strings"
)

// markdownExtensions contains supported markdown file extensions
var markdownExtensions = []string{
	".md", ".mdown", ".mkdn", ".mkd", ".markdown", ".mdc",
}

// isMarkdownFile checks if a filename has a markdown extension.
// This function is used as a file filter for the directory scanner.
func isMarkdownFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return slices.Contains(markdownExtensions, ext)
}

// ScanCurrDirectory recursively scans the current working directory and all its children
// for markdown files and returns a list of FileItem. This function acts as an integration
// point between the generic fileops directory scanner and the filemanager domain logic.
//
// Returns:
//   - []FileItem: List of discovered markdown files
//   - error: Scanning errors
//
// Security: Uses secure directory scanning with protection against path traversal and symlink attacks.
func (fm *FileManager) ScanCurrDirectory() ([]FileItem, error) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Create scanner with markdown-specific options
	opts := &fileops.DirectoryScanOptions{
		SkipUnreadableDirs: true,
		MaxDepth:           20,
		IncludeHidden:      true,
		SkipPatterns:       []string{"node_modules", ".git", "vendor", "target", "build", ".next", "dist", ".cache", "__pycache__", ".vscode", ".idea"},
		FileFilter:         isMarkdownFile,
	}

	// Create secure directory scanner
	scanner, err := fileops.NewDirectoryScanner(cwd, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create directory scanner: %w", err)
	}
	defer scanner.Close()

	// Perform the scan
	files, err := scanner.ScanDirectory()
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	// Convert fileops.FileInfo to filemanager.FileItem
	var result []FileItem
	for _, file := range files {
		if !file.IsDir { // Only include files, not directories
			result = append(result, FileItem{
				Name: file.Name,
				Path: file.Path,
			})
		}
	}

	logging.Debug("Scanned current directory for markdown files", "fileCount", len(result))
	return result, nil
}

// ScanRepository recursively scans the repository directory and all its children
// for markdown files and returns a list of FileItem.
//
// This method scans the FileManager's configured storage directory for markdown files.
// It performs comprehensive security validation including symlink security checks,
// reserved directory protection, and path traversal prevention.
//
// Returns:
//   - []FileItem: List of discovered markdown files with paths relative to storage root
//   - error: Scanning errors including security violations
//
// Security: Uses secure directory scanning with protection against path traversal and symlink attacks.
// Validates storage path and symlinks to prevent access to system directories.
func (fm *FileManager) ScanRepository() ([]FileItem, error) {
	if fm == nil {
		return nil, fmt.Errorf("filemanager is nil")
	}

	storageRoot := fm.storageDir
	if storageRoot == "" {
		return nil, fmt.Errorf("storage directory is not configured")
	}

	// Handle symlinks with security validation
	isSymlink, err := fileops.IsSymlink(storageRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to check if storage directory is a symlink: %w", err)
	}

	if isSymlink {
		fm.logger.Debug("Storage directory is a symlink, validating security", "path", storageRoot)

		// For symlinks, we need to define allowed base paths for security
		// Allow the current working directory and user home directory as safe bases
		allowedPaths := []string{}
		if cwd, err := os.Getwd(); err == nil {
			allowedPaths = append(allowedPaths, cwd)
		}
		if homeDir, err := os.UserHomeDir(); err == nil {
			allowedPaths = append(allowedPaths, homeDir)
		}

		// Validate symlink security
		if err := fileops.ValidateSymlinkSecurity(storageRoot, allowedPaths); err != nil {
			return nil, fmt.Errorf("storage directory symlink security validation failed: %w", err)
		}

		// Resolve the symlink after validation
		absStorageRootPath, err := fileops.ResolveSymlink(storageRoot)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve symlink for storage directory: %w", err)
		}
		storageRoot = absStorageRootPath
	} else {
		// Resolve absolute path
		absPath, err := filepath.Abs(storageRoot)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve storage directory: %w", err)
		}
		storageRoot = absPath
	}

	// Use comprehensive storage path validation from fileops
	if err := fileops.ValidateStoragePath(storageRoot); err != nil {
		return nil, fmt.Errorf("storage directory failed security validation: %w", err)
	}

	// Ensure path exists and is a directory
	info, err := os.Stat(storageRoot)
	if err != nil {
		return nil, fmt.Errorf("storage directory not accessible: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("storage path is not a directory")
	}

	// Create scanner with markdown-specific options
	opts := &fileops.DirectoryScanOptions{
		SkipUnreadableDirs: true,
		MaxDepth:           50,
		IncludeHidden:      true,
		SkipPatterns:       []string{"node_modules", ".git", "vendor", "target", "build", ".next", "dist", ".cache", "__pycache__", ".vscode", ".idea"},
		FileFilter:         isMarkdownFile,
	}

	// Create secure directory scanner
	scanner, err := fileops.NewDirectoryScanner(storageRoot, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create directory scanner: %w", err)
	}
	defer scanner.Close()

	// Perform the scan
	files, err := scanner.ScanDirectory()
	if err != nil {
		return nil, fmt.Errorf("failed to scan storage directory: %w", err)
	}

	// Convert fileops.FileInfo to filemanager.FileItem
	var result []FileItem
	for _, file := range files {
		if !file.IsDir { // Only include files, not directories
			result = append(result, FileItem{
				Name: file.Name,
				Path: file.Path,
			})
		}
	}

	logging.Debug("Scanned central storage for markdown files", "fileCount", len(result))
	return result, nil
}

// GetAbsolutePath returns the absolute path for a FileItem from central repository.
// This is needed for FilePicker which requires absolute paths for file reading.
func (fm *FileManager) GetAbsolutePath(item FileItem) string {
	return filepath.Join(fm.storageDir, item.Path)
}

// GetAbsolutePathCWD returns the absolute path for a FileItem from current directory.
// This is needed for FilePicker which requires absolute paths for file reading.
func (fm *FileManager) GetAbsolutePathCWD(item FileItem) string {
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, item.Path)
}

// ConvertToAbsolutePaths converts a slice of FileItems with relative paths to absolute paths
// for the storage directory. This is needed for FilePicker which requires absolute paths for file reading.
//
// Parameters:
//   - items: List of FileItems with paths relative to storage directory
//
// Returns:
//   - []FileItem: New slice with absolute paths for storage directory
//
// Note: This creates a new slice and does not modify the original items.
func (fm *FileManager) ConvertToAbsolutePaths(items []FileItem) []FileItem {
	if len(items) == 0 {
		return []FileItem{}
	}

	result := make([]FileItem, len(items))
	for i, item := range items {
		result[i] = FileItem{
			Name:           item.Name,
			Path:           fm.GetAbsolutePath(item),
			RepositoryID:   item.RepositoryID,
			RepositoryName: item.RepositoryName,
			RepositoryType: item.RepositoryType,
		}
	}
	return result
}

// ConvertToAbsolutePathsCWD converts a slice of FileItems with relative paths to absolute paths
// for the current working directory. This is needed for FilePicker which requires absolute paths for file reading.
//
// Parameters:
//   - items: List of FileItems with paths relative to current working directory
//
// Returns:
//   - []FileItem: New slice with absolute paths for current working directory
//
// Note: This creates a new slice and does not modify the original items.
func (fm *FileManager) ConvertToAbsolutePathsCWD(items []FileItem) []FileItem {
	if len(items) == 0 {
		return []FileItem{}
	}

	result := make([]FileItem, len(items))
	for i, item := range items {
		result[i] = FileItem{
			Name:           item.Name,
			Path:           fm.GetAbsolutePathCWD(item),
			RepositoryID:   item.RepositoryID,
			RepositoryName: item.RepositoryName,
			RepositoryType: item.RepositoryType,
		}
	}
	return result
}

// ScanAllRepositories scans multiple repositories and merges their file lists.
// This function is the main entry point for multi-repository file discovery.
// Files are tagged with their source repository metadata for display and tracking.
//
// The function maintains repository order - files from earlier repositories appear
// first in the result list. This provides predictable, stable ordering for UI display.
//
// Parameters:
//   - prepared: Slice of prepared repositories with validated paths (from PrepareAllRepositories)
//   - logger: Logger for structured logging (can be nil)
//
// Returns:
//   - []FileItem: Merged list of files from all repositories with source metadata
//   - error: Scanning errors (partial results may be returned with error)
//
// Usage:
//
//	prepared, err := repository.PrepareAllRepositories(cfg.Repositories, logger)
//	files, err := filemanager.ScanAllRepositories(prepared, logger)
//	for _, file := range files {
//	    fmt.Printf("%s from %s (%s)\n", file.Name, file.RepositoryName, file.RepositoryType)
//	}
//
// Security: Paths are pre-validated by PrepareAllRepositories, so this function can safely assume valid paths.
func ScanAllRepositories(prepared []repository.PreparedRepository, logger *logging.AppLogger) ([]FileItem, error) {
	if logger != nil {
		logger.Info("Starting multi-repository scan", "repository_count", len(prepared))
	}

	if len(prepared) == 0 {
		if logger != nil {
			logger.Debug("No repositories to scan")
		}
		return []FileItem{}, nil
	}

	var allFiles []FileItem
	var scanErrors []string

	// Process repositories in order to maintain predictable file ordering
	for _, prep := range prepared {
		if logger != nil {
			logger.Info("Scanning repository",
				"repository_id", prep.ID(),
				"repository_name", prep.Name(),
				"repository_type", string(prep.Type()),
				"path", prep.LocalPath,
			)
		}

		// Determine repository type for metadata
		repoType := string(prep.Type())

		// Create a temporary FileManager for this repository
		// Paths are already validated by PrepareAllRepositories
		fm, err := NewFileManager(prep.LocalPath, logger)
		if err != nil {
			errorMsg := fmt.Sprintf("repository %s (%s): failed to create file manager: %v", prep.ID(), prep.Name(), err)
			scanErrors = append(scanErrors, errorMsg)
			if logger != nil {
				logger.Error("Failed to create file manager", "repository_id", prep.ID(), "error", err)
			}
			continue
		}

		// Scan the repository
		files, err := fm.ScanRepository()
		if err != nil {
			errorMsg := fmt.Sprintf("repository %s (%s): scan failed: %v", prep.ID(), prep.Name(), err)
			scanErrors = append(scanErrors, errorMsg)
			if logger != nil {
				logger.Error("Repository scan failed", "repository_id", prep.ID(), "error", err)
			}
			continue
		}

		// Tag each file with repository metadata
		for i := range files {
			files[i].RepositoryID = prep.ID()
			files[i].RepositoryName = prep.Name()
			files[i].RepositoryType = repoType
		}

		allFiles = append(allFiles, files...)

		if logger != nil {
			logger.Info("Repository scan completed",
				"repository_id", prep.ID(),
				"repository_name", prep.Name(),
				"file_count", len(files),
			)
		}
	}

	if logger != nil {
		logger.Info("Multi-repository scan completed",
			"total_repositories", len(prepared),
			"total_files", len(allFiles),
			"errors", len(scanErrors),
		)
	}

	// Return partial results with error if any scans failed
	if len(scanErrors) > 0 {
		return allFiles, fmt.Errorf("scan errors in %d repositories:\n  - %s",
			len(scanErrors),
			strings.Join(scanErrors, "\n  - "))
	}

	return allFiles, nil
}
