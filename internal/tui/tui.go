package tui

import (
	"fmt"

	"rulemig/internal/config"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AppState represents the current state of the application
type AppState int

const (
	StateMenu AppState = iota
	StateSettings
	StateMigration
	StateHelp
	StateQuitting
)

// MainModel is the root model for the application
type MainModel struct {
	config *config.Config
	state  AppState

	// Sub-models for different screens
	menuModel MenuModel
	// settingsModel SettingsModel
	// migrationModel MigrationModel
	// helpModel     HelpModel

	// UI state
	width  int
	height int
	err    error
}

func NewMainModel(cfg *config.Config) MainModel {
	return MainModel{
		config:    cfg,
		state:     StateMenu,
		menuModel: NewMenuModel(),
		// settingsModel:  NewSettingsModel(cfg),
		// migrationModel: NewMigrationModel(cfg),
		// helpModel:      NewHelpModel(),
	}
}

func (m MainModel) Init() tea.Cmd {
	switch m.state {
	case StateMenu:
		return m.menuModel.Init()
	// case StateSettings:
	// 	return m.settingsModel.Init()
	// case StateMigration:
	// 	return m.migrationModel.Init()
	// case StateHelp:
	// 	return m.helpModel.Init()
	default:
		return nil
	}
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.state == StateMenu {
				m.state = StateQuitting
				return m, tea.Quit
			}
			// Return to menu from sub-screens
			m.state = StateMenu
			return m, nil
		}

	case StateChangeMsg:
		m.state = msg.NewState
		return m, m.Init()

	case ConfigUpdateMsg:
		m.config = msg.Config
		// m.settingsModel.UpdateConfig(msg.Config)
		// m.migrationModel.UpdateConfig(msg.Config)
	}

	// Delegate to current state's model
	switch m.state {
	case StateMenu:
		m.menuModel, cmd = m.menuModel.Update(msg)
		// case StateSettings:
		// 	m.settingsModel, cmd = m.settingsModel.Update(msg)
		// case StateMigration:
		// 	m.migrationModel, cmd = m.migrationModel.Update(msg)
		// case StateHelp:
		// 	m.helpModel, cmd = m.helpModel.Update(msg)
	}

	return m, cmd
}

func (m MainModel) View() string {
	if m.state == StateQuitting {
		return "Goodbye! 👋\n"
	}

	// Header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Render("🔧 RuleMig - Rule Migration Tool")

	// Content based on current state
	var content string
	switch m.state {
	case StateMenu:
		content = m.menuModel.View()
		// case StateSettings:
		//     content = m.settingsModel.View()
		// case StateMigration:
		//     content = m.migrationModel.View()
		// case StateHelp:
		//     content = m.helpModel.View()
	}

	// Footer
	footer := lipgloss.NewStyle().
		Faint(true).
		Render("Press 'q' to quit • 'ctrl+c' to exit")

	return fmt.Sprintf("%s\n\n%s\n\n%s", header, content, footer)
}

// Messages for state management
type StateChangeMsg struct {
	NewState AppState
}

type ConfigUpdateMsg struct {
	Config *config.Config
}
