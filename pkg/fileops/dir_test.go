package fileops

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Tests for Dir.
//
// These are a re-homing of the tests belonging to the nine functions the Dir
// handle replaced. They were written while Dir still delegated to those
// functions, so both suites passed at once - which is what made the translation
// demonstrably faithful rather than merely plausible. The originals are gone
// now; this table is the record of where each property went.
//
// Provenance of every property asserted below:
//
//	Original test                            Re-homed as
//	---------------------------------------  ---------------------------------
//	TestValidateStoragePath                  TestOpenDir
//	TestValidateDirectoryWritable            TestOpenDir (creation + probe)
//	TestValidateCWDPath                      TestDirRelativePathRules
//	TestValidatePathSecurity (traversal)     TestDirRelativePathRules
//	TestValidateFileInDirectory              TestDirContainment
//	TestValidatePathInHome                   TestDirContainment ("outside root")
//	TestValidateFileInDirectorySymlinkEscape TestDirSymlinkEscape
//	TestIsSymlink                            TestDirSymlinkClassification
//	TestResolveSymlink                       TestDirSymlinkClassification
//	TestValidateSymlinkSecurity              TestDirSymlinkClassification
//	TestValidateFileAccess                   TestDirOpenAndRead
//	TestSecureDirectoryScanner_ScanDirectory TestDirScan

// createTestSymlink creates a symlink, skipping the test on Windows where the
// operation needs a privilege the CI account may not hold. It moved here from
// symlink_test.go when that file's subject was deleted.
func createTestSymlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("symlink creation failed on Windows: %v", err)
		}
		t.Fatalf("failed to create symlink: %v", err)
	}
}

func isWindows() bool {
	return runtime.GOOS == "windows"
}

// openTestDir opens a Dir on a fresh temporary directory and closes it on
// cleanup. t.TempDir is removed automatically, so nothing is left behind.
func openTestDir(t *testing.T) (*Dir, string) {
	t.Helper()

	path := t.TempDir()
	dir, err := OpenDir(path)
	if err != nil {
		t.Fatalf("OpenDir(%q) failed: %v", path, err)
	}
	t.Cleanup(func() { _ = dir.Close() })

	return dir, path
}

// writeAt creates a file (and any parent directories) at rel inside root.
func writeAt(t *testing.T, root, rel, content string) string {
	t.Helper()

	abs := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		t.Fatalf("MkdirAll for %q failed: %v", rel, err)
	}
	if err := os.WriteFile(abs, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile %q failed: %v", rel, err)
	}

	return abs
}

