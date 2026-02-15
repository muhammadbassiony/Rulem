// Package settingsmenu provides the settings modification flow for the rulem TUI application.
package settingsmenu

import (
	"fmt"
	"rulem/internal/config"
	"rulem/internal/repository"
	"rulem/internal/tui/components"
	"rulem/internal/tui/helpers/settingshelpers"
	"rulem/internal/tui/styles"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Edit Branch Flow
// Flow: UpdateGitHubBranch → (Dirty Check) → EditBranchConfirm → [EditBranchError | Complete]
//
// This file contains all handlers, transitions, and business logic for editing
// the GitHub branch of an existing repository.
//
// IMPORTANT: This flow includes a dirty state check to ensure the repository
// has no uncommitted changes before changing the branch.

// handleUpdateGitHubBranchKeys processes user input in the UpdateGitHubBranch state.
// Validates the branch name and triggers dirty state check before proceeding to confirmation.
func (m *SettingsModel) handleUpdateGitHubBranchKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		input := strings.TrimSpace(m.textInput.Value())
		m.logger.LogUserAction("settings_github_branch_submit", input)

		// Validate branch name (can be empty for default)
		if input != "" {
			if err := settingshelpers.ValidateBranchName(input); err != nil {
				m.logger.Warn("Branch validation failed", "error", err)
				return m, func() tea.Msg {
					return editBranchErrorMsg{err}
				}
			}
		}

		m.newGitHubBranch = input
		m.hasChanges = true
		m.changeType = ChangeOptionGitHubBranch

		// Check for dirty state before branch change (could require checkout)
		m.logger.Debug("Checking repository dirty state before branch change")
		return m, m.checkDirtyState(func(isDirty bool, err error) tea.Msg {
			return editBranchDirtyStateMsg{isDirty: isDirty, err: err}
		})

	case "esc":
		m.logger.LogUserAction("settings_branch_cancel", "user cancelled branch change")
		m.resetTemporaryChanges()
		return m.transitionTo(SettingsStateRepositoryActions), nil

	default:
		return m.updateTextInput(msg)
	}
}

// handleEditBranchConfirmKeys processes user input in the EditBranchConfirm state.
// Handles confirmation or cancellation of the branch change.
func (m *SettingsModel) handleEditBranchConfirmKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "enter", "y":
		m.logger.LogUserAction("settings_branch_confirm", "user confirmed branch change")
		return m, m.saveChanges()

	case "esc", "n":
		m.logger.LogUserAction("settings_branch_cancel", "user cancelled branch change at confirmation")
		m.resetTemporaryChanges()
		return m.transitionTo(SettingsStateRepositoryActions), nil
	}
	return m, nil
}

// handleEditBranchErrorKeys processes user input in the EditBranchError state.
// Any key dismisses the error and returns to repository actions.
func (m *SettingsModel) handleEditBranchErrorKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	m.logger.LogUserAction("settings_branch_error_dismiss", "user dismissed error")
	m.layout = m.layout.ClearError()
	m.resetTemporaryChanges()
	return m.transitionTo(SettingsStateRepositoryActions), nil
}

// transitionToUpdateGitHubBranch transitions to the GitHub branch update state.
// Sets up the text input with the current branch as default.
func (m *SettingsModel) transitionToUpdateGitHubBranch() (*SettingsModel, tea.Cmd) {
	defaultBranch := ""
	if m.currentConfig != nil {
		if repo, err := m.currentConfig.FindRepositoryByID(m.selectedRepositoryID); err == nil && repo.Branch != nil {
			defaultBranch = *repo.Branch
		}
	}

	m.textInput.SetValue(defaultBranch)
	m.textInput.Placeholder = "main (leave empty for default)"
	m.textInput.EchoMode = textinput.EchoNormal
	m.textInput.Focus()

	return m.transitionTo(SettingsStateUpdateGitHubBranch), nil
}

