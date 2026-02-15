// Package settingsmenu provides the settings modification flow for the rulem TUI application.
package settingsmenu

import (
	"fmt"
	"rulem/internal/config"
	"rulem/internal/repository"
	"rulem/internal/tui/components"
	"rulem/pkg/fileops"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Add Local Repository Flow
// Flow: AddRepositoryType → AddLocalName → AddLocalPath → [AddLocalError | Complete]
//
// This file contains all handlers, transitions, and business logic for adding
// a new local repository to the configuration.

// handleAddLocalNameKeys processes user input in the AddLocalName state.
// Validates the repository name and proceeds to path input if valid.
func (m *SettingsModel) handleAddLocalNameKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		input := strings.TrimSpace(m.textInput.Value())
		m.logger.LogUserAction("settings_add_local_name_submit", input)

		// Validate name
		if input == "" {
			m.layout = m.layout.SetError(fmt.Errorf("repository name cannot be empty"))
			return m, nil
		}
		if len(input) > 100 {
			m.layout = m.layout.SetError(fmt.Errorf("repository name must be 100 characters or less"))
			return m, nil
		}

		// Check for duplicate name
		if _, err := m.currentConfig.FindRepositoryByName(input); err == nil {
			m.layout = m.layout.SetError(fmt.Errorf("repository name already exists"))
			return m, nil
		}

		m.addRepositoryName = input
		m.textInput.SetValue("")
		m.textInput.Placeholder = "e.g., ~/rules or /path/to/rules"
		return m.transitionTo(SettingsStateAddLocalPath), nil
	case "esc":
		m.logger.LogUserAction("settings_add_local_cancelled", "returning to type selection")
		return m.transitionTo(SettingsStateAddRepositoryType), nil
	default:
		return m.updateTextInput(msg)
	}
}

// handleAddLocalPathKeys processes user input in the AddLocalPath state.
// Validates the local directory path and creates the repository if valid.
func (m *SettingsModel) handleAddLocalPathKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		input := strings.TrimSpace(m.textInput.Value())
		m.logger.LogUserAction("settings_add_local_path_submit", input)

		// Validate path
		if input == "" {
			m.layout = m.layout.SetError(fmt.Errorf("path cannot be empty"))
			return m, nil
		}

		expandedPath := fileops.ExpandPath(input)
		if err := fileops.ValidateStoragePath(expandedPath); err != nil {
			m.logger.Warn("Path validation failed", "error", err)
			m.layout = m.layout.SetError(err)
			return m, nil
		}

		// Check for duplicate path
		for _, repo := range m.currentConfig.Repositories {
			if repo.Path == expandedPath {
				m.layout = m.layout.SetError(fmt.Errorf("path already used by another repository"))
				return m, nil
			}
		}

		m.addRepositoryPath = expandedPath

		// Create local repository
		return m, m.createLocalRepository()
	case "esc":
		m.logger.LogUserAction("settings_add_local_path_cancelled", "returning to name input")
		return m.transitionTo(SettingsStateAddLocalName), nil
	default:
		return m.updateTextInput(msg)
	}
}

// handleAddLocalErrorKeys processes input in the AddLocalError state.
// Any key returns to the local path input state.
func (m *SettingsModel) handleAddLocalErrorKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	m.logger.LogUserAction("settings_add_local_error_dismiss", msg.String())
	m.layout = m.layout.ClearError()
	return m.transitionTo(SettingsStateAddLocalPath), nil
}

// createLocalRepository creates a new local repository configuration entry.
// This is adapted from setupmenu.go but appends to existing repositories instead of replacing.
func (m *SettingsModel) createLocalRepository() tea.Cmd {
	return func() tea.Msg {
		// Generate repository ID
		timestamp := time.Now().Unix()
		id := config.GenerateRepositoryID(m.addRepositoryName, timestamp)

		m.logger.Info("Creating local repository",
			"id", id,
			"name", m.addRepositoryName,
			"path", m.addRepositoryPath)

		// Create repository entry
		newRepo := repository.RepositoryEntry{
			ID:        id,
			Name:      m.addRepositoryName,
			Type:      repository.RepositoryTypeLocal,
			Path:      m.addRepositoryPath,
			CreatedAt: timestamp,
		}

		// Append to config (different from setupmenu which replaces)
		m.currentConfig.Repositories = append(m.currentConfig.Repositories, newRepo)

		// Save config
		if err := m.currentConfig.Save(); err != nil {
			return addLocalErrorMsg{fmt.Errorf("failed to save configuration: %w", err)}
		}

		// Reload repositories
		var err error
		m.preparedRepos, err = repository.PrepareAllRepositories(
			m.currentConfig.Repositories,
			m.logger,
		)
		if err != nil {
			m.logger.Warn("Failed to reload repositories after creation", "error", err)
			return addLocalErrorMsg{fmt.Errorf("failed to prepare new repository: %w", err)}
		}

		// Rebuild list with action items
		items := BuildSettingsMainMenuItems(m.preparedRepos)
		m.repoList.SetItems(items)

		m.logger.Info("Local repository created successfully")

		// Reset add flow state
		m.addRepositoryName = ""
		m.addRepositoryPath = ""

		// Transition to main menu
		m.state = SettingsStateMainMenu
		return settingsCompleteMsg{}
	}
}

// transitionToAddLocalName transitions to the AddLocalName state.
// Sets up the text input for repository name entry.
func (m *SettingsModel) transitionToAddLocalName() (*SettingsModel, tea.Cmd) {
	m.textInput.SetValue("")
	m.textInput.Placeholder = "e.g., My Rules Repository"
	m.textInput.CharLimit = 100
	m.textInput.EchoMode = textinput.EchoNormal
	m.textInput.Focus()

	return m.transitionTo(SettingsStateAddLocalName), nil
}

// View Functions

// viewAddLocalName renders the repository name input screen for local repositories.
// This is the first step in the add local repository flow.
func (m *SettingsModel) viewAddLocalName() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "📁 Add Local Repository",
		Subtitle: "Enter a name for this repository",
		HelpText: "Enter to continue • Esc to go back",
	})

	var content strings.Builder
	content.WriteString("Repository Name:\n\n")
	content.WriteString(m.textInput.View())
	content.WriteString("\n\n")
	content.WriteString(lipgloss.NewStyle().Faint(true).Render("This name will help you identify the repository"))

	return m.layout.Render(content.String())
}

// viewAddLocalPath renders the local directory path input screen.
// This is the second and final step in the add local repository flow.
func (m *SettingsModel) viewAddLocalPath() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    fmt.Sprintf("📁 Add Local Repository: %s", m.addRepositoryName),
		Subtitle: "Enter the local directory path",
		HelpText: "Enter to save • Esc to go back",
	})

	var content strings.Builder
	content.WriteString("Local Directory Path:\n\n")
	content.WriteString(m.textInput.View())
	content.WriteString("\n\n")
	content.WriteString(lipgloss.NewStyle().Faint(true).Render("Path to the directory containing your rule files"))

	return m.layout.Render(content.String())
}

// viewAddLocalError renders the error screen when local repository creation fails.
// Displays the error message and instructions to return.
func (m *SettingsModel) viewAddLocalError() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "❌ Failed to Add Local Repository",
		Subtitle: "Cannot create repository",
		HelpText: "Press any key to return",
	})

	var content strings.Builder
	content.WriteString("Failed to add local repository:\n\n")

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
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).
		Render("💡 Common reasons:\n  - Invalid or inaccessible path\n  - Duplicate repository name or path\n  - Permission issues"))

	return m.layout.Render(content.String())
}
