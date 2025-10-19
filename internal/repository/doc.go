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
// # Git Repository Integration
//
// The GitSource implementation provides comprehensive Git repository support:
//
// **Authentication Strategy:**
//   - Public repositories: No authentication required (tries public access first)
//   - Private repositories: Uses GitHub Personal Access Tokens via go-keyring OS credential store
//   - Automatic fallback: Public access → PAT authentication → clear error messages
//   - Token validation: Supports all GitHub PAT formats (ghp_, github_pat_, gho_, ghu_, ghs_)
//
// **Repository Operations:**
//   - Initial clone: Shallow clone (depth=1) for performance, with branch-specific support
//   - Updates: Fetch + reset approach for clean synchronization without merge conflicts
//   - Dirty detection: Preserves local changes and provides user guidance for manual resolution
//   - URL normalization: Automatic SSH → HTTPS conversion for consistent authentication
//
// **Security and Conflict Resolution:**
//   - Directory validation: Prevents overwrites of different repositories
//   - Path security: All paths validated through pkg/fileops security functions
//   - Explicit conflict handling: Manual resolution required for directory conflicts
//   - Safe operations: No automatic deletion of user data or uncommitted changes
//
// **Error Handling:**
//   - User-friendly messages: Technical errors translated to actionable guidance
//   - Settings integration: Authentication errors direct users to Settings → GitHub Authentication
//   - Offline support: Network failures allow continued operation with cached repositories
//   - Recovery guidance: Clear instructions for resolving common issues
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
//   - Graceful degradation when OS credential store is unavailable
//
// Credentials are stored securely in the OS credential store with service name "rulem" and key "github_pat".
// The credential manager provides clear error messages directing users to Settings → GitHub Authentication
// for token management and troubleshooting.
//
// Authentication flow:
//  1. Check if PAT exists in credential store
//  2. If no PAT, attempt public repository access
//  3. If authentication required, prompt user for PAT configuration
//  4. Store PAT securely for subsequent operations
//  5. Validate PAT format and permissions before storage
//
// # Path Resolution and URL Handling
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
// **URL Processing:**
//   - SSH → HTTPS conversion: git@github.com:user/repo.git → https://github.com/user/repo.git
//   - Consistent .git suffix: Ensures reliable remote URL comparison
//   - URL validation: Supports GitHub, GitLab, and custom Git servers
//   - Component extraction: parseGitURL extracts host, owner, and repository name
//
// **Directory Conflict Resolution:**
//   - Empty directory: Safe to clone
//   - Same repository: Safe to fetch/pull updates
//   - Different repository: Manual resolution required
//   - Non-git content: Manual resolution required
//   - Validation uses go-git library for reliable repository detection
//
// # Multi-Repository Support
//
// The package provides comprehensive multi-repository management for handling multiple
// rule repositories simultaneously (added in multi-repository feature):
//
// **Repository Orchestration (multi.go):**
//   - PrepareAllRepositories: Validates, prepares, and syncs multiple repositories in one call
//   - ValidateAllRepositories: Structural validation with uniqueness checks (IDs and names)
//   - Returns map[repositoryID]localPath for efficient lookup by consumers
//   - Resilient design: Preparation failures in one repository don't affect others
//   - Aggregated error reporting with detailed per-repository messages
//
// **Repository Synchronization (sync.go):**
//   - SyncAllRepositories: Synchronizes all GitHub repositories independently
//   - RepositorySyncResult: Detailed status tracking (Success/Failed/Skipped)
//   - SyncStatus enum: Categorizes outcomes for proper UI display and error handling
//   - Dirty repository detection: Skips repos with uncommitted changes to preserve work
//   - Independent operation: Sync failures in one repo don't prevent others from syncing
//
// **Validation (config.go):**
//   - ValidateRepositoryEntry: Structural validation for individual repository entries
//   - ValidateCentralRepositoryConfig: Configuration field validation
//   - ID format validation: Ensures "sanitized-name-timestamp" pattern
//   - Name uniqueness: Case-insensitive duplicate detection
//   - Comprehensive error messages for all validation failures
//
// Usage pattern for multi-repository setup:
//
//	// Prepare all repositories (validates, prepares, and syncs)
//	pathMap, syncResults, err := repository.PrepareAllRepositories(cfg.Repositories, logger)
//	if err != nil {
//	    // Handle errors (partial results may be available in pathMap)
//	}
//
//	// Use path map to access repositories
//	for repoID, localPath := range pathMap {
//	    fm := filemanager.NewFileManager(localPath, logger)
//	    // Work with repository files
//	}
//
//	// Review sync results for UI feedback
//	for _, result := range syncResults {
//	    fmt.Printf("%s: %s\n", result.RepositoryName, result.GetMessage())
//	}
//
// The multi-repository layer maintains the same security and validation standards as
// single-repository operations, with additional checks for uniqueness and consistency
// across the repository collection.
//
// # Security
//
// All path operations leverage the existing pkg/fileops security validators to prevent:
//   - Path traversal attacks (../, symbolic links)
//   - Access to system/reserved directories (/etc, /sys, /proc)
//   - Symlink security vulnerabilities
//   - Invalid or malicious path specifications
//
// **Git-specific Security:**
//   - Directory validation: Two-phase approach with security validation + Git conflict detection
//   - Path isolation: Parent directories created with restricted permissions (0755)
//   - URL validation: parseGitURL prevents malformed or malicious repository URLs
//   - Credential isolation: PATs stored in OS credential store with service-specific keys
//
// **Conflict Resolution Strategy:**
//   - If target directory exists, validate it's a git repository with the same remote URL
//   - If not the same repository, error and ask user to resolve manually
//   - If directory contains non-git content, require manual cleanup
//   - This approach prioritizes safety and user control over automatic conflict resolution
//   - No automatic deletion or overwriting of existing user data
//
// The package maintains the security-first approach established in the application architecture,
// with additional protections specific to Git repository operations and credential management.
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
//   - Comprehensive Git integration with shallow cloning and conflict detection
//   - Robust error handling with actionable user guidance
//
// See the contracts/ directory for detailed interface specifications and expected behaviors.
// Implementation spans multiple files:
//   - source.go: Core interfaces and LocalSource implementation
//   - git.go: GitSource implementation with full Git lifecycle management
//   - credentials.go: Secure PAT management via OS credential store
//   - path.go: URL parsing, path derivation, and directory conflict resolution
//   - config.go: Configuration management and repository preparation
//   - secure_root.go: Local storage directory management with security boundaries
package repository
