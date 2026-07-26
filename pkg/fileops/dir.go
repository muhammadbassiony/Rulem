package fileops

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Dir is a confined handle on a directory rulem is allowed to use for storage.
//
// # Why a handle
//
// Validating a path as *text* - a caller passes a string, a validator proves
// something about it, and a separate syscall then acts on that same string -
// leaves a gap. Everything that goes wrong lives in that gap: a symlink
// swapped in between the two, a component that resolves somewhere else, a
// check that turns out to be unreachable. A Dir closes the gap by holding the
// boundary open. os.OpenRoot is called once, when the path is first accepted,
// and every later operation is named *relative to that open handle*.
//
// # Confinement is held, not re-proved
//
// os.Root resolves each component against an open directory, so a name that
// would leave the tree fails at the syscall rather than being caught by a
// preceding check. There is no separate check to keep in sync, and no window
// between checking and using. What remains in this file is not containment: it
// is the small set of *lexical* rules about which relative names a Dir accepts
// at all (non-empty, relative, no ".." component), which exist so that callers
// get a clear error instead of a confusing ENOENT.
//
// The rule for callers is simple: get a Dir once, carry it, and address
// everything inside it by a relative path. Never take Path() and rebuild an
// absolute path to hand to os.Open - that is the shape this type exists to
// remove.
//
// # Scope
//
// A Dir knows about directories, names, bytes and permissions. It has no
// opinion about what a file means: filtering is supplied by the caller as a
// func(name string) bool, and policy decisions (overwrite? what to call the
// destination?) belong to the caller, not here.
//
// A Dir is not safe for concurrent Close with other operations. Concurrent
// reads and writes through one are as safe as the underlying filesystem calls.
type Dir struct {
	// root is the open boundary. It is the reason a Dir is a capability and
	// not a path, and it performs every operation below.
	root *os.Root
}

// OpenDir validates a user-supplied directory path against rulem's storage
// policy and returns an open, confined handle on it.
//
// It composes, in order, exactly what the application does today before it
// touches a storage directory:
//
//  1. ExpandPath      - "~" is a shell convention, so expand it first.
//  2. ValidateStoragePath - non-empty; no ".." component; absolute or below
//     "~"; not a reserved/system directory, before or after symlink
//     resolution; parent directory must exist.
//  3. ValidateDirectoryWritable - creates the directory if missing, then
//     proves writability with a randomly named O_EXCL probe that is removed
//     again.
//  4. os.OpenRoot     - takes the boundary and holds it open.
//
// Note that step 3 CREATES the directory when it does not exist. That matches
// the setup and settings flows, which have always created the storage
// directory on acceptance; a caller that must not create anything should stat
// the path itself first.
//
// The caller owns the returned handle and must Close it.
func OpenDir(userPath string) (*Dir, error) {
	// ValidateStoragePath and ValidateDirectoryWritable each trim and expand
	// internally; expanding here as well keeps the path handed to os.OpenRoot
	// - and therefore Path() - identical to the one they validated.
	expanded := ExpandPath(strings.TrimSpace(userPath))

	if err := ValidateStoragePath(userPath); err != nil {
		return nil, err
	}

	if err := ValidateDirectoryWritable(expanded); err != nil {
		return nil, err
	}

	root, err := os.OpenRoot(expanded)
	if err != nil {
		return nil, fmt.Errorf("cannot open directory: %w", err)
	}

	return &Dir{root: root}, nil
}

