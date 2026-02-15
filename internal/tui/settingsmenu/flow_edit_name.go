// Package settingsmenu provides the settings modification flow for the rulem TUI application.
package settingsmenu

import (
	"fmt"
	"rulem/internal/config"
	"rulem/internal/tui/components"
	"rulem/internal/tui/helpers/settingshelpers"
	"rulem/internal/tui/styles"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Edit Repository Name Flow
// Flow: UpdateRepoName → EditNameConfirm → [EditNameError | Complete]
//
// This file contains all handlers, transitions, and business logic for editing
// the name of an existing repository (works for both Local and GitHub repositories).

// handleUpdateRepoNameKeys processes user input in the UpdateRepoName state.
// Validates the new name and proceeds to confirmation if valid.
func (m *SettingsModel) handleUpdateRepoNameKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.logger.LogUserAction("settings_repo_name_submit", m.textInput.Value())
		input := strings.TrimSpace(m.textInput.Value())

		// Validate name is not empty
		if input == "" {
			err := fmt.Errorf("repository name cannot be empty")
			m.logger.Warn("Empty repository name", "error", err)
			return m, func() tea.Msg { return editNameErrorMsg{err} }
		}

		// Validate name length
		if len(input) > 100 {
			err := fmt.Errorf("repository name must be 100 characters or less")
			m.logger.Warn("Repository name too long", "error", err, "length", len(input))
			return m, func() tea.Msg { return editNameErrorMsg{err} }
		}

		// Validate name is unique across all repositories
		if err := m.validateRepositoryNameUnique(input); err != nil {
			m.logger.Warn("Repository name validation failed", "error", err, "name", input)
			return m, func() tea.Msg { return editNameErrorMsg{err} }
		}

		// Store the new name
		m.addRepositoryName = input
		m.hasChanges = true
		m.changeType = ChangeOptionChangeRepoName

		// No dirty state check needed for name changes - proceed to confirmation
		m.logger.Debug("Transitioning to name change confirmation")
		return m.transitionTo(SettingsStateEditNameConfirm), nil

	case "esc":
		m.resetTemporaryChanges()
		return m.transitionTo(SettingsStateRepositoryActions), nil

	default:
		return m.updateTextInput(msg)
	}
}

// handleEditNameConfirmKeys processes user input in the EditNameConfirm state.
// Handles confirmation or cancellation of the repository name change.
func (m *SettingsModel) handleEditNameConfirmKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "enter", "y":
		m.logger.LogUserAction("settings_name_confirm", "user confirmed name change")
		return m, m.saveChanges()
	case "esc", "n":
		m.logger.LogUserAction("settings_name_cancel", "user cancelled name change")
		m.resetTemporaryChanges()
		return m.transitionTo(SettingsStateRepositoryActions), nil
	}
	return m, nil
}

// handleEditNameErrorKeys processes user input in the EditNameError state.
// Any key dismisses the error and returns to repository actions.
func (m *SettingsModel) handleEditNameErrorKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	m.logger.LogUserAction("settings_name_error_dismiss", "user dismissed error")
	m.layout = m.layout.ClearError()
	m.resetTemporaryChanges()
	return m.transitionTo(SettingsStateRepositoryActions), nil
}

// transitionToUpdateRepoName transitions to the repository name update state.
// Sets up the text input with current name as default.
func (m *SettingsModel) transitionToUpdateRepoName() (*SettingsModel, tea.Cmd) {
	currentName := ""
	if m.currentConfig != nil {
		if repo, err := m.currentConfig.FindRepositoryByID(m.selectedRepositoryID); err == nil {
			currentName = repo.Name
		}
	}

	return m.transitionTo(SettingsStateUpdateRepoName), settingshelpers.ResetTextInputForState(&m.textInput, currentName, "Enter repository name", textinput.EchoNormal)
}

