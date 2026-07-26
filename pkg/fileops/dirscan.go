package fileops

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// DirectoryScanOptions configures a directory walk.
//
// None of these options is a security mechanism. Confinement comes from the
// os.Root the walk runs inside: names are resolved against an open directory
// handle, so nothing outside it can be reached whatever the options say. What
// is left here is ergonomics - which of the files inside the boundary the
// caller wants to hear about.
type DirectoryScanOptions struct {
	// SkipUnreadableDirs decides what an unreadable directory or file means:
	// skip it and carry on (true), or abort the whole walk (false).
	//
	// This is a resilience choice, not a safety one. A scan of a user's storage
	// directory should survive one permission-denied subdirectory.
	SkipUnreadableDirs bool

	// MaxDepth limits how many directory levels are descended. The scan root
	// itself is level 1, so MaxDepth: 1 reports only the files sitting directly
	// in it.
	//
	// This is a cost bound, not a loop guard: the walk never descends a
	// symlinked directory, so a symlink loop cannot be entered in the first
	// place.
	MaxDepth int

	// IncludeHidden determines whether entries whose name starts with '.' are
	// reported, and whether hidden directories are descended.
	IncludeHidden bool

	// SkipPatterns lists directory names that are not descended. These are
	// exact matches against the directory's own name, not against a full path.
	SkipPatterns []string

	// FileFilter decides which files are reported. If nil, every file is.
	// This package has no opinion about which files are interesting; the
	// caller supplies one.
	FileFilter func(filename string) bool

	// DirFilter decides which directories are descended. If non-nil it takes
	// precedence over IncludeHidden and SkipPatterns.
	DirFilter func(dirname string) bool

	// ValidateFileAccess reports only files that can actually be opened.
	// Off by default: it costs an open per file, and a caller that is about to
	// read the files anyway learns the same thing from the read.
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

// walkFiles walks root with fs.WalkDir and collects the files it finds.
//
// Root.FS() exposes the boundary as an fs.FS, and fs.WalkDir never follows a
// symlink: a link is handed to the walk function as a leaf, whatever it points
// at. Those two facts together are the whole safety argument. Nothing here can
// name a file outside root, and nothing here can be lured into a symlink loop,
// so the walk itself contains no containment logic - only the caller's
// preferences about what to report.
//
// A link is resolved deliberately, once, with fs.Stat: that follows it *inside
// the boundary*, so a link to a file in the root is reported with the target's
// size and mode, and a link that escapes, dangles or is absolute simply fails
// to resolve and is dropped. Dropping it rather than failing the walk matters:
// an out-of-root symlink is an ordinary thing to find on disk, and letting one
// abort the scan would hand anyone who can write to the directory a way to
// break scanning entirely.
func walkFiles(root *os.Root, opts *DirectoryScanOptions) ([]FileInfo, error) {
	if opts == nil {
		opts = getDefaultScanOptions()
	}

	fsys := root.FS()
	results := []FileInfo{}

	err := fs.WalkDir(fsys, ".", func(entryPath string, entry fs.DirEntry, err error) error {
		if err != nil {
			// The directory at entryPath could not be read.
			if opts.SkipUnreadableDirs {
				return nil
			}
			return fmt.Errorf("failed to read directory %s: %w", entryPath, err)
		}

		if entry.IsDir() {
			if skipDirectory(opts, entryPath, entry.Name()) {
				return fs.SkipDir
			}
			return nil
		}

		if !includeFile(opts, entry.Name()) {
			return nil
		}

		info, err := statEntry(fsys, entryPath, entry)
		if err != nil {
			if opts.SkipUnreadableDirs {
				return nil
			}
			return fmt.Errorf("failed to get file info for %s: %w", entryPath, err)
		}
		if info == nil {
			return nil // A symlink that does not resolve inside the root.
		}

		if opts.ValidateFileAccess {
			file, err := root.Open(entryPath)
			if err != nil {
				if opts.SkipUnreadableDirs {
					return nil
				}
				return fmt.Errorf("file access validation failed: %w", err)
			}
			_ = file.Close()
		}

		results = append(results, FileInfo{
			Name: entry.Name(),
			// fs paths are always slash-separated; callers join this onto
			// native paths, so hand it back in the native form.
			Path:    filepath.FromSlash(entryPath),
			IsDir:   info.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			Mode:    info.Mode(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return results, nil
}

// statEntry returns the metadata to report for a non-directory entry, or a nil
// FileInfo when the entry is a symlink that should not be reported at all.
func statEntry(fsys fs.FS, entryPath string, entry fs.DirEntry) (fs.FileInfo, error) {
	if entry.Type()&fs.ModeSymlink == 0 {
		return entry.Info()
	}

	// Resolving through fsys keeps the answer inside the boundary.
	info, err := fs.Stat(fsys, entryPath)
	if err != nil {
		return nil, nil // Escapes the root, is absolute, or dangles.
	}
	if info.IsDir() {
		return nil, nil // Not reported as a file; WalkDir has not descended it.
	}
	return info, nil
}

// skipDirectory reports whether a directory should not be descended.
func skipDirectory(opts *DirectoryScanOptions, entryPath, dirName string) bool {
	// The scan root is always descended - depth permitting.
	if entryPath == "." {
		return depthOf(entryPath) > opts.MaxDepth
	}

	if depthOf(entryPath) > opts.MaxDepth {
		return true
	}

	if opts.DirFilter != nil {
		return !opts.DirFilter(dirName)
	}

	if !opts.IncludeHidden && strings.HasPrefix(dirName, ".") {
		return true
	}

	return slices.Contains(opts.SkipPatterns, dirName)
}

// depthOf converts a walk path into a 1-based directory level: the scan root
// is level 1, its immediate subdirectories are level 2, and so on.
func depthOf(entryPath string) int {
	if entryPath == "." {
		return 1
	}
	return strings.Count(entryPath, "/") + 2
}

// includeFile reports whether a file should appear in the results.
func includeFile(opts *DirectoryScanOptions, fileName string) bool {
	if !opts.IncludeHidden && strings.HasPrefix(fileName, ".") {
		return false
	}
	if opts.FileFilter != nil {
		return opts.FileFilter(fileName)
	}
	return true
}
