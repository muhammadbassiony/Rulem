// Package settingsmenu provides the settings modification flow for the rulem TUI application.
package settingsmenu

import (
	"context"
	"fmt"
	"rulem/internal/repository"
	"rulem/internal/tui/components"
	"rulem/internal/tui/styles"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// === Update PAT Flow ===
// Flow: UpdateGitHubPAT → UpdatePATConfirm → [UpdatePATError | Complete]
//
// This file contains all handlers, transitions, and business logic for updating
// or removing the GitHub Personal Access Token (PAT) for all GitHub repositories.
//
// IMPORTANT: PAT updates are GLOBAL - they affect ALL GitHub repositories.

// handleUpdateGitHubPATKeys processes user input in the UpdateGitHubPAT state.
// Validates the PAT and proceeds to confirmation if valid.
func (m *SettingsModel) handleUpdateGitHubPATKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.logger.LogUserAction("settings_pat_submit", "PAT provided")
		input := strings.TrimSpace(m.textInput.Value())

		if input == "" {
			m.logger.Warn("Empty PAT submitted")
			return m, func() tea.Msg {
				return updatePATErrorMsg{fmt.Errorf("PAT cannot be empty")}
			}
		}

		m.logger.Debug("Validating GitHub PAT format")
		// Validate token format
		if err := m.credManager.ValidateGitHubToken(input); err != nil {
			m.logger.Warn("GitHub PAT format validation failed", "error", err)
			return m, func() tea.Msg {
				return updatePATErrorMsg{fmt.Errorf("invalid PAT format: %w", err)}
			}
		}

		// Validate PAT against all GitHub repositories
		ctx := context.Background()
		if err := m.credManager.ValidateGitHubTokenForRepos(ctx, input, m.currentConfig.Repositories); err != nil {
			m.logger.Warn("GitHub PAT repository validation failed", "error", err)
			return m, func() tea.Msg {
				return updatePATErrorMsg{err}
			}
		}

		m.logger.Debug("GitHub PAT validated successfully")

		// Store the new PAT temporarily
		m.newGitHubPAT = input
		m.hasChanges = true
		m.changeType = ChangeOptionGitHubPAT

		// Transition to confirmation
		return m.transitionTo(SettingsStateUpdatePATConfirm), nil

	case "esc":
		m.logger.LogUserAction("settings_pat_cancel", "user cancelled PAT update")
		m.resetTemporaryChanges()
		return m.transitionTo(SettingsStateMainMenu), nil

	default:
		return m.updateTextInput(msg)
	}
}

// handleUpdatePATConfirmKeys processes user input in the UpdatePATConfirm state.
// Handles confirmation or cancellation of the PAT update.
func (m *SettingsModel) handleUpdatePATConfirmKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "enter", "y":
		m.logger.LogUserAction("settings_pat_confirm", "user confirmed PAT update")

		// Perform the PAT update
		if err := m.updateGitHubPAT(); err != nil {
			m.logger.Error("Failed to update GitHub PAT", "error", err)
			return m, func() tea.Msg {
				return updatePATErrorMsg{err}
			}
		}

		// Success - transition to complete
		m.logger.Info("GitHub PAT updated successfully")
		m.resetTemporaryChanges()
		return m.transitionTo(SettingsStateComplete), nil

	case "esc", "n":
		m.logger.LogUserAction("settings_pat_cancel_confirm", "user cancelled PAT update at confirmation")
		m.resetTemporaryChanges()
		return m.transitionTo(SettingsStateMainMenu), nil
	}
	return m, nil
}

// handleUpdatePATErrorKeys processes user input in the UpdatePATError state.
// Any key dismisses the error and returns to main menu.
func (m *SettingsModel) handleUpdatePATErrorKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	m.logger.LogUserAction("settings_pat_error_dismiss", "user dismissed error")
	m.layout = m.layout.ClearError()
	m.resetTemporaryChanges()
	return m.transitionTo(SettingsStateMainMenu), nil
}

// transitionToUpdateGitHubPAT transitions to the GitHub PAT update state.
// Sets up the text input for PAT entry.
func (m *SettingsModel) transitionToUpdateGitHubPAT() (*SettingsModel, tea.Cmd) {
	m.textInput.SetValue("")
	m.textInput.Placeholder = "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	m.textInput.EchoMode = textinput.EchoPassword
	m.textInput.Focus()

	return m.transitionTo(SettingsStateUpdateGitHubPAT), nil
}

// updateGitHubPAT updates the GitHub Personal Access Token.
// This is a GLOBAL operation that affects ALL GitHub repositories.
func (m *SettingsModel) updateGitHubPAT() error {
	m.logger.Info("Updating GitHub PAT")

	if m.newGitHubPAT == "" {
		return fmt.Errorf("no PAT provided for update")
	}

	if m.currentConfig == nil {
		return fmt.Errorf("current config is nil")
	}

	if err := m.credManager.ValidateGitHubTokenForRepos(m.context, m.newGitHubPAT, m.currentConfig.Repositories); err != nil {
		return err
	}

	m.logger.Debug("Storing GitHub PAT in keyring")
	if err := m.credManager.StoreGitHubToken(m.newGitHubPAT); err != nil {
		return fmt.Errorf("failed to store GitHub token: %w", err)
	}

	m.logger.Info("GitHub PAT updated successfully")
	return nil
}

