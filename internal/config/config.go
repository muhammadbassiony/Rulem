package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// SetStorageDir updates the storage directory and saves the config to the given path.
func (c *Config) SetStorageDir(newDir, path string) error {
	c.StorageDir = newDir
	return c.Save(path)
}

// Config holds user configuration for rulemig.
type Config struct {
	// StorageDir is the directory where instruction files are stored.
	StorageDir string `yaml:"storage_dir"`
}

const defaultConfigFile = ".rulemig.yaml"

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "." // fallback to current dir
	}
	return Config{
		StorageDir: filepath.Join(home, "rulemig-instructions"),
	}
}

// Load loads the config from the given path, or returns default if not found.
func Load(path string) (Config, error) {
	cfg := DefaultConfig()
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	defer f.Close()
	dec := yaml.NewDecoder(f)
	if err := dec.Decode(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// Save writes the config to the given path.
func (c Config) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := yaml.NewEncoder(f)
	defer enc.Close()
	return enc.Encode(c)
}
