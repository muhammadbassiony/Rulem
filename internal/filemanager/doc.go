// Package filemanager holds rulem's file-handling business logic: what a rule
// file is, which repository it came from, whether an existing file may be
// replaced, what the copy should be called, and what gets written to the audit
// log.
//
// # Architecture
//
// There are exactly two layers, and the split is deliberate:
//
//	filemanager  -  policy. Overwrite rules, destination naming, the markdown
//	                extension list, repository provenance, audit logging.
//	pkg/fileops  -  mechanics and storage policy. Confined directory handles,
//	                atomic copies, name sanitizers, reserved-directory rules.
//
// **This package makes no raw filesystem calls.** Every operation on disk goes
// through a *fileops.Dir - a handle on an open directory, obtained once when
// the directory is chosen, that confines everything addressed through it. The
// only permitted uses of os/filepath here are string arithmetic that never
// touches the disk: deriving a destination name from a source path, and
// rendering an absolute path for display.
//
// The reason is not tidiness. Validating a path as text and then acting on
// that same text leaves a gap in which the filesystem can change underneath
// the check. A handle removes the gap: there is no second lookup to disagree
// with the first.
//
// # The FileManager
//
// A FileManager is a storage directory plus a logger. It is constructed from
// an already-open handle, so a FileManager cannot exist for a directory rulem
// was not allowed to open:
//
//	dir, err := fileops.OpenExistingDir(repo.LocalPath)
//	if err != nil {
//	    return err
//	}
//	defer dir.Close()
//
//	fm, err := filemanager.NewFileManager(dir, logger)
//
// Use fileops.OpenDir when the flow is allowed to create the directory (the
// user is choosing it now) and fileops.OpenExistingDir when it is not (the
// directory was configured earlier and must still be there).
//
// Paths crossing this package's API are relative to a directory handle:
// CopyFileFromStorage and CreateSymlinkFromStorage take a path relative to the
// storage directory, and a destination relative to the working directory.
// FileItem.RelPath carries the former; FileItem.Path is the absolute rendering
// and is for display only.
//
// # Multi-repository support
//
// A FileManager operates on one directory. For multiple repositories, use
// ScanAllRepositories, which opens a handle per repository, tags every file
// with its source repository metadata, and aggregates partial failures:
//
//	prepared, _, err := repository.PrepareAllRepositories(ctx, cfg.Repositories, logger)
//	files, err := filemanager.ScanAllRepositories(prepared, logger)
//
// # Security
//
// Confinement, traversal refusal and symlink policy are properties of
// fileops.Dir, not of this package. What this package contributes is the
// decision about *what* to do: whether an existing file may be overwritten,
// which names are acceptable, and what is recorded. All operations are logged
// for debugging and audit purposes.
package filemanager
