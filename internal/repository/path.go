package repository

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"rulem/pkg/fileops"
	"strings"

	"github.com/adrg/xdg"
	"github.com/go-git/go-git/v6"
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

// GetDefaultStorageDir returns the default storage directory in the user's data directory.
// This function was moved from internal/filemanager/storagedir.go to avoid circular dependencies
// when the repository package needs to reference the default storage location.
//
// Returns:
//   - string: Absolute path to the default storage directory (e.g., ~/.local/share/rulem)
func GetDefaultStorageDir() string {
	return filepath.Join(xdg.DataHome, "rulem")
}

// DeriveClonePath derives a simple, user-friendly local clone path from a Git remote URL.
//
// This function creates paths in the format: <DefaultStorageDir>/<repo>
// This simple structure makes repositories directly visible and manageable by users outside the tool.
//
// Examples:
//   - git@github.com:user/rules.git → ~/.local/share/rulem/rules
//   - https://github.com/user/rules.git → ~/.local/share/rulem/rules (same path!)
//   - https://git.company.com/team/project.git → ~/.local/share/rulem/project
//
// Repository name conflicts (same repo name from different sources) are handled explicitly
// rather than through complex nested directory structures.
//
// Parameters:
//   - remoteURL: Git repository URL (SSH or HTTPS format)
//
// Returns:
//   - string: Absolute path where the repository should be cloned
//   - error: Validation or parsing errors
func DeriveClonePath(remoteURL string) (string, error) {
	// Extract repository name from URL
	_, _, repoName, err := parseGitURL(remoteURL)
	if err != nil {
		return "", err
	}

	// Build the simple clone path: <DefaultStorageDir>/<repo>
	clonePath := filepath.Join(GetDefaultStorageDir(), repoName)

	// Validate the resulting path using existing fileops security functions
	cleanPath := filepath.Clean(clonePath)
	if err := fileops.ValidatePathSecurity(cleanPath); err != nil {
		return "", fmt.Errorf("derived path failed security validation: %w", err)
	}

	// Ensure the path is absolute (required for consistent behavior)
	if !filepath.IsAbs(cleanPath) {
		return "", fmt.Errorf("derived clone path must be absolute: %s", cleanPath)
	}

	return cleanPath, nil
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
func parseGitURL(gitURL string) (host, owner, repo string, err error) {
	gitURL = strings.TrimSpace(gitURL)

	// Handle SSH URLs like git@github.com:owner/repo.git
	sshPattern := regexp.MustCompile(`^git@([^:]+):([^/]+)/(.+?)(?:\.git)?$`)
	if matches := sshPattern.FindStringSubmatch(gitURL); matches != nil {
		return matches[1], matches[2], matches[3], nil
	}

	// Handle HTTPS URLs
	parsedURL, err := url.Parse(gitURL)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid URL format: %w", err)
	}

	if parsedURL.Host == "" {
		return "", "", "", fmt.Errorf("URL missing host component")
	}

	// Extract path components (remove leading slash and .git suffix)
	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(pathParts) < 2 {
		return "", "", "", fmt.Errorf("URL path should contain owner/repo: %s", parsedURL.Path)
	}

	// Get owner and repo, removing .git suffix if present
	owner = pathParts[0]
	repo = strings.TrimSuffix(pathParts[1], ".git")

	if owner == "" || repo == "" {
		return "", "", "", fmt.Errorf("could not extract owner/repo from URL path: %s", parsedURL.Path)
	}

	return parsedURL.Host, owner, repo, nil
}

// ValidateCloneDirectory checks if a target clone directory can be used safely for the given repository.
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
func ValidateCloneDirectory(clonePath, expectedRemoteURL string) (DirectoryStatus, error) {
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
	currentRemote, err := getGitRemoteURL(clonePath)
	if err != nil {
		// If it's not a git repository, it's a conflict (contains non-git content)
		if strings.Contains(err.Error(), "not a git repository") {
			return DirectoryStatusConflict, fmt.Errorf("directory contains non-git content: %s", clonePath)
		}
		// Other errors (corrupted repo, permission issues, etc.)
		return DirectoryStatusError, fmt.Errorf("cannot get current git remote URL: %w", err)
	}

	// Normalize URLs for comparison (handle SSH vs HTTPS for same repo)
	if normalizeGitURL(currentRemote) == normalizeGitURL(expectedRemoteURL) {
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
func getGitRemoteURL(repoPath string) (string, error) {
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
func normalizeGitURL(gitURL string) string {
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
