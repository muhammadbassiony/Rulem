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
//   - RepositoryTypeSelect: Select new repository type
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
	"strings"

	"rulem/internal/config"
	"rulem/internal/logging"
	"rulem/internal/repository"
	"rulem/internal/tui/components"
	"rulem/internal/tui/helpers"
	"rulem/internal/tui/helpers/settingshelpers"
	"rulem/internal/tui/styles"
	"rulem/pkg/fileops"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SettingsState represents the current state of the settings flow
type SettingsState int

const (
	// Main navigation states
	SettingsStateMainMenu SettingsState = iota
	SettingsStateSelectChange

	// Repository type change flow
	SettingsStateConfirmTypeChange
	SettingsStateRepositoryTypeSelect

	// Granular update states
	SettingsStateUpdateLocalPath
	SettingsStateUpdateGitHubPAT
	SettingsStateUpdateGitHubURL
	SettingsStateUpdateGitHubBranch
	SettingsStateUpdateGitHubPath

	// GitHub operations
	SettingsStateManualRefresh
	SettingsStateRefreshInProgress

	// Terminal states
	SettingsStateConfirmation
	SettingsStateComplete
	SettingsStateError
)

// String returns a human-readable name for the state
func (s SettingsState) String() string {
	switch s {
	case SettingsStateMainMenu:
		return "MainMenu"
	case SettingsStateSelectChange:
		return "SelectChange"
	case SettingsStateConfirmTypeChange:
		return "ConfirmTypeChange"
	case SettingsStateRepositoryTypeSelect:
		return "RepositoryTypeSelect"
	case SettingsStateUpdateLocalPath:
		return "UpdateLocalPath"
	case SettingsStateUpdateGitHubPAT:
		return "UpdateGitHubPAT"
	case SettingsStateUpdateGitHubURL:
		return "UpdateGitHubURL"
	case SettingsStateUpdateGitHubBranch:
		return "UpdateGitHubBranch"
	case SettingsStateUpdateGitHubPath:
		return "UpdateGitHubPath"
	case SettingsStateManualRefresh:
		return "ManualRefresh"
	case SettingsStateRefreshInProgress:
		return "RefreshInProgress"
	case SettingsStateConfirmation:
		return "Confirmation"
	case SettingsStateComplete:
		return "Complete"
	case SettingsStateError:
		return "Error"
	default:
		return "Unknown"
	}
}

// ChangeOption represents a type of change the user can make
type ChangeOption int

const (
	ChangeOptionRepositoryType ChangeOption = iota
	ChangeOptionLocalPath
	ChangeOptionGitHubPAT
	ChangeOptionGitHubURL
	ChangeOptionGitHubBranch
	ChangeOptionGitHubPath
	ChangeOptionManualRefresh
	ChangeOptionBack
)

// ChangeOptionInfo contains display information for a change option
type ChangeOptionInfo struct {
	Option  ChangeOption
	Display string
	Desc    string
}

// PATAction represents the action to take on the PAT
type PATAction int

const (
	PATActionNone PATAction = iota
	PATActionUpdate
	PATActionRemove
)

// Custom messages for internal state transitions

// settingsErrorMsg signals an error occurred during settings operations.
// Transitions model to SettingsStateError and displays error to user.
type settingsErrorMsg struct{ err error }

// settingsCompleteMsg signals successful completion of settings update.
// Transitions model to SettingsStateComplete and triggers config reload.
type settingsCompleteMsg struct{}

// refreshInitiateMsg signals the start of a manual refresh operation.
// Sent when user confirms manual refresh from GitHub.
type refreshInitiateMsg struct{}

// refreshInProgressMsg indicates refresh operation is currently running.
// Used to update UI during async refresh operations.
type refreshInProgressMsg struct{}

// refreshCompleteMsg signals completion of manual refresh operation.
// Contains sync information or error from the refresh attempt.
type refreshCompleteMsg struct {
	syncInfo *repository.SyncInfo
	err      error
}

// dirtyStateMsg reports the dirty state of the local repository.
// Sent after checking if repository has uncommitted changes.
type dirtyStateMsg struct {
	isDirty bool
	err     error
}

