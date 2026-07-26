// Package repository - setup.go
//
// This file contains functions for setting up and creating local storage directories
// during configuration setup. These functions are distinct from runtime validation:
//
//   - EnsureLocalStorageDirectory: Creates the directory if needed, proves it is
//     writable, and returns an open confined handle on it. This is called during
//     config creation, when the user is choosing the directory and rulem is
//     therefore allowed to bring it into being.
//
//   - Runtime validation (in local.go): LocalSource.Prepare validates that configured
//     directories exist and are accessible during application startup, but does not
//     create them.
//
// This separation ensures that:
// - Config setup can create necessary directories for the user
// - Runtime validation confirms directories are still valid without side effects
// - There is exactly one implementation of "where may rulem write?" (fileops.OpenDir)
package repository

import (
	"fmt"
	"rulem/internal/logging"
	"rulem/pkg/fileops"
	"strings"
)

// EnsureLocalStorageDirectory brings a local storage directory into existence
// and returns an open, confined handle on it.
//
// # The contract
//
// It returns a *fileops.Dir - a capability, not a path. Nothing about the
// directory has to be re-proved afterwards: the handle exists only because the
// directory passed rulem's storage policy, and every name addressed through it
// is resolved against the open directory rather than against a string. The
// caller owns the handle and must Close it.
//
// This is the "the user is choosing this directory now" case, so creating it is
// part of the job. Its counterpart is fileops.OpenExistingDir, for a directory
// that was configured earlier and must still be there.
//
// # What it checks
//
// Everything is delegated to fileops.OpenDir, which is the single answer to
// "where may rulem write?":
//
//   - "~" is expanded
//   - the path must be absolute or home-relative, with no ".." component
//   - it must not be a reserved system directory, before or after symlink
//     resolution
//   - the directory is created if missing
//   - writability is proven with a randomly named, O_EXCL probe that is
//     removed again
//
// Earlier revisions of this function repeated that work here with a
// home-directory root and a fixed ".rulem-write-test" probe name. Two answers
// to the same question is how they drift apart; this is now the thin one.
//
// Example:
//
//	dir, err := EnsureLocalStorageDirectory("~/Documents/rulem-rules")
//	if err != nil {
//	    return fmt.Errorf("failed to setup storage: %w", err)
//	}
//	defer dir.Close()
func EnsureLocalStorageDirectory(userPath string) (*fileops.Dir, error) {
	if strings.TrimSpace(userPath) == "" {
		return nil, fmt.Errorf("local storage directory path cannot be empty")
	}

	dir, err := fileops.OpenDir(userPath)
	if err != nil {
		logging.Error("Local storage directory unusable", "path", userPath, "error", err)
		return nil, err
	}

	logging.Info("Local storage directory ready", "path", dir.Path())
	return dir, nil
}
