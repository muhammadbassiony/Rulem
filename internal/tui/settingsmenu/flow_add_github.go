// Package settingsmenu provides the settings modification flow for the rulem TUI application.
package settingsmenu

import (
	"fmt"
	"os"
	"path/filepath"
	"rulem/internal/config"
	"rulem/internal/repository"
	"rulem/internal/tui/components"
	"rulem/internal/tui/helpers/settingshelpers"
	"rulem/pkg/fileops"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Add GitHub Repository Flow
// Flow: AddRepositoryType → AddGitHubName → AddGitHubURL → AddGitHubBranch → AddGitHubPath → [Optional: AddGitHubPAT] → [AddGitHubError | Complete]
//
// This file contains all handlers, transitions, and business logic for adding
// a new GitHub repository to the configuration.
//
// Optional PAT State:
// If no valid PAT exists or PAT validation fails, the flow transitions to AddGitHubPAT
// where the user can enter a new PAT. This provides a seamless experience for adding
// the first GitHub repository without separately configuring authentication.
//
// Handlers (in order of flow):
// - handleAddGitHubNameKeys: Validates and stores repository name
// - handleAddGitHubURLKeys: Validates and stores GitHub URL
// - handleAddGitHubBranchKeys: Validates and stores branch (optional)
// - handleAddGitHubPathKeys: Validates clone path and triggers creation
// - handleAddGitHubPATKeys: [Optional] Handles PAT input when needed
// - handleAddGitHubErrorKeys: Handles error state dismissal
//
// Business Logic:
// - createGitHubRepository: Checks PAT and creates repository
// - createGitHubRepositoryWithPAT: Creates repository after PAT is entered
//
// Views (at end of file):
// - viewAddGitHubName, viewAddGitHubURL, viewAddGitHubBranch, viewAddGitHubPath
// - viewAddGitHubPAT, viewAddGitHubError

// handleAddGitHubNameKeys processes user input in the AddGitHubName state.
// Validates the repository name and proceeds to URL input if valid.
func (m *SettingsModel) handleAddGitHubNameKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		input := strings.TrimSpace(m.textInput.Value())
		m.logger.LogUserAction("settings_add_github_name_submit", input)

		// Validate name (same as local)
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
		m.textInput.Placeholder = "e.g., https://github.com/user/repo"
		return m.transitionTo(SettingsStateAddGitHubURL), nil
	case "esc":
		m.logger.LogUserAction("settings_add_github_cancelled", "returning to type selection")
		return m.transitionTo(SettingsStateAddRepositoryType), nil
	default:
		return m.updateTextInput(msg)
	}
}

// handleAddGitHubURLKeys processes user input in the AddGitHubURL state.
// Validates the GitHub URL and proceeds to branch input if valid.
func (m *SettingsModel) handleAddGitHubURLKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		input := strings.TrimSpace(m.textInput.Value())
		m.logger.LogUserAction("settings_add_github_url_submit", input)

		// Validate GitHub URL
		if err := settingshelpers.ValidateGitHubURL(input); err != nil {
			m.logger.Warn("GitHub URL validation failed", "error", err)
			m.layout = m.layout.SetError(err)
			return m, nil
		}

		// Check for duplicate URL
		for _, repo := range m.currentConfig.Repositories {
			if repo.RemoteURL != nil && *repo.RemoteURL == input {
				m.layout = m.layout.SetError(fmt.Errorf("GitHub URL already used by another repository"))
				return m, nil
			}
		}

		m.newGitHubURL = input
		m.textInput.SetValue("")
		m.textInput.Placeholder = "e.g., main (leave empty for default)"
		return m.transitionTo(SettingsStateAddGitHubBranch), nil
	case "esc":
		m.logger.LogUserAction("settings_add_github_url_cancelled", "returning to name input")
		return m.transitionTo(SettingsStateAddGitHubName), nil
	default:
		return m.updateTextInput(msg)
	}
}