// OpenExistingDir is OpenDir without the creation step: the directory must
// already exist.
//
// The distinction is a product one, not a safety one. Choosing a storage
// directory is the moment an application is allowed to bring one into being -
// that is OpenDir. Using a directory that was configured earlier is not: if a
// configured directory has been deleted or moved, the user must be told, not
// handed a silently recreated empty one. That is this constructor.
//
// It applies the same storage policy as OpenDir (expansion, no traversal, not
// a reserved system directory) and the same confinement, and it deliberately
// does NOT run the writability probe: a read-only directory can still be
// listed and read from, and an attempt to write to one fails at the write.
//
// The caller owns the returned handle and must Close it.
func OpenExistingDir(userPath string) (*Dir, error) {
	expanded := ExpandPath(strings.TrimSpace(userPath))

	if err := ValidateStoragePath(userPath); err != nil {
		return nil, err
	}

	info, err := os.Stat(expanded)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("directory does not exist: %s", expanded)
		}
		return nil, fmt.Errorf("cannot access directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", expanded)
	}

	root, err := os.OpenRoot(expanded)
	if err != nil {
		return nil, fmt.Errorf("cannot open directory: %w", err)
	}

	return &Dir{root: root}, nil
}

// OpenWorkingDir returns a confined handle on the process's current working
// directory.
//
// It takes no path on purpose: the only directory it can ever open is the one
// the user's shell already chose, so there is nothing here to talk into
// opening somewhere else.
//
// Unlike OpenDir it applies no storage policy. The reserved-directory rules
// answer "where may this application keep its own data?", and the working
// directory is not that: nothing is stored there, files the user explicitly
// asked for are copied into the directory they explicitly ran the command
// from. Refusing to work in an unusual directory would break that flow without
// protecting anything. Confinement still applies in full - nothing outside the
// working directory can be reached through the returned handle.
//
// The caller owns the returned handle and must Close it.
func OpenWorkingDir() (*Dir, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("cannot determine current working directory: %w", err)
	}

	root, err := os.OpenRoot(cwd)
	if err != nil {
		return nil, fmt.Errorf("cannot open current working directory: %w", err)
	}

	return &Dir{root: root}, nil
}

// Path returns the directory this handle is confined to.
//
// FOR DISPLAY ONLY. The returned string is an ordinary path with no boundary
// attached to it; joining a relative name onto it and passing the result to
// os.Open reintroduces precisely the check-then-use gap this type removes.
// Address files through the Dir instead.
func (d *Dir) Path() string {
	if d.root == nil {
		return ""
	}
	return d.root.Name()
}

// DisplayPath renders the absolute path of rel for showing to a user.
//
// FOR DISPLAY ONLY, for the same reason as Path. Callers that must show a user
// where a file lives need an absolute path; this is the one sanctioned way to
// build one, so that the discipline has a single documented home.
func (d *Dir) DisplayPath(rel string) string {
	return filepath.Join(d.Path(), rel)
}

// Close releases the directory handle. It is safe to call more than once.
func (d *Dir) Close() error {
	if d.root == nil {
		return nil
	}
	err := d.root.Close()
	d.root = nil
	return err
}

// check applies the lexical rules for a name addressed inside this directory:
// the handle must be open, and the name must be a non-empty relative path with
// no ".." component.
//
// None of this is containment - os.Root provides that, and would refuse an
// escape whatever this function returned. It exists so that an obviously wrong
// name produces "path traversal not allowed" rather than a bare ENOENT from
// several layers down, and so that "" and absolute paths are named as the
// mistakes they are.
func (d *Dir) check(rel string) error {
	if d.root == nil {
		return fmt.Errorf("directory handle is closed")
	}
	if rel == "" {
		return fmt.Errorf("path cannot be empty")
	}
	if filepath.IsAbs(rel) {
		return fmt.Errorf("path must be relative to the directory: %q", rel)
	}
	// Reject ".." components outright rather than only checking where the path
	// lexically lands: "valid/../file.txt" is a name whose meaning depends on
	// what "valid" turns out to be, and there is no reason to accept it.
	if hasParentTraversal(rel) || !filepath.IsLocal(rel) {
		return fmt.Errorf("path traversal not allowed: %q", rel)
	}
	return nil
}

