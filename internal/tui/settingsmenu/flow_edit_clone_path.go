// Package settingsmenu provides the settings modification flow for the rulem TUI application.
package settingsmenu

import (
	"fmt"
	"rulem/internal/config"
	"rulem/internal/repository"
	"rulem/internal/tui/components"
	"rulem/internal/tui/helpers/settingshelpers"
	"rulem/internal/tui/styles"
	"rulem/pkg/fileops"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Edit Clone Path Flow
// Flow: UpdateGitHubPath → EditClonePathConfirm → [EditClonePathError | Complete]
//
// This file contains all handlers, transitions, and business logic for editing
// the local clone path of an existing GitHub repository.

// handleUpdateGitHubPathKeys processes user input in the UpdateGitHubPath state.
// Validates the new path and initiates dirty state check before proceeding to confirmation.
func (m *SettingsModel) handleUpdateGitHubPathKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.logger.LogUserAction("settings_github_path_submit", m.textInput.Value())
		input := strings.TrimSpace(m.textInput.Value())

		// Get current repository to derive placeholder
		var placeholder string
		if repo, err := m.currentConfig.FindRepositoryByID(m.selectedRepositoryID); err == nil && repo.RemoteURL != nil {
			placeholder = settingshelpers.DeriveClonePath(*repo.RemoteURL)
		}
		if placeholder == "" {
			placeholder = repository.GetDefaultStorageDir()
		}

		if input == "" {
			input = placeholder
		}

		// Validate the path
		if err := fileops.ValidateStoragePath(input); err != nil {
			m.logger.Warn("Path validation failed", "error", err)
			return m, func() tea.Msg { return editClonePathErrorMsg{err} }
		}

		// Check for duplicate paths across repositories
		expandedPath := fileops.ExpandPath(input)
		for _, repo := range m.currentConfig.Repositories {
			if repo.ID != m.selectedRepositoryID && repo.Path == expandedPath {
				err := fmt.Errorf("path already used by repository '%s'", repo.Name)
				m.logger.Warn("Duplicate path detected", "error", err, "path", expandedPath)
				return m, func() tea.Msg { return editClonePathErrorMsg{err} }
			}
		}

		m.newGitHubPath = expandedPath
		m.hasChanges = true
		m.changeType = ChangeOptionGitHubPath

		// Perform dirty state check before confirming
		m.logger.Debug("Initiating dirty state check for clone path change")
		return m, m.checkDirtyState(func(isDirty bool, err error) tea.Msg {
			return editClonePathDirtyStateMsg{isDirty: isDirty, err: err}
		})
	case "esc":
		m.resetTemporaryChanges()
		return m.transitionTo(SettingsStateRepositoryActions), nil
	default:
		return m.updateTextInput(msg)
	}
}

// handleEditClonePathConfirmKeys processes user input in the EditClonePathConfirm state.
// Handles confirmation or cancellation of the clone path change.
func (m *SettingsModel) handleEditClonePathConfirmKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "enter", "y":
		m.logger.LogUserAction("settings_clone_path_confirm", "user confirmed clone path change")
		return m, m.saveChanges()
	case "esc", "n":
		m.logger.LogUserAction("settings_clone_path_cancel", "user cancelled clone path change")
		m.resetTemporaryChanges()
		return m.transitionTo(SettingsStateRepositoryActions), nil
	}
	return m, nil
}

// handleEditClonePathErrorKeys processes user input in the EditClonePathError state.
// Any key dismisses the error and returns to repository actions.
func (m *SettingsModel) handleEditClonePathErrorKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	m.logger.LogUserAction("settings_clone_path_error_dismiss", "user dismissed error")
	m.layout = m.layout.ClearError()
	m.resetTemporaryChanges()
	return m.transitionTo(SettingsStateRepositoryActions), nil
}

