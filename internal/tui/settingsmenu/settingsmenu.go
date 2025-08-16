package settingsmenu

import (
	"fmt"
	"strings"

	"rulem/internal/config"
	"rulem/internal/filemanager"
	"rulem/internal/logging"
	"rulem/internal/tui/components"
	"rulem/internal/tui/helpers"
	"rulem/internal/tui/styles"
	"rulem/pkg/fileops"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// SettingsState represents the current state of the settings flow
type SettingsState int

const (
	SettingsStateStorageInput SettingsState = iota
	SettingsStateConfirmation
	SettingsStateComplete
	SettingsStateError
)

// Custom messages for internal state transitions
type (
	settingsErrorMsg    struct{ err error }
	settingsCompleteMsg struct{}
)

// SettingsModel handles the settings modification flow
type SettingsModel struct {
	state      SettingsState
	StorageDir string
	logger     *logging.AppLogger
	ctx        helpers.UIContext

	// Current configuration
	currentConfig *config.Config

	// Components - centralized layout management
	textInput textinput.Model
	layout    components.LayoutModel

	// State tracking
	hasChanges bool
}

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
		state:     SettingsStateStorageInput,
		textInput: ti,
		layout:    layout,
		logger:    ctx.Logger,
		ctx:       ctx,
	}
}

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
		return m, nil

	case config.LoadConfigMsg:
		if msg.Error != nil {
			m.logger.Error("Failed to load current config", "error", msg.Error)
			return m, func() tea.Msg { return settingsErrorMsg{msg.Error} }
		}
		m.currentConfig = msg.Config
		// Set current storage directory as default
		if m.currentConfig != nil {
			m.textInput.SetValue(m.currentConfig.StorageDir)
			m.textInput.Placeholder = m.currentConfig.StorageDir
		} else {
			// Fallback to system default if no config exists
			defaultDir := filemanager.GetDefaultStorageDir()
			m.textInput.SetValue(defaultDir)
			m.textInput.Placeholder = defaultDir
		}
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

	// Clear error on input change and track changes
	if m.layout.GetError() != nil {
		m.layout = m.layout.ClearError()
	}

	// Check if there are changes from original
	if m.currentConfig != nil {
		m.hasChanges = strings.TrimSpace(m.textInput.Value()) != m.currentConfig.StorageDir
	} else {
		m.hasChanges = strings.TrimSpace(m.textInput.Value()) != filemanager.GetDefaultStorageDir()
	}

	return m, cmd
}

// Key press delegation
func (m *SettingsModel) handleKeyPress(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	// Global navigation commands
	if m.isNavigationKey(msg) {
		return m.handleNavigation(msg)
	}

	switch m.state {
	case SettingsStateStorageInput:
		return m.handleStorageInputKeys(msg)
	case SettingsStateConfirmation:
		return m.handleConfirmationKeys(msg)
	case SettingsStateComplete, SettingsStateError:
		return m.handleCompleteKeys(msg)
	default:
		return m, nil
	}
}

// State-specific key handlers
func (m *SettingsModel) handleStorageInputKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.logger.LogUserAction("settings_storage_input_submit", m.textInput.Value())
		return m.validateAndProceed()
	case "esc":
		if !m.hasChanges {
			return m, func() tea.Msg { return helpers.NavigateToMainMenuMsg{} }
		}
		// If there are changes, go to confirmation to ask about discarding
		m.logger.LogUserAction("settings_discard_attempt", "has changes")
		m.state = SettingsStateConfirmation
		return m, nil
	default:
		return m.updateTextInput(msg)
	}
}