// Stat reports metadata for a regular file inside this directory.
//
// Directories are rejected with "path is a directory, not a file" - enumerating
// one is what Scan is for. Symlinks are followed, but only within the boundary:
// a link leaving this directory fails to resolve rather than being followed
// out of it.
func (d *Dir) Stat(rel string) (fs.FileInfo, error) {
	if err := d.check(rel); err != nil {
		return nil, err
	}

	info, err := d.root.Stat(rel)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file does not exist: %s", filepath.Base(rel))
		}
		return nil, fmt.Errorf("cannot access file within directory: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file")
	}
	return info, nil
}

// Exists reports whether rel names anything at all inside this directory,
// including a directory or a broken symlink.
//
// It answers "is something already here?" - the question asked before writing
// - so it must not follow symlinks; a dangling link still occupies the name.
func (d *Dir) Exists(rel string) bool {
	if err := d.check(rel); err != nil {
		return false
	}
	_, err := d.root.Lstat(rel)
	return err == nil
}

// IsSymlink reports whether rel is itself a symbolic link, without following it.
func (d *Dir) IsSymlink(rel string) (bool, error) {
	if err := d.check(rel); err != nil {
		return false, err
	}
	info, err := d.root.Lstat(rel)
	if err != nil {
		return false, fmt.Errorf("failed to stat path: %w", err)
	}
	return info.Mode()&fs.ModeSymlink != 0, nil
}

// Open opens a regular file inside this directory for reading.
//
// Opening IS the check: the file is proven to exist, to be a regular file and
// to resolve inside the boundary, and the handle you get back is the one that
// was checked. There is no separate "can I read this later?" question to ask.
func (d *Dir) Open(rel string) (*os.File, error) {
	file, _, err := d.openFile(rel)
	return file, err
}

// openFile opens rel and returns it alongside the metadata of the handle that
// was actually opened, so callers never have to stat the name a second time.
func (d *Dir) openFile(rel string) (*os.File, fs.FileInfo, error) {
	if err := d.check(rel); err != nil {
		return nil, nil, err
	}

	file, err := d.root.Open(rel)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("file does not exist: %s", filepath.Base(rel))
		}
		return nil, nil, fmt.Errorf("cannot open file within directory: %w", err)
	}

	// Opening a directory succeeds on Unix, so the type check has to happen on
	// the open handle - which is also the only way to be sure it describes the
	// file that was opened.
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, nil, fmt.Errorf("cannot stat file within directory: %w", err)
	}
	if info.IsDir() {
		_ = file.Close()
		return nil, nil, fmt.Errorf("path is a directory, not a file")
	}

	return file, info, nil
}

// ReadFile reads the whole contents of a regular file inside this directory.
// Callers that care about size must Stat first; this package's size policy is
// ValidateFileSizeLimit.
func (d *Dir) ReadFile(rel string) ([]byte, error) {
	f, err := d.Open(rel)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return io.ReadAll(f)
}

// Remove deletes rel from this directory. It does not remove directories
// recursively.
func (d *Dir) Remove(rel string) error {
	if err := d.check(rel); err != nil {
		return err
	}
	return d.root.Remove(rel)
}

// destination prepares rel for writing: it creates any missing parent
// directories through the boundary and returns a root scoped to the parent
// directory together with the final name.
//
// The returned root is always a fresh handle, even when rel names a file
// directly in this directory, so the caller can close it unconditionally
// without closing the Dir.
func (d *Dir) destination(rel string) (*os.Root, string, error) {
	if err := d.check(rel); err != nil {
		return nil, "", err
	}

	parent, name := filepath.Split(filepath.Clean(rel))
	parent = filepath.Clean(parent) // "" becomes "."
	if name == "" {
		return nil, "", fmt.Errorf("path has no file name: %q", rel)
	}

	if parent != "." {
		if err := d.root.MkdirAll(parent, 0755); err != nil {
			return nil, "", fmt.Errorf("cannot create destination directory: %w", err)
		}
	}

	root, err := d.root.OpenRoot(parent)
	if err != nil {
		return nil, "", fmt.Errorf("cannot open destination directory: %w", err)
	}
	return root, name, nil
}