// TestOpenDir covers the storage policy OpenDir composes: tilde expansion,
// ValidateStoragePath and the writability probe.
func TestOpenDir(t *testing.T) {
	tempDir := t.TempDir()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot determine home directory")
	}
	// A real directory under home, so the tilde case neither depends on
	// ~/Documents existing (it does not on headless CI) nor litters home.
	homeChild, err := os.MkdirTemp(homeDir, "rulem-dir-test-")
	if err != nil {
		t.Fatalf("Failed to create test directory under home: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(homeChild) })

	tests := []struct {
		name      string
		path      string
		wantError bool
		errorText string
	}{
		{
			name:      "empty path",
			path:      "",
			wantError: true,
			errorText: "storage directory cannot be empty",
		},
		{
			name:      "whitespace only",
			path:      "   \t\n  ",
			wantError: true,
			errorText: "storage directory cannot be empty",
		},
		{
			name:      "path traversal",
			path:      "../../../etc/passwd",
			wantError: true,
			errorText: "path traversal not allowed",
		},
		{
			name:      "relative path not under home",
			path:      "relative/path",
			wantError: true,
			errorText: "path must be absolute or relative to home directory",
		},
		{
			name:      "reserved system directory",
			path:      "/etc",
			wantError: true,
			errorText: "path traversal not allowed",
		},
		{
			name:      "user ssh directory",
			path:      filepath.Join(homeDir, ".ssh"),
			wantError: true,
			errorText: "path traversal not allowed",
		},
		{
			name:      "existing absolute directory",
			path:      tempDir,
			wantError: false,
		},
		{
			name:      "home relative directory",
			path:      "~/" + filepath.Base(homeChild),
			wantError: false,
		},
		{
			// ValidateDirectoryWritable creates the directory, so a
			// not-yet-existing child of an existing directory is accepted.
			name:      "missing directory is created",
			path:      filepath.Join(tempDir, "created-on-open"),
			wantError: false,
		},
		{
			name:      "parent directory does not exist",
			path:      filepath.Join(tempDir, "missing", "deeper"),
			wantError: true,
			errorText: "parent directory does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, err := OpenDir(tt.path)
			if dir != nil {
				defer func() { _ = dir.Close() }()
			}

			if tt.wantError {
				if err == nil {
					t.Fatalf("OpenDir(%q) expected an error, got none", tt.path)
				}
				if tt.errorText != "" && !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("OpenDir(%q) error = %v, want error containing %q", tt.path, err, tt.errorText)
				}
				if dir != nil {
					t.Errorf("OpenDir(%q) returned a handle alongside an error", tt.path)
				}
				return
			}

			if err != nil {
				t.Fatalf("OpenDir(%q) unexpected error: %v", tt.path, err)
			}
			if dir == nil {
				t.Fatal("OpenDir returned a nil handle without an error")
			}
			if info, err := os.Stat(dir.Path()); err != nil || !info.IsDir() {
				t.Errorf("OpenDir(%q) did not leave a usable directory at %q: %v", tt.path, dir.Path(), err)
			}
		})
	}
}

// TestOpenDirLeavesNoProbe pins that the writability probe cleans up after
// itself, as ValidateDirectoryWritable promises.
func TestOpenDirLeavesNoProbe(t *testing.T) {
	dir, path := openTestDir(t)
	_ = dir

	entries, err := os.ReadDir(path)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("OpenDir left %d entries behind: %v", len(entries), entries)
	}
}

// TestDirPathIsDisplayOnly documents the one thing Path and DisplayPath are
// for. They are strings, with no boundary attached.
func TestDirPathIsDisplayOnly(t *testing.T) {
	dir, path := openTestDir(t)

	if dir.Path() != path {
		t.Errorf("Path() = %q, want %q", dir.Path(), path)
	}
	if got, want := dir.DisplayPath("a/b.md"), filepath.Join(path, "a", "b.md"); got != want {
		t.Errorf("DisplayPath() = %q, want %q", got, want)
	}
}

// TestDirRelativePathRules re-homes TestValidateCWDPath and the traversal half
// of TestValidatePathSecurity: which relative names a Dir will accept at all,
// before it ever touches the filesystem.
func TestDirRelativePathRules(t *testing.T) {
	dir, path := openTestDir(t)

	tests := []struct {
		name      string
		rel       string
		wantError bool
		errorText string
	}{
		{
			name: "file in the root",
			rel:  "file.txt",
		},
		{
			name: "nested relative path",
			rel:  "deep/nested/path/file.txt",
		},
		{
			name: "current directory reference",
			rel:  "./dotslash.txt",
		},
		{
			// ".." is only traversal when it is a whole component.
			name: "legitimate dots inside a filename",
			rel:  "notes/my..notes.md",
		},
		{
			name:      "empty path",
			rel:       "",
			wantError: true,
			errorText: "cannot be empty",
		},
		{
			name:      "absolute path",
			rel:       filepath.Join(path, "file.txt"),
			wantError: true,
			errorText: "must be relative",
		},
		{
			name:      "parent traversal",
			rel:       "../escape.txt",
			wantError: true,
			errorText: "path traversal not allowed",
		},
		{
			name:      "traversal in the middle",
			rel:       "valid/../escape.txt",
			wantError: true,
			errorText: "path traversal not allowed",
		},
		{
			name:      "escape hidden by a cleanable prefix",
			rel:       "a/b/../../../escape.txt",
			wantError: true,
			errorText: "path traversal not allowed",
		},
		{
			name:      "trailing parent component",
			rel:       "notes/subdir/..",
			wantError: true,
			errorText: "path traversal not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Accepted names must resolve to a real file, so that a failure
			// can only mean the name was rejected by the path rules.
			if !tt.wantError {
				writeAt(t, path, tt.rel, "content")
			}

			_, err := dir.Stat(tt.rel)

			if tt.wantError {
				if err == nil {
					t.Fatalf("Stat(%q) expected an error, got none", tt.rel)
				}
				if !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("Stat(%q) error = %v, want error containing %q", tt.rel, err, tt.errorText)
				}
				return
			}

			if err != nil {
				t.Errorf("Stat(%q) unexpected error: %v", tt.rel, err)
			}
		})
	}
}

