// Package settingsmenu provides the settings modification flow for the rulem TUI application.
package settingsmenu

import (
	"rulem/internal/tui/components"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Add Repository Type Selection Flow
// Flow: MainMenu → AddRepositoryType → [AddLocalName | AddGitHubName]
//
// This file contains the handler & view for repository type selection (Local vs GitHub).

// handleAddRepositoryTypeKeys processes user input in the AddRepositoryType state.
// User can navigate between Local and GitHub options and select one.
func (m *SettingsModel) handleAddRepositoryTypeKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.addRepositoryTypeIndex > 0 {
			m.addRepositoryTypeIndex--
		}
	case "down", "j":
		if m.addRepositoryTypeIndex < 1 {
			m.addRepositoryTypeIndex++
		}
	case "enter", " ":
		// Transition based on selected type
		if m.addRepositoryTypeIndex == 0 {
			// Local repository
			m.logger.LogUserAction("settings_add_type_local", "user selected local repository")
			return m.transitionToAddLocalName()
		} else {
			// GitHub repository
			m.logger.LogUserAction("settings_add_type_github", "user selected GitHub repository")
			return m.transitionToAddGitHubName()
		}
	case "esc":
		m.logger.LogUserAction("settings_add_type_cancelled", "returning to main menu")
		return m.transitionTo(SettingsStateMainMenu), nil
	}
	return m, nil
}

// transitionToAddRepositoryType transitions to the AddRepositoryType state.
// Resets the type selection index to default (Local).
func (m *SettingsModel) transitionToAddRepositoryType() (*SettingsModel, tea.Cmd) {
	m.addRepositoryTypeIndex = 0 // Reset to Local
	return m.transitionTo(SettingsStateAddRepositoryType), nil
}

// viewAddRepositoryType renders the repository type selection screen.
// Allows user to choose between Local and GitHub repository types.
func (m *SettingsModel) viewAddRepositoryType() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "➕ Add Repository",
		Subtitle: "Choose repository type",
		HelpText: "↑/↓ to navigate • Enter to continue • Esc to cancel",
	})

	var content string

	// Local option
	prefix := "  "
	if m.addRepositoryTypeIndex == 0 {
		prefix = "▸ "
	}
	content += lipgloss.NewStyle().Bold(m.addRepositoryTypeIndex == 0).Render(prefix + "📁 Local Repository")
	content += "\n"
	content += lipgloss.NewStyle().Faint(true).Render("   A local directory containing rule files")
	content += "\n\n"

	// GitHub option
	prefix = "  "
	if m.addRepositoryTypeIndex == 1 {
		prefix = "▸ "
	}
	content += lipgloss.NewStyle().Bold(m.addRepositoryTypeIndex == 1).Render(prefix + "🔗 GitHub Repository")
	content += "\n"
	content += lipgloss.NewStyle().Faint(true).Render("   Sync with a remote GitHub repository")

	return m.layout.Render(content)
}
