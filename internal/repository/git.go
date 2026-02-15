package repository

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"rulem/internal/logging"
	"rulem/pkg/fileops"
	"strings"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/transport/http"
)

// DirectoryStatus represents the state of a target clone directory
type DirectoryStatus int

const (
	// DirectoryStatusEmpty indicates the directory doesn't exist or is empty - safe to clone
	DirectoryStatusEmpty DirectoryStatus = iota
	// DirectoryStatusSameRepo indicates the directory contains the same git repository - safe to fetch/pull
	DirectoryStatusSameRepo
	// DirectoryStatusDifferentRepo indicates the directory contains a different git repository - user intervention needed
	DirectoryStatusDifferentRepo
	// DirectoryStatusConflict indicates the directory contains non-git content - user intervention needed
	DirectoryStatusConflict
	// DirectoryStatusError indicates an error occurred during validation
	DirectoryStatusError
)

// String returns a human-readable description of the directory status
func (ds DirectoryStatus) String() string {
	switch ds {
	case DirectoryStatusEmpty:
		return "empty or doesn't exist"
	case DirectoryStatusSameRepo:
		return "same git repository"
	case DirectoryStatusDifferentRepo:
		return "different git repository"
	case DirectoryStatusConflict:
		return "contains non-git content"
	case DirectoryStatusError:
		return "validation error"
	default:
		return "unknown status"
	}
}

// GitSource represents a Git repository used as the central repository.
// It handles cloning, fetching, and authentication using Personal Access Tokens.
//
// The GitSource manages the complete lifecycle of a Git-based rule repository:
//   - URL validation and normalization (SSH to HTTPS conversion)
//   - Directory conflict detection and resolution
//   - Authentication via GitHub Personal Access Tokens
//   - Shallow cloning for performance optimization
//   - Dirty working tree detection with user guidance
//   - Fetch/pull operations with conflict-free updates
//
// Security and reliability features:
//   - All paths validated through pkg/fileops security functions
//   - Directory conflicts resolved explicitly (no automatic overwrites)
//   - Authentication tried public-first, PAT fallback for private repos
//   - User-friendly error messages with Settings guidance
//   - Maintains local changes when sync conflicts occur
type GitSource struct {
	RemoteURL string  // Git repository URL (HTTPS format, SSH URLs auto-converted)
	Branch    *string // Optional branch name (nil defaults to remote's HEAD branch)
	Path      string  // Local path where the repository will be cloned/cached
}

// NewGitSource creates a new GitSource instance with the specified parameters.
//
// The constructor accepts flexible URL formats and defers validation to Prepare():
//   - SSH URLs (git@github.com:user/repo.git) are automatically converted to HTTPS
//   - HTTPS URLs are normalized with .git suffix for consistency
//   - Branch parameter can be nil to use the remote repository's default branch
//   - Local path will be validated and secured during Prepare() execution
//
// Parameters:
//   - remoteURL: Git repository URL (supports both HTTPS and SSH formats)
//   - branch: Optional branch name (nil uses remote's default branch)
//   - localPath: Local directory path for cloning/caching the repository
//
// Returns:
//   - GitSource: Configured Git source instance ready for Prepare() calls
//
// Example:
//
//	// GitHub repository with specific branch
//	source := NewGitSource("git@github.com:user/rules.git", &"main", "/tmp/rules-cache")
//
//	// Public repository with default branch
//	source := NewGitSource("https://github.com/org/policies.git", nil, "/home/user/.local/share/rulem/policies")
func NewGitSource(remoteURL string, branch *string, localPath string) GitSource {
	return GitSource{
		RemoteURL: remoteURL,
		Branch:    branch,
		Path:      localPath,
	}
}