// handleAddGitHubBranchKeys processes user input in the AddGitHubBranch state.
// Validates the branch name (optional) and proceeds to path input.
func (m *SettingsModel) handleAddGitHubBranchKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		input := strings.TrimSpace(m.textInput.Value())
		m.logger.LogUserAction("settings_add_github_branch_submit", input)

		// Validate branch (optional, can be empty for default)
		if input != "" {
			if err := settingshelpers.ValidateBranchName(input); err != nil {
				m.logger.Warn("Branch validation failed", "error", err)
				m.layout = m.layout.SetError(err)
				return m, nil
			}
		}

		m.newGitHubBranch = input

		// Derive default clone path from URL
		placeholder := settingshelpers.DeriveClonePath(m.newGitHubURL)
		if placeholder == "" {
			placeholder = repository.GetDefaultStorageDir()
		}

		m.textInput.SetValue("")
		m.textInput.Placeholder = placeholder
		return m.transitionTo(SettingsStateAddGitHubPath), nil
	case "esc":
		m.logger.LogUserAction("settings_add_github_branch_cancelled", "returning to URL input")
		return m.transitionTo(SettingsStateAddGitHubURL), nil
	default:
		return m.updateTextInput(msg)
	}
}

// handleAddGitHubPathKeys processes user input in the AddGitHubPath state.
// Validates the clone path and creates the repository if valid.
func (m *SettingsModel) handleAddGitHubPathKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	if m.layout.GetError() != nil {
		m.layout = m.layout.ClearError()
	}
	switch msg.String() {
	case "enter":
		input := strings.TrimSpace(m.textInput.Value())
		m.logger.LogUserAction("settings_add_github_path_submit", input)

		// Use placeholder if empty
		if input == "" {
			input = m.textInput.Placeholder
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

		// Check if directory exists and contains content
		if info, err := os.Stat(expandedPath); err == nil && info.IsDir() {
			// Directory exists, check if it's empty
			isEmpty, err := fileops.IsDirEmpty(expandedPath)
			if err == nil && !isEmpty {
				// Check if it's a git repository
				gitDirPath := filepath.Join(expandedPath, ".git")
				if _, err := os.Stat(gitDirPath); err == nil {
					m.layout = m.layout.SetError(fmt.Errorf("directory already contains a Git repository.\n\nPlease choose an empty directory or remove the existing repository at:\n%s", expandedPath))
					return m, nil
				}
				// Directory is not empty but not a git repo - warn user
				m.logger.Warn("Directory is not empty", "path", expandedPath)
				m.layout = m.layout.SetError(fmt.Errorf("directory is not empty.\n\nCloning will fail if the directory contains files. Please use an empty directory:\n%s", expandedPath))
				return m, nil
			}
		}

		m.addRepositoryPath = expandedPath
		m.layout = m.layout.ClearError()
		return m, m.createGitHubRepository()
	case "esc":
		m.logger.LogUserAction("settings_add_github_path_cancelled", "returning to branch input")
		return m.transitionTo(SettingsStateAddGitHubBranch), nil
	default:
		return m.updateTextInput(msg)
	}
}

// handleAddGitHubPATKeys processes user input in the AddGitHubPAT state.
// This is an optional flow state - only entered when PAT is missing or invalid during Add GitHub flow.
// Validates the PAT and continues with repository creation if valid.
func (m *SettingsModel) handleAddGitHubPATKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		input := strings.TrimSpace(m.textInput.Value())
		m.logger.LogUserAction("settings_add_github_pat_submit", "PAT provided")

		if input == "" {
			m.logger.Warn("Empty PAT submitted in Add GitHub flow")
			m.layout = m.layout.SetError(fmt.Errorf("PAT cannot be empty"))
			return m, nil
		}

		// Validate token format
		m.logger.Debug("Validating GitHub PAT format")
		if err := m.credManager.ValidateGitHubToken(input); err != nil {
			m.logger.Warn("GitHub PAT format validation failed", "error", err)
			m.layout = m.layout.SetError(fmt.Errorf("invalid PAT format: %w", err))
			return m, nil
		}

		// Validate PAT with the repository being added
		m.logger.Debug("Validating GitHub PAT with repository", "url", m.newGitHubURL)
		if err := m.credManager.ValidateGitHubTokenWithRepo(m.context, input, m.newGitHubURL); err != nil {
			m.logger.Warn("GitHub PAT repository validation failed", "error", err)
			m.layout = m.layout.SetError(fmt.Errorf("PAT validation failed: %w", err))
			return m, nil
		}

		// Store the PAT
		m.logger.Debug("Storing GitHub PAT")
		if err := m.credManager.StoreGitHubToken(input); err != nil {
			m.logger.Error("Failed to store GitHub PAT", "error", err)
			m.layout = m.layout.SetError(fmt.Errorf("failed to store PAT: %w", err))
			return m, nil
		}

		m.logger.Info("GitHub PAT validated and stored successfully")

		// Clear the input and continue with repository creation
		m.textInput.SetValue("")
		m.layout = m.layout.ClearError()

		// Continue with repository creation using the newly stored PAT
		return m, m.createGitHubRepositoryWithPAT(input)

	case "esc":
		m.logger.LogUserAction("settings_add_github_pat_cancelled", "returning to path input")
		m.layout = m.layout.ClearError()
		return m.transitionTo(SettingsStateAddGitHubPath), nil

	default:
		return m.updateTextInput(msg)
	}
}

