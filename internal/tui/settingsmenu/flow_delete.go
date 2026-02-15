// Package settingsmenu provides the settings modification flow for the rulem TUI application.
package settingsmenu

import (
	"fmt"
	"rulem/internal/repository"
	"rulem/internal/tui/components"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Delete Repository Flow
// Flow: RepositoryActions → ConfirmDelete → [DeleteError | Complete]
//
// This file contains all handlers, transitions, and business logic for deleting
// a repository from the configuration.

// handleConfirmDeleteKeys processes user input in the ConfirmDelete state.
// Handles confirmation or cancellation of the repository deletion.
func (m *SettingsModel) handleConfirmDeleteKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.logger.LogUserAction("settings_delete_confirmed", m.selectedRepositoryID)
		return m, m.deleteRepository()
	case "n", "N", "esc":
		m.logger.LogUserAction("settings_delete_cancelled", "returning to actions menu")
		return m.transitionTo(SettingsStateRepositoryActions), nil
	}
	return m, nil
}

// handleDeleteErrorKeys processes user input in the DeleteError state.
// Any key dismisses the error and returns to repository actions.
func (m *SettingsModel) handleDeleteErrorKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	m.logger.LogUserAction("settings_delete_error_dismiss", "user dismissed error")
	m.layout = m.layout.ClearError()
	return m.transitionTo(SettingsStateRepositoryActions), nil
}

// deleteRepository removes a repository from the configuration and reloads the list.
// This function includes safety checks to prevent deletion of the last repository.
func (m *SettingsModel) deleteRepository() tea.Cmd {
	return func() tea.Msg {
		// Safety check: Prevent deletion of the last repository
		if len(m.currentConfig.Repositories) <= 1 {
			m.logger.Warn("Cannot delete last repository", "count", len(m.currentConfig.Repositories))
			return deleteErrorMsg{fmt.Errorf("cannot delete the last repository.\n\nYou must have at least one repository configured.")}
		}

		// Find repository index
		idx := -1
		var deletedRepo *repository.RepositoryEntry
		for i, repo := range m.currentConfig.Repositories {
			if repo.ID == m.selectedRepositoryID {
				idx = i
				deletedRepo = &m.currentConfig.Repositories[i]
				break
			}
		}

		if idx == -1 {
			m.logger.Error("Repository not found for deletion", "id", m.selectedRepositoryID)
			return deleteErrorMsg{fmt.Errorf("repository not found: %s", m.selectedRepositoryID)}
		}

		m.logger.Info("Deleting repository",
			"id", m.selectedRepositoryID,
			"name", deletedRepo.Name,
			"type", deletedRepo.Type,
			"path", deletedRepo.Path)

		// Remove from slice
		m.currentConfig.Repositories = append(
			m.currentConfig.Repositories[:idx],
			m.currentConfig.Repositories[idx+1:]...,
		)

		// Save config
		if err := m.currentConfig.Save(); err != nil {
			m.logger.Error("Failed to save configuration after delete", "error", err)
			return deleteErrorMsg{fmt.Errorf("failed to save configuration: %w", err)}
		}

		// Reload repositories
		var err error
		m.preparedRepos, err = repository.PrepareAllRepositories(
			m.currentConfig.Repositories,
			m.logger,
		)
		if err != nil {
			m.logger.Warn("Failed to reload repositories after delete", "error", err)
			// Continue anyway - the repository is deleted from config
		}

		// Rebuild list with action items
		items := BuildSettingsMainMenuItems(m.preparedRepos)
		m.repoList.SetItems(items)

		m.logger.Info("Repository deleted successfully",
			"deleted", deletedRepo.Name,
			"remaining_count", len(m.currentConfig.Repositories))

		// Clear selection
		m.selectedRepositoryID = ""

		// Transition to main menu
		m.state = SettingsStateMainMenu
		return settingsCompleteMsg{}
	}
}

// getCleanupWarning returns a warning message about manual cleanup of local files.
// For GitHub repositories, it warns about the clone directory that remains.
// For local repositories, it notes that the directory remains unchanged.
func (m *SettingsModel) getCleanupWarning() string {
	if !m.isGitHubRepo() {
		return "Your current local directory will remain unchanged."
	}

	currentPath := ""
	if repo, err := m.currentConfig.FindRepositoryByID(m.selectedRepositoryID); err == nil {
		currentPath = repo.Path
	}

	if currentPath == "" {
		return "⚠️  Your GitHub clone directory will NOT be automatically deleted.\nYou may want to clean it up manually."
	}

	return fmt.Sprintf(`⚠️  Your current GitHub clone at:

  %s

will NOT be automatically deleted. You may want to clean it up manually.`, currentPath)
}

// Views

// viewConfirmDelete renders the repository deletion confirmation screen.
// Shows full repository information and a warning about manual cleanup.
func (m *SettingsModel) viewConfirmDelete() string {
	// Get selected repository info
	selectedRepo, err := m.currentConfig.FindRepositoryByID(m.selectedRepositoryID)
	var repoInfo string
	if err != nil {
		repoInfo = "Unknown Repository"
	} else {
		repoInfo = fmt.Sprintf("%s (%s: %s)", selectedRepo.Name, string(selectedRepo.Type), selectedRepo.Path)
	}

	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "⚠️  Confirm Repository Deletion",
		Subtitle: "This action cannot be undone",
		HelpText: "y to confirm deletion • n/Esc to cancel",
	})

	// Build warning message
	warningText := fmt.Sprintf(`You are about to delete the repository:

  %s

This will remove the repository from your configuration.

%s

Are you sure you want to proceed? (y/N)`,
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ff5f5f")).Render(repoInfo),
		lipgloss.NewStyle().Faint(true).Render(m.getCleanupWarning()))

	return m.layout.Render(warningText)
}

// viewDeleteError renders the repository deletion error screen.
// Displays the error message and instructions to return.
func (m *SettingsModel) viewDeleteError() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "❌ Deletion Failed",
		Subtitle: "Cannot delete repository",
		HelpText: "Press any key to return",
	})

	var content string
	content = "Failed to delete repository:\n\n"

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
		Render("💡 Common reasons:\n  - Cannot delete the last repository\n  - Repository not found\n  - Failed to save configuration")

	return m.layout.Render(content)
}