// TestDirContainment re-homes TestValidateFileInDirectory: a Dir addresses
// regular files inside itself and nothing else. The "outside the root" case
// also carries TestValidatePathInHome's property - containment is measured
// against the handle's own directory, whatever that directory is.
func TestDirContainment(t *testing.T) {
	dir, path := openTestDir(t)
	outside := t.TempDir()

	writeAt(t, path, "valid.txt", "content")
	writeAt(t, path, "subdir/nested.txt", "nested content")
	writeAt(t, path, "my..notes.md", "content")
	writeAt(t, outside, "outside.txt", "content")
	if err := os.MkdirAll(filepath.Join(path, "testdir"), 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	tests := []struct {
		name      string
		rel       string
		wantError bool
		errorText string
	}{
		{
			name: "file in the directory",
			rel:  "valid.txt",
		},
		{
			name: "nested file in the directory",
			rel:  "subdir/nested.txt",
		},
		{
			name: "file name containing dots is not traversal",
			rel:  "my..notes.md",
		},
		{
			name:      "file outside the directory by absolute path",
			rel:       filepath.Join(outside, "outside.txt"),
			wantError: true,
			errorText: "must be relative",
		},
		{
			name:      "file outside the directory by traversal",
			rel:       filepath.Join("..", filepath.Base(outside), "outside.txt"),
			wantError: true,
			errorText: "path traversal not allowed",
		},
		{
			name:      "non-existent file",
			rel:       "nonexistent.txt",
			wantError: true,
			errorText: "does not exist",
		},
		{
			name:      "path is a directory",
			rel:       "testdir",
			wantError: true,
			errorText: "directory, not a file",
		},
		{
			name:      "empty path",
			rel:       "",
			wantError: true,
			errorText: "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := dir.Stat(tt.rel)

			if tt.wantError {
				if err == nil {
					t.Fatalf("Stat(%q) expected an error, got none", tt.rel)
				}
				if !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("Stat(%q) error = %v, want error containing %q", tt.rel, err, tt.errorText)
				}
				return
			}

			if err != nil {
				t.Fatalf("Stat(%q) unexpected error: %v", tt.rel, err)
			}
			if info.IsDir() {
				t.Errorf("Stat(%q) reported a directory", tt.rel)
			}
		})
	}
}

