// Package settingsmenu provides the settings modification flow for the rulem TUI application.
package settingsmenu

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"rulem/internal/tui/components"
)

// === Common View Functions ===
// This file contains shared view functions used across multiple flows.
// Includes completion views, confirmation views, and shared formatting helpers.

// viewComplete renders the completion screen after successful operations.
// Shows a success message and instructions to continue.
func (m *SettingsModel) viewComplete() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "✅ Settings Updated",
		Subtitle: "Your changes have been saved",
		HelpText: "Press any key to continue",
	})

	content := "Your settings have been updated successfully!\n\nThe changes will take effect immediately."

	return m.layout.Render(content)
}

// viewConfirmation renders the generic confirmation screen for reviewing changes.
// Shows a summary of changes and prompts for confirmation.
// DEPRECATED: Most flows now use flow-specific confirmation views.
func (m *SettingsModel) viewConfirmation() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "💾 Confirm Changes",
		Subtitle: "Review your changes",
		HelpText: "y to save • n to discard • Esc to go back",
	})

	content := m.formatChangesSummary()
	content += "\n\nSave these changes? (Y/n)"

	return m.layout.Render(content)
}

// viewError renders a generic error screen.
// DEPRECATED: Most flows now use flow-specific error views.
func (m *SettingsModel) viewError() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "❌ Error",
		Subtitle: "Failed to update settings",
		HelpText: "Press any key to continue",
	})

	content := "An error occurred while updating your settings.\nPlease check the error message above and try again."

	return m.layout.Render(content)
}

// formatChangesSummary formats a summary of pending changes for confirmation views.
// Shows what will be changed when the user confirms.
func (m *SettingsModel) formatChangesSummary() string {
	var summary strings.Builder

	// Get selected repository info
	selectedRepo, err := m.currentConfig.FindRepositoryByID(m.selectedRepositoryID)
	if err != nil {
		summary.WriteString("Repository: Unknown\n")
		return summary.String()
	}

	summary.WriteString(fmt.Sprintf("Repository: %s\n", lipgloss.NewStyle().Bold(true).Render(selectedRepo.Name)))
	summary.WriteString(fmt.Sprintf("Type: %s\n\n", selectedRepo.Type))

	// Show what's changing based on changeType
	summary.WriteString(lipgloss.NewStyle().Bold(true).Render("Changes:\n"))

	switch m.changeType {
	case ChangeOptionGitHubBranch:
		oldBranch := "default"
		if selectedRepo.Branch != nil {
			oldBranch = *selectedRepo.Branch
		}
		summary.WriteString(fmt.Sprintf("  Branch: %s → %s\n",
			lipgloss.NewStyle().Faint(true).Render(oldBranch),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")).Render(m.newGitHubBranch)))

	case ChangeOptionGitHubPath:
		summary.WriteString(fmt.Sprintf("  Clone Path: %s → %s\n",
			lipgloss.NewStyle().Faint(true).Render(selectedRepo.Path),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")).Render(m.newGitHubPath)))

	case ChangeOptionChangeRepoName:
		summary.WriteString(fmt.Sprintf("  Name: %s → %s\n",
			lipgloss.NewStyle().Faint(true).Render(selectedRepo.Name),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")).Render(m.addRepositoryName)))

	default:
		summary.WriteString("  Unknown change type\n")
	}

	return summary.String()
}

// getCleanupWarning is defined in flow_delete.go

// formatCurrentConfig formats the current configuration for display purposes.
// Shows a summary of all repositories and their settings.
func (m *SettingsModel) formatCurrentConfig() string {
	var content strings.Builder

	content.WriteString(lipgloss.NewStyle().Bold(true).Render("Current Configuration\n\n"))

	if len(m.currentConfig.Repositories) == 0 {
		content.WriteString(lipgloss.NewStyle().Faint(true).Render("No repositories configured"))
		return content.String()
	}

	for i, repo := range m.currentConfig.Repositories {
		// Repository name and type
		content.WriteString(fmt.Sprintf("%d. %s (%s)\n",
			i+1,
			lipgloss.NewStyle().Bold(true).Render(repo.Name),
			repo.Type))

		// Path
		content.WriteString(fmt.Sprintf("   Path: %s\n",
			lipgloss.NewStyle().Faint(true).Render(repo.Path)))

		// GitHub-specific details
		if repo.Type == "github" {
			if repo.RemoteURL != nil {
				content.WriteString(fmt.Sprintf("   URL: %s\n",
					lipgloss.NewStyle().Faint(true).Render(*repo.RemoteURL)))
			}
			if repo.Branch != nil {
				content.WriteString(fmt.Sprintf("   Branch: %s\n",
					lipgloss.NewStyle().Faint(true).Render(*repo.Branch)))
			}
		}

		// Add spacing between repositories
		if i < len(m.currentConfig.Repositories)-1 {
			content.WriteString("\n")
		}
	}

	return content.String()
}

// renderErrorWithContext renders an error message with contextual information.
// Includes the error, common causes, and suggested next steps.
func renderErrorWithContext(title, subtitle string, err error, commonCauses []string) string {
	var content strings.Builder

	// Error message
	if err != nil {
		content.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff5f87")).
			Render(fmt.Sprintf("• %s", err.Error())))
	} else {
		content.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff5f87")).
			Render("• Unknown error occurred"))
	}

	// Common causes
	if len(commonCauses) > 0 {
		content.WriteString("\n\n")
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).
			Render("💡 Common reasons:\n"))
		for _, cause := range commonCauses {
			content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).
				Render(fmt.Sprintf("  - %s\n", cause)))
		}
	}

	return content.String()
}
