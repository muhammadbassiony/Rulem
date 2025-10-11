// Package settingshelpers provides shared utilities for setup and settings menu flows.
// This package extracts common logic for repository configuration, validation,
// and UI patterns to promote code reuse and consistency across setup and settings menus.
package settingshelpers

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"rulem/internal/repository"
	"rulem/pkg/fileops"
)

// RepositoryTypeOption represents a repository type choice in the UI.
type RepositoryTypeOption struct {
	Type  string // "local" or "github"
	Title string // Display text with emoji
	Desc  string // Description of the option
}

// GetRepositoryTypeOptions returns the available repository type options
// for display in setup/settings menus.
func GetRepositoryTypeOptions() []RepositoryTypeOption {
	return []RepositoryTypeOption{
		{
			Type:  "local",
			Title: "📁 Local Directory",
			Desc:  "Store rules in a local directory on your filesystem",
		},
		{
			Type:  "github",
			Title: "🌐 GitHub Repository",
			Desc:  "Sync rules with a GitHub repository (requires authentication)",
		},
	}
}

// ValidateGitHubURL validates a GitHub repository URL format by attempting to parse it.
// This is a convenience wrapper around repository.ParseGitURL that provides
// user-friendly error messages for UI validation.
//
// Returns:
//   - error: Validation error if URL format is invalid
func ValidateGitHubURL(urlStr string) error {
	urlStr = strings.TrimSpace(urlStr)

	if urlStr == "" {
		return fmt.Errorf("repository URL cannot be empty")
	}

	// Use existing ParseGitURL for validation
	_, err := repository.ParseGitURL(urlStr)
	if err != nil {
		return fmt.Errorf("invalid repository URL format: %w", err)
	}

	return nil
}

// ValidateBranchName validates a git branch name format.
// Follows git naming conventions:
//   - Empty is allowed (means use default branch)
//   - Cannot contain spaces
//   - Cannot contain certain special characters
//   - Cannot start or end with /
//   - Cannot have consecutive dots (..)
//
// Returns:
//   - error: Validation error if branch name is invalid
func ValidateBranchName(branch string) error {
	branch = strings.TrimSpace(branch)

	// Empty is allowed (means use default branch)
	if branch == "" {
		return nil
	}

	// Cannot contain spaces
	if strings.Contains(branch, " ") {
		return fmt.Errorf("branch name cannot contain spaces")
	}

	// Cannot start or end with /
	if strings.HasPrefix(branch, "/") || strings.HasSuffix(branch, "/") {
		return fmt.Errorf("branch name cannot start or end with /")
	}

	// Cannot have consecutive dots
	if strings.Contains(branch, "..") {
		return fmt.Errorf("branch name cannot contain consecutive dots (..)")
	}

	// Cannot contain certain special characters that git doesn't allow
	invalidChars := []string{"~", "^", ":", "?", "*", "[", "\\", " ", "\t", "\n"}
	for _, char := range invalidChars {
		if strings.Contains(branch, char) {
			return fmt.Errorf("branch name cannot contain '%s'", char)
		}
	}

	// Cannot be just a dot or end with .lock
	if branch == "." || strings.HasSuffix(branch, ".lock") {
		return fmt.Errorf("invalid branch name: %s", branch)
	}

	return nil
}

// DeriveClonePath suggests a default clone path based on the repository URL.
// It uses repository.ParseGitURL to extract the repository name and creates
// a path under the default storage directory.
//
// Examples:
//   - https://github.com/user/repo.git -> ~/.local/share/rulem/repo
//   - git@github.com:user/my-rules.git -> ~/.local/share/rulem/my-rules
//
// Parameters:
//   - repoURL: GitHub repository URL
//
// Returns:
//   - string: Suggested clone path
func DeriveClonePath(repoURL string) string {
	repoURL = strings.TrimSpace(repoURL)

	// Use existing ParseGitURL to extract repository name
	info, err := repository.ParseGitURL(repoURL)
	if err != nil || info.Repo == "" {
		// Fallback to default if parsing fails
		return repository.GetDefaultStorageDir()
	}

	// Combine repository name with default storage directory
	return filepath.Join(repository.GetDefaultStorageDir(), info.Repo)
}