// TestDirSymlinkEscape re-homes TestValidateFileInDirectorySymlinkEscape: the
// guarantee that a name resolving out of the directory through a symlink is
// refused rather than followed. This is the case the old implementation
// advertised but could not reach (S2), so it is the single most important
// property in this file.
func TestDirSymlinkEscape(t *testing.T) {
	if isWindows() {
		t.Skip("Skipping symlink tests on Windows")
	}

	dir, path := openTestDir(t)
	outside := t.TempDir()

	outsideFile := writeAt(t, outside, "secret.txt", "secret content")
	writeAt(t, path, "inside.txt", "inside content")

	tests := []struct {
		name      string
		rel       string
		target    string // symlink target, created at rel before the check
		linkTo    string // directory symlink: created at this name instead
		wantError bool
	}{
		{
			name:      "symlink pointing outside the directory is rejected",
			rel:       "escaping-link.txt",
			target:    outsideFile,
			wantError: true,
		},
		{
			name:      "symlink via a parent traversal is rejected",
			rel:       "traversing-link.txt",
			target:    filepath.Join("..", filepath.Base(outside), "secret.txt"),
			wantError: true,
		},
		{
			name:      "symlink staying inside the directory is accepted",
			rel:       "internal-link.txt",
			target:    "inside.txt",
			wantError: false,
		},
		{
			name:      "file reached through an escaping directory symlink is rejected",
			rel:       "escaping-dir/secret.txt",
			linkTo:    "escaping-dir",
			target:    outside,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			linkName := tt.rel
			if tt.linkTo != "" {
				linkName = tt.linkTo
			}
			link := filepath.Join(path, linkName)
			createTestSymlink(t, tt.target, link)
			defer func() { _ = os.Remove(link) }()

			_, err := dir.Stat(tt.rel)

			if tt.wantError && err == nil {
				t.Errorf("Stat(%q) expected an error for a link that leaves the directory", tt.rel)
			}
			if !tt.wantError && err != nil {
				t.Errorf("Stat(%q) unexpected error for a link that stays inside: %v", tt.rel, err)
			}
		})
	}
}

// TestDirSymlinkClassification re-homes TestIsSymlink, TestResolveSymlink and
// TestValidateSymlinkSecurity. Classification (is this a link?) and resolution
// (what does it lead to?) are now answered without leaving the directory:
// IsSymlink reports on the name itself, and Open/Stat report on the target -
// but only if the target is still inside.
func TestDirSymlinkClassification(t *testing.T) {
	if isWindows() {
		t.Skip("Skipping symlink tests on Windows")
	}

	dir, path := openTestDir(t)
	outside := t.TempDir()
	outsideFile := writeAt(t, outside, "forbidden.txt", "forbidden content")

	writeAt(t, path, "regular.txt", "content")
	if err := os.MkdirAll(filepath.Join(path, "testdir"), 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	createTestSymlink(t, "regular.txt", filepath.Join(path, "file_link"))
	createTestSymlink(t, "testdir", filepath.Join(path, "dir_link"))
	createTestSymlink(t, "nonexistent.txt", filepath.Join(path, "broken_link"))
	createTestSymlink(t, "chain-1", filepath.Join(path, "chain-2"))
	createTestSymlink(t, "regular.txt", filepath.Join(path, "chain-1"))
	createTestSymlink(t, outsideFile, filepath.Join(path, "escaping_link"))

	tests := []struct {
		name string
		rel  string

		wantIsSymlink    bool
		wantIsSymlinkErr bool

		// wantContent is checked through Open when readable is true.
		readable    bool
		wantContent string
	}{
		{
			name:        "regular file is not a symlink",
			rel:         "regular.txt",
			readable:    true,
			wantContent: "content",
		},
		{
			name: "directory is not a symlink",
			rel:  "testdir",
		},
		{
			name:          "symlink to a file inside is a symlink and is readable",
			rel:           "file_link",
			wantIsSymlink: true,
			readable:      true,
			wantContent:   "content",
		},
		{
			name:          "symlink to a directory is a symlink but is not a file",
			rel:           "dir_link",
			wantIsSymlink: true,
		},
		{
			name:          "symlink chain resolves to the final target",
			rel:           "chain-2",
			wantIsSymlink: true,
			readable:      true,
			wantContent:   "content",
		},
		{
			name:          "broken symlink is a symlink but cannot be opened",
			rel:           "broken_link",
			wantIsSymlink: true,
		},
		{
			name:          "symlink to a forbidden location cannot be opened",
			rel:           "escaping_link",
			wantIsSymlink: true,
		},
		{
			name:             "non-existent path cannot be classified",
			rel:              "nonexistent",
			wantIsSymlinkErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isLink, err := dir.IsSymlink(tt.rel)
			if tt.wantIsSymlinkErr {
				if err == nil {
					t.Errorf("IsSymlink(%q) expected an error, got none", tt.rel)
				}
			} else {
				if err != nil {
					t.Fatalf("IsSymlink(%q) unexpected error: %v", tt.rel, err)
				}
				if isLink != tt.wantIsSymlink {
					t.Errorf("IsSymlink(%q) = %v, want %v", tt.rel, isLink, tt.wantIsSymlink)
				}
			}

			content, err := dir.ReadFile(tt.rel)
			if !tt.readable {
				if err == nil {
					t.Errorf("ReadFile(%q) expected an error, got none", tt.rel)
				}
				return
			}
			if err != nil {
				t.Fatalf("ReadFile(%q) unexpected error: %v", tt.rel, err)
			}
			if string(content) != tt.wantContent {
				t.Errorf("ReadFile(%q) = %q, want %q", tt.rel, content, tt.wantContent)
			}
		})
	}
}

