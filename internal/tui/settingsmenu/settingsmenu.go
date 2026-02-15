// Package settingsmenu provides the settings modification flow for the rulem TUI application.
//
// This package implements a comprehensive settings menu that allows users to modify
// their existing configuration, including repository type switching, path updates,
// GitHub configuration changes, and manual refresh operations.
//
// The settings flow consists of several states:
//   - MainMenu: Display current configuration with available options
//   - SelectChange: Choose what setting to modify
//   - ConfirmTypeChange: Confirm repository type switch (destructive operation)
//   - UpdateLocalPath: Modify local directory path
//   - UpdateGitHubPAT: Update or remove GitHub Personal Access Token
//   - UpdateGitHubURL: Change GitHub repository URL
//   - UpdateGitHubBranch: Change GitHub branch
//   - UpdateGitHubPath: Change local clone path
//   - ManualRefresh: Trigger manual sync from GitHub
//   - Confirmation: Review and confirm all changes
//   - Complete: Settings successfully updated
//   - Error: Error occurred during settings modification
//
// Key features:
//   - Full repository type switching (Local ↔ GitHub)
//   - Granular updates for individual settings
//   - Dirty state detection for GitHub repositories
//   - Manual refresh operations
//   - Comprehensive validation and error handling
//   - State history for back navigation
package settingsmenu