// FormatPATDisplay formats a Personal Access Token for display by masking
// most characters but showing the first and last few characters for identification.
//
// Examples:
//   - "ghp_1234567890abcdef" -> "ghp_•••••••def"
//   - "github_pat_11A..." -> "gith•••••..."
//
// Parameters:
//   - pat: Personal Access Token to format
//
// Returns:
//   - string: Masked PAT for display
func FormatPATDisplay(pat string) string {
	pat = strings.TrimSpace(pat)

	// If very short, mask entirely
	if len(pat) <= 8 {
		return strings.Repeat("•", len(pat))
	}

	// Show first 4 and last 4 characters, mask the middle
	first := pat[:4]
	last := pat[len(pat)-4:]
	middleLen := len(pat) - 8

	return first + strings.Repeat("•", middleLen) + last
}

// GetRepositoryTypeName returns a human-readable name for the repository type.
func GetRepositoryTypeName(isRemote bool) string {
	if isRemote {
		return "GitHub Repository"
	}
	return "Local Directory"
}

// GetManualCleanupWarning returns a warning message about manual cleanup
// when switching repository types.
//
// Parameters:
//   - fromGitHub: true if switching from GitHub to Local
//   - oldPath: the old repository path
//   - newPath: the new repository path
//
// Returns:
//   - string: Formatted cleanup warning message
func GetManualCleanupWarning(fromGitHub bool, oldPath, newPath string) string {
	if fromGitHub {
		return fmt.Sprintf(`⚠️  Manual Cleanup Required

The GitHub clone directory at:
  %s

will remain on your system and will NOT be deleted automatically.

If you no longer need it, you should manually delete it:
  rm -rf "%s"

Your new local repository will be at:
  %s`, oldPath, oldPath, newPath)
	}

	return fmt.Sprintf(`ℹ️  Note

Your old local directory at:
  %s

will remain unchanged and will NOT be deleted automatically.

The new GitHub repository will be cloned to:
  %s

You may want to manually clean up the old directory if no longer needed.`, oldPath, newPath)
}

// GetDirtyStateWarning returns a formatted warning message for repositories
// with uncommitted changes.
//
// Parameters:
//   - repoPath: Path to the repository with dirty state
//
// Returns:
//   - string: Formatted warning message with cleanup instructions
func GetDirtyStateWarning(repoPath string) string {
	return fmt.Sprintf(`⚠️  Uncommitted Changes Detected

Your local repository has uncommitted changes that haven't been pushed to GitHub.

⛔ Please clean your repository before proceeding:

Option 1 - Commit and push your changes:
  cd "%s"
  git add .
  git commit -m "Your commit message"
  git push

Option 2 - Discard changes (careful - this is destructive!):
  cd "%s"
  git reset --hard HEAD

After cleaning, return to settings and retry the operation.

We do NOT automatically clean or modify your repository.
All cleanup must be done manually to prevent accidental data loss.`, repoPath, repoPath)
}

// ResetTextInputForState is a helper to reset and configure text input for a new state.
// This is commonly used in setup and settings flows to transition between input fields.
//
// Parameters:
//   - textInput: The text input model to configure
//   - value: Initial value for the input
//   - placeholder: Placeholder text to display
//   - echoMode: Echo mode (Normal, Password, etc.)
//
// Returns:
//   - tea.Cmd: Command to blink the cursor
func ResetTextInputForState(textInput *textinput.Model, value, placeholder string, echoMode textinput.EchoMode) tea.Cmd {
	textInput.Reset()
	textInput.SetValue(value)
	textInput.Placeholder = placeholder
	textInput.EchoMode = echoMode
	textInput.Focus()
	return textinput.Blink
}

// ValidateAndExpandLocalPath validates a local storage path and expands it to an absolute path.
// This is used by both setup and settings flows to ensure consistent path validation.
//
// The function:
//   - Trims whitespace from input
//   - Validates path security and accessibility using fileops.ValidateStoragePath
//   - Expands path shortcuts like ~/ to absolute paths
//
// Parameters:
//   - path: The path to validate and expand
//
// Returns:
//   - string: The expanded absolute path
//   - error: Validation error if the path is invalid
func ValidateAndExpandLocalPath(path string) (string, error) {
	input := strings.TrimSpace(path)

	// Validate the storage path
	if err := fileops.ValidateStoragePath(input); err != nil {
		return "", err
	}

	// Expand the path (handles ~/ and other shortcuts)
	expandedPath := fileops.ExpandPath(input)

	return expandedPath, nil
}
