package fileops

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// DirectoryScanOptions configures the behavior of directory scanning operations.
// These options provide fine-grained control over what gets scanned and how.
type DirectoryScanOptions struct {
	// SkipUnreadableDirs determines whether to skip directories that cannot be read
	// or to return an error. Setting to true makes scanning more resilient.
	SkipUnreadableDirs bool

	// MaxDepth limits the maximum recursion depth for directory traversal.
	// This prevents infinite loops and stack overflow from deep directory structures.
	MaxDepth int

	// IncludeHidden determines whether to include files and directories that start with '.'
	// Setting to false will skip hidden files and directories.
	IncludeHidden bool

	// SkipPatterns contains directory names that should be skipped during scanning.
	// These are exact matches against directory names (not full paths).
	SkipPatterns []string

	// FileFilter is an optional function that determines whether a file should be included.
	// If nil, all files are included. If provided, only files for which this returns true are included.
	FileFilter func(filename string) bool

	// DirFilter is an optional function that determines whether a directory should be scanned.
	// If nil, standard skip patterns are used. If provided, this takes precedence.
	DirFilter func(dirname string) bool

	// ValidateFileAccess enables file access validation for each discovered file.
	// Uses ValidateFileAccess from validation.go to ensure files are readable.
	// This is optional for performance reasons in cases where you trust the file system.
	ValidateFileAccess bool
}

// FileInfo represents information about a discovered file during directory scanning.
// This provides a platform-independent view of file metadata.
type FileInfo struct {
	// Name is the base filename without path components
	Name string

	// Path is the relative path from the scan root to this file
	Path string

	// IsDir indicates whether this entry represents a directory
	IsDir bool

	// Size is the file size in bytes (0 for directories)
	Size int64

	// ModTime is the last modification time
	ModTime time.Time

	// Mode contains the file mode and permission bits
	Mode os.FileMode
}

// SecureDirectoryScanner provides secure, configurable directory scanning with
// built-in protection against directory traversal and symlink attacks.
//
// The scanner operates within a security boundary defined by an os.Root,
// preventing access to files outside the designated scan area.
type SecureDirectoryScanner struct {
	// root defines the security boundary for scanning operations
	root *os.Root

	// opts contains the scanning configuration
	opts *DirectoryScanOptions

	// results stores discovered files during scanning
	results []FileInfo

	// visited tracks visited directories to prevent infinite loops
	visited map[string]bool

	// scanRoot stores the absolute path of the scan root for security validation
	scanRoot string
}

// NewDirectoryScanner creates a new secure directory scanner for the given path.
//
// Parameters:
//   - scanPath: The directory path to scan (can be relative or absolute)
//   - opts: Scanning options (if nil, sensible defaults are used)
//
// Returns:
//   - *SecureDirectoryScanner: Configured scanner instance
//   - error: Setup errors including path validation and access issues
//
// Security considerations:
//   - Creates a secure root to prevent directory escapes
//   - Validates the scan path before creating the scanner
//   - Initializes loop detection to prevent symlink attacks
//
// Usage example:
//
//	opts := &fileops.DirectoryScanOptions{
//	    MaxDepth: 10,
//	    IncludeHidden: false,
//	    FileFilter: func(name string) bool {
//	        return strings.HasSuffix(name, ".go")
//	    },
//	}
//	scanner, err := fileops.NewDirectoryScanner("/project/src", opts)
//	if err != nil {
//	    return fmt.Errorf("failed to create scanner: %w", err)
//	}
//	defer scanner.Close()
func NewDirectoryScanner(scanPath string, opts *DirectoryScanOptions) (*SecureDirectoryScanner, error) {
	// Apply default options if none provided
	if opts == nil {
		opts = getDefaultScanOptions()
	}

	// Validate the scan path
	if strings.TrimSpace(scanPath) == "" {
		return nil, fmt.Errorf("scan path cannot be empty")
	}

	// Expand tilde and resolve to absolute path for security
	expandedPath := ExpandPath(scanPath)
	absPath, err := filepath.Abs(expandedPath)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve scan path: %w", err)
	}

	// Validate path security
	if err := ValidatePathSecurity(absPath); err != nil {
		return nil, fmt.Errorf("scan path security validation failed: %w", err)
	}

	// Always block reserved directories - no legitimate use case for scanning system directories
	if IsReservedDirectory(absPath) {
		return nil, fmt.Errorf("cannot scan reserved/system directory: %s", absPath)
	}

	// Check that the path exists and is a directory
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("cannot access scan path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scan path is not a directory: %s", absPath)
	}

	// Create secure root for the scan area
	root, err := os.OpenRoot(absPath)
	if err != nil {
		return nil, fmt.Errorf("cannot create secure scan root: %w", err)
	}

	return &SecureDirectoryScanner{
		root:     root,
		opts:     opts,
		results:  []FileInfo{},
		visited:  make(map[string]bool),
		scanRoot: absPath,
	}, nil
}