// getGitHubRepositories filters and returns only GitHub repositories from the config.
// This is a helper function to identify which repositories will be affected by PAT changes.
func getGitHubRepositories(repos []repository.RepositoryEntry) []repository.RepositoryEntry {
	var github []repository.RepositoryEntry
	for _, repo := range repos {
		if repo.Type == repository.RepositoryTypeGitHub {
			github = append(github, repo)
		}
	}
	return github
}

// Views

// viewUpdateGitHubPAT renders the GitHub PAT input screen.
// Shows a multi-repo warning and lists all affected GitHub repositories.
func (m *SettingsModel) viewUpdateGitHubPAT() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "🔑 Update GitHub PAT",
		Subtitle: "Global Personal Access Token for all GitHub repositories",
		HelpText: "Enter to save • Esc to cancel",
	})

	var content strings.Builder

	// Multi-repo warning box
	warningStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ff8700")).
		Bold(true)

	content.WriteString(warningStyle.Render("⚠️  This PAT will be used for ALL GitHub repositories"))
	content.WriteString("\n\n")

	// List affected GitHub repositories
	githubRepos := getGitHubRepositories(m.currentConfig.Repositories)
	if len(githubRepos) > 0 {
		content.WriteString(lipgloss.NewStyle().Faint(true).Render("Affected repositories:"))
		content.WriteString("\n")
		for i, repo := range githubRepos {
			if i < 5 { // Show first 5 repositories
				content.WriteString(fmt.Sprintf("  • %s", repo.Name))
				if repo.RemoteURL != nil {
					content.WriteString(lipgloss.NewStyle().Faint(true).Render(fmt.Sprintf(" (%s)", *repo.RemoteURL)))
				}
				content.WriteString("\n")
			}
		}
		if len(githubRepos) > 5 {
			content.WriteString(lipgloss.NewStyle().Faint(true).Render(fmt.Sprintf("  ... and %d more", len(githubRepos)-5)))
			content.WriteString("\n")
		}
		content.WriteString("\n")
	} else {
		content.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Render("ℹ️  No GitHub repositories found. Add a GitHub repository first."))
		content.WriteString("\n\n")
	}

	// PAT input
	content.WriteString("Personal Access Token (PAT):\n")
	content.WriteString(styles.InputStyle.Render(m.textInput.View()))
	content.WriteString("\n\n")

	// Required permissions note
	content.WriteString(lipgloss.NewStyle().Faint(true).Render("Required permissions: repo (full control)"))
	content.WriteString("\n")
	content.WriteString(lipgloss.NewStyle().Faint(true).Render("Your PAT will be stored securely in your system keyring"))

	return m.layout.Render(content.String())
}

// viewUpdatePATConfirm renders the PAT update confirmation screen.
// Shows the number of repositories that will use the new PAT.
func (m *SettingsModel) viewUpdatePATConfirm() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "🔑 Confirm PAT Update",
		Subtitle: "Review changes before applying",
		HelpText: "Enter/y to confirm • Esc/n to cancel",
	})

	var content strings.Builder
	highlightStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#5fd7ff"))

	content.WriteString("You are about to update the GitHub Personal Access Token.\n\n")

	// Show affected repositories count
	githubRepos := getGitHubRepositories(m.currentConfig.Repositories)
	repoCount := len(githubRepos)

	if repoCount > 0 {
		content.WriteString(fmt.Sprintf("This will affect %s:\n\n",
			highlightStyle.Render(fmt.Sprintf("%d GitHub %s", repoCount, pluralize("repository", repoCount)))))

		// List up to 5 repositories
		for i, repo := range githubRepos {
			if i < 5 {
				content.WriteString(fmt.Sprintf("  • %s\n", repo.Name))
			}
		}
		if repoCount > 5 {
			content.WriteString(fmt.Sprintf("  ... and %d more\n", repoCount-5))
		}
		content.WriteString("\n")
	}

	// Security note
	content.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Render("🔐 Your PAT will be encrypted and stored in your system keyring."))

	return m.layout.Render(content.String())
}

// viewUpdatePATError renders the PAT update error screen.
// Displays the error message and helpful troubleshooting information.
func (m *SettingsModel) viewUpdatePATError() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "❌ PAT Update Failed",
		Subtitle: "Cannot update Personal Access Token",
		HelpText: "Press any key to return",
	})

	var content strings.Builder
	content.WriteString("Failed to update GitHub PAT:\n\n")

	if err := m.layout.GetError(); err != nil {
		content.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff5f87")).
			Render(fmt.Sprintf("• %s", err.Error())))
	} else {
		content.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff5f87")).
			Render("• Unknown error occurred"))
	}

	content.WriteString("\n\n")

	// Helpful troubleshooting information
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).
		Render("💡 Common issues:\n"))
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).
		Render("  • PAT format is invalid (should start with 'ghp_')\n"))
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).
		Render("  • PAT doesn't have 'repo' permissions\n"))
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).
		Render("  • PAT has expired or been revoked\n"))
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).
		Render("  • Network connection issues\n"))
	content.WriteString("\n")
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).
		Render("Create a new PAT at: https://github.com/settings/tokens"))

	return m.layout.Render(content.String())
}

// pluralize returns the singular or plural form of a word based on count.
func pluralize(word string, count int) string {
	if count == 1 {
		return word
	}
	return word + "ies" // Simple pluralization for "repository" -> "repositories"
}