// TestDirOpenAndRead re-homes TestValidateFileAccess. "Check it is readable,
// then open it later" is replaced by opening it: the open is the check, and it
// hands back the file that was checked.
func TestDirOpenAndRead(t *testing.T) {
	dir, path := openTestDir(t)

	writeAt(t, path, "readable.txt", "content")
	if err := os.MkdirAll(filepath.Join(path, "testdir"), 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	unreadable := writeAt(t, path, "unreadable.txt", "content")
	permissionsEnforced := os.Getenv("CI") == "" && os.Geteuid() != 0
	if permissionsEnforced {
		if err := os.Chmod(unreadable, 0000); err != nil {
			permissionsEnforced = false
		} else {
			t.Cleanup(func() { _ = os.Chmod(unreadable, 0644) })
		}
	}

	tests := []struct {
		name      string
		rel       string
		skip      bool
		wantError bool
		errorText string
	}{
		{
			name: "readable file",
			rel:  "readable.txt",
		},
		{
			name:      "non-existent file",
			rel:       "nonexistent.txt",
			wantError: true,
			errorText: "does not exist",
		},
		{
			name:      "directory instead of file",
			rel:       "testdir",
			wantError: true,
			errorText: "directory, not a file",
		},
		{
			name:      "unreadable file",
			rel:       "unreadable.txt",
			skip:      !permissionsEnforced,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip("File permissions are not enforced in this environment")
			}

			f, err := dir.Open(tt.rel)
			if f != nil {
				defer func() { _ = f.Close() }()
			}

			if tt.wantError {
				if err == nil {
					t.Fatalf("Open(%q) expected an error, got none", tt.rel)
				}
				if tt.errorText != "" && !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("Open(%q) error = %v, want error containing %q", tt.rel, err, tt.errorText)
				}
				return
			}

			if err != nil {
				t.Fatalf("Open(%q) unexpected error: %v", tt.rel, err)
			}
		})
	}
}

// TestDirExists covers the "is something already here?" question asked before
// writing. It must see broken symlinks, which still occupy the name.
func TestDirExists(t *testing.T) {
	dir, path := openTestDir(t)

	writeAt(t, path, "present.txt", "content")
	if err := os.MkdirAll(filepath.Join(path, "subdir"), 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if !isWindows() {
		createTestSymlink(t, "nowhere.txt", filepath.Join(path, "broken_link"))
	}

	tests := []struct {
		name string
		rel  string
		skip bool
		want bool
	}{
		{name: "existing file", rel: "present.txt", want: true},
		{name: "existing directory", rel: "subdir", want: true},
		{name: "broken symlink still occupies the name", rel: "broken_link", skip: isWindows(), want: true},
		{name: "missing name", rel: "absent.txt", want: false},
		{name: "escaping name is never reported as present", rel: "../escape.txt", want: false},
		{name: "empty name", rel: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip("Skipping symlink case on Windows")
			}
			if got := dir.Exists(tt.rel); got != tt.want {
				t.Errorf("Exists(%q) = %v, want %v", tt.rel, got, tt.want)
			}
		})
	}
}

