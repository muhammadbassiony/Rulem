// Package repository provides abstractions for resolving central rule sources to local filesystem paths.
//
// This package implements the repository abstraction layer described in the GitHub-backed central repo
// specification. It handles both local directory sources and remote Git repository sources, providing
// a unified interface for accessing rule files regardless of their origin.
//
// # Architecture
//
// The package defines a clean, simple architecture for repository management:
//
//   - CentralRepositoryConfig: Configuration struct for both local and remote repositories
//     Contains Path, RemoteURL, Branch, and LastSyncTime fields with IsRemote() method
//   - PrepareRepository(config, logger): Creates appropriate Source and prepares it
//     One-line solution for most use cases: get config, return ready-to-use local path
//   - Source interface: Prepare(logger) (localPath, SyncInfo, error)
//     LocalSource validates existing directories, GitSource handles clone/sync operations
//   - SyncInfo struct: Reports cloned/updated/dirty status with user-friendly messages
//
// This eliminates the need for separate startup packages and provides a unified API:
//
//	// Simple usage pattern for all components:
//	localPath, syncInfo, err := repository.PrepareRepository(config.Central, logger)
//	if err != nil { /* handle error */ }
//	// Use syncInfo.Message for UI feedback
//	fm := filemanager.NewFileManager(localPath, logger)
//
// LocalSource behavior:
//   - Validates non-empty path, expands "~/" and cleans it
//   - Enforces security and suitability using fileops.ValidateStoragePath
//   - Requires the directory to already exist and be a directory (no auto-create)
//   - Performs no network access; Message typically "Using local repository"
//   - Returns the absolute path for FileManager consumption
//
// All sources resolve to a local filesystem path that can be used by the FileManager for
// consistent file operations, maintaining the existing public API contracts.
//
// # Local Storage Directory Management
//
// The package provides distinct functions for different local storage scenarios:
//
//   - EnsureLocalStorageDirectory: Creates and validates local directories during config setup
//     Used by config.CreateNewConfig and config.UpdateCentralPath to ensure user-specified
//     directories exist and are writable before saving configuration.
//   - LocalSource.Prepare: Validates existing directories at runtime for FileManager use
//     Used during startup/refresh to validate that configured local paths are ready for use.
//
// This separation maintains clear responsibilities:
//   - Config setup: "Make sure this directory exists and is usable"
//   - Runtime validation: "Confirm this directory is ready for file operations"
//
// Usage (Recommended - via config):
//
//	cfg := repository.CentralRepositoryConfig{
//	    Path: "/path/to/rules",
//	    // RemoteURL: nil for local, or &"https://github.com/user/repo.git" for Git
//	}
//	localPath, syncInfo, err := repository.PrepareRepository(cfg, logger)
//	if err != nil {
//	    return fmt.Errorf("repository preparation failed: %w", err)
//	}
//	// Use syncInfo.Message for user feedback
//	fm := filemanager.NewFileManager(localPath, logger)
//
// Usage (Direct LocalSource):
//
//	ls := LocalSource{Path: "/path/to/rules"}
//	p, info, err := ls.Prepare(logging.GetDefault())
//	// p is an absolute path; info.Message explains state (e.g., "Using local repository")
//
// Usage (Config setup):
//
//	root, err := EnsureLocalStorageDirectory("~/Documents/rulem-rules")
//	if err != nil {
//	    return fmt.Errorf("setup failed: %w", err)
//	}
//	defer root.Close()
//	// Directory now exists and is validated for use
//
// # Credential Management
//
// For GitHub repository authentication, this package provides secure credential management:
//   - CredentialManager: Handles GitHub Personal Access Token (PAT) storage and retrieval
//   - OS credential store integration using go-keyring library
//   - Cross-platform support (Keychain on macOS, libsecret on Linux, Credential Manager on Windows)
//   - Token format validation for GitHub PATs (ghp_, github_pat_, gho_, ghu_, ghs_ prefixes)
//
// Credentials are stored securely in the OS credential store with service name "rulem" and key "github_pat".
// The credential manager provides clear error messages directing users to Settings → GitHub Authentication
// for token management and troubleshooting.
//
// # Path Resolution
//
// For remote Git repositories, this package provides simple, user-friendly path derivation:
//   - Clone path: <DefaultStorageDir>/<repo>
//   - Examples:
//   - git@github.com:user/rules.git → ~/.local/share/rulem/rules
//   - https://git.company.com/team/policies.git → ~/.local/share/rulem/policies
//
// This simple structure makes repositories directly visible and manageable by users outside the tool.
// Repository name conflicts are handled explicitly rather than through complex nested directory structures.
//
// # Security
//
// All path operations leverage the existing pkg/fileops security validators to prevent:
//   - Path traversal attacks
//   - Access to system/reserved directories
//   - Symlink security vulnerabilities
//
// Directory conflicts are resolved explicitly:
//   - If target directory exists, validate it's a git repository with the same remote URL
//   - If not the same repository, error and ask user to resolve manually
//   - This approach prioritizes safety and user control over automatic conflict resolution
//
// The package maintains the security-first approach established in the application architecture.
//
// # Usage Flow
//
//  1. Configuration (CentralRepositoryConfig) specifies local path and optional remote Git URL
//  2. Call repository.PrepareRepository(config, logger)
//  3. For Git sources, retrieve Personal Access Token from credential store
//  4. Source validates/clones/syncs and resolves to local path
//  5. Local path is passed to FileManager for normal operations
//  6. All subsequent file operations work transparently
//
// The simplified flow eliminates intermediate packages and provides a clean, unified API.
//
// # Design Goals
//
//   - Maintain existing FileManager public API unchanged
//   - Support offline operation with cached repositories
//   - Provide security through existing fileops validators
//   - Use simple, user-friendly repository paths for easy external management
//   - Handle directory conflicts explicitly and safely
//   - Support both local and remote repository sources seamlessly
//   - Secure credential management with clear error messaging
//
// See the contracts/ directory for detailed interface specifications and expected behaviors.
package repository