// getDefaultScanOptions returns sensible default scanning options.
func getDefaultScanOptions() *DirectoryScanOptions {
	return &DirectoryScanOptions{
		SkipUnreadableDirs: true,
		MaxDepth:           20,
		IncludeHidden:      true,
		SkipPatterns:       getDefaultSkipPatterns(),
		FileFilter:         nil,   // Include all files by default
		DirFilter:          nil,   // Use default directory filtering
		ValidateFileAccess: false, // Disabled by default for performance
	}
}

// getDefaultSkipPatterns returns commonly skipped directory patterns.
func getDefaultSkipPatterns() []string {
	return []string{
		"node_modules",
		".git",
		"vendor",
		"target",
		"build",
		".next",
		"dist",
		".cache",
		"__pycache__",
		".vscode",
		".idea",
	}
}

// Close releases resources associated with the scanner.
// This should be called when the scanner is no longer needed.
func (s *SecureDirectoryScanner) Close() error {
	if s.root != nil {
		err := s.root.Close()
		s.root = nil
		return err
	}
	return nil
}

// ScanDirectory performs a recursive scan of the configured directory.
//
// Returns:
//   - []FileInfo: List of discovered files matching the configured criteria
//   - error: Scanning errors
//
// The scan respects all configured options including depth limits, skip patterns,
// and file filters. The scan is performed securely within the root boundary.
//
// Usage example:
//
//	files, err := scanner.ScanDirectory()
//	if err != nil {
//	    return fmt.Errorf("scan failed: %w", err)
//	}
//	for _, file := range files {
//	    fmt.Printf("Found: %s (%d bytes)\n", file.Path, file.Size)
//	}
func (s *SecureDirectoryScanner) ScanDirectory() ([]FileInfo, error) {
	if s.root == nil {
		return nil, fmt.Errorf("scanner has been closed")
	}

	// Reset state for new scan
	s.results = []FileInfo{}
	s.visited = make(map[string]bool)

	// Start recursive scan from root
	if err := s.scanRecursive(".", 1); err != nil {
		return nil, fmt.Errorf("directory scan failed: %w", err)
	}

	// Return a copy of results to prevent external modification
	resultsCopy := make([]FileInfo, len(s.results))
	copy(resultsCopy, s.results)
	return resultsCopy, nil
}

// scanRecursive performs the actual recursive directory scanning.
func (s *SecureDirectoryScanner) scanRecursive(relativePath string, depth int) error {
	// Check depth limit
	if depth > s.opts.MaxDepth {
		return nil // Silently stop at max depth
	}

	// Clean path and check for loops
	cleanPath := filepath.Clean(relativePath)
	if s.visited[cleanPath] {
		return nil // Skip already visited directory (prevents symlink loops)
	}
	s.visited[cleanPath] = true

	// Check if directory should be skipped
	dirName := filepath.Base(relativePath)
	if s.shouldSkipDirectory(dirName) {
		return nil
	}

	// Open directory within secure root
	dir, err := s.root.Open(relativePath)
	if err != nil {
		if s.opts.SkipUnreadableDirs {
			return nil // Skip unreadable directories
		}
		return fmt.Errorf("failed to open directory %s: %w", relativePath, err)
	}
	defer dir.Close()

	// Read directory entries
	entries, err := dir.ReadDir(-1)
	if err != nil {
		if s.opts.SkipUnreadableDirs {
			return nil
		}
		return fmt.Errorf("failed to read directory %s: %w", relativePath, err)
	}

	// Process each entry
	for _, entry := range entries {
		entryPath := filepath.Join(relativePath, entry.Name())

		if entry.IsDir() {
			// Built-in symlink security validation - always enabled
			fullEntryPath := filepath.Join(s.scanRoot, entryPath)
			if isLink, err := IsSymlink(fullEntryPath); err == nil && isLink {
				// Validate symlink security with scan root as allowed path
				if err := ValidateSymlinkSecurity(fullEntryPath, []string{s.scanRoot}); err != nil {
					if s.opts.SkipUnreadableDirs {
						continue // Skip unsafe symlinks
					}
					return fmt.Errorf("symlink security check failed for %s: %w", entryPath, err)
				}
			}

			// Recursively scan subdirectory
			if err := s.scanRecursive(entryPath, depth+1); err != nil {
				return err
			}
		} else {
			// Process file entry
			if s.shouldIncludeFile(entry.Name()) {
				fileInfo, err := s.createFileInfo(entry, entryPath)
				if err != nil {
					if s.opts.SkipUnreadableDirs {
						continue // Skip files we can't stat
					}
					return fmt.Errorf("failed to get file info for %s: %w", entryPath, err)
				}
				s.results = append(s.results, fileInfo)
			}
		}
	}

	return nil
}