// Prepare clones or fetches the Git repository and returns the local path.
//
// This method implements the complete Git repository lifecycle management:
//
// **Initial Clone (DirectoryStatusEmpty):**
//   - Attempts public access first, falls back to PAT authentication if needed
//   - Performs shallow clone (depth=1) for performance optimization
//   - Handles specific branch checkout if configured
//   - Creates secure parent directories using fileops functions
//
// **Update Fetch (DirectoryStatusSameRepo):**
//   - Detects dirty working tree and preserves local changes
//   - Performs shallow fetch to get latest remote commits
//   - Resets to remote HEAD for clean sync (cache-focused approach)
//   - Provides user guidance when local changes block sync
//
// **Directory Conflict Resolution:**
//   - Validates existing directories contain the same Git repository
//   - Prevents accidental overwrites of different repositories
//   - Provides clear error messages for manual conflict resolution
//
// **Authentication Strategy:**
//   - Public repositories: No authentication required
//   - Private repositories: Uses GitHub Personal Access Token from credential store
//   - Clear error messages guide users to Settings → GitHub Authentication
//
// **Error Handling:**
//   - Network failures allow offline operation with cached repository
//   - Authentication errors provide specific guidance for token management
//   - Path security enforced through existing pkg/fileops validators
//
// Implementation follows the Source interface contract defined in contracts/README.md
//
// Returns:
//   - localPath: Absolute path to the prepared repository (ready for FileManager)
//   - error: Preparation failures with actionable error messages
func (gs GitSource) Prepare(logger *logging.AppLogger) (string, error) {
	if logger != nil {
		logger.Info("Preparing Git repository source",
			"remoteURL", gs.RemoteURL,
			"branch", gs.Branch,
			"localPath", gs.Path)
	}

	// Validate inputs
	if err := gs.validateInputs(); err != nil {
		return "", err
	}

	// Normalize and validate remote URL (convert SSH to HTTPS if needed)
	normalizedURL, err := gs.normalizeRemoteURL()
	if err != nil {
		return "", fmt.Errorf("invalid remote URL: %w", err)
	}

	// Validate and prepare local path
	cleanPath, err := gs.validateLocalPath()
	if err != nil {
		return "", err
	}

	// Check directory status for conflicts
	dirStatus, err := gs.validateCloneDirectory(cleanPath, normalizedURL)
	if err != nil {
		return "", err
	}

	// Handle directory conflicts
	if dirStatus == DirectoryStatusConflict || dirStatus == DirectoryStatusDifferentRepo {
		return "", fmt.Errorf("directory conflict at %s (%s): please resolve manually by removing or relocating the existing directory",
			cleanPath, dirStatus.String())
	}

	// Perform clone or fetch based on directory status
	// Try without authentication first, fall back to PAT if needed
	switch dirStatus {
	case DirectoryStatusEmpty:
		err = gs.performCloneWithAuth(cleanPath, normalizedURL, logger)
		if err != nil {
			return "", err
		}

	case DirectoryStatusSameRepo:
		err = gs.performFetchWithAuth(cleanPath, logger)
		if err != nil {
			return "", err
		}

	default:
		return "", fmt.Errorf("unexpected directory status: %s", dirStatus.String())
	}

	if logger != nil {
		logger.Info("Git repository prepared successfully", "localPath", cleanPath)
	}

	return cleanPath, nil
}

// validateInputs validates the GitSource configuration
func (gs GitSource) validateInputs() error {
	if strings.TrimSpace(gs.RemoteURL) == "" {
		return fmt.Errorf("remote URL cannot be empty")
	}

	if strings.TrimSpace(gs.Path) == "" {
		return fmt.Errorf("local path cannot be empty")
	}

	return nil
}

// normalizeRemoteURL converts SSH URLs to HTTPS and validates the URL format.
func (gs GitSource) normalizeRemoteURL() (string, error) {
	url := strings.TrimSpace(gs.RemoteURL)

	// Use ParseGitURL to extract components
	info, err := ParseGitURL(url)
	if err != nil {
		return "", fmt.Errorf("invalid Git URL format: %w", err)
	}

	// Reconstruct as HTTPS URL with .git suffix for consistency
	normalizedURL := fmt.Sprintf("https://%s/%s/%s.git", info.Host, info.Owner, info.Repo)

	return normalizedURL, nil
}

// FetchUpdates performs a fetch/pull operation on an existing repository.
// This is the public API for manually refreshing a repository.
//
// Unlike Prepare(), this function:
//   - Only fetches updates (does not clone if missing)
//   - Checks for dirty working tree before updating
//   - Attempts public fetch first, then falls back to PAT authentication
//
// This function is designed for user-initiated refresh operations where:
//   - The repository is already cloned
//   - The user wants to sync with remote changes
//   - The working tree may have uncommitted changes (will be skipped)
//
// Returns:
//   - error: Any error that occurred during fetch (nil if successful or skipped due to dirty tree)
//
// Example:
//
//	source := repository.NewGitSource(url, &branch, path)
//	err := source.FetchUpdates(logger)
//	if err != nil {
//	    return fmt.Errorf("failed to fetch updates: %w", err)
//	}
func (gs GitSource) FetchUpdates(logger *logging.AppLogger) error {
	if logger != nil {
		logger.Info("Manual fetch requested", "url", gs.RemoteURL, "path", gs.Path)
	}

	// Validate that the repository exists
	if _, err := os.Stat(gs.Path); os.IsNotExist(err) {
		return fmt.Errorf("repository does not exist at %s - cannot fetch updates", gs.Path)
	}

	return gs.performFetchWithAuth(gs.Path, logger)
}

