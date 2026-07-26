// Package fileops is rulem's storage-policy module: where the application may
// keep files, what those files may be called, and how bytes get moved between
// them safely.
//
// # The model: a confined handle, not a validated string
//
// The centre of the package is [Dir], a handle on a directory that rulem is
// allowed to use. Open it once from a user-supplied path with [OpenDir], then
// address everything inside it by a relative name:
//
//	dir, err := fileops.OpenDir("~/rules")
//	if err != nil {
//	    return err
//	}
//	defer dir.Close()
//
//	files, err := dir.Scan(nil)             // walk it
//	content, err := dir.ReadFile("api.md")  // read one file
//	err = dir.CopyFrom(picked, "api.md")    // copy one in, atomically
//
// A Dir wraps an [os.Root]. Names are resolved against an open directory
// handle, one component at a time, so a path that would leave the tree fails
// at the syscall itself. Confinement is *held*, not re-proved: there is no
// separate check that could be forgotten, go stale, or turn out to be
// unreachable, because there is no separate check.
//
// There are three constructors, because "open a directory" is three questions:
// [OpenDir] when the user is choosing the directory now (it is created),
// [OpenExistingDir] when it was configured earlier (a missing one is reported,
// not conjured), and [OpenWorkingDir] for the directory the shell chose, which
// gets confinement but no storage policy.
//
// The one discipline the type system cannot enforce: [Dir.Path] and
// [Dir.DisplayPath] return ordinary strings for showing to a user. Never feed
// one back into an operation.
//
// # What else is here
//
// Everything that is genuinely rulem's policy rather than the standard
// library's job:
//
//   - Storage policy - [ValidateStoragePath], [IsReservedDirectory] and the
//     per-OS lists of directories an application has no business writing to.
//     [ValidatePathSecurity] applies the same policy to a path that does not
//     exist yet and so cannot be opened.
//   - Name sanitizers - [SanitizeFilename] and [SanitizeRelativePath] for the
//     filesystem, [SanitizeIdentifier] for protocol identifiers.
//   - [AtomicCopy] - unguessable temporary name, O_EXCL, source permissions
//     preserved, fsync, atomic rename, cleanup on every failure path. The
//     destination either appears complete or is not touched.
//   - Content limits - [ValidateContentSecurity] rejects control characters.
//   - Confined walking - [Dir.Scan] over [io/fs.WalkDir]; the caller supplies
//     the filter.
//
// # Scope
//
// This package deals in directories, names, bytes and permissions. It has no
// notion of what any file *means* - which files are interesting is expressed
// by the caller as a func(name string) bool, and decisions like "may this be
// overwritten?" or "what should the destination be called?" belong to the
// caller. Domain vocabulary stays out of this package entirely.
//
// It is not a security library. [os.Root] is, and its limits are its own: it
// does not defend against mount points, bind mounts, /proc special files or
// device nodes, and Chmod/Chown/Chtimes race on Unix.
//
// # Migration note: what used to be here, and why it is not
//
// This package once exported nine path validators and four symlink helpers:
// ValidateFileInDirectory, ValidateCWDPath, ValidatePathInHome,
// ValidateFileAccess, ValidateSymlinkSecurity, IsSymlink, ResolveSymlink,
// CreateRelativeSymlink and SecureDirectoryScanner. They are gone, and they
// should not come back.
//
// They existed because when this package was written there was no obvious
// alternative: they proved things about path *strings* and let a later syscall
// act on the same string. [os.Root] (Go 1.24, completed in 1.25) closed that
// gap by making confinement a property of an open handle, and
// [path/filepath.IsLocal] (Go 1.20) had already replaced the lexical half. See
// [Traversal-resistant file APIs] for the reasoning.
//
// The mistake worth remembering is not that the validators were written, but
// that they were stacked *around* [os.Root] - which the scanner already used -
// without asking whether it subsumed them. It did, and two of the guards turned
// out to be unreachable code that nobody noticed because the layer underneath
// was quietly doing the job. Defence in depth costs nothing only when each
// layer is known to be reachable.
//
// So: if you find yourself about to add a function that takes a path string and
// returns an error about where that path points, open a [Dir] instead.
//
// [Traversal-resistant file APIs]: https://go.dev/blog/osroot
package fileops