// SettingsModel handles the settings modification flow
type SettingsModel struct {
	// State management
	state         SettingsState
	previousState SettingsState

	// Current configuration
	currentConfig *config.Config

	// Temporary changes (not yet saved)
	changeType        ChangeOption
	newRepositoryType string // "local" or "github"
	newStorageDir     string
	newGitHubURL      string
	newGitHubBranch   string
	newGitHubPath     string
	newGitHubPAT      string
	patAction         PATAction

	// Components
	textInput textinput.Model
	layout    components.LayoutModel

	// Selection state
	selectedOption    int
	repositoryOptions []settingshelpers.RepositoryTypeOption

	// Change tracking
	hasChanges bool

	// GitHub repository state
	isDirty           bool
	refreshInProgress bool
	lastSyncInfo      *repository.SyncInfo

	// Dependencies
	logger      *logging.AppLogger
	credManager *repository.CredentialManager
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

	// Apply window sizing if available
	if ctx.HasValidDimensions() {
		windowMsg := tea.WindowSizeMsg{Width: ctx.Width, Height: ctx.Height}
		layout, _ = layout.Update(windowMsg)
		ti.Width = layout.InputWidth()
	}

	return &SettingsModel{
		state:             SettingsStateMainMenu,
		textInput:         ti,
		layout:            layout,
		logger:            ctx.Logger,
		credManager:       repository.NewCredentialManager(),
		ctx:               ctx,
		context:           context.Background(),
		repositoryOptions: settingshelpers.GetRepositoryTypeOptions(),
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

	case settingsErrorMsg:
		m.state = SettingsStateError
		m.layout = m.layout.SetError(msg.err)
		return m, nil

	case settingsCompleteMsg:
		m.state = SettingsStateComplete
		m.layout = m.layout.ClearError()
		// Trigger config reload in parent after successful update
		return m, config.ReloadConfig()

	case config.LoadConfigMsg:
		if msg.Error != nil {
			m.logger.Error("Failed to load current config", "error", msg.Error)
			return m, func() tea.Msg { return settingsErrorMsg{msg.Error} }
		}
		m.currentConfig = msg.Config
		// Load last sync info if GitHub repo
		if m.isGitHubRepo() {
			m.lastSyncInfo = &repository.SyncInfo{
				// TODO: Load actual sync info from repository
				// For now, placeholder
			}
		}
		return m, nil

	case refreshCompleteMsg:
		m.refreshInProgress = false
		if msg.err != nil {
			m.logger.Error("Refresh failed", "error", msg.err)
			return m, func() tea.Msg { return settingsErrorMsg{msg.err} }
		}
		m.lastSyncInfo = msg.syncInfo
		m.state = SettingsStateMainMenu
		m.layout = m.layout.ClearError()
		// TODO: Update config.Central.LastSyncTime after successful refresh
		return m, nil

	case dirtyStateMsg:
		m.isDirty = msg.isDirty
		if msg.err != nil {
			m.logger.Warn("Dirty state check failed", "error", msg.err)
		}
		// TODO: Display dirty state warning in UI if isDirty is true
		// TODO: Prevent certain operations (like URL change) if repository is dirty
		return m, nil
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
	case SettingsStateSelectChange:
		return m.handleSelectChangeKeys(msg)
	case SettingsStateConfirmTypeChange:
		return m.handleConfirmTypeChangeKeys(msg)
	case SettingsStateRepositoryTypeSelect:
		return m.handleRepositoryTypeSelectKeys(msg)
	case SettingsStateUpdateLocalPath:
		return m.handleUpdateLocalPathKeys(msg)
	case SettingsStateUpdateGitHubPAT:
		return m.handleUpdateGitHubPATKeys(msg)
	case SettingsStateUpdateGitHubURL:
		return m.handleUpdateGitHubURLKeys(msg)
	case SettingsStateUpdateGitHubBranch:
		return m.handleUpdateGitHubBranchKeys(msg)
	case SettingsStateUpdateGitHubPath:
		return m.handleUpdateGitHubPathKeys(msg)
	case SettingsStateManualRefresh:
		return m.handleManualRefreshKeys(msg)
	case SettingsStateRefreshInProgress:
		// No input during refresh
		return m, nil
	case SettingsStateConfirmation:
		return m.handleConfirmationKeys(msg)
	case SettingsStateComplete, SettingsStateError:
		return m.handleCompleteKeys(msg)
	default:
		return m, nil
	}
}

// State-specific key handlers

func (m *SettingsModel) handleMainMenuKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "enter", " ":
		m.logger.LogUserAction("settings_main_menu_select", "proceeding to change selection")
		return m.transitionTo(SettingsStateSelectChange), nil
	case "esc":
		return m, func() tea.Msg { return helpers.NavigateToMainMenuMsg{} }
	}
	return m, nil
}

func (m *SettingsModel) handleSelectChangeKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	options := m.getMenuOptions()

	switch msg.String() {
	case "up", "k":
		if m.selectedOption > 0 {
			m.selectedOption--
		}
	case "down", "j":
		if m.selectedOption < len(options)-1 {
			m.selectedOption++
		}
	case "enter", " ":
		selected := options[m.selectedOption]
		m.changeType = selected.Option
		m.logger.LogUserAction("settings_change_selected", selected.Display)

		switch selected.Option {
		case ChangeOptionBack:
			return m.transitionTo(SettingsStateMainMenu), nil
		case ChangeOptionRepositoryType:
			return m.transitionTo(SettingsStateConfirmTypeChange), nil
		case ChangeOptionLocalPath:
			return m.transitionToUpdateLocalPath()
		case ChangeOptionGitHubPAT:
			return m.transitionToUpdateGitHubPAT()
		case ChangeOptionGitHubURL:
			return m.transitionToUpdateGitHubURL()
		case ChangeOptionGitHubBranch:
			return m.transitionToUpdateGitHubBranch()
		case ChangeOptionGitHubPath:
			return m.transitionToUpdateGitHubPath()
		case ChangeOptionManualRefresh:
			return m.transitionTo(SettingsStateManualRefresh), nil
		}
	case "esc":
		return m.transitionTo(SettingsStateMainMenu), nil
	}
	return m, nil
}

func (m *SettingsModel) handleConfirmTypeChangeKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		m.logger.LogUserAction("settings_type_change_confirmed", "proceeding to type selection")
		return m.transitionTo(SettingsStateRepositoryTypeSelect), nil
	case "n", "N", "esc":
		m.logger.LogUserAction("settings_type_change_cancelled", "returning to menu")
		return m.transitionTo(SettingsStateSelectChange), nil
	}
	return m, nil
}

