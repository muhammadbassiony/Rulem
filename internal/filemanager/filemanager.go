package filemanager

import (
	"fmt"
	"path/filepath"
	"rulem/internal/logging"
	"rulem/pkg/fileops"
)

// FileManager applies rulem's file-handling policy to one storage directory.
//
// It holds an open handle on that directory rather than its path. Everything
// below therefore splits cleanly in two: the decisions (may this be
// overwritten? what should the copy be called? what gets logged?) live here,
// and the filesystem work happens through the handle.
type FileManager struct {
	logger *logging.AppLogger

	// dir is the storage directory, already opened and confined. Holding the
	// handle instead of a string is what makes "inside the storage directory"
	// a property of the operation rather than a claim about a path.
	dir *fileops.Dir
}

// NewFileManager wraps an open storage directory in rulem's file policy.
//
// The handle is supplied by the caller, which is where the choice of whether
// the directory may be created belongs:
//
//	fileops.OpenDir         - the user is choosing this directory now; create it.
//	fileops.OpenExistingDir - it was configured earlier; it must already exist.
//
// The FileManager does not take ownership of the handle: whoever opened it
// closes it.
func NewFileManager(dir *fileops.Dir, logger *logging.AppLogger) (*FileManager, error) {
	if dir == nil {
		return nil, fmt.Errorf("storage directory handle is required")
	}

	return &FileManager{
		logger: logger,
		dir:    dir,
	}, nil
}

// CopyFileToStorage copies a file into the storage directory.
//
// Parameters:
//   - srcPath: source file, an ordinary path outside every boundary - it is
//     whatever file the user picked, and opening it is what validates it
//   - newFileName: destination name relative to the storage root, optionally
//     containing subdirectories ("backend/api-rules.md"); nil keeps the
//     source's base name
//   - overwrite: whether an existing destination may be replaced
//
// Returns the absolute destination path, for display.
//
// Policy applied here: the destination name, and whether an existing file may
// be replaced. The copy itself is atomic and confined to the storage
// directory - the destination either appears complete or not at all, and a
// name that would leave the directory cannot be written.
func (fm *FileManager) CopyFileToStorage(srcPath string, newFileName *string, overwrite bool) (string, error) {
	// POLICY - destination naming.
	// A supplied name may contain forward-slash separated subdirectories;
	// SanitizeRelativePath validates each segment and rejects traversal,
	// absolute and empty segments. With no name supplied, the source file
	// keeps its own. Both are string arithmetic - nothing is touched on disk.
	var fileName string
	if newFileName != nil {
		cleanName, err := fileops.SanitizeRelativePath(*newFileName)
		if err != nil {
			return "", fmt.Errorf("invalid filename: %w", err)
		}
		fileName = cleanName
	} else {
		fileName = filepath.Base(srcPath)
	}

	// POLICY - overwrite. Exists reports anything occupying the name,
	// including a broken symlink, which still occupies it.
	if fm.dir.Exists(fileName) {
		if !overwrite {
			return "", fmt.Errorf("destination file already exists: %s (use overwrite=true to replace)", fileName)
		}
		fm.logger.Debug("Overwriting existing file", "dest", fm.dir.DisplayPath(fileName))
	}

	// MECHANICS - atomic copy through the storage handle, which creates any
	// intermediate subdirectories the name asks for.
	if err := fm.dir.CopyFrom(srcPath, fileName); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	destPath := fm.dir.DisplayPath(fileName)
	fm.logger.Info("File copied successfully", "src", srcPath, "dest", destPath)
	return destPath, nil
}

// CopyFileFromStorage copies a file out of the storage directory into the
// current working directory.
//
// Parameters:
//   - storagePath: source file, relative to the storage root (FileItem.RelPath)
//   - destPath: destination, relative to the working directory
//   - overwrite: whether an existing destination may be replaced
//
// Returns the absolute destination path, for display.
//
// Both ends are directory handles, so neither the source nor the destination
// can be argued out of its directory by a crafted name or a symlinked
// subdirectory.
func (fm *FileManager) CopyFileFromStorage(storagePath string, destPath string, overwrite bool) (string, error) {
	fm.logger.Debug("Copying file from storage", "src", storagePath, "dest", destPath, "overwrite", overwrite)

	cwd, err := fileops.OpenWorkingDir()
	if err != nil {
		return "", fmt.Errorf("cannot open current working directory: %w", err)
	}
	defer func() { _ = cwd.Close() }()

	// POLICY - overwrite.
	if cwd.Exists(destPath) {
		if !overwrite {
			return "", fmt.Errorf("destination file already exists: %s (use overwrite=true to replace)", destPath)
		}
		fm.logger.Debug("Overwriting existing file", "dest", cwd.DisplayPath(destPath))
	}

	// MECHANICS - atomic copy from one handle to the other.
	if err := fm.dir.CopyTo(storagePath, cwd, destPath); err != nil {
		return "", fmt.Errorf("failed to copy file from storage: %w", err)
	}

	absDestPath := cwd.DisplayPath(destPath)
	fm.logger.Info("File copied from storage successfully", "src", fm.dir.DisplayPath(storagePath), "dest", absDestPath)
	return absDestPath, nil
}

// CreateSymlinkFromStorage links a file in the storage directory into the
// current working directory instead of copying it, so that editing either side
// edits the same file.
//
// Parameters:
//   - storagePath: target file, relative to the storage root (FileItem.RelPath)
//   - destPath: link location, relative to the working directory
//   - overwrite: whether an existing destination may be replaced
//
// Returns the absolute path of the created link, for display.
//
// The link is written with a relative target so the pair survives the tree
// being moved.
func (fm *FileManager) CreateSymlinkFromStorage(storagePath string, destPath string, overwrite bool) (string, error) {
	cwd, err := fileops.OpenWorkingDir()
	if err != nil {
		return "", fmt.Errorf("cannot open current working directory: %w", err)
	}
	defer func() { _ = cwd.Close() }()

	// POLICY - overwrite. A symlink cannot be created over an existing name,
	// so replacing means removing first; that is a decision, and it is made
	// here.
	if cwd.Exists(destPath) {
		if !overwrite {
			return "", fmt.Errorf("destination file already exists: %s (use overwrite=true to replace)", destPath)
		}
		if err := cwd.Remove(destPath); err != nil {
			return "", fmt.Errorf("cannot remove existing destination: %w", err)
		}
		fm.logger.Debug("Removed existing file for symlink", "dest", cwd.DisplayPath(destPath))
	}

	// MECHANICS - the target is proven to be a regular file inside storage and
	// the link is created through the working directory's handle.
	if err := fm.dir.SymlinkTo(storagePath, cwd, destPath); err != nil {
		return "", fmt.Errorf("failed to create symlink: %w", err)
	}

	absDestPath := cwd.DisplayPath(destPath)
	fm.logger.Info("Symlink created successfully", "target", fm.dir.DisplayPath(storagePath), "link", absDestPath)
	return absDestPath, nil
}

// GetStorageDir returns the storage directory path for display.
//
// It is the path behind the handle, with no boundary attached to it. Do not
// join a name onto it and open the result - address files through the
// FileManager instead.
func (fm *FileManager) GetStorageDir() string {
	return fm.dir.Path()
}
