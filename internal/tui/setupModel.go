package tui

import (
	"fmt"
	"strings"

	"rulemig/internal/config"
	"rulemig/internal/filemanager"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// SetupModel handles the first-time setup flow
type SetupModel struct {
	state      SetupState
	StorageDir string
	cursor     int
	err        error
	Cancelled  bool
	width      int
	height     int

	// UI styling
	titleStyle      lipgloss.Style
	subtitleStyle   lipgloss.Style
	inputStyle      lipgloss.Style
	errorStyle      lipgloss.Style
	successStyle    lipgloss.Style
	normalTextStyle lipgloss.Style

	// Text input component
	textInput textinput.Model
}

func NewSetupModel() SetupModel {
	ti := textinput.New()
	ti.Placeholder = filemanager.GetDefaultStorageDir()
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 40

	return SetupModel{
		state:      SetupStateWelcome,
		StorageDir: "",
		cursor:     0,
		Cancelled:  false,
		width:      80,
		height:     24,

		titleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1),

		subtitleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginBottom(1),

		inputStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1),

		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),

		successStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true),

		normalTextStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#fff")).
			MarginBottom(1),

		textInput: ti,
	}
}

func (m SetupModel) Init() tea.Cmd {
	return nil
}

func (m SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Update text input width responsively
		m.textInput.Width = m.getInputWidth()
		return m, nil
	case tea.KeyMsg:
		switch m.state {
		case SetupStateWelcome:
			return m.handleWelcomeInput(msg)
		case SetupStateStorageInput:
			// Update text input first
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m.handleStorageInput(msg, cmd)
		case SetupStateConfirmation:
			return m.handleConfirmationInput(msg)
		case SetupStateComplete, SetupStateCancelled:
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m SetupModel) handleWelcomeInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q", "esc":
		m.Cancelled = true
		m.state = SetupStateCancelled
		return m, tea.Quit
	case "enter", " ":
		m.state = SetupStateStorageInput
		// // Set default storage directory
		// m.input = filemanager.GetDefaultStorageDir()
		m.textInput.SetValue(filemanager.GetDefaultStorageDir())
		m.textInput.Focus()
	}
	return m, nil
}

func (m SetupModel) handleStorageInput(msg tea.KeyMsg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		m.Cancelled = true
		m.state = SetupStateCancelled
		return m, tea.Quit
	case "enter":
		input := m.textInput.Value()
		if err := filemanager.ValidateStorageDir(input); err != nil {
			m.err = err
			return m, nil
		}
		m.StorageDir = filemanager.ExpandPath(strings.TrimSpace(input))
		m.state = SetupStateConfirmation
		m.err = nil
	default:
		m.err = nil
	}
	return m, cmd
}

func (m SetupModel) handleConfirmationInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "n", "N":
		m.state = SetupStateStorageInput
		m.err = nil
	case "ctrl+c", "q", "esc":
		m.Cancelled = true
		m.state = SetupStateCancelled
		return m, tea.Quit
	case "y", "Y", "enter":
		// Create the storage directory
		if err := m.createNewConfig(); err != nil {
			m.err = err
			return m, nil
		}
		m.state = SetupStateComplete
	}
	return m, nil
}

/**
 * View renders the current state of the setup process.
 */
func (m SetupModel) View() string {
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

func (m SetupModel) viewWelcome() string {
	title := m.titleStyle.Render("🔧 Welcome to RuleMig!")
	subtitle := m.subtitleStyle.Render("Let's set up your configuration.")

	content := `This is your first time running RuleMig. We need to configure a few settings to get you started.

We'll need to set up:
• Storage directory for your migration rules

Press Enter to continue, or Esc to cancel.`

	return fmt.Sprintf("%s\n%s\n%s", title, subtitle, content)
}

func (m SetupModel) viewStorageInput() string {
	title := m.titleStyle.Render("📁 Storage Directory")
	subtitle := m.subtitleStyle.Render("Where should we store your migration rules?")
	explanation := m.normalTextStyle.Render("This directory will be used as a central location to save & \norganize your migration rules and configurations.\nChoose a path that is accessible and writable.")

	prompt := "Enter storage directory path:"
	input := m.inputStyle.Render(m.textInput.View())

	help := lipgloss.NewStyle().
		Faint(true).
		Render("Press Enter to continue • Esc to cancel • Use ~ for home directory")

	content := fmt.Sprintf("%s\n%s\n%s\n\n%s\n%s\n\n%s",
		title, subtitle, explanation, prompt, input, help)

	if m.err != nil {
		errorMsg := m.errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
		content = fmt.Sprintf("%s\n\n%s", content, errorMsg)
	}

	return content
}

func (m SetupModel) viewConfirmation() string {
	title := m.titleStyle.Render("✅ Confirm Configuration")
	subtitle := m.subtitleStyle.Render("Please review your settings:")

	settings := fmt.Sprintf("Storage Directory: %s", m.StorageDir)

	prompt := "Is this correct? (Y/n)"
	help := lipgloss.NewStyle().
		Faint(true).
		Render("Press y to confirm • n to go back • Esc to cancel")

	content := fmt.Sprintf("%s\n%s\n\n%s\n\n%s\n\n%s",
		title, subtitle, settings, prompt, help)

	if m.err != nil {
		errorMsg := m.errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
		content = fmt.Sprintf("%s\n\n%s", content, errorMsg)
	}

	return content
}

func (m SetupModel) viewComplete() string {
	title := m.successStyle.Render("🎉 Setup Complete!")
	message := "RuleMig has been configured successfully."

	details := fmt.Sprintf("Storage Directory: %s", m.StorageDir)
	next := "You can now start using RuleMig!"

	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s", title, message, details, next)
}

func (m SetupModel) viewCancelled() string {
	return m.errorStyle.Render("Setup cancelled. RuleMig will not be configured.")
}

// Helper methods
func (m SetupModel) createNewConfig() error {
	if err := filemanager.CreateStorageDir(m.StorageDir); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Create config with the selected storage directory
	cfg := config.DefaultConfig()
	if err := cfg.SetStorageDir(m.StorageDir); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	return nil
}

func (m SetupModel) getInputWidth() int {
	// Calculate responsive input width
	maxWidth := m.width - 10 // Leave margin
	if maxWidth < 30 {
		maxWidth = 30 // Minimum width
	}
	if maxWidth > 80 {
		maxWidth = 80 // Maximum width for readability
	}
	return maxWidth
}