import (
	"context"
	"fmt"
	"log/slog"
	"rulem/internal/config"
	"rulem/internal/logging"
	"rulem/internal/repository"
	"rulem/internal/tui/components"
	"rulem/internal/tui/helpers"
	"rulem/internal/tui/helpers/repolist"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Type definitions have been moved to types.go for better organization.
// This includes: SettingsState, ChangeOption, ChangeOptionInfo,
// and all message types (settingsCompleteMsg, refreshInitiateMsg, etc.).

type credentialManager interface {
	ValidateGitHubToken(token string) error
	ValidateGitHubTokenWithRepo(ctx context.Context, token string, repoURL string) error
	ValidateGitHubTokenForRepos(ctx context.Context, token string, repos []repository.RepositoryEntry) error
	StoreGitHubToken(token string) error
	GetGitHubToken() (string, error)
}

// SettingsModel handles the settings modification flow
type SettingsModel struct {
	// State management
	state         SettingsState
	previousState SettingsState

	// Current configuration - contains loaded repositories
	currentConfig *config.Config
	preparedRepos []repository.PreparedRepository

	// Temporary changes (not yet saved)
	changeType      ChangeOption
	newStorageDir   string
	newGitHubURL    string // Used in Add GitHub flow
	newGitHubBranch string
	newGitHubPath   string
	newGitHubPAT    string // Used in global PAT management

	// Add repository flow state
	addRepositoryTypeIndex int    // 0=Local, 1=GitHub
	addRepositoryName      string // name for new repository
	addRepositoryPath      string // path for new repository (local or github clone)

	// Components
	textInput textinput.Model
	layout    components.LayoutModel
	repoList  list.Model

	// Selection state
	selectedRepositoryActionOption int
	selectedRepositoryID           string // ID of currently selected repository for operations

	// Change tracking
	hasChanges bool

	// GitHub repository state
	isDirty           bool
	refreshInProgress bool
	lastRefreshError  error

	// Dependencies
	logger      *logging.AppLogger
	credManager credentialManager
	ctx         helpers.UIContext
	context     context.Context
}

// NewSettingsModel creates a new settings model
func NewSettingsModel(ctx helpers.UIContext) *SettingsModel {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 256

	// Create centralized layout
	layout := components.NewLayout(components.LayoutConfig{
		MarginX:  2,
		MarginY:  1,
		MaxWidth: 100,
	})

	var preparedRepos []repository.PreparedRepository
	if ctx.Config != nil {
		var err error
		preparedRepos, err = repository.PrepareAllRepositories(ctx.Config.Repositories, ctx.Logger)
		if err != nil {
			ctx.Logger.Error("Failed to prepare repositories for settings menu", "error", err)
		}
		if len(preparedRepos) == 0 {
			ctx.Logger.Warn("No repositories prepared for settings menu")
		}
	}

	// Apply window sizing if available
	if ctx.HasValidDimensions() {
		windowMsg := tea.WindowSizeMsg{Width: ctx.Width, Height: ctx.Height}
		layout, _ = layout.Update(windowMsg)
		ti.Width = layout.InputWidth()
	}

	// Initialize repository list with action items
	repoItems := BuildSettingsMainMenuItems(preparedRepos)
	ctx.Logger.Info("Repository list items count", "count", len(repoItems), "items", repoItems)
	repoList := repolist.BuildRepositoryList(repoItems, ctx.Width-4, ctx.Height-10)

	return &SettingsModel{
		state:         SettingsStateMainMenu,
		currentConfig: ctx.Config,
		preparedRepos: preparedRepos,
		textInput:     ti,
		layout:        layout,
		repoList:      repoList,
		logger:        ctx.Logger,
		credManager:   repository.NewCredentialManager(),
		ctx:           ctx,
		context:       context.Background(),
	}
}

// Init initializes the settings model
func (m *SettingsModel) Init() tea.Cmd {
	m.logger.Info("Settings model initialized")

	// Load current configuration
	return tea.Batch(
		textinput.Blink,
		m.loadCurrentConfig(),
	)
}

// Update handles all message types and delegates to state-specific handlers
func (m *SettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Log strategic messages for debugging
	m.logger.LogMessage(msg)

	// Update layout with window size changes
	m.layout, _ = m.layout.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Update layout and text input width responsively
		m.layout, _ = m.layout.Update(msg)
		m.textInput.Width = m.layout.InputWidth()
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case settingsCompleteMsg:
		m.state = SettingsStateComplete
		m.layout = m.layout.ClearError()
		// Trigger config reload in parent after successful update
		return m, config.ReloadConfig()

	case config.LoadConfigMsg:
		// Reload prepared repositories with the new config
		if msg.Config != nil {
			m.currentConfig = msg.Config
			var err error
			m.preparedRepos, err = repository.PrepareAllRepositories(msg.Config.Repositories, m.logger)
			if err != nil {
				m.logger.Warn("Failed to reload repositories after config load", "error", err)
				// Don't fail completely - let the user work with what we have
			}

			// Rebuild repository list with the new prepared repositories
			items := BuildSettingsMainMenuItems(m.preparedRepos)
			m.repoList.SetItems(items)
		} else {
			// Handle nil config case
			m.currentConfig = nil
		}

		return m, nil

	case refreshCompleteMsg:
		m.refreshInProgress = false
		if msg.err != nil {
			m.logger.Error("Refresh failed", "error", msg.err)
			m.lastRefreshError = msg.err
		}
		m.lastRefreshError = nil
		m.state = SettingsStateMainMenu
		m.layout = m.layout.ClearError()
		return m, nil

	case editBranchDirtyStateMsg:
		// Handle dirty state check result for branch editing
		m.isDirty = msg.isDirty

		if msg.err != nil {
			m.logger.Warn("Dirty state check failed during branch edit", "error", msg.err)
			return m.transitionTo(SettingsStateEditBranchError), func() tea.Msg {
				return editBranchErrorMsg{fmt.Errorf("failed to check repository status: %w", msg.err)}
			}
		}

		if msg.isDirty {
			// Repository has uncommitted changes
			m.logger.Info("Branch edit blocked - repository has uncommitted changes")
			return m.transitionTo(SettingsStateEditBranchError), func() tea.Msg {
				return editBranchErrorMsg{fmt.Errorf("repository has uncommitted changes - please commit or stash them before changing branch")}
			}
		}

		// Repository is clean, proceed to confirmation
		m.logger.Debug("Repository clean, proceeding to branch confirmation")
		return m.transitionTo(SettingsStateEditBranchConfirm), nil

	case editBranchErrorMsg:
		// Transition to error state and display error
		m.logger.Error("Branch edit error", "error", msg.err)
		m.layout = m.layout.SetError(msg.err)
		return m.transitionTo(SettingsStateEditBranchError), nil

	case refreshDirtyStateMsg:
		// Handle dirty state check result for manual refresh
		m.isDirty = msg.isDirty

		if msg.err != nil {
			m.logger.Warn("Dirty state check failed during refresh", "error", msg.err)
		}

		if msg.isDirty {
			// Repository has uncommitted changes
			m.logger.Info("Refresh blocked - repository has uncommitted changes")
			m.lastRefreshError = fmt.Errorf("repository has uncommitted changes - please commit or stash them before refreshing")
			return m.transitionTo(SettingsStateRefreshError), nil
		}

		// Repository is clean, proceed with refresh
		m.logger.Debug("Repository clean, proceeding with refresh")
		m.refreshInProgress = true
		m.state = SettingsStateRefreshInProgress
		return m, m.triggerRefresh()

	case editClonePathDirtyStateMsg:
		// Handle dirty state check result for clone path editing
		m.isDirty = msg.isDirty

		if msg.err != nil {
			m.logger.Warn("Dirty state check failed during clone path edit", "error", msg.err)
			return m.transitionTo(SettingsStateEditClonePathError), func() tea.Msg {
				return editClonePathErrorMsg{fmt.Errorf("failed to check repository status: %w", msg.err)}
			}
		}

		if msg.isDirty {
			// Repository has uncommitted changes
			m.logger.Info("Clone path edit blocked - repository has uncommitted changes")
			return m.transitionTo(SettingsStateEditClonePathError), func() tea.Msg {
				return editClonePathErrorMsg{fmt.Errorf("repository has uncommitted changes - please commit or stash them before changing clone path")}
			}
		}

		// Repository is clean, proceed to confirmation
		m.logger.Debug("Repository clean, proceeding to clone path confirmation")
		return m.transitionTo(SettingsStateEditClonePathConfirm), nil

	case editClonePathErrorMsg:
		// Transition to error state and display error
		m.logger.Error("Clone path edit error", "error", msg.err)
		m.layout = m.layout.SetError(msg.err)
		return m.transitionTo(SettingsStateEditClonePathError), nil

	case editNameErrorMsg:
		// Transition to error state and display error
		m.logger.Error("Repository name edit error", "error", msg.err)
		m.layout = m.layout.SetError(msg.err)
		return m.transitionTo(SettingsStateEditNameError), nil

	case deleteErrorMsg:
		// Transition to error state and display error
		m.logger.Error("Repository deletion error", "error", msg.err)
		m.layout = m.layout.SetError(msg.err)
		return m.transitionTo(SettingsStateDeleteError), nil

	case updatePATErrorMsg:
		// Transition to error state and display error
		m.logger.Error("PAT update error", "error", msg.err)
		m.layout = m.layout.SetError(msg.err)
		return m.transitionTo(SettingsStateUpdatePATError), nil

	case addLocalErrorMsg:
		m.logger.Error("Add local repository error", "error", msg.err)
		m.layout = m.layout.SetError(msg.err)
		return m.transitionTo(SettingsStateAddLocalError), nil

	case addGitHubErrorMsg:
		m.logger.Error("Add GitHub repository error", "error", msg.err)
		m.layout = m.layout.SetError(msg.err)
		return m.transitionTo(SettingsStateAddGitHubError), nil

	case addGitHubPATNeededMsg:
		// PAT is missing - transition to PAT input state
		m.logger.Info("GitHub PAT needed for repository creation, transitioning to PAT input")
		m.textInput.SetValue("")
		m.textInput.Placeholder = "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
		m.textInput.EchoMode = textinput.EchoPassword
		m.textInput.CharLimit = 100
		m.textInput.Focus()
		return m.transitionTo(SettingsStateAddGitHubPAT), nil
	}

	return m, cmd
}