func (m *SettingsModel) handleConfirmationKeys(msg tea.KeyMsg) (*SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "n", "N":
		m.logger.LogUserAction("settings_confirmation_reject", "going back to input")
		m.state = SettingsStateStorageInput
		return m, nil
	case "y", "Y", "enter":
		m.logger.LogUserAction("settings_confirmation_accept", "saving config")
		return m, m.saveConfig()
	case "esc":
		if m.hasChanges {
			// Go back to input if there are unsaved changes
			m.state = SettingsStateStorageInput
			return m, nil
		}
		return m, func() tea.Msg { return helpers.NavigateToMainMenuMsg{} }
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
func (m *SettingsModel) validateAndProceed() (*SettingsModel, tea.Cmd) {
	input := strings.TrimSpace(m.textInput.Value())
	m.logger.Debug("Validating storage directory", "path", input)

	if err := fileops.ValidateStoragePath(input); err != nil {
		m.logger.Warn("Storage directory validation failed", "error", err)
		return m, func() tea.Msg { return settingsErrorMsg{err} }
	}

	m.StorageDir = fileops.ExpandPath(input)

	// If no changes, go directly back to menu
	if !m.hasChanges {
		m.logger.Debug("No changes detected, returning to menu")
		return m, func() tea.Msg { return helpers.NavigateToMainMenuMsg{} }
	}

	m.logger.LogStateTransition("SettingsModel", "SettingsStateStorageInput", "SettingsStateConfirmation")
	m.state = SettingsStateConfirmation
	m.layout = m.layout.ClearError()
	return m, nil
}

func (m *SettingsModel) saveConfig() tea.Cmd {
	return func() tea.Msg {
		m.logger.Info("Updating configuration", "storage_dir", m.StorageDir)
		if err := m.performConfigUpdate(); err != nil {
			m.logger.Error("Configuration update failed", "error", err)
			return settingsErrorMsg{err}
		}
		m.logger.Info("Configuration updated successfully")
		return settingsCompleteMsg{}
	}
}

func (m *SettingsModel) performConfigUpdate() error {
	// Update existing config or create new one
	if m.currentConfig != nil {
		// Use UpdateStorageDir to ensure directory is created
		if err := config.UpdateStorageDir(m.currentConfig, m.StorageDir); err != nil {
			return fmt.Errorf("failed to update configuration: %w", err)
		}
	} else {
		if err := config.CreateNewConfig(m.StorageDir); err != nil {
			return fmt.Errorf("failed to create new configuration: %w", err)
		}
	}
	return nil
}

// View renders the current state using the centralized layout
func (m *SettingsModel) View() string {
	switch m.state {
	case SettingsStateStorageInput:
		return m.viewStorageInput()
	case SettingsStateConfirmation:
		return m.viewConfirmation()
	case SettingsStateComplete:
		return m.viewComplete()
	case SettingsStateError:
		return m.viewError()
	}
	return ""
}

// View methods using the centralized layout
func (m *SettingsModel) viewStorageInput() string {
	title := "⚙️ Update Settings"
	subtitle := "Modify your Rulem configuration"
	helpText := "Press Enter to save • Esc to cancel"

	if m.hasChanges {
		helpText = "Press Enter to save changes • Esc to cancel"
	}

	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    title,
		Subtitle: subtitle,
		HelpText: helpText,
	})

	explanation := `Update your Rulem configuration settings. Changes will be saved to your configuration file.`

	currentValue := ""
	if m.currentConfig != nil {
		currentValue = fmt.Sprintf("Current: %s", m.currentConfig.StorageDir)
	}

	prompt := "Storage directory path:"
	input := styles.InputStyle.Render(m.textInput.View())

	content := fmt.Sprintf("%s\n\n%s\n\n%s\n%s", explanation, currentValue, prompt, input)

	return m.layout.Render(content)
}

func (m *SettingsModel) viewConfirmation() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "💾 Save Changes",
		Subtitle: "Confirm your configuration changes:",
		HelpText: "Press y to save • n to go back • Esc to cancel",
	})

	oldValue := "Not set"
	if m.currentConfig != nil {
		oldValue = m.currentConfig.StorageDir
	}

	changes := fmt.Sprintf("Old storage directory: %s\nNew storage directory: %s", oldValue, m.StorageDir)
	prompt := "Save these changes? (Y/n)"
	content := fmt.Sprintf("%s\n\n%s", changes, prompt)

	return m.layout.Render(content)
}

func (m *SettingsModel) viewComplete() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "✅ Settings Updated!",
		Subtitle: "Your configuration has been saved successfully.",
		HelpText: "Press any key to continue",
	})

	content := fmt.Sprintf(`Configuration updated successfully!

Storage Directory: %s

Your changes have been saved and will take effect immediately.`, m.StorageDir)

	return m.layout.Render(content)
}

func (m *SettingsModel) viewError() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "❌ Settings Error",
		Subtitle: "Failed to update configuration",
		HelpText: "Press Enter to try again • Esc to cancel",
	})

	content := `An error occurred while updating your settings. Please check the error message above and try again.`

	return m.layout.Render(content)
}

// TODO when the settings menu updates the config, we need to refresh it in the main model