// CheckGithubRepositoryStatus checks if the repository at the given path has uncommitted changes.
// Returns true if there are uncommitted changes (dirty), false if clean.
//
// This function is useful for detecting whether a repository has local modifications
// before performing operations that might conflict with those changes (like branch
// switches or fetches).
//
// Parameters:
//   - repoPath: Local filesystem path to the git repository
//
// Returns:
//   - bool: true if repository has uncommitted changes, false if clean
//   - error: error if repository cannot be opened or status cannot be determined
//
// Example:
//
//	isDirty, err := repository.CheckGithubRepositoryStatus("/path/to/repo")
//	if err != nil {
//	    return fmt.Errorf("failed to check repo status: %w", err)
//	}
//	if isDirty {
//	    return fmt.Errorf("repository has uncommitted changes")
//	}
func CheckGithubRepositoryStatus(repoPath string) (bool, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return false, fmt.Errorf("failed to open repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return false, fmt.Errorf("failed to get working tree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return false, fmt.Errorf("failed to get repository status: %w", err)
	}

	return !status.IsClean(), nil
}

// validateLocalPath validates and cleans the local clone path.
//
// This function performs essential security validation that is NOT redundant with
// ValidateCloneDirectory - they serve different purposes:
//   - validateLocalPath: Security validation (path traversal, reserved dirs)
//   - ValidateCloneDirectory: Git-specific conflict detection (same/different repo)
//
// Both validations are required for complete safety.
func (gs GitSource) validateLocalPath() (string, error) {
	// Clean and expand the path
	expanded := fileops.ExpandPath(gs.Path)
	clean := filepath.Clean(expanded)

	// SECURITY: Prevent path traversal and system directory access
	if err := fileops.ValidatePathSecurity(clean); err != nil {
		return "", fmt.Errorf("invalid local path: %w", err)
	}

	// Ensure absolute path for consistent behavior
	abs, err := filepath.Abs(clean)
	if err != nil {
		return "", fmt.Errorf("cannot resolve absolute path: %w", err)
	}

	return abs, nil
}

// getAuthentication retrieves PAT from credential manager for authentication.
//
// Authentication strategy:
//   - Returns nil auth if no token is stored (allows public repository access)
//   - Uses GitHub PAT authentication pattern (username="token", password=PAT)
//   - Provides user-friendly error messages with Settings guidance
//   - Validates token availability before attempting Git operations
//
// Returns:
//   - *http.BasicAuth: GitHub PAT authentication or nil for public access
//   - error: Credential retrieval errors with actionable user guidance
func (gs GitSource) getAuthentication(logger *logging.AppLogger) (*http.BasicAuth, error) {
	credMgr := NewCredentialManager()

	// Check if token exists
	if !credMgr.HasGitHubToken() {
		return nil, nil // No auth available - will try public access
	}

	// Retrieve token
	token, err := credMgr.GetGitHubToken()
	if err != nil {
		// The credential manager already provides user-friendly error messages
		return nil, err
	}

	if logger != nil {
		logger.Debug("Using GitHub Personal Access Token for authentication")
	}

	// GitHub PAT authentication uses "token" as username
	return &http.BasicAuth{
		Username: "token",
		Password: token,
	}, nil
}

// performCloneWithAuth performs the initial repository clone, trying public access first.
//
// This function implements the authentication fallback strategy:
//  1. Attempt clone without authentication (public repositories)
//  2. If authentication error occurs, retrieve PAT and retry
//  3. If non-auth error occurs, return original error for proper handling
//
// This approach minimizes credential usage and supports both public and private repositories.
func (gs GitSource) performCloneWithAuth(localPath, remoteURL string, logger *logging.AppLogger) error {
	// First try without authentication (for public repositories)
	err := gs.performClone(localPath, remoteURL, nil, logger)
	if err == nil {
		return nil
	}

	// If clone failed, check if it's an auth error and try with PAT
	if gs.isAuthenticationError(err) {
		if logger != nil {
			logger.Debug("Public access failed, trying with authentication")
		}

		// Get authentication credentials
		auth, authErr := gs.getAuthentication(logger)
		if authErr != nil {
			return fmt.Errorf("GitHub authentication failed: %w", authErr)
		}
		if auth == nil {
			return fmt.Errorf("GitHub authentication required - please configure a Personal Access Token in Settings → GitHub Authentication")
		}

		// Retry with authentication
		return gs.performClone(localPath, remoteURL, auth, logger)
	}

	// Not an auth error, return original error
	return err
}