// Load current configuration
func (m *SettingsModel) loadCurrentConfig() tea.Cmd {
	return func() tea.Msg {
		cfg, err := config.LoadConfig()
		return config.LoadConfigMsg{Config: cfg, Error: err}
	}
}

// Text input updates
func (m *SettingsModel) updateTextInput(msg tea.Msg) (*SettingsModel, tea.Cmd) {
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)

	// Clear error on input change
	if m.layout.GetError() != nil {
		m.layout = m.layout.ClearError()
	}

	// Mark as having changes
	m.hasChanges = true

	return m, cmd
}

// Key press delegation
func (m *SettingsModel) handleKeyPress(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	// Global navigation commands
	if m.isNavigationKey(msg) {
		return m.handleNavigation(msg)
	}

	switch m.state {
	case SettingsStateMainMenu:
		return m.handleMainMenuKeys(msg)
	case SettingsStateRepositoryActions:
		return m.handleRepositoryActionsKeys(msg)
	case SettingsStateConfirmDelete:
		return m.handleConfirmDeleteKeys(msg)
	case SettingsStateDeleteError:
		return m.handleDeleteErrorKeys(msg)
	case SettingsStateUpdateGitHubPath:
		return m.handleUpdateGitHubPathKeys(msg)
	case SettingsStateEditClonePathConfirm:
		return m.handleEditClonePathConfirmKeys(msg)
	case SettingsStateEditClonePathError:
		return m.handleEditClonePathErrorKeys(msg)
	case SettingsStateUpdateGitHubBranch:
		return m.handleUpdateGitHubBranchKeys(msg)
	case SettingsStateEditBranchConfirm:
		return m.handleEditBranchConfirmKeys(msg)
	case SettingsStateEditBranchError:
		return m.handleEditBranchErrorKeys(msg)
	case SettingsStateUpdateRepoName:
		return m.handleUpdateRepoNameKeys(msg)
	case SettingsStateEditNameConfirm:
		return m.handleEditNameConfirmKeys(msg)
	case SettingsStateEditNameError:
		return m.handleEditNameErrorKeys(msg)
	case SettingsStateUpdateGitHubPAT:
		return m.handleUpdateGitHubPATKeys(msg)
	case SettingsStateUpdatePATConfirm:
		return m.handleUpdatePATConfirmKeys(msg)
	case SettingsStateUpdatePATError:
		return m.handleUpdatePATErrorKeys(msg)
	case SettingsStateManualRefresh:
		return m.handleManualRefreshKeys(msg)
	case SettingsStateRefreshInProgress:
		return m.handleRefreshInProgressKeys(msg)
	case SettingsStateRefreshError:
		return m.handleRefreshErrorKeys(msg)
	case SettingsStateAddRepositoryType:
		return m.handleAddRepositoryTypeKeys(msg)
	case SettingsStateAddLocalName:
		return m.handleAddLocalNameKeys(msg)
	case SettingsStateAddLocalPath:
		return m.handleAddLocalPathKeys(msg)
	case SettingsStateAddLocalError:
		return m.handleAddLocalErrorKeys(msg)
	case SettingsStateAddGitHubName:
		return m.handleAddGitHubNameKeys(msg)
	case SettingsStateAddGitHubURL:
		return m.handleAddGitHubURLKeys(msg)
	case SettingsStateAddGitHubBranch:
		return m.handleAddGitHubBranchKeys(msg)
	case SettingsStateAddGitHubPath:
		return m.handleAddGitHubPathKeys(msg)
	case SettingsStateAddGitHubPAT:
		return m.handleAddGitHubPATKeys(msg)
	case SettingsStateAddGitHubError:
		return m.handleAddGitHubErrorKeys(msg)
	case SettingsStateComplete:
		return m.handleCompleteKeys(msg)
	default:
		slog.Warn("Unhandled settings menu state in key handler", "state", m.state.String())
		return m, nil
	}
}

