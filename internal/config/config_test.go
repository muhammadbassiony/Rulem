package config

import (
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.StorageDir == "" {
		t.Error("DefaultConfig should set a non-empty StorageDir")
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "testconfig.yaml")
	cfg := Config{StorageDir: filepath.Join(dir, "instructions")}
	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.StorageDir != cfg.StorageDir {
		t.Errorf("Expected StorageDir %q, got %q", cfg.StorageDir, loaded.StorageDir)
	}
}

func TestSetStorageDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "testconfig.yaml")
	cfg := DefaultConfig()
	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	newDir := filepath.Join(dir, "new-instructions")
	if err := cfg.SetStorageDir(newDir, path); err != nil {
		t.Fatalf("SetStorageDir failed: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.StorageDir != newDir {
		t.Errorf("Expected StorageDir %q, got %q", newDir, loaded.StorageDir)
	}
}