func (m *SettingsModel) handleRepositoryTypeSelectKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedOption > 0 {
			m.selectedOption--
		}
	case "down", "j":
		if m.selectedOption < len(m.repositoryOptions)-1 {
			m.selectedOption++
		}
	case "enter", " ":
		selected := m.repositoryOptions[m.selectedOption]
		m.newRepositoryType = selected.Type
		m.logger.LogUserAction("settings_new_repo_type_selected", selected.Type)

		// Transition to appropriate setup flow
		if selected.Type == "local" {
			return m.transitionToUpdateLocalPath()
		}
		return m.transitionToUpdateGitHubURL()
	case "esc":
		return m.transitionTo(SettingsStateConfirmTypeChange), nil
	}
	return m, nil
}

func (m *SettingsModel) handleUpdateLocalPathKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.logger.LogUserAction("settings_local_path_submit", m.textInput.Value())
		return m.validateAndProceedLocalPath()
	case "esc":
		m.resetTemporaryChanges()
		if m.changeType == ChangeOptionRepositoryType {
			return m.transitionTo(SettingsStateRepositoryTypeSelect), nil
		}
		return m.transitionTo(SettingsStateSelectChange), nil
	default:
		return m.updateTextInput(msg)
	}
}

func (m *SettingsModel) handleUpdateGitHubPATKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "ctrl+d":
		// Remove PAT
		m.logger.LogUserAction("settings_pat_remove", "user requested PAT removal")
		m.patAction = PATActionRemove
		m.newGitHubPAT = ""
		// TODO: Validate that removing PAT won't break repository access
		return m.transitionTo(SettingsStateConfirmation), nil
	case "enter":
		m.logger.LogUserAction("settings_pat_submit", "PAT provided")
		input := strings.TrimSpace(m.textInput.Value())
		if input == "" {
			return m, func() tea.Msg {
				return settingsErrorMsg{fmt.Errorf("PAT cannot be empty (use Ctrl+D to remove)")}
			}
		}
		// TODO: Validate PAT format using credManager.ValidateGitHubToken()
		m.patAction = PATActionUpdate
		m.newGitHubPAT = input
		return m.transitionTo(SettingsStateConfirmation), nil
	case "esc":
		m.resetTemporaryChanges()
		if m.changeType == ChangeOptionRepositoryType {
			return m.transitionToUpdateGitHubPath()
		}
		return m.transitionTo(SettingsStateSelectChange), nil
	default:
		return m.updateTextInput(msg)
	}
}

func (m *SettingsModel) handleUpdateGitHubURLKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.logger.LogUserAction("settings_github_url_submit", m.textInput.Value())
		input := strings.TrimSpace(m.textInput.Value())
		if err := settingshelpers.ValidateGitHubURL(input); err != nil {
			m.logger.Warn("GitHub URL validation failed", "error", err)
			return m, func() tea.Msg { return settingsErrorMsg{err} }
		}
		m.newGitHubURL = input
		// TODO: Optionally test repository accessibility before proceeding

		// If changing repository type, continue setup flow
		if m.changeType == ChangeOptionRepositoryType {
			return m.transitionToUpdateGitHubBranch()
		}
		// Otherwise, go to confirmation
		return m.transitionTo(SettingsStateConfirmation), nil
	case "esc":
		m.resetTemporaryChanges()
		if m.changeType == ChangeOptionRepositoryType {
			return m.transitionTo(SettingsStateRepositoryTypeSelect), nil
		}
		return m.transitionTo(SettingsStateSelectChange), nil
	default:
		return m.updateTextInput(msg)
	}
}

func (m *SettingsModel) handleUpdateGitHubBranchKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		input := strings.TrimSpace(m.textInput.Value())
		m.logger.LogUserAction("settings_github_branch_submit", input)
		if err := settingshelpers.ValidateBranchName(input); err != nil {
			m.logger.Warn("Branch validation failed", "error", err)
			return m, func() tea.Msg { return settingsErrorMsg{err} }
		}
		m.newGitHubBranch = input

		// If changing repository type, continue setup flow
		if m.changeType == ChangeOptionRepositoryType {
			return m.transitionToUpdateGitHubPath()
		}
		// Otherwise, go to confirmation
		return m.transitionTo(SettingsStateConfirmation), nil
	case "esc":
		m.resetTemporaryChanges()
		if m.changeType == ChangeOptionRepositoryType {
			return m.transitionToUpdateGitHubURL()
		}
		return m.transitionTo(SettingsStateSelectChange), nil
	default:
		return m.updateTextInput(msg)
	}
}

func (m *SettingsModel) handleUpdateGitHubPathKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.logger.LogUserAction("settings_github_path_submit", m.textInput.Value())
		input := strings.TrimSpace(m.textInput.Value())
		if err := fileops.ValidateStoragePath(input); err != nil {
			m.logger.Warn("Path validation failed", "error", err)
			return m, func() tea.Msg { return settingsErrorMsg{err} }
		}
		m.newGitHubPath = fileops.ExpandPath(input)

		// If changing repository type, continue setup flow
		if m.changeType == ChangeOptionRepositoryType {
			return m.transitionToUpdateGitHubPAT()
		}
		// Otherwise, go to confirmation
		return m.transitionTo(SettingsStateConfirmation), nil
	case "esc":
		m.resetTemporaryChanges()
		if m.changeType == ChangeOptionRepositoryType {
			return m.transitionToUpdateGitHubBranch()
		}
		return m.transitionTo(SettingsStateSelectChange), nil
	default:
		return m.updateTextInput(msg)
	}
}

