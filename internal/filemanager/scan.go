package filemanager

import (
	"fmt"
	"os"
	"path/filepath"
	"rulem/internal/logging"
	"slices"
	"strings"
)

type ScanOptions struct {
	SkipUnreadableDirs bool
	MaxDepth           int
	IncludeHidden      bool
}

type DirScanner struct {
	root    *os.Root
	opts    *ScanOptions
	files   []FileItem
	visited map[string]bool
}

func NewDefaultDirScanner(root *os.Root, opts *ScanOptions) *DirScanner {
	if opts == nil {
		opts = &ScanOptions{
			SkipUnreadableDirs: true,
			MaxDepth:           20,
			IncludeHidden:      true,
		}
	}

	return &DirScanner{
		root:    root,
		opts:    opts,
		files:   []FileItem{},
		visited: make(map[string]bool),
	}
}

// ScanCurrDirectory recursively scans the CWD, and all its children
// for all md files and returns a list of FileItem
func ScanCurrDirectory() ([]FileItem, error) {
	// get CWD
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("Failed to get CWD with err: %v", err)
	}

	// opening the CWD - using root for security
	root, err := os.OpenRoot(cwd)
	if err != nil {
		return nil, fmt.Errorf("Failed to open CWD with err: %v", err)
	}
	defer root.Close()

	// Create a new DirScanner with the root and default options
	dirScanner := NewDefaultDirScanner(root, nil)

	err = dirScanner.scanDirRecursive(".", 1)
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	logging.Debug("Scanned curr directory for md files - returning ", "files", dirScanner.files)
	return dirScanner.files, nil
}

// scanDirRecursive recursively scans a directory using the secure root
func (d *DirScanner) scanDirRecursive(relativePath string, depth int) error {
	// Check depth limit first
	if depth > d.opts.MaxDepth {
		logging.Warn("Maximum recursion depth reached. Skipping directory", "dir", relativePath, "depth", depth)
		return nil
	}

	// Use clean relative path for loop detection
	cleanPath := filepath.Clean(relativePath)
	if d.visited[cleanPath] {
		logging.Debug("Directory already visited, skipping", "dir", relativePath)
		return nil // Skip already visited directory (symlink loop)
	}
	d.visited[cleanPath] = true

	// Check if we should skip this directory before trying to open it
	if shouldSkipDirectory(filepath.Base(relativePath), d.opts.IncludeHidden) {
		logging.Info("Skipping directory", "dir", relativePath)
		return nil
	}

	// Open the current directory relative to root
	dir, err := d.root.Open(relativePath)
	if err != nil {
		if d.opts.SkipUnreadableDirs {
			logging.Info("Cannot scan directory", "dir", relativePath, "error", err)
			return nil
		}

		return fmt.Errorf("failed to open directory %s: %w", relativePath, err)
	}
	defer dir.Close()

	// Read directory entries
	entries, err := dir.ReadDir(-1)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", relativePath, err)
	}

	logging.Info("Scanning directory", "dir", relativePath)
	for _, entry := range entries {
		entryPath := filepath.Join(relativePath, entry.Name())

		if entry.IsDir() {
			// Recursively scan subdirectory
			err := d.scanDirRecursive(entryPath, depth+1)
			if err != nil {
				return err
			}
		} else {
			// Check if it's a markdown file
			if isMarkdownFile(entry.Name()) {
				d.files = append(d.files, FileItem{
					Name: entry.Name(),
					Path: entryPath,
				})
			}
		}
	}

	return nil
}

var markdownExtensions = []string{
	".md", ".mdown", ".mkdn", ".mkd", ".markdown",
}

// isMarkdownFile checks if a filename has a markdown extension
func isMarkdownFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return slices.Contains(markdownExtensions, ext)
}

func shouldSkipDirectory(name string, includeHidden bool) bool {
	// Don't skip current directory "." or parent directory ".."
	if name == "." || name == ".." {
		return false
	}

	if !includeHidden && strings.HasPrefix(name, ".") {
		return true
	}

	// Skip common build/cache directories
	skipDirs := []string{"node_modules", ".git", "vendor", "target", "build", ".next"}
	if slices.Contains(skipDirs, name) {
		return true
	}

	return false
}
