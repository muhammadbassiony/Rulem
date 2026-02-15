// Package settingsmenu provides the settings modification flow for the rulem TUI application.
package settingsmenu

import (
	"fmt"
	"rulem/internal/tui/components"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Repository Actions
// This file contains the update & view function for the repository actions menu.
// Flow: MainMenu → RepositoryActions → [various action flows based on selection]

// handleRepositoryActionsKeys handles actions for selected repository (renamed from handleSelectChangeKeys)
func (m *SettingsModel) handleRepositoryActionsKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	options := m.getMenuOptions()

	switch msg.String() {
	case "up", "k":
		if m.selectedRepositoryActionOption > 0 {
			m.selectedRepositoryActionOption--
		}
	case "down", "j":
		if m.selectedRepositoryActionOption < len(options)-1 {
			m.selectedRepositoryActionOption++
		}
	case "enter", " ":
		selected := options[m.selectedRepositoryActionOption]
		m.changeType = selected.Option
		m.logger.LogUserAction("settings_change_selected", selected.Title)

		switch selected.Option {
		case ChangeOptionBack:
			return m.transitionTo(SettingsStateMainMenu), nil
		case ChangeOptionGitHubBranch:
			return m.transitionToUpdateGitHubBranch()
		case ChangeOptionGitHubPath:
			return m.transitionToUpdateGitHubPath()
		case ChangeOptionChangeRepoName:
			return m.transitionToUpdateRepoName()
		case ChangeOptionManualRefresh:
			return m.transitionTo(SettingsStateManualRefresh), nil
		case ChangeOption(999): // Delete option
			m.logger.LogUserAction("settings_delete_repository", "user selected delete from menu")
			return m.transitionTo(SettingsStateConfirmDelete), nil
		}
	case "esc":
		return m.transitionTo(SettingsStateMainMenu), nil
	}
	return m, nil
}

// viewRepositoryActions renders the repository actions menu for a selected repository.
// Shows available actions based on repository type (Local vs GitHub).
// Local repositories: Delete, Rename
// GitHub repositories: Delete, Rename, Edit Branch, Edit Clone Path, Manual Refresh
func (m *SettingsModel) viewRepositoryActions() string {
	// Get selected repository info
	selectedRepo, err := m.currentConfig.FindRepositoryByID(m.selectedRepositoryID)
	var repoName string
	if err != nil {
		repoName = "Unknown Repository"
	} else {
		repoName = selectedRepo.Name
	}

	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    fmt.Sprintf("⚙️  Repository Actions: %s", repoName),
		Subtitle: "Choose an action",
		HelpText: "↑/↓ to navigate • Enter to select • Esc to go back",
	})

	var content strings.Builder

	// Get menu options for this repository
	options := m.getMenuOptions()

	// Render each option
	for i, option := range options {
		prefix := "  "
		if i == m.selectedRepositoryActionOption {
			prefix = "▸ "
		}

		// Render option title
		content.WriteString(lipgloss.NewStyle().Bold(i == m.selectedRepositoryActionOption).Render(prefix + option.Title))
		content.WriteString("\n")

		// Render option description
		if option.Description != "" {
			content.WriteString(lipgloss.NewStyle().Faint(true).Render("   " + option.Description))
			content.WriteString("\n")
		}

		// Add spacing between options
		if i < len(options)-1 {
			content.WriteString("\n")
		}
	}

	return m.layout.Render(content.String())
}

// getMenuOptions returns the list of available change options based on repository type
// this is used in the repository actions menu
func (m *SettingsModel) getMenuOptions() []ChangeOptionInfo {
	options := []ChangeOptionInfo{}

	if m.isGitHubRepo() {
		// GitHub repository options
		options = append(options,
			ChangeOptionInfo{
				Option:      ChangeOptionGitHubBranch,
				Title:       "🌿 Update GitHub Branch",
				Description: "Change the branch to sync with",
			},
			ChangeOptionInfo{
				Option:      ChangeOptionGitHubPath,
				Title:       "📂 Update Clone Path",
				Description: "Change where the repository is cloned locally",
			},
			ChangeOptionInfo{
				Option:      ChangeOptionManualRefresh,
				Title:       "🔄 Manual Refresh",
				Description: "Pull latest changes from GitHub now",
			},
		)
	}

	// Repository name editing (available for both Local and GitHub)
	options = append(options, ChangeOptionInfo{
		Option:      ChangeOptionChangeRepoName,
		Title:       "✏️ Change Repository Name",
		Description: "Update the display name for this repository",
	})

	// Delete option (always available if >1 repo)
	if len(m.currentConfig.Repositories) > 1 {
		options = append(options, ChangeOptionInfo{
			Option:      ChangeOption(999), // Use special value for delete
			Title:       "🗑️  Delete Repository",
			Description: "Remove this repository from configuration",
		})
	}

	// Back option
	options = append(options, ChangeOptionInfo{
		Option:      ChangeOptionBack,
		Title:       "← Back to Repository List",
		Description: "",
	})

	return options
}