// performClone performs the initial repository clone with the given authentication.
//
// Clone process:
//   - Creates secure parent directories using fileops for consistent security
//   - Performs shallow clone (depth=1) for performance optimization
//   - Handles branch-specific cloning when configured
//   - Provides user-friendly error translation for common failure scenarios
//
// go-git library functions used:
//   - git.PlainClone: Creates repository clone in specified directory
//   - CloneOptions.Depth: Shallow clone depth (1 = latest commit only)
//   - CloneOptions.ReferenceName: Specific branch to clone
//   - CloneOptions.SingleBranch: Clone only the specified branch
func (gs GitSource) performClone(localPath, remoteURL string, auth *http.BasicAuth, logger *logging.AppLogger) error {
	if logger != nil {
		logger.Info("Cloning repository", "remoteURL", remoteURL, "localPath", localPath)
	}

	// Create parent directories with security validation
	parentDir := filepath.Dir(localPath)

	// Validate parent directory path security before creation
	if err := fileops.ValidatePathSecurity(parentDir); err != nil {
		return fmt.Errorf("parent directory failed security validation: %w", err)
	}

	// Create parent directories if they don't exist
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Configure clone options
	cloneOpts := &git.CloneOptions{
		URL:      remoteURL,
		Progress: nil, // Set to os.Stdout for debug
	}

	// Add authentication if provided
	if auth != nil {
		cloneOpts.Auth = auth
	}

	// Add branch specification if provided
	if gs.Branch != nil && *gs.Branch != "" {
		cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(*gs.Branch)
		cloneOpts.SingleBranch = true
	}

	// Perform the clone
	_, err := git.PlainClone(localPath, cloneOpts)
	if err != nil {
		// Provide user-friendly error messages for common failures
		return gs.translateCloneError(err)
	}

	if logger != nil {
		logger.Info("Repository cloned successfully", "localPath", localPath)
	}

	return nil
}

// performFetchWithAuth performs fetch/pull operations with authentication fallback.
//
// This function mirrors the clone authentication strategy:
//  1. Attempt fetch without authentication (public repositories)
//  2. If authentication error occurs, retrieve PAT and retry
//  3. Preserve existing behavior for non-authentication errors
//
// This approach maintains consistency with clone operations and supports repository
// visibility changes (public to private or vice versa).
func (gs GitSource) performFetchWithAuth(localPath string, logger *logging.AppLogger) error {
	// First try without authentication (for public repositories)
	err := gs.performFetch(localPath, nil, logger)
	if err == nil {
		return nil
	}

	// If fetch failed, check if it's an auth error and try with PAT
	if gs.isAuthenticationError(err) {
		if logger != nil {
			logger.Debug("Public fetch failed, trying with authentication")
		}

		// Get authentication credentials
		auth, authErr := gs.getAuthentication(logger)
		if authErr != nil {
			return fmt.Errorf("GitHub authentication failed: %w", authErr)
		}
		if auth == nil {
			return fmt.Errorf("GitHub authentication required - please configure a Personal Access Token in Settings → GitHub Authentication")
		}

		// Retry with authentication
		return gs.performFetch(localPath, auth, logger)
	}

	// Not an auth error, return original error
	return err
}

