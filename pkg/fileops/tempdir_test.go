package fileops

import (
	"runtime"
	"testing"
)

// Tests for the temp-directory exception inside IsReservedDirectory (B12).
//
// isUserTempDirectory is an allow-list: it can wave a path through *after* it
// has been found under a reserved directory. It is therefore the one matcher in
// this package where being too permissive weakens a guard, and the two matches
// it used to perform had no component boundary at all - a bare
// strings.HasPrefix on $TMPDIR, and a strings.Contains for `\temp\` on Windows.

func TestIsUserTempDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Cases use Unix path syntax")
	}

	tests := []struct {
		name string
		// tmpdir is exported as $TMPDIR, which os.TempDir reads. Empty leaves
		// the environment alone.
		tmpdir      string
		path        string
		reservedDir string
		want        bool
		// onlyOn restricts a case to one GOOS, for the hardcoded per-platform
		// temp roots. Empty means every platform.
		onlyOn string
	}{
		{
			// The B12 case. $TMPDIR is user-controlled, so a temp root at or
			// above the reserved directory must not exempt the whole tree.
			name:        "TMPDIR set to a parent of a reserved directory does not exempt it",
			tmpdir:      "/var",
			path:        "/var/log/rulem",
			reservedDir: "/var/log",
			want:        false,
		},
		{
			name:        "TMPDIR equal to the reserved directory does not exempt it",
			tmpdir:      "/var/log",
			path:        "/var/log/rulem",
			reservedDir: "/var/log",
			want:        false,
		},
		{
			// The exception's legitimate use: a temp root nested inside the
			// reserved directory carves out that one hole and nothing more.
			name:        "TMPDIR nested inside the reserved directory exempts only itself",
			tmpdir:      "/var/log/rulem-tmp",
			path:        "/var/log/rulem-tmp/probe",
			reservedDir: "/var/log",
			want:        true,
		},
		{
			name:        "the temp root itself is exempt",
			tmpdir:      "/var/log/rulem-tmp",
			path:        "/var/log/rulem-tmp",
			reservedDir: "/var/log",
			want:        true,
		},
		{
			// The old bare strings.HasPrefix matched mid-component.
			name:        "a sibling sharing the temp root's prefix is not exempt",
			tmpdir:      "/var/log/tmp",
			path:        "/var/log/tmpfoo/probe",
			reservedDir: "/var/log",
			want:        false,
		},
		{
			// The old Windows branch exempted any path containing a component
			// named temp or tmp, at any depth.
			name:        "a directory merely named temp is not exempt",
			tmpdir:      "/tmp",
			path:        "/var/log/temp/probe",
			reservedDir: "/var/log",
			want:        false,
		},
		{
			name:        "a path outside every temp root is not exempt",
			tmpdir:      "/tmp",
			path:        "/var/log/rulem",
			reservedDir: "/var/log",
			want:        false,
		},
		{
			// The per-platform defaults stay valid even when $TMPDIR points
			// somewhere else entirely.
			name:        "the linux default temp root is honoured regardless of TMPDIR",
			tmpdir:      "/nowhere",
			path:        "/var/tmp/rulem/probe",
			reservedDir: "/var",
			want:        true,
			onlyOn:      "linux",
		},
		{
			name:        "the macOS per-user temp root is honoured regardless of TMPDIR",
			tmpdir:      "/nowhere",
			path:        "/var/folders/aa/bb/T/probe",
			reservedDir: "/var",
			want:        true,
			onlyOn:      "darwin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.onlyOn != "" && runtime.GOOS != tt.onlyOn {
				t.Skipf("Case applies to %s only", tt.onlyOn)
			}
			if tt.tmpdir != "" {
				t.Setenv("TMPDIR", tt.tmpdir)
			}

			if got := isUserTempDirectory(tt.path, tt.reservedDir); got != tt.want {
				t.Errorf("isUserTempDirectory(%q, %q) = %v, want %v",
					tt.path, tt.reservedDir, got, tt.want)
			}
		})
	}
}

// TestIsReservedDirectoryTempExemption checks the same defect one level up,
// where it matters: a user-set $TMPDIR must not be able to talk
// IsReservedDirectory out of protecting a system directory.
func TestIsReservedDirectoryTempExemption(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Cases use Unix path syntax")
	}

	tests := []struct {
		name   string
		tmpdir string
		path   string
		want   bool
	}{
		{
			name:   "TMPDIR=/var does not unprotect /var/log",
			tmpdir: "/var",
			path:   "/var/log/rulem-storage",
			want:   true,
		},
		{
			name:   "TMPDIR=/ does not unprotect anything",
			tmpdir: "/",
			path:   "/etc/rulem-storage",
			want:   true,
		},
		{
			name:   "an ordinary temp path stays usable",
			tmpdir: "/tmp",
			path:   "/tmp/rulem-storage",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TMPDIR", tt.tmpdir)

			if got := IsReservedDirectory(tt.path); got != tt.want {
				t.Errorf("IsReservedDirectory(%q) with TMPDIR=%q = %v, want %v",
					tt.path, tt.tmpdir, got, tt.want)
			}
		})
	}
}