// handleAddGitHubErrorKeys processes input in the AddGitHubError state.
// Any key returns to the clone path input state.
func (m *SettingsModel) handleAddGitHubErrorKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	m.logger.LogUserAction("settings_add_github_error_dismiss", msg.String())
	m.layout = m.layout.ClearError()
	return m.transitionTo(SettingsStateAddGitHubPath), nil
}

// createGitHubRepository creates a new GitHub repository configuration entry.
// This is adapted from setupmenu.go but appends to existing repositories instead of replacing.
// It validates the PAT, creates the repository entry, saves config, and triggers repository preparation.
func (m *SettingsModel) createGitHubRepository() tea.Cmd {
	return func() tea.Msg {
		// Get PAT from credential manager
		pat, err := m.credManager.GetGitHubToken()
		if err != nil {
			// PAT not found - transition to PAT input instead of failing
			m.logger.Info("GitHub PAT not found, prompting user to enter PAT", "error", err)
			return addGitHubPATNeededMsg{}
		}

		// Validate PAT with repository
		m.logger.Info("Validating GitHub PAT with repository", "url", m.newGitHubURL)
		if err := m.credManager.ValidateGitHubTokenWithRepo(m.context, pat, m.newGitHubURL); err != nil {
			// PAT validation failed - could be invalid/corrupted token
			// Offer user to re-enter PAT instead of showing error
			m.logger.Warn("GitHub PAT validation failed, prompting user to re-enter PAT", "error", err)
			return addGitHubPATNeededMsg{}
		}

		// Generate repository ID
		timestamp := time.Now().Unix()

		// Extract repository name from URL
		gitInfo, err := repository.ParseGitURL(m.newGitHubURL)
		if err != nil {
			m.logger.Warn("Failed to parse GitHub URL for name extraction", "error", err, "url", m.newGitHubURL)
			gitInfo.Repo = m.addRepositoryName // Use user-provided name as fallback
		}

		id := config.GenerateRepositoryID(m.addRepositoryName, timestamp)

		m.logger.Info("Creating GitHub repository",
			"id", id,
			"name", m.addRepositoryName,
			"url", m.newGitHubURL,
			"branch", m.newGitHubBranch,
			"path", m.addRepositoryPath)

		// Create repository entry
		newRepo := repository.RepositoryEntry{
			ID:        id,
			Name:      m.addRepositoryName,
			Type:      repository.RepositoryTypeGitHub,
			Path:      m.addRepositoryPath,
			RemoteURL: &m.newGitHubURL,
			CreatedAt: timestamp,
		}

		// Handle optional branch (nil if empty, pointer if specified)
		if m.newGitHubBranch != "" {
			newRepo.Branch = &m.newGitHubBranch
		}

		// Append to config (different from setupmenu which replaces)
		m.currentConfig.Repositories = append(m.currentConfig.Repositories, newRepo)

		// Save config
		if err := m.currentConfig.Save(); err != nil {
			return addGitHubErrorMsg{fmt.Errorf("failed to save configuration: %w", err)}
		}

		// Reload repositories (this will trigger clone if needed)
		m.preparedRepos, err = repository.PrepareAllRepositories(
			m.currentConfig.Repositories,
			m.logger,
		)
		if err != nil {
			m.logger.Warn("Failed to reload repositories after creation", "error", err)
			// Provide user-friendly error message for common cases
			errMsg := err.Error()
			if strings.Contains(errMsg, "directory contains different git repository") {
				return addGitHubErrorMsg{fmt.Errorf("the chosen directory already contains a different Git repository.\n\nPlease choose an empty directory or remove the existing repository at:\n%s", m.addRepositoryPath)}
			}
			return addGitHubErrorMsg{fmt.Errorf("failed to prepare new repository: %w", err)}
		}

		// Rebuild list with action items
		items := BuildSettingsMainMenuItems(m.preparedRepos)
		m.repoList.SetItems(items)

		m.logger.Info("GitHub repository created successfully")

		// Reset add flow state
		m.addRepositoryName = ""
		m.addRepositoryPath = ""
		m.newGitHubURL = ""
		m.newGitHubBranch = ""

		// Transition to main menu
		m.state = SettingsStateMainMenu
		return settingsCompleteMsg{}
	}
}