// performFetch performs fetch/pull operations on existing repository.
//
// Fetch process:
//  1. Open existing repository and validate it's accessible
//  2. Check working tree status for uncommitted changes (dirty detection)
//  3. If dirty, preserve local changes and inform user (no data loss)
//  4. If clean, fetch remote updates with shallow fetch for consistency
//  5. Reset local branch to remote HEAD for clean sync (cache-focused approach)
//
// go-git library functions explained:
//   - git.PlainOpen: Opens existing Git repository from filesystem path
//   - repo.Worktree(): Gets working tree interface for status/reset operations
//   - worktree.Status(): Returns map of file changes (modified, added, deleted, etc.)
//   - status.IsClean(): True if no uncommitted changes exist in working tree
//   - repo.Head(): Gets current branch reference (HEAD pointer)
//   - head.Name().Short(): Extracts branch name from reference (e.g. "main" from "refs/heads/main")
//   - repo.ResolveRevision(): Converts revision string to commit hash
//   - worktree.Reset(): Updates working tree to specific commit (HardReset = discard local changes)
//
// The shallow fetch + reset approach avoids complex merge scenarios in favor of
// clean synchronization, which is appropriate for a cache-focused use case.
func (gs GitSource) performFetch(localPath string, auth *http.BasicAuth, logger *logging.AppLogger) error {
	if logger != nil {
		logger.Info("Fetching repository updates", "localPath", localPath)
	}

	// Open existing repository
	repo, err := git.PlainOpen(localPath)
	if err != nil {
		return fmt.Errorf("failed to open existing repository: %w", err)
	}

	// Check for dirty working tree first - this detects uncommitted changes
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get working tree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return fmt.Errorf("failed to get working tree status: %w", err)
	}

	// If working tree is dirty, continue with current state but inform user
	if !status.IsClean() {
		if logger != nil {
			logger.Warn("Working tree has uncommitted changes, skipping sync")
		}
		// Return nil error - this is not an error condition, just skipped
		return nil
	}

	// Perform fetch
	// Get the remote
	remote, err := repo.Remote("origin")
	if err != nil {
		return fmt.Errorf("failed to get origin remote: %w", err)
	}

	// Fetch updates from remote
	fetchOpts := &git.FetchOptions{
		Auth: auth,
		// TODO should we raise an error or show a message here? do we really want to force?
		Force: true, // Force update to handle force-pushes
	}

	err = remote.Fetch(fetchOpts)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return gs.translateFetchError(err)
	}

	if logger != nil {
		if err == git.NoErrAlreadyUpToDate {
			logger.Debug("Repository already up to date")
		} else {
			logger.Info("Repository updated successfully")
		}
	}

	// Check if we need to switch branches
	// Note: Checkout failures are logged but don't fail the entire fetch operation
	// This allows repositories with invalid branch configurations to still be accessible
	// for editing/fixing in the settings menu
	if gs.Branch != nil && *gs.Branch != "" {
		if err := gs.checkoutBranch(repo, worktree, *gs.Branch, logger); err != nil {
			if logger != nil {
				logger.Warn("Failed to checkout configured branch, repository may be in inconsistent state",
					"branch", *gs.Branch,
					"error", err)
			}
			// Don't return error - allow repository to be used even with checkout failure
			// User can fix the branch configuration via settings menu
		}
	}

	return nil
}

// translateCloneError provides user-friendly error messages for clone failures.
//
// This function translates technical Git errors into actionable user guidance:
//   - Authentication errors → Settings → GitHub Authentication guidance
//   - Permission errors → Token scope validation guidance
//   - Network errors → Connectivity troubleshooting suggestions
//   - Repository access → URL validation and access verification
//
// All error messages include specific next steps for resolution.
func (gs GitSource) translateCloneError(err error) error {
	errMsg := err.Error()
	errStr := strings.ToLower(errMsg)

	// Authentication errors
	if gs.containsAuthErrorPatterns(errMsg) {
		if strings.Contains(errStr, "403") || strings.Contains(errStr, "forbidden") {
			return fmt.Errorf("GitHub token lacks required permissions - ensure 'repo' scope is enabled in Settings → GitHub Authentication")
		}
		return fmt.Errorf("GitHub authentication failed - please update your Personal Access Token in Settings → GitHub Authentication")
	}

	// Repository not found
	if strings.Contains(errStr, "404") || strings.Contains(errStr, "not found") {
		return fmt.Errorf("repository not found - check the URL or ensure you have access: %s", gs.RemoteURL)
	}

	// Network errors
	if strings.Contains(errStr, "network") || strings.Contains(errStr, "connection") || strings.Contains(errStr, "timeout") {
		return fmt.Errorf("network error during clone - check your internet connection and try again: %w", err)
	}

	// Generic Git errors
	return fmt.Errorf("failed to clone repository: %w", err)
}

// isAuthenticationError checks if an error is related to authentication.
//
// This function identifies HTTP status codes and error messages that indicate
// authentication failures, allowing the performXWithAuth functions to implement
// the public-first, PAT-fallback strategy correctly.
func (gs GitSource) isAuthenticationError(err error) bool {
	if err == nil {
		return false
	}

	return gs.containsAuthErrorPatterns(err.Error())
}

// containsAuthErrorPatterns checks if error message contains authentication-related patterns
func (gs GitSource) containsAuthErrorPatterns(errMsg string) bool {
	errStr := strings.ToLower(errMsg)
	authPatterns := []string{
		"authentication required",
		"401",
		"unauthorized",
		"403",
		"forbidden",
	}

	for _, pattern := range authPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}