// validateRepositoryNameUnique checks if the repository name is unique in the configuration.
// Returns an error if the name already exists (excluding the current repository being edited).
func (m *SettingsModel) validateRepositoryNameUnique(name string) error {
	if m.currentConfig == nil {
		return fmt.Errorf("configuration not loaded")
	}

	for _, repo := range m.currentConfig.Repositories {
		// Skip the repository being edited
		if repo.ID == m.selectedRepositoryID {
			continue
		}
		// Check for duplicate name
		if repo.Name == name {
			return fmt.Errorf("repository name '%s' already exists", name)
		}
	}

	return nil
}

// updateRepositoryName updates the repository name in the configuration.
// Works for both Local and GitHub repositories.
func (m *SettingsModel) updateRepositoryName(cfg *config.Config) error {
	repo, err := cfg.FindRepositoryByID(m.selectedRepositoryID)
	if err != nil {
		return fmt.Errorf("failed to get repository: %w", err)
	}

	oldName := repo.Name
	m.logger.Info("Updating repository name", "id", m.selectedRepositoryID, "old", oldName, "new", m.addRepositoryName)

	if m.addRepositoryName == "" {
		return fmt.Errorf("no repository name provided")
	}

	// Validate uniqueness one more time before saving
	if err := m.validateRepositoryNameUnique(m.addRepositoryName); err != nil {
		return err
	}

	repo.Name = m.addRepositoryName

	// Update in the config array
	for i := range cfg.Repositories {
		if cfg.Repositories[i].ID == m.selectedRepositoryID {
			cfg.Repositories[i] = *repo
			break
		}
	}

	// Save the config
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save repository name configuration: %w", err)
	}

	m.logger.Info("Repository name updated successfully", "old", oldName, "new", m.addRepositoryName)

	return nil
}

// Views

// viewUpdateRepoName renders the repository name input screen.
func (m *SettingsModel) viewUpdateRepoName() string {
	title := "✏️ Update Repository Name"

	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    title,
		Subtitle: "Enter a new name for the repository",
		HelpText: "Enter to save • Esc to cancel",
	})

	var content strings.Builder

	// Show current repository info
	if repo, err := m.currentConfig.FindRepositoryByID(m.selectedRepositoryID); err == nil {
		content.WriteString(fmt.Sprintf("Current name: %s\n\n", lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(repo.Name)))
	}

	content.WriteString("New name:\n")
	content.WriteString(styles.InputStyle.Render(m.textInput.View()))
	content.WriteString("\n\n")
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("💡 The repository name is used for display purposes only."))

	return m.layout.Render(content.String())
}

// viewEditNameConfirm renders the repository name change confirmation screen.
func (m *SettingsModel) viewEditNameConfirm() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "✏️ Confirm Name Change",
		Subtitle: "Review your changes",
		HelpText: "Enter/y to confirm • Esc/n to cancel",
	})

	var content strings.Builder
	highlightStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#5fd7ff"))

	// Get current repository info
	if repo, err := m.currentConfig.FindRepositoryByID(m.selectedRepositoryID); err == nil {
		content.WriteString("Repository name will be updated:\n\n")
		content.WriteString(fmt.Sprintf("  Old: %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(repo.Name)))
		content.WriteString(fmt.Sprintf("  New: %s\n\n", highlightStyle.Render(m.addRepositoryName)))

		// Show repository type context
		if repo.RemoteURL != nil {
			content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(
				fmt.Sprintf("📦 GitHub Repository: %s", *repo.RemoteURL),
			))
		} else {
			content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(
				fmt.Sprintf("📁 Local Repository: %s", repo.Path),
			))
		}
	}

	return m.layout.Render(content.String())
}

// viewEditNameError renders the repository name change error screen.
func (m *SettingsModel) viewEditNameError() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "❌ Name Update Failed",
		Subtitle: "Cannot change repository name",
		HelpText: "Press any key to return",
	})

	var content strings.Builder
	content.WriteString("Failed to update repository name:\n\n")

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
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("💡 Make sure the name is unique and not empty."))

	return m.layout.Render(content.String())
}
