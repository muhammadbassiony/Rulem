package filemanager

import (
	"fmt"
	"os"
	"path/filepath"
	"rulem/internal/logging"
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

// ScanCentralRepo recursively scans the central repository and all its children
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
func (fm *FileManager) ScanCentralRepo() ([]FileItem, error) {
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