// shouldSkipDirectory determines if a directory should be skipped based on configured rules.
func (s *SecureDirectoryScanner) shouldSkipDirectory(dirName string) bool {
	// Never skip current or parent directory references
	if dirName == "." || dirName == ".." {
		return false
	}

	// Use custom directory filter if provided
	if s.opts.DirFilter != nil {
		return !s.opts.DirFilter(dirName)
	}

	// Skip hidden directories if not including hidden files
	if !s.opts.IncludeHidden && strings.HasPrefix(dirName, ".") {
		return true
	}

	// Check against skip patterns
	return slices.Contains(s.opts.SkipPatterns, dirName)
}

// shouldIncludeFile determines if a file should be included based on configured rules.
func (s *SecureDirectoryScanner) shouldIncludeFile(fileName string) bool {
	// Skip hidden files if not including hidden files
	if !s.opts.IncludeHidden && strings.HasPrefix(fileName, ".") {
		return false
	}

	// Apply custom file filter if provided
	if s.opts.FileFilter != nil {
		return s.opts.FileFilter(fileName)
	}

	// Include all files by default
	return true
}

// createFileInfo creates a FileInfo struct from directory entry information.
func (s *SecureDirectoryScanner) createFileInfo(entry os.DirEntry, path string) (FileInfo, error) {
	info, err := entry.Info()
	if err != nil {
		return FileInfo{}, fmt.Errorf("failed to get file info: %w", err)
	}

	// Optional file access validation for enhanced security
	if s.opts.ValidateFileAccess && !entry.IsDir() {
		// Construct full path for validation (root.Open gives us a relative path)
		// We use the scan root + relative path to get the full path
		fullPath := filepath.Join(s.scanRoot, path)
		if err := ValidateFileAccess(fullPath, false); err != nil {
			return FileInfo{}, fmt.Errorf("file access validation failed: %w", err)
		}
	}

	return FileInfo{
		Name:    entry.Name(),
		Path:    path,
		IsDir:   entry.IsDir(),
		Size:    info.Size(),
		ModTime: info.ModTime(),
		Mode:    info.Mode(),
	}, nil
}

// GetResults returns the current scan results without performing a new scan.
// This is useful when you want to access results from a previous scan.
func (s *SecureDirectoryScanner) GetResults() []FileInfo {
	resultsCopy := make([]FileInfo, len(s.results))
	copy(resultsCopy, s.results)
	return resultsCopy
}

// GetStats returns statistics about the last scan operation.
type ScanStats struct {
	TotalFiles       int
	TotalDirectories int
	SkippedDirs      int
	LargestFile      int64
	TotalSize        int64
}

// GetScanStats calculates and returns statistics about the current scan results.
func (s *SecureDirectoryScanner) GetScanStats() ScanStats {
	stats := ScanStats{}

	for _, file := range s.results {
		if file.IsDir {
			stats.TotalDirectories++
		} else {
			stats.TotalFiles++
			stats.TotalSize += file.Size
			if file.Size > stats.LargestFile {
				stats.LargestFile = file.Size
			}
		}
	}

	return stats
}

// ScanWithFilter is a convenience function that creates a scanner with a file filter
// and immediately performs a scan.
//
// Parameters:
//   - scanPath: Directory to scan
//   - fileFilter: Function to filter files (nil to include all)
//   - maxDepth: Maximum recursion depth
//
// Returns:
//   - []FileInfo: Discovered files
//   - error: Any scanning errors
//
// Usage example:
//
//	goFiles, err := fileops.ScanWithFilter("/project", func(name string) bool {
//	    return strings.HasSuffix(name, ".go")
//	}, 10)
func ScanWithFilter(scanPath string, fileFilter func(string) bool, maxDepth int) ([]FileInfo, error) {
	opts := &DirectoryScanOptions{
		SkipUnreadableDirs: true,
		MaxDepth:           maxDepth,
		IncludeHidden:      false,
		SkipPatterns:       getDefaultSkipPatterns(),
		FileFilter:         fileFilter,
		ValidateFileAccess: false, // Disabled by default for performance
	}

	scanner, err := NewDirectoryScanner(scanPath, opts)
	if err != nil {
		return nil, err
	}
	defer scanner.Close()

	return scanner.ScanDirectory()
}