// TestDirCopyFrom covers copying an external file in. The source is a raw path
// on purpose - it is whatever the user picked - but the destination carries
// the boundary.
func TestDirCopyFrom(t *testing.T) {
	dir, path := openTestDir(t)
	srcDir := t.TempDir()

	srcFile := writeAt(t, srcDir, "source.md", "source content")
	srcSubdir := filepath.Join(srcDir, "adir")
	if err := os.MkdirAll(srcSubdir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	tests := []struct {
		name      string
		src       string
		rel       string
		wantError bool
	}{
		{
			name: "copy into the root",
			src:  srcFile,
			rel:  "copied.md",
		},
		{
			name: "copy into a subdirectory that does not exist yet",
			src:  srcFile,
			rel:  "backend/api-rules.md",
		},
		{
			name:      "destination escaping the directory is rejected",
			src:       srcFile,
			rel:       "../escaped.md",
			wantError: true,
		},
		{
			name:      "absolute destination is rejected",
			src:       srcFile,
			rel:       filepath.Join(path, "absolute.md"),
			wantError: true,
		},
		{
			name:      "missing source is rejected",
			src:       filepath.Join(srcDir, "nonexistent.md"),
			rel:       "missing.md",
			wantError: true,
		},
		{
			name:      "directory source is rejected",
			src:       srcSubdir,
			rel:       "adir.md",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := dir.CopyFrom(tt.src, tt.rel)

			if tt.wantError {
				if err == nil {
					t.Fatalf("CopyFrom(%q, %q) expected an error, got none", tt.src, tt.rel)
				}
				return
			}

			if err != nil {
				t.Fatalf("CopyFrom(%q, %q) unexpected error: %v", tt.src, tt.rel, err)
			}
			got, err := dir.ReadFile(tt.rel)
			if err != nil {
				t.Fatalf("ReadFile(%q) after copy failed: %v", tt.rel, err)
			}
			if string(got) != "source content" {
				t.Errorf("copied content = %q, want %q", got, "source content")
			}
		})
	}
}

// TestDirCopyTo covers copying between two handles. Neither end can be talked
// out of its boundary by a crafted relative path.
func TestDirCopyTo(t *testing.T) {
	src, srcPath := openTestDir(t)
	dst, _ := openTestDir(t)

	writeAt(t, srcPath, "rules.md", "rule content")
	if err := os.MkdirAll(filepath.Join(srcPath, "adir"), 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	tests := []struct {
		name      string
		rel       string
		dstRel    string
		wantError bool
	}{
		{name: "copy to the destination root", rel: "rules.md", dstRel: "rules.md"},
		{name: "copy into a new destination subdirectory", rel: "rules.md", dstRel: "nested/rules.md"},
		{name: "escaping source is rejected", rel: "../rules.md", dstRel: "rules.md", wantError: true},
		{name: "escaping destination is rejected", rel: "rules.md", dstRel: "../escaped.md", wantError: true},
		{name: "missing source is rejected", rel: "absent.md", dstRel: "rules.md", wantError: true},
		{name: "directory source is rejected", rel: "adir", dstRel: "adir.md", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := src.CopyTo(tt.rel, dst, tt.dstRel)

			if tt.wantError {
				if err == nil {
					t.Fatalf("CopyTo(%q, %q) expected an error, got none", tt.rel, tt.dstRel)
				}
				return
			}

			if err != nil {
				t.Fatalf("CopyTo(%q, %q) unexpected error: %v", tt.rel, tt.dstRel, err)
			}
			got, err := dst.ReadFile(tt.dstRel)
			if err != nil {
				t.Fatalf("ReadFile(%q) after copy failed: %v", tt.dstRel, err)
			}
			if string(got) != "rule content" {
				t.Errorf("copied content = %q, want %q", got, "rule content")
			}
		})
	}

	t.Run("nil destination is rejected", func(t *testing.T) {
		if err := src.CopyTo("rules.md", nil, "rules.md"); err == nil {
			t.Error("CopyTo with a nil destination expected an error, got none")
		}
	})
}

