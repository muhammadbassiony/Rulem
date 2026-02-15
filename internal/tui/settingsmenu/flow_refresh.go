// Package settingsmenu provides the settings modification flow for the rulem TUI application.
package settingsmenu

import (
	"fmt"
	"rulem/internal/repository"
	"rulem/internal/tui/components"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Manual Refresh Flow
// Flow: RepositoryActions → ManualRefresh → RefreshInProgress → [RefreshError | Complete]
//
// This file contains all handlers, transitions, and business logic for manually
// refreshing a GitHub repository from its remote source.

// handleManualRefreshKeys processes user input in the ManualRefresh confirmation state.
// User can confirm (y/Y/Enter) or cancel (n/N/Esc) the refresh operation.
func (m *SettingsModel) handleManualRefreshKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		m.logger.LogUserAction("settings_manual_refresh_confirmed", "triggering refresh")
		// Check for dirty state before refresh
		return m, m.checkDirtyState(func(isDirty bool, err error) tea.Msg {
			return refreshDirtyStateMsg{isDirty: isDirty, err: err}
		})
	case "n", "N", "esc":
		m.logger.LogUserAction("settings_manual_refresh_cancelled", "returning to menu")
		return m.transitionTo(SettingsStateRepositoryActions), nil
	}
	return m, nil
}

// handleRefreshInProgressKeys processes user input during an active refresh operation.
// Most input is blocked during refresh, only allows Esc (for potential future cancellation).
func (m *SettingsModel) handleRefreshInProgressKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	// Block most input during refresh, only allow escape
	switch msg.String() {
	case "esc":
		m.logger.LogUserAction("settings_refresh_cancel_attempt", "user tried to cancel refresh")
		// Could implement cancellation here if needed
	}
	return m, nil
}

// handleRefreshErrorKeys processes user input on the refresh error screen.
// Any key dismisses the error and returns to repository actions menu.
func (m *SettingsModel) handleRefreshErrorKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	// Any key returns to repository actions
	m.logger.LogUserAction("settings_refresh_error_dismiss", "user dismissed error")
	m.lastRefreshError = nil
	return m.transitionTo(SettingsStateRepositoryActions), nil
}

// triggerRefresh initiates a manual refresh operation for a GitHub repository.
// This performs a git pull operation to sync with the remote repository.
func (m *SettingsModel) triggerRefresh() tea.Cmd {
	return func() tea.Msg {
		m.logger.Info("Starting manual refresh", "repositoryID", m.selectedRepositoryID)

		// Find the repository in current config
		selectedRepo, err := m.currentConfig.FindRepositoryByID(m.selectedRepositoryID)
		if err != nil {
			m.logger.Error("Repository not found for refresh", "error", err, "id", m.selectedRepositoryID)
			return refreshCompleteMsg{success: false, err: err}
		}

		// Only GitHub repositories can be refreshed
		if selectedRepo.Type != repository.RepositoryTypeGitHub {
			m.logger.Warn("Attempted to refresh non-GitHub repository", "type", selectedRepo.Type)
			return refreshCompleteMsg{
				success: false,
				err:     fmt.Errorf("cannot refresh: not a GitHub repository"),
			}
		}

		// Perform the refresh (git pull)
		// Create a git source and fetch updates
		if selectedRepo.RemoteURL == nil {
			return refreshCompleteMsg{
				success: false,
				err:     fmt.Errorf("GitHub repository missing remote URL"),
			}
		}

		source := repository.NewGitSource(
			*selectedRepo.RemoteURL,
			selectedRepo.Branch,
			selectedRepo.Path,
		)

		err = source.FetchUpdates(m.logger)
		if err != nil {
			m.logger.Error("Failed to refresh repository", "error", err, "path", selectedRepo.Path)
			return refreshCompleteMsg{success: false, err: err}
		}

		m.logger.Info("Repository refreshed successfully", "repositoryID", m.selectedRepositoryID)
		return refreshCompleteMsg{success: true, err: nil}
	}
}

// transitionToManualRefresh transitions to the ManualRefresh confirmation state.
// Sets up the state for confirming a manual refresh operation.
func (m *SettingsModel) transitionToManualRefresh() (*SettingsModel, tea.Cmd) {
	return m.transitionTo(SettingsStateManualRefresh), nil
}

// Views

// viewManualRefresh renders the manual refresh confirmation screen.
// Shows a confirmation prompt to pull latest changes from GitHub.
func (m *SettingsModel) viewManualRefresh() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "🔄 Manual Refresh",
		Subtitle: "Pull latest changes from GitHub",
		HelpText: "y to confirm • n to cancel • Esc to go back",
	})

	content := `This will pull the latest changes from your GitHub repository.

Any local changes that have been committed and pushed will be preserved.

Do you want to continue? (Y/n)`

	return m.layout.Render(content)
}

// viewRefreshInProgress renders the in-progress screen during a refresh operation.
// Shows a spinner or progress indicator while the git pull is executing.
func (m *SettingsModel) viewRefreshInProgress() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "🔄 Refreshing...",
		Subtitle: "Pulling changes from GitHub",
		HelpText: "Please wait",
	})

	content := lipgloss.NewStyle().Faint(true).Render("Syncing with remote repository...")

	return m.layout.Render(content)
}

// viewRefreshError renders the error screen when a refresh operation fails.
// Displays the error message and instructions to return.
func (m *SettingsModel) viewRefreshError() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "❌ Refresh Failed",
		Subtitle: "Cannot refresh repository",
		HelpText: "Press any key to return",
	})

	var content strings.Builder
	content.WriteString("Failed to refresh repository:\n\n")

	if m.lastRefreshError != nil {
		content.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff5f87")).
			Render(fmt.Sprintf("• %s", m.lastRefreshError.Error())))
	} else {
		content.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff5f87")).
			Render("• Unknown error occurred"))
	}

	content.WriteString("\n\n")
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).
		Render("💡 Common reasons:\n  - Network connectivity issues\n  - Invalid or expired GitHub PAT\n  - Local repository has uncommitted changes\n  - Merge conflicts with remote changes\n  - Repository not found or access denied"))

	return m.layout.Render(content.String())
}
