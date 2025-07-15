package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type MenuModel struct {
	choices  []string
	cursor   int
	selected int
}

func NewMenuModel() MenuModel {
	return MenuModel{
		choices: []string{
			"🚀 Start Migration",
			"⚙️  Settings",
			"❓ Help",
			"🚪 Exit",
		},
		cursor: 0,
	}
}

func (m MenuModel) Init() tea.Cmd {
	return nil
}

func (m MenuModel) Update(msg tea.Msg) (MenuModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter", " ":
			m.selected = m.cursor
			return m, m.handleSelection()
		}
	}
	return m, nil
}

func (m MenuModel) handleSelection() tea.Cmd {
	switch m.selected {
	case 0: // Start Migration
		return func() tea.Msg {
			return StateChangeMsg{NewState: StateMigration}
		}
	case 1: // Settings
		return func() tea.Msg {
			return StateChangeMsg{NewState: StateSettings}
		}
	case 2: // Help
		return func() tea.Msg {
			return StateChangeMsg{NewState: StateHelp}
		}
	case 3: // Exit
		return tea.Quit
	}
	return nil
}

func (m MenuModel) View() string {
	s := "What would you like to do?\n\n"

	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = "▶"
			choice = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205")).
				Render(choice)
		}
		s += fmt.Sprintf("%s %s\n", cursor, choice)
	}

	s += "\nUse ↑/↓ or j/k to navigate, Enter to select"
	return s
}