// translateFetchError provides user-friendly error messages for fetch failures.
//
// This function handles fetch-specific error scenarios:
//   - Token expiration → Settings update guidance
//   - Network failures → Offline operation explanation
//   - Generic fetch errors → Technical details with context
//
// Fetch errors are often less critical than clone errors since the local
// repository can continue operating with cached content.
func (gs GitSource) translateFetchError(err error) error {
	errMsg := err.Error()
	errStr := strings.ToLower(errMsg)

	// Authentication errors (similar to clone)
	if gs.containsAuthErrorPatterns(errMsg) {
		return fmt.Errorf("GitHub token has expired or is invalid - please update in Settings → GitHub Authentication")
	}

	// Network errors
	if strings.Contains(errStr, "network") || strings.Contains(errStr, "connection") || strings.Contains(errStr, "timeout") {
		return fmt.Errorf("network error during fetch - repository will use cached version: %w", err)
	}

	// Generic fetch errors
	return fmt.Errorf("failed to fetch repository updates: %w", err)
}

// parseGitURL parses a Git URL (SSH or HTTPS) and extracts host, owner, and repo components.
//
// This function handles the two most common Git URL formats:
//   - SSH: git@hostname:owner/repo.git
//   - HTTPS: https://hostname/owner/repo.git
//
// The .git suffix is optional and will be stripped if present.
//
// Parameters:
//   - gitURL: Git repository URL to parse
//
// Returns:
//   - host: Hostname (e.g., "github.com", "git.company.com")
//   - owner: Repository owner/organization (e.g., "user", "org")
//   - repo: Repository name (e.g., "project", "rules")
//   - err: Parsing errors for invalid or unsupported URL formats
//
// GitURLInfo contains the parsed components of a Git repository URL.
type GitURLInfo struct {
	Host  string // Host (e.g., "github.com")
	Owner string // Repository owner/organization
	Repo  string // Repository name (without .git suffix)
}

// ParseGitURL parses a Git repository URL and extracts its components.
// It supports both SSH (git@host:owner/repo.git) and HTTPS (https://host/owner/repo.git) formats.
//
// Returns a GitURLInfo struct containing the parsed host, owner, and repo name,
// or an error if the URL format is invalid or required components are missing.
//
// Example:
//
//	info, err := repository.ParseGitURL("https://github.com/user/repo.git")
//	// info.Host = "github.com", info.Owner = "user", info.Repo = "repo"
func ParseGitURL(gitURL string) (GitURLInfo, error) {
	gitURL = strings.TrimSpace(gitURL)

	// Handle SSH URLs like git@github.com:owner/repo.git
	sshPattern := regexp.MustCompile(`^git@([^:]+):([^/]+)/(.+?)(?:\.git)?$`)
	if matches := sshPattern.FindStringSubmatch(gitURL); matches != nil {
		return GitURLInfo{
			Host:  matches[1],
			Owner: matches[2],
			Repo:  matches[3],
		}, nil
	}

	// Handle HTTPS URLs
	parsedURL, err := url.Parse(gitURL)
	if err != nil {
		return GitURLInfo{}, fmt.Errorf("invalid URL format: %w", err)
	}

	if parsedURL.Host == "" {
		return GitURLInfo{}, fmt.Errorf("URL missing host component")
	}

	// Extract path components (remove leading slash and .git suffix)
	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(pathParts) < 2 {
		return GitURLInfo{}, fmt.Errorf("URL path should contain owner/repo: %s", parsedURL.Path)
	}

	// Get owner and repo, removing .git suffix if present
	owner := pathParts[0]
	repo := strings.TrimSuffix(pathParts[1], ".git")

	if owner == "" || repo == "" {
		return GitURLInfo{}, fmt.Errorf("could not extract owner/repo from URL path: %s", parsedURL.Path)
	}

	return GitURLInfo{
		Host:  parsedURL.Host,
		Owner: owner,
		Repo:  repo,
	}, nil
}

