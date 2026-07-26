package filemanager

import (
	"fmt"
	"path/filepath"
	"rulem/internal/logging"
	"rulem/internal/repository"
	"rulem/pkg/fileops"
	"slices"
	"strings"
)

// markdownExtensions contains supported markdown file extensions.
//
// This is rulem's definition of "a rule file" and it stays in this package.
// fileops takes a func(name string) bool and has no opinion about which files
// are interesting.
var markdownExtensions = []string{
	".md", ".mdown", ".mkdn", ".mkd", ".markdown", ".mdc",
}

// isMarkdownFile checks if a filename has a markdown extension.
// It is passed to fileops as the scan filter.
func isMarkdownFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return slices.Contains(markdownExtensions, ext)
}

// markdownScanOptions describes the walk rulem wants when looking for rule
// files. None of these are security settings - confinement comes from the
// directory handle the walk runs inside - they are about which of the files
// inside the boundary are worth reporting.
func markdownScanOptions(maxDepth int) *fileops.DirectoryScanOptions {
	return &fileops.DirectoryScanOptions{
		SkipUnreadableDirs: true,
		MaxDepth:           maxDepth,
		IncludeHidden:      true,
		SkipPatterns:       []string{"node_modules", ".git", "vendor", "target", "build", ".next", "dist", ".cache", "__pycache__", ".vscode", ".idea"},
		FileFilter:         isMarkdownFile,
	}
}

// toFileItems converts a scan result into rule-file items.
//
// dir supplies both ends of the pair every item carries: RelPath is the name
// to address the file by, and Path is the same file rendered absolutely for
// display and filtering.
func toFileItems(dir *fileops.Dir, files []fileops.FileInfo) []FileItem {
	var result []FileItem
	for _, file := range files {
		if file.IsDir {
			continue
		}
		result = append(result, FileItem{
			Name:    file.Name,
			RelPath: file.Path,
			Path:    dir.DisplayPath(file.Path),
		})
	}
	return result
}

// ScanCurrDirectory recursively scans the current working directory for
// markdown files.
//
// It is a package-level function, not a FileManager method: it has nothing to
// do with the storage directory, and pretending otherwise previously forced
// callers to construct a FileManager they had no use for.
//
// Returns:
//   - []FileItem: discovered markdown files, addressed relative to the working
//     directory and rendered absolutely for display
//   - error: scanning errors
func ScanCurrDirectory() ([]FileItem, error) {
	cwd, err := fileops.OpenWorkingDir()
	if err != nil {
		return nil, fmt.Errorf("failed to open current working directory: %w", err)
	}
	defer func() { _ = cwd.Close() }()

	files, err := cwd.Scan(markdownScanOptions(20))
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	result := toFileItems(cwd, files)
	logging.Debug("Scanned current directory for markdown files", "fileCount", len(result))
	return result, nil
}

// ScanRepository recursively scans this FileManager's storage directory for
// markdown files.
//
// Everything that used to precede the walk - resolving the path, checking
// whether it was a symlink, re-validating it against storage policy, statting
// it - was proving facts about a string. The handle already carries them: it
// could not have been opened otherwise.
//
// Returns:
//   - []FileItem: discovered markdown files, addressed relative to the storage
//     root and rendered absolutely for display
//   - error: scanning errors
func (fm *FileManager) ScanRepository() ([]FileItem, error) {
	if fm == nil {
		return nil, fmt.Errorf("filemanager is nil")
	}

	files, err := fm.dir.Scan(markdownScanOptions(50))
	if err != nil {
		return nil, fmt.Errorf("failed to scan storage directory: %w", err)
	}

	result := toFileItems(fm.dir, files)
	logging.Debug("Scanned central storage for markdown files", "fileCount", len(result))
	return result, nil
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
//	prepared, err := repository.PrepareAllRepositories(ctx, cfg.Repositories, logger)
//	files, err := filemanager.ScanAllRepositories(prepared, logger)
//	for _, file := range files {
//	    fmt.Printf("%s from %s (%s)\n", file.Name, file.RepositoryName, file.RepositoryType)
//	}
//
// A repository directory that has since been deleted or replaced by a file
// fails to open and is reported as a scan error rather than being silently
// recreated - that is the point of OpenExistingDir.
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
		// Repositories that failed preparation (e.g. missing local path) are
		// listed for repair in settings but have nothing to scan.
		if !prep.IsAvailable() {
			if logger != nil {
				logger.Info("Skipping unavailable repository",
					"repository_id", prep.ID(),
					"repository_name", prep.Name(),
				)
			}
			continue
		}
		if logger != nil {
			logger.Info("Scanning repository",
				"repository_id", prep.ID(),
				"repository_name", prep.Name(),
				"repository_type", string(prep.Type()),
				"path", prep.LocalPath,
			)
		}

		files, err := scanOneRepository(prep.LocalPath, logger)
		if err != nil {
			scanErrors = append(scanErrors, fmt.Sprintf("repository %s (%s): %v", prep.ID(), prep.Name(), err))
			if logger != nil {
				logger.Error("Repository scan failed", "repository_id", prep.ID(), "error", err)
			}
			continue
		}

		// Tag each file with repository metadata
		repoType := string(prep.Type())
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

// scanOneRepository opens one repository directory, scans it, and closes the
// handle again. It exists so the handle's lifetime is a single scope rather
// than a deferred close inside a loop.
func scanOneRepository(localPath string, logger *logging.AppLogger) ([]FileItem, error) {
	dir, err := fileops.OpenExistingDir(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository directory: %w", err)
	}
	defer func() { _ = dir.Close() }()

	fm, err := NewFileManager(dir, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create file manager: %w", err)
	}

	return fm.ScanRepository()
}
