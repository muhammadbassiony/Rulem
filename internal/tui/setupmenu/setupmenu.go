package setupmenu

import (
	"fmt"
	"strings"

	"rulemig/internal/config"
	"rulemig/internal/filemanager"
	"rulemig/internal/tui/components"
	"rulemig/internal/tui/styles"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// SetupState represents the current state of the setup process
type SetupState int

const (
	SetupStateWelcome SetupState = iota
	SetupStateStorageInput
	SetupStateConfirmation
	SetupStateComplete
	SetupStateCancelled
)

// Custom messages for internal state transitions
type (
	setupErrorMsg    struct{ err error }
	setupCompleteMsg struct{}
)

// SetupModel handles the first-time setup flow
type SetupModel struct {
	state      SetupState
	StorageDir string
	Cancelled  bool

	// Components - centralized layout management
	textInput textinput.Model
	layout    components.LayoutModel
}

func NewSetupModel() SetupModel {
	ti := textinput.New()
	ti.Placeholder = filemanager.GetDefaultStorageDir()
	ti.Focus()
	ti.CharLimit = 256

	// Create centralized layout - will be reconfigured per state
	layout := components.NewLayout(components.LayoutConfig{
		MarginX:  2,
		MarginY:  1,
		MaxWidth: 100,
	})

	return SetupModel{
		state:     SetupStateWelcome,
		textInput: ti,
		layout:    layout,
	}
}

func (m SetupModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles all message types and delegates to state-specific handlers
func (m SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// TODO move to switch case?
	// Update layout with window size changes
	m.layout, _ = m.layout.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Update text input width responsively
		m.textInput.Width = m.layout.InputWidth()
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case setupErrorMsg:
		m.layout = m.layout.SetError(msg.err)
		return m, nil

	case setupCompleteMsg:
		m.state = SetupStateComplete
		m.layout = m.layout.ClearError()
		return m, nil
	}

	return m, cmd
}

// Text input updates
func (m SetupModel) updateTextInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	// Clear error on input change
	if m.layout.GetError() != nil {
		m.layout = m.layout.ClearError()
	}
	return m, cmd
}

// Key press delegation
func (m SetupModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global quit commands
	if m.isQuitKey(msg) {
		return m.handleQuit()
	}

	switch m.state {
	case SetupStateWelcome:
		return m.handleWelcomeKeys(msg)
	case SetupStateStorageInput:
		return m.handleStorageInputKeys(msg)
	case SetupStateConfirmation:
		return m.handleConfirmationKeys(msg)
	default:
		return m, tea.Quit
	}
}

// State-specific key handlers
func (m SetupModel) handleWelcomeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", " ":
		return m.transitionToStorageInput()
	}
	return m, nil
}

func (m SetupModel) handleStorageInputKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		return m.validateAndProceed()
	default:
		return m.updateTextInput(msg)
	}
}

func (m SetupModel) handleConfirmationKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "n", "N":
		return m.transitionToStorageInput()
	case "y", "Y", "enter":
		return m, m.createConfig()
	}
	return m, nil
}

// State transition helpers
func (m SetupModel) transitionToStorageInput() (tea.Model, tea.Cmd) {
	m.state = SetupStateStorageInput
	m.textInput.SetValue(filemanager.GetDefaultStorageDir())
	m.textInput.Focus()
	m.layout = m.layout.ClearError()
	return m, textinput.Blink
}

func (m SetupModel) validateAndProceed() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.textInput.Value())
	if err := filemanager.ValidateStorageDir(input); err != nil {
		return m, func() tea.Msg { return setupErrorMsg{err} }
	}

	m.StorageDir = filemanager.ExpandPath(input)
	m.state = SetupStateConfirmation
	m.layout = m.layout.ClearError()
	return m, nil
}

func (m SetupModel) createConfig() tea.Cmd {
	return func() tea.Msg {
		if err := m.performConfigCreation(); err != nil {
			return setupErrorMsg{err}
		}
		return setupCompleteMsg{}
	}
}

func (m SetupModel) handleQuit() (tea.Model, tea.Cmd) {
	m.Cancelled = true
	m.state = SetupStateCancelled
	return m, tea.Quit
}

// Utility functions
func (m SetupModel) isQuitKey(msg tea.KeyMsg) bool {
	return msg.String() == "ctrl+c" || msg.String() == "q" || msg.String() == "esc"
}

// TODO implement in config pkg
func (m SetupModel) performConfigCreation() error {

	if err := config.CreateNewConfig(m.StorageDir); err != nil {
		return fmt.Errorf("failed to create configuration: %w", err)
	}
	return nil
}

// View renders the current state using the centralized layout
func (m SetupModel) View() string {
	// Ensure layout is configured for current state

	switch m.state {
	case SetupStateWelcome:
		return m.viewWelcome()
	case SetupStateStorageInput:
		return m.viewStorageInput()
	case SetupStateConfirmation:
		return m.viewConfirmation()
	case SetupStateComplete:
		return m.viewComplete()
	case SetupStateCancelled:
		return m.viewCancelled()
	}
	return ""
}

// View methods now use the centralized layout
func (m SetupModel) viewWelcome() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "🔧 Welcome to RuleMig!",
		Subtitle: "Let's set up your configuration.",
		HelpText: "Press Enter to continue, or Esc to cancel",
	})

	content := `This is your first time running RuleMig. We need to configure a few settings to get you started.

We'll need to set up:
• Storage directory for your migration rules

This directory will be used as a central location to save and organize your migration rules and configurations.`

	return m.layout.Render(content)
}

func (m SetupModel) viewStorageInput() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "📁 Storage Directory",
		Subtitle: "Where should we store your migration rules?",
		HelpText: "Press Enter to continue • Esc to cancel • Use ~ for home directory",
	})

	explanation := `This directory will be used as a central location to save and organize your migration rules and configurations. Choose a path that is accessible and writable.`

	prompt := "Enter storage directory path:"
	input := styles.InputStyle.Render(m.textInput.View())

	content := fmt.Sprintf("%s\n\n%s\n%s", explanation, prompt, input)

	return m.layout.Render(content)
}

func (m SetupModel) viewConfirmation() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "✅ Confirm Configuration",
		Subtitle: "Please review your settings:",
		HelpText: "Press y to confirm • n to go back • Esc to cancel",
	})

	settings := fmt.Sprintf("Storage Directory: %s", m.StorageDir)
	prompt := "Is this correct? (Y/n)"
	content := fmt.Sprintf("%s\n\n%s", settings, prompt)

	return m.layout.Render(content)
}

func (m SetupModel) viewComplete() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "🎉 Setup Complete!",
		Subtitle: "RuleMig has been configured successfully.",
		HelpText: "", // No help text needed for final state
	})

	content := fmt.Sprintf(`Configuration saved successfully!

Storage Directory: %s

You can now start using RuleMig to manage your migration rules. The application will store all your rules and configurations in the directory you specified.`, m.StorageDir)

	return m.layout.Render(content)
}

func (m SetupModel) viewCancelled() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "❌ Setup Cancelled",
		Subtitle: "RuleMig will not be configured.",
		HelpText: "",
	})

	content := `Setup was cancelled. RuleMig has not been configured and will need to be set up before you can use it.`

	return m.layout.Render(content)
}