// updateGitHubBranch updates the GitHub branch for a repository in the configuration.
// Handles both setting a specific branch and using the default branch (nil).
// After updating the configuration, it triggers an immediate fetch to checkout the new branch.
func (m *SettingsModel) updateGitHubBranch(cfg *config.Config) error {
	repo, err := cfg.FindRepositoryByID(m.selectedRepositoryID)
	if err != nil {
		return fmt.Errorf("failed to get repository: %w", err)
	}

	oldBranch := "(default)"
	if repo.Branch != nil {
		oldBranch = *repo.Branch
	}

	m.logger.Info("Updating GitHub branch",
		"id", m.selectedRepositoryID,
		"repo", repo.Name,
		"old", oldBranch,
		"new", m.newGitHubBranch)

	// Validate that the branch exists on the remote before saving
	// (skip validation for empty branch as it means "use default")
	if m.newGitHubBranch != "" {
		m.logger.Debug("Validating branch exists on remote", "branch", m.newGitHubBranch, "path", repo.Path)
		if err := repository.ValidateRemoteBranchExists(repo.Path, m.newGitHubBranch, m.logger); err != nil {
			return fmt.Errorf("branch validation failed: %w", err)
		}
		m.logger.Debug("Branch exists on remote", "branch", m.newGitHubBranch)
	}

	// Handle branch value
	if m.newGitHubBranch == "" {
		// Empty branch means use default
		repo.Branch = nil
	} else {
		repo.Branch = &m.newGitHubBranch
	}

	// Update in the config array
	for i := range cfg.Repositories {
		if cfg.Repositories[i].ID == m.selectedRepositoryID {
			cfg.Repositories[i] = *repo
			break
		}
	}

	// Save the config
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save GitHub branch configuration: %w", err)
	}

	m.logger.Info("Branch updated successfully",
		"repo", repo.Name,
		"new_branch", m.newGitHubBranch)

	// Trigger an immediate fetch to checkout the new branch
	// This ensures the working directory reflects the new branch immediately
	m.logger.Info("Triggering fetch to checkout new branch", "branch", m.newGitHubBranch)

	if repo.RemoteURL == nil {
		m.logger.Warn("Cannot fetch: repository missing remote URL")
		return nil // Don't fail the update, just skip the fetch
	}

	source := repository.NewGitSource(
		*repo.RemoteURL,
		repo.Branch,
		repo.Path,
	)

	if err := source.FetchUpdates(m.logger); err != nil {
		m.logger.Warn("Failed to fetch after branch update (config saved successfully)", "error", err)
		// Don't return error - config was saved successfully
		// The fetch will happen on next manual refresh
	} else {
		m.logger.Info("Successfully fetched and checked out new branch")
	}

	return nil
}

// VIews

// viewUpdateGitHubBranch renders the GitHub branch input screen.
// Shows the current branch and allows entering a new branch name.
func (m *SettingsModel) viewUpdateGitHubBranch() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "🌿 Update GitHub Branch",
		Subtitle: "Change the branch to sync",
		HelpText: "Enter to save • Esc to cancel",
	})

	var content string

	// Show current branch if available
	if m.currentConfig != nil {
		if repo, err := m.currentConfig.FindRepositoryByID(m.selectedRepositoryID); err == nil {
			currentBranch := "(default)"
			if repo.Branch != nil {
				currentBranch = *repo.Branch
			}
			content += fmt.Sprintf("Repository: %s\n", lipgloss.NewStyle().Bold(true).Render(repo.Name))
			content += fmt.Sprintf("Current branch: %s\n\n", lipgloss.NewStyle().Faint(true).Render(currentBranch))
		}
	}

	content += "Branch name (leave empty for default):\n"
	content += styles.InputStyle.Render(m.textInput.View())
	content += "\n\n"
	content += lipgloss.NewStyle().Faint(true).Render("💡 The repository will checkout to the new branch on next sync.")

	return m.layout.Render(content)
}

// viewEditBranchConfirm renders the branch change confirmation screen.
// Shows the old and new branch names for review before applying.
func (m *SettingsModel) viewEditBranchConfirm() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "🌿 Confirm Branch Change",
		Subtitle: "Review your changes",
		HelpText: "Enter/y to confirm • Esc/n to cancel",
	})

	var content string
	highlightStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#5fd7ff"))

	// Get current repository info
	if repo, err := m.currentConfig.FindRepositoryByID(m.selectedRepositoryID); err == nil {
		content += fmt.Sprintf("Repository: %s\n\n", highlightStyle.Render(repo.Name))

		currentBranch := "(default)"
		if repo.Branch != nil {
			currentBranch = *repo.Branch
		}

		newBranch := m.newGitHubBranch
		if newBranch == "" {
			newBranch = "(default)"
		}

		content += fmt.Sprintf("Current branch: %s\n", lipgloss.NewStyle().Faint(true).Render(currentBranch))
		content += fmt.Sprintf("New branch:     %s\n\n", highlightStyle.Render(newBranch))

		content += lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")).Render("⚠️  Note:") + "\n"
		content += "The repository will checkout to the new branch on next sync.\n"
		content += lipgloss.NewStyle().Faint(true).Render("Make sure all changes are committed before syncing.\n\n")
	}

	content += "Do you want to proceed? (y/N)"

	return m.layout.Render(content)
}

// viewEditBranchError renders the branch change error screen.
// Displays the error message and instructions to return.
func (m *SettingsModel) viewEditBranchError() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "❌ Branch Update Failed",
		Subtitle: "Cannot change branch",
		HelpText: "Press any key to return",
	})

	var content string
	content = "Failed to update branch:\n\n"

	if err := m.layout.GetError(); err != nil {
		content += lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff5f87")).
			Render(fmt.Sprintf("• %s", err.Error()))
	} else {
		content += lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff5f87")).
			Render("• Unknown error occurred")
	}

	content += "\n\n"
	content += lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).
		Render("💡 Common issues:\n")
	content += lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).
		Render("  • Repository has uncommitted changes\n")
	content += lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).
		Render("  • Invalid branch name\n")
	content += lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).
		Render("  • Failed to save configuration\n")
	content += "\n"
	content += "Press any key to return to repository actions."

	return m.layout.Render(content)
}