// transitionToUpdateGitHubPath transitions to the clone path update state.
// Sets up the text input with current path as default.
func (m *SettingsModel) transitionToUpdateGitHubPath() (*SettingsModel, tea.Cmd) {
	placeholder := repository.GetDefaultStorageDir()
	if m.currentConfig != nil {
		if repo, err := m.currentConfig.FindRepositoryByID(m.selectedRepositoryID); err == nil {
			placeholder = repo.Path
		}
	}

	return m.transitionTo(SettingsStateUpdateGitHubPath), settingshelpers.ResetTextInputForState(&m.textInput, "", placeholder, textinput.EchoNormal)
}

// updateGitHubPath updates the local clone path for the GitHub repository in the configuration.
func (m *SettingsModel) updateGitHubPath(cfg *config.Config) error {
	repo, err := cfg.FindRepositoryByID(m.selectedRepositoryID)
	if err != nil {
		return fmt.Errorf("failed to get repository: %w", err)
	}
	m.logger.Info("Updating GitHub path", "id", m.selectedRepositoryID, "old", repo.Path, "new", m.newGitHubPath)

	if m.newGitHubPath == "" {
		return fmt.Errorf("no GitHub path provided")
	}

	repo.Path = m.newGitHubPath

	// Update in the config array
	for i := range cfg.Repositories {
		if cfg.Repositories[i].ID == m.selectedRepositoryID {
			cfg.Repositories[i] = *repo
			break
		}
	}
	// Save the config
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save GitHub path configuration: %w", err)
	}

	// Note: Repository will be re-cloned to new path on next sync
	m.logger.Info("GitHub path updated, repository will be cloned to new path on next sync")

	return nil
}

// Views

// viewUpdateGitHubPath renders the clone path edit input view.
func (m *SettingsModel) viewUpdateGitHubPath() string {
	title := "📂 Update Clone Path"

	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    title,
		Subtitle: "Enter where to clone the repository",
		HelpText: "Enter to save • Esc to cancel",
	})

	var content strings.Builder

	if m.currentConfig != nil {
		if repo, err := m.currentConfig.FindRepositoryByID(m.selectedRepositoryID); err == nil {
			content.WriteString(fmt.Sprintf("Current: %s\n\n", lipgloss.NewStyle().Faint(true).Render(repo.Path)))
		}
	}

	content.WriteString("Clone path:\n")
	content.WriteString(styles.InputStyle.Render(m.textInput.View()))

	return m.layout.Render(content.String())
}

// viewEditClonePathConfirm renders the clone path change confirmation view.
func (m *SettingsModel) viewEditClonePathConfirm() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "📂 Confirm Clone Path Change",
		Subtitle: "Review your changes",
		HelpText: "Enter/y to confirm • Esc/n to cancel",
	})

	var content strings.Builder
	highlightStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#5fd7ff"))

	// Get current repository info
	if repo, err := m.currentConfig.FindRepositoryByID(m.selectedRepositoryID); err == nil {
		content.WriteString(fmt.Sprintf("Repository: %s\n\n", highlightStyle.Render(repo.Name)))
		content.WriteString(fmt.Sprintf("Current path: %s\n", lipgloss.NewStyle().Faint(true).Render(repo.Path)))
		content.WriteString(fmt.Sprintf("New path:     %s\n\n", highlightStyle.Render(m.newGitHubPath)))

		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")).Render("⚠️  Warning:\n"))
		content.WriteString("The repository will be re-cloned to the new path on next sync.\n")
		content.WriteString("The old clone will remain at the current path (manual cleanup may be needed).\n\n")
	}

	content.WriteString("Do you want to proceed? (y/N)")

	return m.layout.Render(content.String())
}

// viewEditClonePathError renders the clone path error view.
func (m *SettingsModel) viewEditClonePathError() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "❌ Clone Path Update Failed",
		Subtitle: "Cannot change clone path",
		HelpText: "Press any key to return",
	})

	var content strings.Builder
	content.WriteString("Failed to update clone path:\n\n")

	if err := m.layout.GetError(); err != nil {
		content.WriteString(fmt.Sprintf("%s\n\n",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render(err.Error())))
	}

	content.WriteString("Press any key to return to repository actions.")

	return m.layout.Render(content.String())
}