func (m *SettingsModel) handleManualRefreshKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		m.logger.LogUserAction("settings_manual_refresh_confirmed", "triggering refresh")
		return m, m.triggerRefresh()
	case "n", "N", "esc":
		m.logger.LogUserAction("settings_manual_refresh_cancelled", "returning to menu")
		return m.transitionTo(SettingsStateSelectChange), nil
	}
	return m, nil
}

func (m *SettingsModel) handleConfirmationKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		m.logger.LogUserAction("settings_confirmation_accept", "saving changes")
		return m, m.saveChanges()
	case "n", "N":
		m.logger.LogUserAction("settings_confirmation_reject", "discarding changes")
		m.resetTemporaryChanges()
		return m.transitionTo(SettingsStateSelectChange), nil
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
	m.layout = m.layout.ClearError()
	m.selectedOption = 0
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
	m.newRepositoryType = ""
	m.newStorageDir = ""
	m.newGitHubURL = ""
	m.newGitHubBranch = ""
	m.newGitHubPath = ""
	m.newGitHubPAT = ""
	m.patAction = PATActionNone
	m.hasChanges = false
}

// isGitHubRepo returns true if current configuration is a GitHub repository.
func (m *SettingsModel) isGitHubRepo() bool {
	return m.currentConfig != nil && m.currentConfig.Central.IsRemote()
}

// Transition helpers with input setup

// transitionToUpdateLocalPath transitions to local path update state.
// Sets up text input with current local path as default.
func (m *SettingsModel) transitionToUpdateLocalPath() (*SettingsModel, tea.Cmd) {
	defaultPath := repository.GetDefaultStorageDir()
	if m.currentConfig != nil {
		defaultPath = m.currentConfig.Central.Path
	}

	return m.transitionTo(SettingsStateUpdateLocalPath), settingshelpers.ResetTextInputForState(&m.textInput, defaultPath, defaultPath, textinput.EchoNormal)
}

// transitionToUpdateGitHubPAT transitions to GitHub PAT update state.
// Sets up password-masked text input for PAT entry.
func (m *SettingsModel) transitionToUpdateGitHubPAT() (*SettingsModel, tea.Cmd) {
	return m.transitionTo(SettingsStateUpdateGitHubPAT), settingshelpers.ResetTextInputForState(&m.textInput, "", "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", textinput.EchoPassword)
}

// transitionToUpdateGitHubURL transitions to GitHub URL update state.
// Sets up text input with current GitHub URL as default.
func (m *SettingsModel) transitionToUpdateGitHubURL() (*SettingsModel, tea.Cmd) {
	defaultURL := ""
	if m.currentConfig != nil && m.currentConfig.Central.RemoteURL != nil {
		defaultURL = *m.currentConfig.Central.RemoteURL
	}

	return m.transitionTo(SettingsStateUpdateGitHubURL), settingshelpers.ResetTextInputForState(&m.textInput, defaultURL, "https://github.com/user/repo.git", textinput.EchoNormal)
}

// transitionToUpdateGitHubBranch transitions to GitHub branch update state.
// Sets up text input with current branch as default.
func (m *SettingsModel) transitionToUpdateGitHubBranch() (*SettingsModel, tea.Cmd) {
	defaultBranch := ""
	if m.currentConfig != nil && m.currentConfig.Central.Branch != nil {
		defaultBranch = *m.currentConfig.Central.Branch
	}

	return m.transitionTo(SettingsStateUpdateGitHubBranch), settingshelpers.ResetTextInputForState(&m.textInput, defaultBranch, "main (leave empty for default)", textinput.EchoNormal)
}

// transitionToUpdateGitHubPath transitions to GitHub clone path update state.
// Derives smart default path from GitHub URL if available.
func (m *SettingsModel) transitionToUpdateGitHubPath() (*SettingsModel, tea.Cmd) {
	placeholder := settingshelpers.DeriveClonePath(m.newGitHubURL)
	if placeholder == "" {
		placeholder = repository.GetDefaultStorageDir()
	}

	return m.transitionTo(SettingsStateUpdateGitHubPath), settingshelpers.ResetTextInputForState(&m.textInput, "", placeholder, textinput.EchoNormal)
}

// Validation helpers

// validateAndProceedLocalPath validates the entered local path and proceeds to confirmation
// if the path is valid and different from the current path.
func (m *SettingsModel) validateAndProceedLocalPath() (*SettingsModel, tea.Cmd) {
	input := m.textInput.Value()
	m.logger.Debug("Validating local path", "path", input)

	// Validate and expand the storage path using shared helper
	expandedPath, err := settingshelpers.ValidateAndExpandLocalPath(input)
	if err != nil {
		m.logger.Warn("Local path validation failed", "error", err)
		return m, func() tea.Msg { return settingsErrorMsg{err} }
	}

	m.newStorageDir = expandedPath

	// Check if the path has actually changed
	currentPath := m.currentConfig.Central.Path
	if expandedPath == currentPath {
		m.logger.Info("Path unchanged, returning to main menu")
		m.resetTemporaryChanges()
		return m.transitionTo(SettingsStateMainMenu), nil
	}

	// Mark that we have changes to save
	m.hasChanges = true
	m.logger.LogStateTransition("SettingsModel", "UpdateLocalPath", "Confirmation")
	m.logger.Info("Local path validated and changed", "old", currentPath, "new", expandedPath)

	return m.transitionTo(SettingsStateConfirmation), nil
}