// TestDirSymlinkTo re-homes the parts of TestCreateRelativeSymlink that
// concern containment: the link target must be a real file inside the source
// handle, the link itself must land inside the destination handle, and the
// link is written with a relative target so the pair survives being moved.
func TestDirSymlinkTo(t *testing.T) {
	if isWindows() {
		t.Skip("Skipping symlink tests on Windows")
	}

	src, srcPath := openTestDir(t)
	dst, dstPath := openTestDir(t)

	writeAt(t, srcPath, "rules.md", "rule content")

	tests := []struct {
		name      string
		rel       string
		dstRel    string
		wantError bool
	}{
		{name: "link in the destination root", rel: "rules.md", dstRel: "rules.md"},
		{name: "link in a destination subdirectory", rel: "rules.md", dstRel: "nested/rules.md"},
		{name: "escaping target is rejected", rel: "../rules.md", dstRel: "rules.md", wantError: true},
		{name: "escaping link location is rejected", rel: "rules.md", dstRel: "../escaped.md", wantError: true},
		{name: "missing target is rejected", rel: "absent.md", dstRel: "absent.md", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := src.SymlinkTo(tt.rel, dst, tt.dstRel)

			if tt.wantError {
				if err == nil {
					t.Fatalf("SymlinkTo(%q, %q) expected an error, got none", tt.rel, tt.dstRel)
				}
				return
			}

			if err != nil {
				t.Fatalf("SymlinkTo(%q, %q) unexpected error: %v", tt.rel, tt.dstRel, err)
			}

			link := filepath.Join(dstPath, tt.dstRel)
			target, err := os.Readlink(link)
			if err != nil {
				t.Fatalf("Readlink(%q) failed: %v", link, err)
			}
			if filepath.IsAbs(target) {
				t.Errorf("expected a relative link target, got %q", target)
			}
			if got := readFileContent(t, link); got != "rule content" {
				t.Errorf("content through the link = %q, want %q", got, "rule content")
			}
		})
	}
}

