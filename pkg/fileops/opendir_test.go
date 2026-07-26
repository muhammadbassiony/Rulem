package fileops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Tests for the two constructors added alongside OpenDir when the application
// moved onto Dir handles.
//
// They exist because "open a directory" turned out to be three questions, not
// one, and the differences between them are user-visible:
//
//	OpenDir         - the user is choosing this directory now; create it.
//	OpenExistingDir - it was configured earlier; it must still be there.
//	OpenWorkingDir  - the directory the shell chose; no storage policy.
//
// This file is separate from dir_test.go so that the six test files the
// migration froze stay byte-identical.

func TestOpenExistingDir(t *testing.T) {
	tempDir := t.TempDir()

	existing := filepath.Join(tempDir, "existing")
	if err := os.Mkdir(existing, 0755); err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	regularFile := filepath.Join(tempDir, "a-file.md")
	if err := os.WriteFile(regularFile, []byte("x"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	tests := []struct {
		name      string
		path      string
		wantError bool
		errorText string
	}{
		{
			name:      "existing directory",
			path:      existing,
			wantError: false,
		},
		{
			// The difference from OpenDir that the whole distinction exists
			// for: a missing directory is reported, not conjured.
			name:      "missing directory is not created",
			path:      filepath.Join(tempDir, "not-here"),
			wantError: true,
			errorText: "directory does not exist",
		},
		{
			name:      "path is a regular file",
			path:      regularFile,
			wantError: true,
			errorText: "path is not a directory",
		},
		{
			name:      "empty path",
			path:      "",
			wantError: true,
			errorText: "cannot be empty",
		},
		{
			name:      "path traversal",
			path:      "../../../etc/passwd",
			wantError: true,
			errorText: "path traversal not allowed",
		},
		{
			name:      "reserved system directory",
			path:      "/etc",
			wantError: true,
			errorText: "path traversal not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, err := OpenExistingDir(tt.path)
			if dir != nil {
				defer func() { _ = dir.Close() }()
			}

			if tt.wantError {
				if err == nil {
					t.Fatalf("OpenExistingDir(%q) expected an error, got none", tt.path)
				}
				if tt.errorText != "" && !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("OpenExistingDir(%q) error = %v, want error containing %q", tt.path, err, tt.errorText)
				}
				if dir != nil {
					t.Errorf("OpenExistingDir(%q) returned a handle alongside an error", tt.path)
				}
				return
			}

			if err != nil {
				t.Fatalf("OpenExistingDir(%q) unexpected error: %v", tt.path, err)
			}
			if dir == nil {
				t.Fatal("OpenExistingDir returned a nil handle without an error")
			}
		})
	}
}

// TestOpenExistingDirDoesNotCreate pins the negative: nothing appears on disk
// when the directory is missing.
func TestOpenExistingDirDoesNotCreate(t *testing.T) {
	tempDir := t.TempDir()
	missing := filepath.Join(tempDir, "should-not-appear")

	if _, err := OpenExistingDir(missing); err == nil {
		t.Fatal("OpenExistingDir succeeded on a missing directory")
	}

	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Errorf("OpenExistingDir created %q; stat err = %v", missing, err)
	}
}

// TestOpenExistingDirLeavesNoProbe pins that, unlike OpenDir, this constructor
// does not write anything at all - a read-only directory must still open.
func TestOpenExistingDirLeavesNoProbe(t *testing.T) {
	tempDir := t.TempDir()

	dir, err := OpenExistingDir(tempDir)
	if err != nil {
		t.Fatalf("OpenExistingDir failed: %v", err)
	}
	defer func() { _ = dir.Close() }()

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("OpenExistingDir left %d entries behind: %v", len(entries), entries)
	}
}

func TestOpenWorkingDir(t *testing.T) {
	// t.Chdir restores the previous working directory when the test ends and
	// makes the test non-parallel, so nothing leaks into other tests.
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	if err := os.WriteFile("inside.md", []byte("content"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	dir, err := OpenWorkingDir()
	if err != nil {
		t.Fatalf("OpenWorkingDir failed: %v", err)
	}
	defer func() { _ = dir.Close() }()

	t.Run("addresses files in the working directory", func(t *testing.T) {
		got, err := dir.ReadFile("inside.md")
		if err != nil {
			t.Fatalf("ReadFile failed: %v", err)
		}
		if string(got) != "content" {
			t.Errorf("ReadFile = %q, want %q", got, "content")
		}
	})

	t.Run("confinement still applies", func(t *testing.T) {
		if _, err := dir.ReadFile("../escape.md"); err == nil {
			t.Error("expected a traversal rejection, got none")
		}
	})
}

// TestOpenWorkingDirSkipsStoragePolicy is the point of the separate
// constructor: rulem must keep working when it is run from a directory the
// storage policy would refuse, because the working directory is not storage.
func TestOpenWorkingDirSkipsStoragePolicy(t *testing.T) {
	reserved := reservedDirectoryForTest(t)

	// Sanity check the premise - if this path were acceptable to the storage
	// policy the test would prove nothing.
	if err := ValidateStoragePath(reserved); err == nil {
		t.Skipf("%q is not refused by the storage policy on this platform", reserved)
	}

	t.Chdir(reserved)

	dir, err := OpenWorkingDir()
	if err != nil {
		t.Fatalf("OpenWorkingDir refused a reserved working directory: %v", err)
	}
	defer func() { _ = dir.Close() }()
}

// reservedDirectoryForTest returns a directory that exists, is readable, and is
// refused by the reserved-directory policy on the current platform.
func reservedDirectoryForTest(t *testing.T) string {
	t.Helper()

	for _, candidate := range getReservedDirectories() {
		info, err := os.Stat(candidate)
		if err != nil || !info.IsDir() {
			continue
		}
		if entries, err := os.ReadDir(candidate); err != nil || len(entries) == 0 {
			continue // unreadable; chdir would fail for an unrelated reason
		}
		return candidate
	}

	t.Skip("no readable reserved directory available on this platform")
	return ""
}