// Save and operations

func (m *SettingsModel) saveChanges() tea.Cmd {
	return func() tea.Msg {
		m.logger.Info("Saving settings changes", "change_type", m.changeType)

		if !m.hasChanges {
			m.logger.Info("No changes to save, returning to main menu")
			return settingsCompleteMsg{}
		}

		// Perform the actual config update based on change type
		if err := m.performConfigUpdate(); err != nil {
			m.logger.Error("Settings update failed", "error", err)
			return settingsErrorMsg{err}
		}

		m.logger.Info("Settings updated successfully")
		return settingsCompleteMsg{}
	}
}

// performConfigUpdate performs the actual configuration update based on the change type.
// It loads the current config, applies the changes, and saves it back.
func (m *SettingsModel) performConfigUpdate() error {
	// Load current config
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	switch m.changeType {
	case ChangeOptionLocalPath:
		return m.updateLocalPath(cfg)

	case ChangeOptionGitHubPAT:
		return m.updateGitHubPAT(cfg)

	case ChangeOptionGitHubURL:
		return m.updateGitHubURL(cfg)

	case ChangeOptionGitHubBranch:
		return m.updateGitHubBranch(cfg)

	case ChangeOptionGitHubPath:
		return m.updateGitHubPath(cfg)

	case ChangeOptionRepositoryType:
		return m.updateRepositoryType(cfg)

	default:
		return fmt.Errorf("unknown change type: %v", m.changeType)
	}
}

// updateLocalPath updates the local storage path in the configuration.
func (m *SettingsModel) updateLocalPath(cfg *config.Config) error {
	m.logger.Info("Updating local path", "old", cfg.Central.Path, "new", m.newStorageDir)

	// Use the config helper to update the path, which also ensures the directory exists
	if err := config.UpdateCentralPath(cfg, m.newStorageDir); err != nil {
		return fmt.Errorf("failed to update local path: %w", err)
	}

	m.logger.Info("Local path updated successfully", "path", m.newStorageDir)
	return nil
}

// updateGitHubPAT updates or removes the GitHub Personal Access Token.
func (m *SettingsModel) updateGitHubPAT(cfg *config.Config) error {
	m.logger.Info("Updating GitHub PAT", "action", m.patAction)

	// TODO Implement PAT update logic

	return nil
}

// updateGitHubURL updates the GitHub repository URL in the configuration.
func (m *SettingsModel) updateGitHubURL(cfg *config.Config) error {
	m.logger.Info("Updating GitHub URL", "old", cfg.Central.RemoteURL, "new", m.newGitHubURL)

	// TODO: Implement GitHub URL update
	// This will require:
	// 1. Validate the new URL
	// 2. Update cfg.Central.RemoteURL
	// 3. Possibly update clone path
	// 4. Save config
	// 5. May require re-cloning the repository

	return fmt.Errorf("GitHub URL update not yet implemented")
}

// updateGitHubBranch updates the GitHub branch in the configuration.
func (m *SettingsModel) updateGitHubBranch(cfg *config.Config) error {
	m.logger.Info("Updating GitHub branch", "old", cfg.Central.Branch, "new", m.newGitHubBranch)

	// TODO: Implement GitHub branch update
	// This will require:
	// 1. Validate the new branch
	// 2. Update cfg.Central.Branch
	// 3. Save config
	// 4. May require checking out the new branch

	return fmt.Errorf("GitHub branch update not yet implemented")
}

// updateGitHubPath updates the path within the GitHub repository in the configuration.
func (m *SettingsModel) updateGitHubPath(cfg *config.Config) error {
	m.logger.Info("Updating GitHub path", "old", cfg.Central.Path, "new", m.newGitHubPath)

	// TODO: Implement GitHub path update
	// This will require:
	// 1. Validate the new path
	// 2. Update cfg.Central.Path
	// 3. Save config

	return fmt.Errorf("GitHub path update not yet implemented")
}

// updateRepositoryType performs a full repository type switch (local <-> GitHub).
func (m *SettingsModel) updateRepositoryType(cfg *config.Config) error {
	m.logger.Info("Switching repository type", "old", cfg.Central.RemoteURL != nil, "new", m.newRepositoryType)

	// TODO: Implement repository type switching
	// This is a complex operation that requires:
	// 1. If switching to local: remove remote config, update path
	// 2. If switching to GitHub: add remote config, clone repository
	// 3. Handle data migration
	// 4. Clean up old repository (with user confirmation)

	return fmt.Errorf("repository type switching not yet implemented")
}

func (m *SettingsModel) triggerRefresh() tea.Cmd {
	m.refreshInProgress = true
	m.state = SettingsStateRefreshInProgress

	return func() tea.Msg {
		m.logger.Info("Triggering manual refresh")

		// TODO: Implement actual refresh logic
		// TODO: Call GitSource.Prepare() or equivalent sync operation
		// TODO: Handle authentication with stored PAT
		// TODO: Detect and report dirty state before refresh
		// TODO: Handle merge conflicts if any
		//
		// Example:
		// source := repository.NewGitSource(
		//     *m.currentConfig.Central.RemoteURL,
		//     m.currentConfig.Central.Branch,
		//     m.currentConfig.Central.Path,
		// )
		// _, syncInfo, err := source.Prepare(m.logger)
		// return refreshCompleteMsg{syncInfo: &syncInfo, err: err}

		return refreshCompleteMsg{
			syncInfo: &repository.SyncInfo{},
			err:      nil,
		}
	}
}