// State-specific key handlers

// handleMainMenuKeys handles the repository list and adding new repositories
func (m *SettingsModel) handleMainMenuKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "enter", " ":
		selectedRepo, selectedItem := repolist.GetSelectedRepository(m.repoList)
		// Handle action items that are not repositories
		if actionItem, ok := selectedItem.(SettingsActionListItem); ok {
			switch actionItem.Action {
			case ChangeOptionAddNewRepository:
				m.logger.LogUserAction("settings_add_repository", "user selected add repository")
				m.addRepositoryTypeIndex = 0 // Reset to Local
				return m.transitionTo(SettingsStateAddRepositoryType), nil
			case ChangeOptionGitHubPAT:
				m.logger.LogUserAction("settings_update_pat", "user selected update GitHub PAT")
				return m.transitionToUpdateGitHubPAT()
			}
			return m, nil
		}

		if selectedRepo != nil {
			m.selectedRepositoryID = selectedRepo.ID
			m.logger.LogUserAction("settings_repository_selected", selectedRepo.Name)
			// Transition to repository actions menu
			return m.transitionTo(SettingsStateRepositoryActions), nil
		}
		return m, nil
	case "esc":
		return m, func() tea.Msg { return helpers.NavigateToMainMenuMsg{} }
	default:
		// Update the list with navigation keys
		m.repoList, cmd = m.repoList.Update(msg)
		return m, cmd
	}
}

func (m *SettingsModel) handleConfirmationKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		m.logger.LogUserAction("settings_confirmation_accept", "saving changes")
		return m, m.saveChanges()
	case "n", "N":
		m.logger.LogUserAction("settings_confirmation_reject", "discarding changes")
		m.resetTemporaryChanges()
		return m.transitionTo(SettingsStateRepositoryActions), nil
	case "esc":
		// Go back to the appropriate input state
		return m.transitionBack(), nil
	}
	return m, nil
}

func (m *SettingsModel) handleCompleteKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "enter", " ", "esc":
		return m, func() tea.Msg { return helpers.NavigateToMainMenuMsg{} }
	}
	return m, nil
}