// TestDirScan covers the walk: relative paths, caller-supplied filtering, and
// an empty result that is empty rather than nil.
func TestDirScan(t *testing.T) {
	dir, path := openTestDir(t)

	writeAt(t, path, "top.md", "x")
	writeAt(t, path, "notes.txt", "x")
	writeAt(t, path, "sub/nested.md", "x")
	writeAt(t, path, "node_modules/ignored.md", "x")

	empty, _ := openTestDir(t)

	onlyMarkdown := func(name string) bool { return strings.HasSuffix(name, ".md") }

	tests := []struct {
		name string
		dir  *Dir
		opts *DirectoryScanOptions
		want []string
	}{
		{
			name: "default options include everything outside the skip patterns",
			dir:  dir,
			opts: nil,
			want: []string{"notes.txt", "sub/nested.md", "top.md"},
		},
		{
			name: "caller supplied filter",
			dir:  dir,
			opts: &DirectoryScanOptions{
				SkipUnreadableDirs: true,
				MaxDepth:           20,
				IncludeHidden:      true,
				SkipPatterns:       []string{"node_modules"},
				FileFilter:         onlyMarkdown,
			},
			want: []string{"sub/nested.md", "top.md"},
		},
		{
			name: "empty directory yields an empty, non-nil result",
			dir:  empty,
			opts: nil,
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := tt.dir.Scan(tt.opts)
			if err != nil {
				t.Fatalf("Scan failed: %v", err)
			}
			if files == nil {
				t.Fatal("Scan returned a nil slice")
			}

			got := make([]string, 0, len(files))
			for _, f := range files {
				if filepath.IsAbs(f.Path) {
					t.Errorf("Scan returned an absolute path %q, want a path relative to the root", f.Path)
				}
				got = append(got, filepath.ToSlash(f.Path))
			}

			if !equalUnordered(got, tt.want) {
				t.Errorf("Scan returned %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDirAfterClose pins that a closed handle is inert: no operation silently
// falls back to a bare path.
func TestDirAfterClose(t *testing.T) {
	path := t.TempDir()
	if err := os.WriteFile(filepath.Join(path, "file.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	dir, err := OpenDir(path)
	if err != nil {
		t.Fatalf("OpenDir failed: %v", err)
	}
	if err := dir.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	tests := []struct {
		name string
		call func() error
	}{
		{"Stat", func() error { _, err := dir.Stat("file.txt"); return err }},
		{"Open", func() error { _, err := dir.Open("file.txt"); return err }},
		{"ReadFile", func() error { _, err := dir.ReadFile("file.txt"); return err }},
		{"IsSymlink", func() error { _, err := dir.IsSymlink("file.txt"); return err }},
		{"Remove", func() error { return dir.Remove("file.txt") }},
		{"CopyFrom", func() error { return dir.CopyFrom(filepath.Join(path, "file.txt"), "copy.txt") }},
		{"Scan", func() error { _, err := dir.Scan(nil); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if err == nil {
				t.Fatalf("%s on a closed Dir expected an error, got none", tt.name)
			}
			// Carried over from TestSecureDirectoryScanner_Close: the failure
			// must name the closed handle rather than surface as a bare ENOENT.
			if !strings.Contains(err.Error(), "closed") {
				t.Errorf("%s on a closed Dir: error = %v, want one mentioning %q", tt.name, err, "closed")
			}
		})
	}

	t.Run("Close is idempotent", func(t *testing.T) {
		if err := dir.Close(); err != nil {
			t.Errorf("second Close returned %v, want nil", err)
		}
	})
	t.Run("Exists reports false", func(t *testing.T) {
		if dir.Exists("file.txt") {
			t.Error("Exists on a closed Dir returned true")
		}
	})
}

// TestDirRemove covers deletion, including that an escaping name cannot be
// used to delete something outside the directory.
func TestDirRemove(t *testing.T) {
	dir, path := openTestDir(t)
	outside := t.TempDir()

	writeAt(t, path, "doomed.txt", "x")
	outsideFile := writeAt(t, outside, "safe.txt", "x")

	tests := []struct {
		name      string
		rel       string
		wantError bool
	}{
		{name: "removes a file inside", rel: "doomed.txt"},
		{name: "missing file is an error", rel: "absent.txt", wantError: true},
		{name: "escaping name is rejected", rel: filepath.Join("..", filepath.Base(outside), "safe.txt"), wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := dir.Remove(tt.rel)
			if tt.wantError && err == nil {
				t.Errorf("Remove(%q) expected an error, got none", tt.rel)
			}
			if !tt.wantError && err != nil {
				t.Errorf("Remove(%q) unexpected error: %v", tt.rel, err)
			}
		})
	}

	if !fileExists(outsideFile) {
		t.Error("Remove deleted a file outside the directory")
	}
}

// equalUnordered compares two string slices ignoring order.
func equalUnordered(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	seen := make(map[string]int, len(got))
	for _, s := range got {
		seen[s]++
	}
	for _, s := range want {
		seen[s]--
		if seen[s] < 0 {
			return false
		}
	}
	return true
}