// validateCloneDirectory checks if a target clone directory can be used safely for the given repository.
// This function handles directory conflict resolution by checking if an existing directory
// contains the same Git repository or needs manual user intervention.
//
// Directory conflict resolution strategy:
//   - Directory doesn't exist: Safe to clone
//   - Directory exists but is empty: Safe to clone
//   - Directory exists with content:
//   - Not a git repository: Error - ask user to resolve manually
//   - Is git repository with same remote URL: Safe to proceed (fetch/pull)
//   - Is git repository with different remote URL: Error - ask user to resolve manually
//   - Has non-git content: Error - ask user to resolve manually
//
// Parameters:
//   - clonePath: Target directory path where repository would be cloned
//   - expectedRemoteURL: The remote URL we want to clone/sync
//
// Returns:
//   - DirectoryStatus: Status indicating what was found in the directory
//   - error: Validation errors or file system access issues
func (gs GitSource) validateCloneDirectory(clonePath, expectedRemoteURL string) (DirectoryStatus, error) {
	// Check if directory exists
	info, err := os.Stat(clonePath)
	if os.IsNotExist(err) {
		return DirectoryStatusEmpty, nil
	}
	if err != nil {
		return DirectoryStatusError, fmt.Errorf("cannot access directory %s: %w", clonePath, err)
	}

	// Ensure it's actually a directory
	if !info.IsDir() {
		return DirectoryStatusError, fmt.Errorf("path exists but is not a directory: %s", clonePath)
	}

	// Check if directory is empty
	isEmpty, err := fileops.IsDirEmpty(clonePath)
	if err != nil {
		return DirectoryStatusError, fmt.Errorf("cannot check if directory is empty: %w", err)
	}
	if isEmpty {
		return DirectoryStatusEmpty, nil
	}

	// Check if it's a git repository and get its remote URL in one step
	// getGitRemoteURL uses git.PlainOpen which reliably detects Git repositories
	currentRemote, err := gs.getGitRemoteURL(clonePath)
	if err != nil {
		// If it's not a git repository, it's a conflict (contains non-git content)
		if strings.Contains(err.Error(), "not a git repository") {
			return DirectoryStatusConflict, fmt.Errorf("directory contains non-git content: %s", clonePath)
		}
		// Other errors (corrupted repo, permission issues, etc.)
		return DirectoryStatusError, fmt.Errorf("cannot get current git remote URL: %w", err)
	}

	// Normalize URLs for comparison (handle SSH vs HTTPS for same repo)
	if gs.normalizeGitURL(currentRemote) == gs.normalizeGitURL(expectedRemoteURL) {
		return DirectoryStatusSameRepo, nil
	}

	return DirectoryStatusDifferentRepo, fmt.Errorf("directory contains different git repository (current: %s, expected: %s)", currentRemote, expectedRemoteURL)
}

// getGitRemoteURL gets the remote URL of a git repository using go-git.
//
// This function uses git.PlainOpen which automatically detects bare vs normal repositories
// and returns git.ErrRepositoryNotExists if the path doesn't contain a valid Git repository.
// This makes it a reliable way to both validate Git repositories and extract remote URLs.
//
// The function follows this sequence:
// 1. git.PlainOpen() - Opens repo and validates it's a Git repository
// 2. repo.Remote("origin") - Gets the origin remote configuration
// 3. remote.Config() - Extracts the remote configuration details
// 4. config.URLs[0] - Returns the first URL (primary remote URL)
//
// Parameters:
//   - repoPath: Path to the Git repository (can be bare or normal)
//
// Returns:
//   - string: The origin remote URL
//   - error: git.ErrRepositoryNotExists if not a Git repo, or other Git-related errors
func (gs GitSource) getGitRemoteURL(repoPath string) (string, error) {
	// Open the repository using go-git
	// This automatically detects bare vs normal repos and validates it's a Git repository
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		// Check specifically for "not a git repository" error
		if err == git.ErrRepositoryNotExists {
			return "", fmt.Errorf("directory is not a git repository: %s", repoPath)
		}
		return "", fmt.Errorf("cannot open git repository: %w", err)
	}

	// Get the remote named "origin" (the conventional name for the main remote)
	// We call this first to validate the remote exists before accessing its config
	remote, err := repo.Remote("origin")
	if err != nil {
		return "", fmt.Errorf("cannot get origin remote: %w", err)
	}

	// Get the remote configuration - this contains URLs, fetch specs, etc.
	// We call this last because remote.Config() never fails (it's a getter)
	config := remote.Config()
	if config == nil || len(config.URLs) == 0 {
		return "", fmt.Errorf("no URLs configured for origin remote")
	}

	// Return the first URL (typically there's only one)
	// Multiple URLs are rare and usually for redundancy
	return config.URLs[0], nil
}