// transitionToAddGitHubName transitions to the AddGitHubName state.
// Sets up the text input for repository name entry.
func (m *SettingsModel) transitionToAddGitHubName() (*SettingsModel, tea.Cmd) {
	m.textInput.SetValue("")
	m.textInput.Placeholder = "e.g., My GitHub Repository"
	m.textInput.CharLimit = 100
	m.textInput.EchoMode = textinput.EchoNormal
	m.textInput.Focus()

	return m.transitionTo(SettingsStateAddGitHubName), nil
}

// Views

// viewAddGitHubName renders the repository name input screen for GitHub repositories.
// This is the first step in the add GitHub repository flow.
func (m *SettingsModel) viewAddGitHubName() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "🔗 Add GitHub Repository",
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

// viewAddGitHubURL renders the GitHub URL input screen.
// This is the second step in the add GitHub repository flow.
func (m *SettingsModel) viewAddGitHubURL() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    fmt.Sprintf("🔗 Add GitHub Repository: %s", m.addRepositoryName),
		Subtitle: "Enter the GitHub repository URL",
		HelpText: "Enter to continue • Esc to go back",
	})

	var content strings.Builder
	content.WriteString("GitHub URL:\n\n")
	content.WriteString(m.textInput.View())
	content.WriteString("\n\n")
	content.WriteString(lipgloss.NewStyle().Faint(true).Render("Format: https://github.com/username/repository"))

	return m.layout.Render(content.String())
}

// viewAddGitHubBranch renders the branch name input screen.
// This is the third step in the add GitHub repository flow.
// Branch selection is optional - users can leave it empty to use the default branch.
func (m *SettingsModel) viewAddGitHubBranch() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    fmt.Sprintf("🔗 Add GitHub Repository: %s", m.addRepositoryName),
		Subtitle: "Enter the branch to sync (optional)",
		HelpText: "Enter to continue • Esc to go back",
	})

	var content strings.Builder
	content.WriteString(fmt.Sprintf("URL: %s\n\n", lipgloss.NewStyle().Faint(true).Render(m.newGitHubURL)))
	content.WriteString("Branch:\n\n")
	content.WriteString(m.textInput.View())
	content.WriteString("\n\n")
	content.WriteString(lipgloss.NewStyle().Faint(true).Render("Leave empty to use the default branch"))

	return m.layout.Render(content.String())
}

// viewAddGitHubPath renders the local clone path input screen.
// This is the fourth and final step in the add GitHub repository flow.
// Shows a summary of the URL and branch (if specified) before path entry.
func (m *SettingsModel) viewAddGitHubPath() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    fmt.Sprintf("🔗 Add GitHub Repository: %s", m.addRepositoryName),
		Subtitle: "Enter the local clone path",
		HelpText: "Enter to save • Esc to go back",
	})

	var content strings.Builder
	content.WriteString(fmt.Sprintf("URL: %s\n", lipgloss.NewStyle().Faint(true).Render(m.newGitHubURL)))
	if m.newGitHubBranch != "" {
		content.WriteString(fmt.Sprintf("Branch: %s\n", lipgloss.NewStyle().Faint(true).Render(m.newGitHubBranch)))
	}
	content.WriteString("\n")
	content.WriteString("Local Clone Path:\n\n")
	content.WriteString(m.textInput.View())
	content.WriteString("\n\n")
	content.WriteString(lipgloss.NewStyle().Faint(true).Render("Where to clone the repository locally"))

	return m.layout.Render(content.String())
}

// viewAddGitHubError renders the error screen when GitHub repository creation fails.
// Displays the error message and instructions to return.
// Common failures include PAT validation, clone issues, and configuration save errors.
func (m *SettingsModel) viewAddGitHubError() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "❌ Failed to Add GitHub Repository",
		Subtitle: "Cannot create repository",
		HelpText: "Press any key to return",
	})

	var content strings.Builder
	content.WriteString("Failed to add GitHub repository:\n\n")

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
		Render("💡 Common reasons:\n  - Invalid GitHub PAT or insufficient permissions\n  - Repository URL not accessible\n  - Clone directory already exists or has content\n  - Network connectivity issues\n  - Duplicate repository name or URL"))

	return m.layout.Render(content.String())
}