// View renders the current state
func (m *SettingsModel) View() string {
	switch m.state {
	case SettingsStateMainMenu:
		return m.viewMainMenu()
	case SettingsStateSelectChange:
		return m.viewSelectChange()
	case SettingsStateConfirmTypeChange:
		return m.viewConfirmTypeChange()
	case SettingsStateRepositoryTypeSelect:
		return m.viewRepositoryTypeSelect()
	case SettingsStateUpdateLocalPath:
		return m.viewUpdateLocalPath()
	case SettingsStateUpdateGitHubPAT:
		return m.viewUpdateGitHubPAT()
	case SettingsStateUpdateGitHubURL:
		return m.viewUpdateGitHubURL()
	case SettingsStateUpdateGitHubBranch:
		return m.viewUpdateGitHubBranch()
	case SettingsStateUpdateGitHubPath:
		return m.viewUpdateGitHubPath()
	case SettingsStateManualRefresh:
		return m.viewManualRefresh()
	case SettingsStateRefreshInProgress:
		return m.viewRefreshInProgress()
	case SettingsStateConfirmation:
		return m.viewConfirmation()
	case SettingsStateComplete:
		return m.viewComplete()
	case SettingsStateError:
		return m.viewError()
	}
	return ""
}

// View methods

func (m *SettingsModel) viewMainMenu() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "⚙️  Settings",
		Subtitle: "Current Configuration",
		HelpText: "Press Enter to modify settings • Esc to go back",
	})

	content := m.formatCurrentConfig()
	content += "\n\n" + lipgloss.NewStyle().Faint(true).Render("Press Enter to modify settings")

	return m.layout.Render(content)
}

func (m *SettingsModel) viewSelectChange() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "⚙️  Modify Settings",
		Subtitle: "Choose what to change",
		HelpText: "↑/↓ to navigate • Enter to select • Esc to go back",
	})

	options := m.getMenuOptions()
	var content strings.Builder

	for i, opt := range options {
		prefix := "  "
		if i == m.selectedOption {
			prefix = "▶ "
		}

		content.WriteString(fmt.Sprintf("%s%s\n", prefix, opt.Display))
		if opt.Desc != "" {
			content.WriteString(fmt.Sprintf("  %s\n", lipgloss.NewStyle().Faint(true).Render(opt.Desc)))
		}
		content.WriteString("\n")
	}

	return m.layout.Render(content.String())
}

func (m *SettingsModel) viewConfirmTypeChange() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "⚠️  Confirm Repository Type Change",
		Subtitle: "This is a significant change",
		HelpText: "y to confirm • n to cancel • Esc to go back",
	})

	currentType := "Local Directory"
	if m.isGitHubRepo() {
		currentType = "GitHub Repository"
	}

	warning := fmt.Sprintf(`You are about to change your repository type from:
  %s

This will require you to reconfigure your repository settings.

%s

Are you sure you want to proceed? (y/N)`,
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#5fd7ff")).Render(currentType),
		m.getCleanupWarning())

	return m.layout.Render(warning)
}

func (m *SettingsModel) viewRepositoryTypeSelect() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "📁 Select Repository Type",
		Subtitle: "Choose your new repository type",
		HelpText: "↑/↓ to navigate • Enter to select • Esc to go back",
	})

	var content strings.Builder

	for i, opt := range m.repositoryOptions {
		prefix := "  "
		if i == m.selectedOption {
			prefix = "▶ "
		}

		content.WriteString(fmt.Sprintf("%s%s\n", prefix, opt.Title))
		content.WriteString(fmt.Sprintf("  %s\n\n", lipgloss.NewStyle().Faint(true).Render(opt.Desc)))
	}

	return m.layout.Render(content.String())
}

func (m *SettingsModel) viewUpdateLocalPath() string {
	title := "📁 Update Local Path"
	if m.changeType == ChangeOptionRepositoryType {
		title = "📁 Local Repository Path"
	}

	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    title,
		Subtitle: "Enter the local directory path",
		HelpText: "Enter to save • Esc to cancel",
	})

	var content strings.Builder

	if m.currentConfig != nil {
		content.WriteString(fmt.Sprintf("Current: %s\n\n", lipgloss.NewStyle().Faint(true).Render(m.currentConfig.Central.Path)))
	}

	content.WriteString("Local directory path:\n")
	content.WriteString(styles.InputStyle.Render(m.textInput.View()))

	return m.layout.Render(content.String())
}

func (m *SettingsModel) viewUpdateGitHubPAT() string {
	title := "🔑 Update GitHub PAT"
	if m.changeType == ChangeOptionRepositoryType {
		title = "🔑 GitHub Personal Access Token"
	}

	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    title,
		Subtitle: "Enter your GitHub Personal Access Token",
		HelpText: "Enter to save • Ctrl+D to remove • Esc to cancel",
	})

	var content strings.Builder

	content.WriteString("Personal Access Token (PAT):\n")
	content.WriteString(styles.InputStyle.Render(m.textInput.View()))
	content.WriteString("\n\n")
	content.WriteString(lipgloss.NewStyle().Faint(true).Render("Your PAT will be stored securely in your system keyring"))

	return m.layout.Render(content.String())
}