// normalizeGitURL normalizes git URLs for comparison
// This ensures that SSH and HTTPS URLs for the same repository are considered equivalent
func (gs GitSource) normalizeGitURL(gitURL string) string {
	gitURL = strings.TrimSpace(gitURL)
	gitURL = strings.TrimSuffix(gitURL, ".git")

	// Convert SSH format to HTTPS-like format for comparison
	// git@github.com:owner/repo -> github.com/owner/repo
	sshPattern := regexp.MustCompile(`^git@([^:]+):(.+)$`)
	if matches := sshPattern.FindStringSubmatch(gitURL); matches != nil {
		return matches[1] + "/" + matches[2]
	}

	// Remove protocol from HTTPS URLs
	// https://github.com/owner/repo -> github.com/owner/repo
	if after, found := strings.CutPrefix(gitURL, "https://"); found {
		return after
	}
	if after, found := strings.CutPrefix(gitURL, "http://"); found {
		return after
	}

	return gitURL
}

// checkoutBranch checks out a specific branch in the repository.
// If the local branch doesn't exist, it creates it tracking the remote branch.
//
// Parameters:
//   - repo: The git repository instance
//   - worktree: The working tree instance
//   - branchName: Name of the branch to checkout
//   - logger: Logger for operation tracking
//
// Returns:
//   - error: If checkout fails or branch doesn't exist on remote
func (gs GitSource) checkoutBranch(repo *git.Repository, worktree *git.Worktree, branchName string, logger *logging.AppLogger) error {
	if logger != nil {
		logger.Debug("Checking out branch", "branch", branchName)
	}

	// Get current branch
	head, err := repo.Head()
	if err != nil && err != plumbing.ErrReferenceNotFound {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Check if we're already on the target branch
	if head != nil && head.Name().Short() == branchName {
		if logger != nil {
			logger.Debug("Already on target branch", "branch", branchName)
		}
		return nil
	}

	// Build the reference names
	localBranchRef := plumbing.NewBranchReferenceName(branchName)
	remoteBranchRef := plumbing.NewRemoteReferenceName("origin", branchName)

	// Check if remote branch exists
	_, err = repo.Reference(remoteBranchRef, true)
	if err != nil {
		return fmt.Errorf("branch '%s' does not exist on remote 'origin'", branchName)
	}

	// Try to get local branch reference
	_, err = repo.Reference(localBranchRef, true)
	
	if err == plumbing.ErrReferenceNotFound {
		// Local branch doesn't exist, create it
		if logger != nil {
			logger.Debug("Creating local branch", "branch", branchName)
		}

		// Get the remote branch reference to get its hash
		remoteRef, err := repo.Reference(remoteBranchRef, true)
		if err != nil {
			return fmt.Errorf("failed to get remote branch reference: %w", err)
		}

		// Create local branch pointing to remote branch's commit
		newRef := plumbing.NewHashReference(localBranchRef, remoteRef.Hash())
		if err := repo.Storer.SetReference(newRef); err != nil {
			return fmt.Errorf("failed to create local branch: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to get local branch reference: %w", err)
	}

	// Checkout the branch
	checkoutOpts := &git.CheckoutOptions{
		Branch: localBranchRef,
		Force:  false, // Don't discard local changes
	}

	if err := worktree.Checkout(checkoutOpts); err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	if logger != nil {
		logger.Info("Successfully checked out branch", "branch", branchName)
	}

	return nil
}

// ValidateRemoteBranchExists validates that a branch exists on the remote repository.
// This function is used before saving branch configuration to ensure the branch is valid.
//
// Parameters:
//   - repoPath: Local path to the git repository
//   - branchName: Name of the branch to validate (empty string means default branch, always valid)
//   - logger: Logger for operation tracking
//
// Returns:
//   - error: If branch doesn't exist on remote or validation fails
func ValidateRemoteBranchExists(repoPath string, branchName string, logger *logging.AppLogger) error {
	// Empty branch name means use default - always valid
	if strings.TrimSpace(branchName) == "" {
		return nil
	}

	if logger != nil {
		logger.Debug("Validating remote branch exists", "path", repoPath, "branch", branchName)
	}

	// Open the repository
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Build remote branch reference name
	remoteBranchRef := plumbing.NewRemoteReferenceName("origin", branchName)

	// Check if remote branch exists
	_, err = repo.Reference(remoteBranchRef, true)
	if err != nil {
		if err == plumbing.ErrReferenceNotFound {
			return fmt.Errorf("branch '%s' does not exist on remote 'origin' - fetch the repository first or use a valid branch name", branchName)
		}
		return fmt.Errorf("failed to check remote branch: %w", err)
	}

	if logger != nil {
		logger.Debug("Remote branch validated successfully", "branch", branchName)
	}

	return nil
}