// CopyFrom atomically copies an external file into this directory at rel,
// creating intermediate directories as needed.
//
// srcPath is a raw path because the source genuinely is outside every
// boundary: it is whatever file the user picked. It is the one place a bare
// string still enters, and opening it is what validates it. The destination
// side carries the boundary.
//
// The copy is atomic: rel either appears complete or is not touched at all.
// Existing files are overwritten - deciding whether that is allowed is the
// caller's policy, not this package's.
func (d *Dir) CopyFrom(srcPath, rel string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("cannot open source file: %w", err)
	}
	defer func() { _ = src.Close() }()

	info, err := src.Stat()
	if err != nil {
		return fmt.Errorf("cannot stat source file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("source is a directory, not a file: %s", srcPath)
	}

	dest, name, err := d.destination(rel)
	if err != nil {
		return err
	}
	defer func() { _ = dest.Close() }()

	return atomicCopyInto(dest, name, src, info.Mode().Perm())
}

// CopyTo atomically copies a regular file from this directory into another
// Dir, creating intermediate directories as needed.
//
// Both ends are handles, so neither the source nor the destination can be
// talked out of its boundary by a crafted relative path - or by a symlinked
// subdirectory, since no path string is ever re-resolved in between.
func (d *Dir) CopyTo(rel string, dst *Dir, dstRel string) error {
	if dst == nil {
		return fmt.Errorf("destination directory is nil")
	}

	src, info, err := d.openFile(rel)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	dest, name, err := dst.destination(dstRel)
	if err != nil {
		return err
	}
	defer func() { _ = dest.Close() }()

	return atomicCopyInto(dest, name, src, info.Mode().Perm())
}

// SymlinkTo creates a symbolic link at dstRel inside dst pointing at rel
// inside this directory.
//
// The link is written with a relative target so the pair keeps working if the
// tree is moved. The target is proven to be a regular file inside this
// directory first, and the link itself is created through dst's boundary.
//
// The link *text* is ordinary path arithmetic between the two directories -
// a symlink target is just a string the kernel interprets later, so this is
// one of the sanctioned uses of Path().
func (d *Dir) SymlinkTo(rel string, dst *Dir, dstRel string) error {
	if dst == nil {
		return fmt.Errorf("destination directory is nil")
	}

	if _, err := d.Stat(rel); err != nil {
		return err
	}
	if err := dst.check(dstRel); err != nil {
		return err
	}

	linkDir := filepath.Dir(dstRel)
	if linkDir != "." {
		if err := dst.root.MkdirAll(linkDir, 0755); err != nil {
			return fmt.Errorf("failed to create symlink directory: %w", err)
		}
	}

	target, err := filepath.Rel(filepath.Join(dst.Path(), linkDir), filepath.Join(d.Path(), rel))
	if err != nil {
		return fmt.Errorf("cannot calculate relative path: %w", err)
	}

	if err := dst.root.Symlink(target, dstRel); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}
	return nil
}

// Scan walks this directory and returns the files it contains, each with a
// Path relative to the directory root. The result is never nil.
//
// opts may be nil, in which case sensible defaults are used. The filter is
// supplied by the caller as a func(name string) bool: this package has no
// notion of which files are interesting.
//
// Symlink policy is the walk's: a relative link to a regular file inside the
// boundary is reported with its target's size and mode; links to directories
// are neither reported nor followed, which is what keeps the walk free of
// loops; links that escape, dangle or are absolute are skipped rather than
// turned into errors.
func (d *Dir) Scan(opts *DirectoryScanOptions) ([]FileInfo, error) {
	if d.root == nil {
		return nil, fmt.Errorf("directory handle is closed")
	}
	return walkFiles(d.root, opts)
}
