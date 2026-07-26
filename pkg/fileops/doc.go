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
// This is why the package no longer offers a "combine these validators in this
// order" recipe. That recipe existed to close the gap between proving
// something about a path string and then acting on that string; holding the
// boundary open removes the gap instead of policing it.
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
//   - Name sanitizers - [SanitizeFilename] and [SanitizeRelativePath] for the
//     filesystem, [SanitizeIdentifier] for protocol identifiers.
//   - [AtomicCopy] - unguessable temporary name, O_EXCL, source permissions
//     preserved, fsync, atomic rename, cleanup on every failure path. The
//     destination either appears complete or is not touched.
//   - Content and size limits - [ValidateContentSecurity] rejects control
//     characters, [ValidateFileSizeLimit] caps a read.
//   - Confined walking - [Dir.Scan] over [fs.WalkDir]; the caller supplies the
//     filter.
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
package fileops
