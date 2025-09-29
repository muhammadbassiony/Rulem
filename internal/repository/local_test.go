package repository

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rulem/internal/logging"
)

func TestLocalSource_Prepare_ValidPath(t *testing.T) {
	tempDir := t.TempDir()
	logger, _ := logging.NewTestLogger()

	ls := LocalSource{Path: tempDir}
	gotPath, info, err := ls.Prepare(logger)
	if err != nil {
		t.Fatalf("Prepare() unexpected error for valid path: %v", err)
	}

	// Path should be absolute and equal to tempDir (t.TempDir returns absolute)
	if !filepath.IsAbs(gotPath) {
		t.Errorf("Prepare() returned non-absolute path: %s", gotPath)
	}
	if gotPath != tempDir {
		t.Errorf("Prepare() returned path = %s, want %s", gotPath, tempDir)
	}

	// No sync activity for LocalSource
	if info.Cloned || info.Updated || info.Dirty {
		t.Errorf("Prepare() unexpected SyncInfo flags for local source: %+v", info)
	}
	if info.Message == "" || !strings.Contains(strings.ToLower(info.Message), "local") {
		t.Errorf("Prepare() expected user message about local repository, got: %q", info.Message)
	}
}

func TestLocalSource_Prepare_MissingDirectory(t *testing.T) {
	parent := t.TempDir() // ensure parent exists to pass parent-dir validation
	missingPath := filepath.Join(parent, "missing-subdir")
	logger, _ := logging.NewTestLogger()

	ls := LocalSource{Path: missingPath}
	_, _, err := ls.Prepare(logger)
	if err == nil {
		t.Fatalf("Prepare() expected error for missing directory but got none")
	}

	// Expect an error indicating the directory does not exist
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "does not exist") {
		t.Errorf("Prepare() error = %v, want error containing 'does not exist'", err)
	}
}

func TestLocalSource_Prepare_PathTraversalRejected(t *testing.T) {
	// Attempt path traversal; should be rejected by storage path validation
	invalidPath := filepath.Join("..", "..", "etc")
	logger, _ := logging.NewTestLogger()

	ls := LocalSource{Path: invalidPath}
	_, _, err := ls.Prepare(logger)
	if err == nil {
		t.Fatalf("Prepare() expected error for traversal path but got none")
	}

	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "path traversal") && !strings.Contains(errStr, "invalid local source path") {
		t.Errorf("Prepare() error = %v, want error indicating path traversal/invalid path", err)
	}
}

func TestLocalSource_Prepare_NotADirectory(t *testing.T) {
	tempDir := t.TempDir()
	// Create a regular file and pass its path
	filePath := filepath.Join(tempDir, "not-a-dir.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	logger, _ := logging.NewTestLogger()
	ls := LocalSource{Path: filePath}
	_, _, err := ls.Prepare(logger)
	if err == nil {
		t.Fatalf("Prepare() expected error for file path but got none")
	}

	if !strings.Contains(strings.ToLower(err.Error()), "not a directory") {
		t.Errorf("Prepare() error = %v, want error containing 'not a directory'", err)
	}
}
