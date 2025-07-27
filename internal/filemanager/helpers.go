package filemanager

import (
	"os"
	"runtime"
	"testing"
)

// CreateSymlink creates a symbolic link for testing
func CreateSymlink(t *testing.T, oldname, newname string) {
	t.Helper()
	if err := os.Symlink(oldname, newname); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("symlink creation failed on Windows: %v", err)
		}
		t.Fatalf("failed to create symlink: %v", err)
	}
}
