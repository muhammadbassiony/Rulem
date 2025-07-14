package core

import (
	"os"
	"path/filepath"
	"testing"
)

func createTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	return path
}

func TestCopyFiles(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	createTestFile(t, srcDir, "a.md", "A")
	createTestFile(t, srcDir, "b.md", "B")
	files := []string{"a.md", "b.md"}
	results := CopyFiles(files, srcDir, dstDir, false)
	for _, name := range files {
		if err := results[name]; err != nil {
			t.Errorf("expected no error for %s, got %v", name, err)
		}
		data, err := os.ReadFile(filepath.Join(dstDir, name))
		if err != nil || string(data) == "" {
			t.Errorf("file %s not copied correctly", name)
		}
	}
	// Test overwrite prevention
	results = CopyFiles(files, srcDir, dstDir, false)
	for _, name := range files {
		if results[name] == nil {
			t.Errorf("expected error for existing file %s, got nil", name)
		}
	}
	// Test force overwrite
	results = CopyFiles(files, srcDir, dstDir, true)
	for _, name := range files {
		if err := results[name]; err != nil {
			t.Errorf("expected no error for force overwrite %s, got %v", name, err)
		}
	}
}

func TestCreateSymlinks(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	createTestFile(t, srcDir, "a.md", "A")
	files := []string{"a.md"}
	results := CreateSymlinks(files, srcDir, dstDir, false)
	for _, name := range files {
		if err := results[name]; err != nil {
			t.Errorf("expected no error for %s, got %v", name, err)
		}
		link := filepath.Join(dstDir, name)
		info, err := os.Lstat(link)
		if err != nil || info.Mode()&os.ModeSymlink == 0 {
			t.Errorf("%s is not a symlink", name)
		}
	}
	// Test overwrite prevention
	results = CreateSymlinks(files, srcDir, dstDir, false)
	for _, name := range files {
		if results[name] == nil {
			t.Errorf("expected error for existing symlink %s, got nil", name)
		}
	}
	// Test force overwrite
	results = CreateSymlinks(files, srcDir, dstDir, true)
	for _, name := range files {
		if err := results[name]; err != nil {
			t.Errorf("expected no error for force overwrite %s, got %v", name, err)
		}
	}
}