// createGitHubRepositoryWithPAT creates a GitHub repository using the provided PAT.
// This is called after the user enters a PAT in the optional AddGitHubPAT state.
// The PAT has already been validated and stored by handleAddGitHubPATKeys.
func (m *SettingsModel) createGitHubRepositoryWithPAT(pat string) tea.Cmd {
	return func() tea.Msg {
		// Generate repository ID
		timestamp := time.Now().Unix()

		// Extract repository name from URL
		gitInfo, err := repository.ParseGitURL(m.newGitHubURL)
		if err != nil {
			m.logger.Warn("Failed to parse GitHub URL for name extraction", "error", err, "url", m.newGitHubURL)
			gitInfo.Repo = m.addRepositoryName // Use user-provided name as fallback
		}

		id := config.GenerateRepositoryID(m.addRepositoryName, timestamp)

		m.logger.Info("Creating GitHub repository with new PAT",
			"id", id,
			"name", m.addRepositoryName,
			"url", m.newGitHubURL,
			"branch", m.newGitHubBranch,
			"path", m.addRepositoryPath)

		// Create repository entry
		newRepo := repository.RepositoryEntry{
			ID:        id,
			Name:      m.addRepositoryName,
			Type:      repository.RepositoryTypeGitHub,
			Path:      m.addRepositoryPath,
			RemoteURL: &m.newGitHubURL,
			CreatedAt: timestamp,
		}

		// Handle optional branch (nil if empty, pointer if specified)
		if m.newGitHubBranch != "" {
			newRepo.Branch = &m.newGitHubBranch
		}

		// Append to config (different from setupmenu which replaces)
		m.currentConfig.Repositories = append(m.currentConfig.Repositories, newRepo)

		// Save config
		if err := m.currentConfig.Save(); err != nil {
			return addGitHubErrorMsg{fmt.Errorf("failed to save configuration: %w", err)}
		}

		// Reload repositories (this will trigger clone if needed)
		m.preparedRepos, err = repository.PrepareAllRepositories(
			m.currentConfig.Repositories,
			m.logger,
		)
		if err != nil {
			m.logger.Warn("Failed to reload repositories after creation", "error", err)
			// Provide user-friendly error message for common cases
			errMsg := err.Error()
			if strings.Contains(errMsg, "directory contains different git repository") {
				return addGitHubErrorMsg{fmt.Errorf("the chosen directory already contains a different Git repository.\n\nPlease choose an empty directory or remove the existing repository at:\n%s", m.addRepositoryPath)}
			}
			return addGitHubErrorMsg{fmt.Errorf("failed to prepare new repository: %w", err)}
		}

		// Rebuild list with action items
		items := BuildSettingsMainMenuItems(m.preparedRepos)
		m.repoList.SetItems(items)

		m.logger.Info("GitHub repository created successfully")

		// Reset add flow state
		m.addRepositoryName = ""
		m.addRepositoryPath = ""
		m.newGitHubURL = ""
		m.newGitHubBranch = ""

		// Transition to main menu
		m.state = SettingsStateMainMenu
		return settingsCompleteMsg{}
	}
}

// viewAddGitHubPAT renders the PAT input screen during Add GitHub flow.
// This is an optional flow screen - only shown when PAT is missing.
// Provides context about the repository being added.
func (m *SettingsModel) viewAddGitHubPAT() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    fmt.Sprintf("🔑 GitHub Access Required: %s", m.addRepositoryName),
		Subtitle: "Enter your GitHub Personal Access Token",
		HelpText: "Enter to continue • Esc to go back",
	})

	var content strings.Builder
	content.WriteString("To add this GitHub repository, you need to provide a Personal Access Token (PAT).\n\n")
	content.WriteString(fmt.Sprintf("Repository: %s\n", lipgloss.NewStyle().Faint(true).Render(m.newGitHubURL)))
	if m.newGitHubBranch != "" {
		content.WriteString(fmt.Sprintf("Branch: %s\n", lipgloss.NewStyle().Faint(true).Render(m.newGitHubBranch)))
	}
	content.WriteString("\n")
	content.WriteString("Personal Access Token:\n\n")
	content.WriteString(m.textInput.View())
	content.WriteString("\n\n")
	content.WriteString(lipgloss.NewStyle().Faint(true).Render("💡 Need a token? Visit: https://github.com/settings/tokens\n   Required scopes: repo (for private repos) or public_repo (for public repos)"))

	return m.layout.Render(content.String())
}