func (m *SettingsModel) viewUpdateGitHubURL() string {
	title := "🌐 Update GitHub URL"
	if m.changeType == ChangeOptionRepositoryType {
		title = "🌐 GitHub Repository URL"
	}

	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    title,
		Subtitle: "Enter the GitHub repository URL",
		HelpText: "Enter to save • Esc to cancel",
	})

	var content strings.Builder

	if m.currentConfig != nil && m.currentConfig.Central.RemoteURL != nil {
		content.WriteString(fmt.Sprintf("Current: %s\n\n", lipgloss.NewStyle().Faint(true).Render(*m.currentConfig.Central.RemoteURL)))
	}

	content.WriteString("Repository URL:\n")
	content.WriteString(styles.InputStyle.Render(m.textInput.View()))

	return m.layout.Render(content.String())
}

func (m *SettingsModel) viewUpdateGitHubBranch() string {
	title := "🌿 Update GitHub Branch"
	if m.changeType == ChangeOptionRepositoryType {
		title = "🌿 GitHub Branch"
	}

	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    title,
		Subtitle: "Enter the branch name",
		HelpText: "Enter to save • Esc to cancel",
	})

	var content strings.Builder

	if m.currentConfig != nil && m.currentConfig.Central.Branch != nil {
		content.WriteString(fmt.Sprintf("Current: %s\n\n", lipgloss.NewStyle().Faint(true).Render(*m.currentConfig.Central.Branch)))
	}

	content.WriteString("Branch name (leave empty for default):\n")
	content.WriteString(styles.InputStyle.Render(m.textInput.View()))

	return m.layout.Render(content.String())
}

func (m *SettingsModel) viewUpdateGitHubPath() string {
	title := "📂 Update Clone Path"
	if m.changeType == ChangeOptionRepositoryType {
		title = "📂 Local Clone Path"
	}

	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    title,
		Subtitle: "Enter where to clone the repository",
		HelpText: "Enter to save • Esc to cancel",
	})

	var content strings.Builder

	if m.currentConfig != nil {
		content.WriteString(fmt.Sprintf("Current: %s\n\n", lipgloss.NewStyle().Faint(true).Render(m.currentConfig.Central.Path)))
	}

	content.WriteString("Clone path:\n")
	content.WriteString(styles.InputStyle.Render(m.textInput.View()))

	return m.layout.Render(content.String())
}

func (m *SettingsModel) viewManualRefresh() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "🔄 Manual Refresh",
		Subtitle: "Pull latest changes from GitHub",
		HelpText: "y to confirm • n to cancel • Esc to go back",
	})

	content := `This will pull the latest changes from your GitHub repository.

Any local changes that have been committed and pushed will be preserved.

Proceed with refresh? (Y/n)`

	return m.layout.Render(content)
}

func (m *SettingsModel) viewRefreshInProgress() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "🔄 Refreshing...",
		Subtitle: "Pulling changes from GitHub",
		HelpText: "Please wait",
	})

	content := lipgloss.NewStyle().Faint(true).Render("Syncing with remote repository...")

	return m.layout.Render(content)
}

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

func (m *SettingsModel) viewComplete() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "✅ Settings Updated",
		Subtitle: "Your changes have been saved",
		HelpText: "Press any key to continue",
	})

	content := "Your settings have been updated successfully!\n\nThe changes will take effect immediately."

	return m.layout.Render(content)
}

func (m *SettingsModel) viewError() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "❌ Error",
		Subtitle: "Failed to update settings",
		HelpText: "Press any key to continue",
	})

	content := "An error occurred while updating your settings.\nPlease check the error message above and try again."

	return m.layout.Render(content)
}

// Helper functions for views

func (m *SettingsModel) formatCurrentConfig() string {
	if m.currentConfig == nil {
		return lipgloss.NewStyle().Faint(true).Render("No configuration found")
	}

	var content strings.Builder
	highlightStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#5fd7ff"))

	// Repository type
	repoType := "Local Directory"
	if m.isGitHubRepo() {
		repoType = "GitHub Repository"
	}
	content.WriteString(fmt.Sprintf("Repository Type: %s\n", highlightStyle.Render(repoType)))

	// Path
	content.WriteString(fmt.Sprintf("Storage Path:    %s\n", highlightStyle.Render(m.currentConfig.Central.Path)))

	// GitHub-specific info
	if m.isGitHubRepo() {
		if m.currentConfig.Central.RemoteURL != nil {
			content.WriteString(fmt.Sprintf("GitHub URL:      %s\n", highlightStyle.Render(*m.currentConfig.Central.RemoteURL)))
		}
		if m.currentConfig.Central.Branch != nil {
			content.WriteString(fmt.Sprintf("Branch:          %s\n", highlightStyle.Render(*m.currentConfig.Central.Branch)))
		}

		// PAT status
		patStatus := "Not set"
		if pat, err := m.credManager.GetGitHubToken(); err == nil && pat != "" {
			patStatus = settingshelpers.FormatPATDisplay(pat)
		}
		content.WriteString(fmt.Sprintf("PAT:             %s\n", highlightStyle.Render(patStatus)))

		// Last sync info
		if m.lastSyncInfo != nil {
			content.WriteString(fmt.Sprintf("\n%s\n", m.formatLastSync()))
		}
	}

	return content.String()
}

