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
// The rest of this package validates paths as *text*: a caller passes a string,
// a validator proves something about it, and a separate syscall then acts on
// that same string. Everything that goes wrong in between - a symlink swapped
// in, a component that resolves elsewhere, a check that was never reachable -
// lives in that gap. A Dir closes the gap by holding the boundary open:
// os.OpenRoot is called once, when the path is first accepted, and every later
// operation is named *relative to that open handle* instead of being re-derived
// from a string.
//
// # Confinement is held, not re-proved
//
// The rule for callers is simple: get a Dir once, carry it, and address
// everything inside it by a relative path. Never take Path() and rebuild an
// absolute path to hand to os.Open - that is the shape this type exists to
// remove.
//
// # Status: this is a facade, deliberately
//
// Every method below currently DELEGATES to the existing package functions
// (ValidateFileInDirectory, IsSymlink, AtomicCopy, NewDirectoryScanner, ...).
// The *os.Root is opened and held - it is what makes a Dir a capability rather
// than a string - but it does not yet perform the operations. That swap is a
// separate, isolated change.
//
// The delegation is the point: while Dir runs on the old implementation, tests
// written against Dir exercise the old behaviour through the new API, so they
// can be shown to be a faithful translation *before* anything underneath
// changes. After the swap, any test that fails is an unambiguous behaviour
// change rather than a translation mistake.
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
	// not a path, and it backs Path() and the handle's lifetime.
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

// resolve turns a caller-supplied relative name into the absolute path the
// delegated validators expect, rejecting anything that is not a plain relative
// name inside this directory.
//
// The rejection is delegated to ValidateCWDPath, which is the existing
// "relative, non-empty, no traversal, stays inside" predicate. Lexical
// analysis alone is not containment - a component could still be a symlink
// pointing elsewhere - so every method that resolves also delegates to a
// filesystem-level check before acting.
func (d *Dir) resolve(rel string) (string, error) {
	if d.root == nil {
		return "", fmt.Errorf("directory handle is closed")
	}
	if err := ValidateCWDPath(rel); err != nil {
		return "", err
	}
	return filepath.Join(d.root.Name(), rel), nil
}

// contained resolves rel and additionally proves, against the filesystem, that
// it names an existing regular file inside this directory.
func (d *Dir) contained(rel string) (string, error) {
	abs, err := d.resolve(rel)
	if err != nil {
		return "", err
	}
	if err := ValidateFileInDirectory(abs, d.root.Name()); err != nil {
		return "", err
	}
	return abs, nil
}

// Stat reports metadata for a regular file inside this directory.
//
// Directories are rejected with "path is a directory, not a file" - that is
// the guarantee the containment validator carries, and enumerating a directory
// is what Scan is for. Symlinks are followed, but only within the boundary: a
// link leaving this directory fails here rather than being resolved outside it.
func (d *Dir) Stat(rel string) (fs.FileInfo, error) {
	abs, err := d.contained(rel)
	if err != nil {
		return nil, err
	}
	return os.Stat(abs)
}

// Exists reports whether rel names anything at all inside this directory,
// including a directory or a broken symlink.
//
// It answers "is something already here?" - the question asked before writing
// - so it must not follow symlinks; a dangling link still occupies the name.
func (d *Dir) Exists(rel string) bool {
	abs, err := d.resolve(rel)
	if err != nil {
		return false
	}
	_, err = os.Lstat(abs)
	return err == nil
}

// IsSymlink reports whether rel is itself a symbolic link, without following it.
func (d *Dir) IsSymlink(rel string) (bool, error) {
	abs, err := d.resolve(rel)
	if err != nil {
		return false, err
	}
	return IsSymlink(abs)
}

// Open opens a regular file inside this directory for reading.
//
// Opening IS the check: the file is proven to exist, to be a regular file and
// to resolve inside the boundary, and the handle you get back is the one that
// was checked. There is no separate "can I read this later?" question to ask.
func (d *Dir) Open(rel string) (*os.File, error) {
	abs, err := d.contained(rel)
	if err != nil {
		return nil, err
	}
	return os.Open(abs)
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
	abs, err := d.resolve(rel)
	if err != nil {
		return err
	}
	return os.Remove(abs)
}

// CopyFrom atomically copies an external file into this directory at rel,
// creating intermediate directories as needed.
//
// srcPath is a raw path because the source genuinely is outside every
// boundary: it is whatever file the user picked. It is the one place a bare
// string still enters, and it is checked with ValidateFileAccess before use.
// The destination side carries the boundary.
//
// The copy is atomic: rel either appears complete or is not touched at all.
// Existing files are overwritten - deciding whether that is allowed is the
// caller's policy, not this package's.
func (d *Dir) CopyFrom(srcPath, rel string) error {
	if err := ValidateFileAccess(srcPath, false); err != nil {
		return fmt.Errorf("source file validation failed: %w", err)
	}

	dest, err := d.resolve(rel)
	if err != nil {
		return err
	}

	if err := EnsureDirectoryExists(filepath.Dir(dest)); err != nil {
		return fmt.Errorf("cannot create destination directory: %w", err)
	}

	return AtomicCopy(srcPath, dest)
}

// CopyTo atomically copies a regular file from this directory into another
// Dir, creating intermediate directories as needed.
//
// Both ends are handles, so neither the source nor the destination can be
// talked out of its boundary by a crafted relative path.
func (d *Dir) CopyTo(rel string, dst *Dir, dstRel string) error {
	if dst == nil {
		return fmt.Errorf("destination directory is nil")
	}

	src, err := d.contained(rel)
	if err != nil {
		return err
	}

	dest, err := dst.resolve(dstRel)
	if err != nil {
		return err
	}

	if err := EnsureDirectoryExists(filepath.Dir(dest)); err != nil {
		return fmt.Errorf("cannot create destination directory: %w", err)
	}

	return AtomicCopy(src, dest)
}

// SymlinkTo creates a symbolic link at dstRel inside dst pointing at rel
// inside this directory.
//
// The link is written with a relative target so the pair keeps working if the
// tree is moved. The link target is proven to be a regular file inside this
// directory first; the destination name is proven to be inside dst.
func (d *Dir) SymlinkTo(rel string, dst *Dir, dstRel string) error {
	if dst == nil {
		return fmt.Errorf("destination directory is nil")
	}

	target, err := d.contained(rel)
	if err != nil {
		return err
	}

	link, err := dst.resolve(dstRel)
	if err != nil {
		return err
	}

	return CreateRelativeSymlink(target, link)
}

// Scan walks this directory and returns the files it contains, each with a
// Path relative to the directory root.
//
// opts may be nil, in which case sensible defaults are used. The filter is
// supplied by the caller as a func(name string) bool: this package has no
// notion of which files are interesting.
//
// Symlink policy is the scanner's: a relative link to a regular file inside
// the boundary is reported with its target's size and mode; links to
// directories are neither reported nor followed, which is what keeps the walk
// free of loops; links that escape, dangle or are absolute are skipped rather
// than turned into errors.
//
// MaxDepth and SkipUnreadableDirs are ergonomics, not safety mechanisms - the
// boundary is what provides safety - and they are not load-bearing here.
func (d *Dir) Scan(opts *DirectoryScanOptions) ([]FileInfo, error) {
	if d.root == nil {
		return nil, fmt.Errorf("directory handle is closed")
	}

	scanner, err := NewDirectoryScanner(d.root.Name(), opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = scanner.Close() }()

	return scanner.ScanDirectory()
}
