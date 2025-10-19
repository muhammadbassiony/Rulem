// Package repository provides abstractions for resolving central rule sources to local filesystem paths.
//
// This package implements the repository abstraction layer for multi-repository support.
// It handles both local directory sources and remote Git repository sources, providing
// a unified interface for accessing rule files regardless of their origin.
//
// # Package Organization
//
// The package is organized into logical groups:
//
// Core Types & Interfaces (types.go):
//   - Source: Interface for repository preparation (LocalSource, GitSource)
//   - RepositoryEntry: Domain entity representing a configured repository
//   - RepositoryType: Enum for repository types (local, github)
//   - PreparedRepository: Bundles repository entry, local path, and sync results
//
// Implementations (local.go, git.go, credentials.go):
//   - LocalSource: Validates existing local directories
//   - GitSource: Handles Git clone/sync operations with authentication
//   - CredentialManager: Secure GitHub PAT management via OS credential store
//
// Operations (preparation.go, validation.go, sync.go):
//   - PrepareRepository: Prepares a single repository for use
//   - PrepareAllRepositories: Orchestrates multi-repository preparation
//   - ValidateRepositoryEntry: Validates repository configuration
//   - SyncAllRepositories: Synchronizes all GitHub repositories
//
// Utilities (defaults.go, setup.go):
//   - GetDefaultStorageDir: Returns default repository storage location
//   - EnsureLocalStorageDirectory: Creates and validates directories for config setup
//
// # Architecture
//
// The package provides a clean API for repository management:
//
//   - RepositoryEntry: Configuration for each repository (ID, name, type, path, etc.)
//   - PrepareRepository(repo, logger): Prepares single repository, returns local path
//   - PrepareAllRepositories(repos, logger): Prepares all repositories, returns []PreparedRepository
//   - Source interface: Prepare(logger) (localPath, error)
//     LocalSource validates existing directories, GitSource handles clone/sync operations
//
// Usage pattern for multi-repository setup:
//
//	// Prepare all repositories (validates, prepares, and syncs)
//	prepared, err := repository.PrepareAllRepositories(cfg.Repositories, logger)
//	if err != nil { /* handle errors */ }
//
//	// Access prepared repositories
//	for _, prep := range prepared {
//	    fm := filemanager.NewFileManager(prep.LocalPath, logger)
//	    // Work with repository files
//	}
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
// Source Implementations:
//
// LocalSource (local.go):
//   - Validates non-empty path, expands "~/" and cleans it
//   - Enforces security via fileops.ValidateStoragePath (prevents traversal, system dirs)
//   - Requires the directory to already exist (no auto-create during preparation)
//   - Returns the absolute path for FileManager consumption
//   - No network operations performed
//
// GitSource (git.go):
//   - Clones repository if not present (shallow clone for performance)
//   - Updates repository if already cloned and clean
//   - Detects uncommitted changes and skips updates to preserve work
//   - Uses CredentialManager for secure GitHub PAT authentication
//   - Returns the absolute path to the cloned repository
//
// All sources resolve to a local filesystem path for use by FileManager.
//
// # Directory Management
//
// The package provides distinct functions for different scenarios:
//
//   - EnsureLocalStorageDirectory (setup.go): Creates and validates directories during config setup
//     Used by configuration management to ensure user-specified directories exist and are writable
//     before saving configuration. Creates parent directories as needed.
//
//   - LocalSource.Prepare (local.go): Validates existing directories at runtime
//     Used during application startup to confirm configured paths are ready for file operations.
//     Does not create directories - they must already exist.
//
// This separation maintains clear responsibilities:
//   - Config setup: "Make sure this directory exists and is usable"
//   - Runtime validation: "Confirm this directory is ready for file operations"
//
// # Usage Examples
//
// Single Repository:
//
//	repo := repository.RepositoryEntry{
//	    ID:        "my-rules-1234567890",
//	    Name:      "My Rules",
//	    Type:      repository.RepositoryTypeLocal,
//	    Path:      "/home/user/rules",
//	    CreatedAt: 1234567890,
//	}
//	localPath, err := repository.PrepareRepository(repo, logger)
//	if err != nil {
//	    return fmt.Errorf("preparation failed: %w", err)
//	}
//	fm := filemanager.NewFileManager(localPath, logger)
//
// Multiple Repositories:
//
//	prepared, err := repository.PrepareAllRepositories(cfg.Repositories, logger)
//	if err != nil {
//	    return fmt.Errorf("multi-repo preparation failed: %w", err)
//	}
//	for _, prep := range prepared {
//	    fmt.Printf("%s: %s\n", prep.Name(), prep.GetStatusMessage())
//	    if !prep.HasError() {
//	        fm := filemanager.NewFileManager(prep.LocalPath, logger)
//	        // Work with repository
//	    }
//	}
//
// Directory Setup (Config):
//
//	root, err := repository.EnsureLocalStorageDirectory("~/Documents/rulem-rules")
//	if err != nil {
//	    return fmt.Errorf("setup failed: %w", err)
//	}
//	defer root.Close()
//	// Directory now exists and is validated
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
// The package provides comprehensive multi-repository management:
//
// **Core Types:**
//   - PreparedRepository: Bundles RepositoryEntry, local path, and sync results
//   - RepositorySyncResult: Detailed sync status (Success/Failed/Skipped)
//   - SyncStatus: Enum for categorizing sync outcomes
//
// **Orchestration (multi.go):**
//   - PrepareAllRepositories: One-call preparation, validation, and sync
//   - Returns []PreparedRepository with complete status for each repository
//   - Resilient: Failures in one repository don't affect others
//   - Aggregated error reporting with per-repository details
//
// **Validation (validation.go):**
//   - ValidateRepositoryEntry: Structural validation for single entries
//   - ValidateAllRepositories: Batch validation with uniqueness checks
//   - ID format: "sanitized-name-timestamp" pattern enforcement
//   - Name uniqueness: Case-insensitive duplicate detection
//   - Type-specific validation: GitHub repos must have RemoteURL, etc.
//
// **Synchronization (sync.go):**
//   - SyncAllRepositories: Independent sync for all GitHub repositories
//   - Dirty detection: Skips repos with uncommitted changes
//   - Returns RepositorySyncResult for each repository
//   - Failures are isolated - one repo's failure doesn't prevent others
//
// **API Benefits:**
//   - Single source of truth: PreparedRepository contains all repository state
//   - Easy status access: WasSynced(), HasError(), GetStatusMessage()
//   - Simplified error handling: Check individual repo status or aggregate
//   - Type-safe access: ID(), Name(), Type(), IsRemote(), etc.
//
// The multi-repository layer maintains the same security and validation standards as
// single-repository operations, with additional checks for uniqueness and consistency.
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
//  1. Create RepositoryEntry with configuration (ID, name, type, path, etc.)
//  2. Call PrepareRepository(repo, logger) or PrepareAllRepositories(repos, logger)
//  3. For Git sources, retrieve Personal Access Token from credential store
//  4. Source validates/clones/syncs and resolves to local path
//  5. Receive PreparedRepository with local path and sync status
//  6. Pass local path to FileManager for file operations
//  7. All subsequent operations work transparently with validated paths
//
// The architecture provides a clean, type-safe API with comprehensive error handling.
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
//   - Clean separation of concerns with logical file organization
//   - Type-safe API with PreparedRepository for unified access
//
// # File Organization
//
// Core:
//   - types.go: Domain types (RepositoryEntry, PreparedRepository, RepositoryType)
//   - source.go: Source interface definition
//
// Implementations:
//   - local.go: LocalSource implementation
//   - git.go: GitSource with Git operations
//   - credentials.go: Secure credential management
//
// Operations:
//   - preparation.go: Single repository preparation
//   - validation.go: Repository validation logic
//   - multi.go: Multi-repository orchestration
//   - sync.go: Repository synchronization
//
// Utilities:
//   - defaults.go: Default paths and constants
//   - setup.go: Directory creation for config setup
//   - doc.go: Package documentation
package repository
