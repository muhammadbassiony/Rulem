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
	".md", ".mdown", ".mkdn", ".mkd", ".markdown",
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
func ScanCurrDirectory() ([]FileItem, error) {
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

// isMarkdownFile checks if a filename has a markdown extension.
// This function is used as a file filter for the directory scanner.
func isMarkdownFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return slices.Contains(markdownExtensions, ext)
}