func (m *SettingsModel) getMenuOptions() []ChangeOptionInfo {
	options := []ChangeOptionInfo{
		{
			Option:  ChangeOptionRepositoryType,
			Display: "🔄 Change Repository Type",
			Desc:    "Switch between Local and GitHub repository",
		},
	}

	if m.isGitHubRepo() {
		// GitHub repository options
		options = append(options,
			ChangeOptionInfo{
				Option:  ChangeOptionGitHubURL,
				Display: "🌐 Update GitHub URL",
				Desc:    "Change the GitHub repository URL",
			},
			ChangeOptionInfo{
				Option:  ChangeOptionGitHubBranch,
				Display: "🌿 Update GitHub Branch",
				Desc:    "Change the branch to sync with",
			},
			ChangeOptionInfo{
				Option:  ChangeOptionGitHubPath,
				Display: "📂 Update Clone Path",
				Desc:    "Change where the repository is cloned locally",
			},
			ChangeOptionInfo{
				Option:  ChangeOptionGitHubPAT,
				Display: "🔑 Update GitHub PAT",
				Desc:    "Update or remove your Personal Access Token",
			},
			ChangeOptionInfo{
				Option:  ChangeOptionManualRefresh,
				Display: "🔄 Manual Refresh",
				Desc:    "Pull latest changes from GitHub now",
			},
		)
	} else {
		// Local repository options
		options = append(options,
			ChangeOptionInfo{
				Option:  ChangeOptionLocalPath,
				Display: "📁 Update Local Path",
				Desc:    "Change the local directory path",
			},
		)
	}

	// Back option
	options = append(options, ChangeOptionInfo{
		Option:  ChangeOptionBack,
		Display: "← Back to Main Menu",
		Desc:    "",
	})

	return options
}

func (m *SettingsModel) formatLastSync() string {
	if m.lastSyncInfo == nil {
		return lipgloss.NewStyle().Faint(true).Render("No sync information available")
	}

	// TODO: Format actual sync info
	return lipgloss.NewStyle().Faint(true).Render("Last synced: Never")
}

func (m *SettingsModel) formatChangesSummary() string {
	var content strings.Builder
	highlightStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#5fd7ff"))

	content.WriteString("Changes to be saved:\n\n")

	switch m.changeType {
	case ChangeOptionRepositoryType:
		newType := "Local Directory"
		if m.newRepositoryType == "github" {
			newType = "GitHub Repository"
		}
		content.WriteString(fmt.Sprintf("• Repository Type → %s\n", highlightStyle.Render(newType)))

		if m.newRepositoryType == "local" {
			content.WriteString(fmt.Sprintf("• Local Path → %s\n", highlightStyle.Render(m.newStorageDir)))
		} else {
			content.WriteString(fmt.Sprintf("• GitHub URL → %s\n", highlightStyle.Render(m.newGitHubURL)))
			if m.newGitHubBranch != "" {
				content.WriteString(fmt.Sprintf("• Branch → %s\n", highlightStyle.Render(m.newGitHubBranch)))
			}
			content.WriteString(fmt.Sprintf("• Clone Path → %s\n", highlightStyle.Render(m.newGitHubPath)))
			if m.patAction == PATActionUpdate {
				content.WriteString(fmt.Sprintf("• PAT → %s\n", highlightStyle.Render("(will be stored securely)")))
			}
		}

	case ChangeOptionLocalPath:
		content.WriteString(fmt.Sprintf("• Local Path → %s\n", highlightStyle.Render(m.newStorageDir)))

	case ChangeOptionGitHubURL:
		content.WriteString(fmt.Sprintf("• GitHub URL → %s\n", highlightStyle.Render(m.newGitHubURL)))

	case ChangeOptionGitHubBranch:
		branch := m.newGitHubBranch
		if branch == "" {
			branch = "(default)"
		}
		content.WriteString(fmt.Sprintf("• Branch → %s\n", highlightStyle.Render(branch)))

	case ChangeOptionGitHubPath:
		content.WriteString(fmt.Sprintf("• Clone Path → %s\n", highlightStyle.Render(m.newGitHubPath)))

	case ChangeOptionGitHubPAT:
		if m.patAction == PATActionUpdate {
			content.WriteString(fmt.Sprintf("• PAT → %s\n", highlightStyle.Render("(will be stored securely)")))
		} else if m.patAction == PATActionRemove {
			content.WriteString(fmt.Sprintf("• PAT → %s\n", highlightStyle.Render("(will be removed)")))
		}
	}

	return content.String()
}

func (m *SettingsModel) getCleanupWarning() string {
	if !m.isGitHubRepo() {
		return lipgloss.NewStyle().Faint(true).Render("Your current local directory will remain unchanged.")
	}

	return lipgloss.NewStyle().Foreground(lipgloss.Color("#ff8700")).Render(fmt.Sprintf(`⚠️  Your current GitHub clone at:
  %s

will NOT be automatically deleted. You may want to clean it up manually.`, m.currentConfig.Central.Path))
}

// func pluralize(count int, singular, plural string) string {
// 	if count == 1 {
// 		return fmt.Sprintf("%d %s", count, singular)
// 	}
// 	return fmt.Sprintf("%d %s", count, plural)
// }