// Navigation and utility functions
func (m *SettingsModel) isNavigationKey(msg tea.KeyMsg) bool {
	return msg.String() == "ctrl+c"
}

func (m *SettingsModel) handleNavigation(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

// State transition helpers

// transitionTo transitions to a new state with proper state tracking.
// Clears errors, resets selection, and logs the transition.
func (m *SettingsModel) transitionTo(newState SettingsState) *SettingsModel {
	m.logger.LogStateTransition("SettingsModel", m.state.String(), newState.String())
	m.previousState = m.state
	m.state = newState
	// Only clear error if we're actually changing states
	// This prevents clearing errors when transitioning to the same error state
	if m.previousState != newState {
		m.layout = m.layout.ClearError()
	}
	m.selectedRepositoryActionOption = 0
	return m
}

// transitionBack navigates back to the previous state.
// Used for escape key handling to maintain navigation history.
func (m *SettingsModel) transitionBack() *SettingsModel {
	m.logger.LogStateTransition("SettingsModel", m.state.String(), m.previousState.String())
	m.state = m.previousState
	m.layout = m.layout.ClearError()
	return m
}

// resetTemporaryChanges clears all pending changes.
// Called when user cancels or discards changes.
func (m *SettingsModel) resetTemporaryChanges() {
	m.changeType = 0
	m.newStorageDir = ""
	m.newGitHubURL = "" // Reset for Add GitHub flow
	m.newGitHubBranch = ""
	m.newGitHubPath = ""
	m.newGitHubPAT = "" // Reset for global PAT management
	m.hasChanges = false
}

// isGitHubRepo returns true if current configuration is a GitHub repository.
func (m *SettingsModel) isGitHubRepo() bool {
	if m.currentConfig == nil {
		return false
	}
	// Use selected repository instead of primary
	repo, err := m.currentConfig.FindRepositoryByID(m.selectedRepositoryID)
	if err != nil {
		return false
	}
	return repo.IsRemote()
}

// Save and operations

func (m *SettingsModel) saveChanges() tea.Cmd {
	return func() tea.Msg {
		m.logger.Info("Saving settings changes", "change_type", m.changeType)

		if !m.hasChanges {
			m.logger.Info("No changes to save, returning to main menu")
			return settingsCompleteMsg{}
		}

		// Perform actual config update
		if err := m.performConfigUpdate(); err != nil {
			m.logger.Error("Settings update failed", "error", err)
			// Return flow-specific error message based on change type
			switch m.changeType {
			case ChangeOptionChangeRepoName:
				return editNameErrorMsg{err}
			case ChangeOptionGitHubBranch:
				return editBranchErrorMsg{err}
			case ChangeOptionGitHubPath:
				return editClonePathErrorMsg{err}
			case ChangeOptionGitHubPAT:
				return updatePATErrorMsg{err}
			}
		}

		m.logger.Info("Settings updated successfully")
		return settingsCompleteMsg{}
	}
}

// performConfigUpdate performs the actual configuration update based on the change type.
// It uses the current config (m.currentConfig), applies the changes, and saves it back.
func (m *SettingsModel) performConfigUpdate() error {
	m.logger.Info("performConfigUpdate called", "change_type", m.changeType)

	if m.currentConfig == nil {
		return fmt.Errorf("current config is nil")
	}

	m.logger.Debug("Using current config for update.", "config", m.currentConfig, "change_type", m.changeType)

	switch m.changeType {
	case ChangeOptionGitHubPAT:
		return m.updateGitHubPAT()

	case ChangeOptionGitHubBranch:
		return m.updateGitHubBranch(m.currentConfig)

	case ChangeOptionGitHubPath:
		return m.updateGitHubPath(m.currentConfig)

	case ChangeOptionChangeRepoName:
		return m.updateRepositoryName(m.currentConfig)

	default:
		return fmt.Errorf("unknown change type: %v", m.changeType)
	}
}

// checkDirtyState performs an async check for uncommitted changes in a GitHub repository.
// It uses a message factory pattern to return flow-specific dirty state messages.
// This allows one underlying async status check while maintaining mutually exclusive states.
//
// Parameters:
//   - msgFactory: A function that takes (isDirty bool, err error) and returns a flow-specific message
//
// Example usage:
//
//	return m, m.checkDirtyState(func(isDirty bool, err error) tea.Msg {
//	    return editBranchDirtyStateMsg{isDirty: isDirty, err: err}
//	})
func (m *SettingsModel) checkDirtyState(msgFactory func(isDirty bool, err error) tea.Msg) tea.Cmd {
	return func() tea.Msg {
		// Find the repository we're checking
		repo, err := m.currentConfig.FindRepositoryByID(m.selectedRepositoryID)
		if err != nil {
			m.logger.Warn("Failed to find repository for dirty check", "error", err, "id", m.selectedRepositoryID)
			return msgFactory(false, fmt.Errorf("repository not found: %w", err))
		}

		// Only check GitHub repositories
		if !repo.IsRemote() {
			m.logger.Debug("Skipping dirty check for local repository")
			return msgFactory(false, nil)
		}

		// Check if repository path exists
		if repo.Path == "" {
			m.logger.Warn("Repository path is empty, cannot check dirty state")
			return msgFactory(false, fmt.Errorf("repository path not configured"))
		}

		// Perform the actual dirty state check
		m.logger.Debug("Checking repository for uncommitted changes", "path", repo.Path)
		isDirty, err := repository.CheckGithubRepositoryStatus(repo.Path)
		if err != nil {
			m.logger.Warn("Dirty state check failed", "error", err, "path", repo.Path)
			// Return error via factory - let flow decide how to handle
			return msgFactory(false, err)
		}

		m.logger.Debug("Dirty state check completed", "isDirty", isDirty, "path", repo.Path)
		return msgFactory(isDirty, nil)
	}
}

// View renders the current state
func (m *SettingsModel) View() string {
	switch m.state {
	case SettingsStateMainMenu:
		return m.viewMainMenu()
	case SettingsStateRepositoryActions:
		return m.viewRepositoryActions()
	case SettingsStateConfirmDelete:
		return m.viewConfirmDelete()
	case SettingsStateDeleteError:
		return m.viewDeleteError()
	case SettingsStateUpdateGitHubPath:
		return m.viewUpdateGitHubPath()
	case SettingsStateEditClonePathConfirm:
		return m.viewEditClonePathConfirm()
	case SettingsStateEditClonePathError:
		return m.viewEditClonePathError()
	case SettingsStateUpdateGitHubBranch:
		return m.viewUpdateGitHubBranch()
	case SettingsStateEditBranchConfirm:
		return m.viewEditBranchConfirm()
	case SettingsStateEditBranchError:
		return m.viewEditBranchError()
	case SettingsStateUpdateRepoName:
		return m.viewUpdateRepoName()
	case SettingsStateEditNameConfirm:
		return m.viewEditNameConfirm()
	case SettingsStateEditNameError:
		return m.viewEditNameError()
	case SettingsStateUpdateGitHubPAT:
		return m.viewUpdateGitHubPAT()
	case SettingsStateUpdatePATConfirm:
		return m.viewUpdatePATConfirm()
	case SettingsStateUpdatePATError:
		return m.viewUpdatePATError()
	case SettingsStateManualRefresh:
		return m.viewManualRefresh()
	case SettingsStateRefreshInProgress:
		return m.viewRefreshInProgress()
	case SettingsStateRefreshError:
		return m.viewRefreshError()
	case SettingsStateAddRepositoryType:
		return m.viewAddRepositoryType()
	case SettingsStateAddLocalName:
		return m.viewAddLocalName()
	case SettingsStateAddLocalPath:
		return m.viewAddLocalPath()
	case SettingsStateAddLocalError:
		return m.viewAddLocalError()
	case SettingsStateAddGitHubName:
		return m.viewAddGitHubName()
	case SettingsStateAddGitHubURL:
		return m.viewAddGitHubURL()
	case SettingsStateAddGitHubBranch:
		return m.viewAddGitHubBranch()
	case SettingsStateAddGitHubPath:
		return m.viewAddGitHubPath()
	case SettingsStateAddGitHubPAT:
		return m.viewAddGitHubPAT()
	case SettingsStateAddGitHubError:
		return m.viewAddGitHubError()
	case SettingsStateComplete:
		return m.viewComplete()
	}
	return ""
}

// viewMainMenu renders the main menu screen showing all configured repositories.
// This is the landing state for the settings menu where users can:
// - View all configured repositories
// - Select a repository to manage
// - Add a new repository (Local or GitHub)
func (m *SettingsModel) viewMainMenu() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "⚙️  Settings - Repositories",
		Subtitle: "Manage your rule repositories",
		HelpText: "↑/↓ to navigate • Enter to select • Esc to go back",
	})

	var content strings.Builder

	// Render repository list (includes "Add Repository" action item)
	content.WriteString(m.repoList.View())

	return m.layout.Render(content.String())
}
