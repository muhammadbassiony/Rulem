package tui

import (
	"rulem/internal/config"
	"rulem/internal/logging"
	"rulem/internal/tui/components"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// AppState represents the current state of the application
type AppState int

const (
	StateMenu AppState = iota
	StateError
	StateComingSoon
	StateQuitting

	StateSettings
	StateMigration
	StateSaveRules
	StateImportCopy
	StateImportLink
	StateFetchGithub
)

// Custom messages for internal state transitions
type (
	NavigateMsg struct {
		State AppState
	}

	ErrorMsg struct {
		Err error
	}

	ComingSoonMsg struct {
		Feature string
	}
)

// MenuItemModel interface for menu item models
type MenuItemModel interface {
	tea.Model
	GetTitle() string
	GetDescription() string
}

// Enhanced item struct with model reference
type item struct {
	title       string
	description string
	state       AppState
	model       MenuItemModel // Reference to the model for this menu item
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.description }
func (i item) FilterValue() string { return i.title }

// MainModel is the root model for the application
type MainModel struct {
	config    *config.Config
	logger    *logging.AppLogger
	state     AppState
	prevState AppState // For returning from error states

	// Main menu list
	menu list.Model

	// Sub-models for different screens - initialized lazily
	settingsModel    MenuItemModel
	migrationModel   MenuItemModel
	saveRulesModel   MenuItemModel
	importCopyModel  MenuItemModel
	importLinkModel  MenuItemModel
	fetchGithubModel MenuItemModel
	helpModel        MenuItemModel

	// Current active model
	activeModel MenuItemModel

	// Layout for consistent UI
	layout components.LayoutModel

	// UI state
	err               error
	loading           bool
	loadingMessage    string
	comingSoonFeature string
}

func NewMainModel(cfg *config.Config, logger *logging.AppLogger) MainModel {
	// Create menu items with model references
	items := []list.Item{
		item{
			title:       "💾  Save rules file",
			description: "Save a rules file from current directory to the central rules repository",
			state:       StateSaveRules,
			model:       nil, // Will be initialized when needed
		},
		item{
			title:       "📄  Import rules (Copy)",
			description: "Import a rule file from the central rules repository, to the current directory.\nThis will copy the file to the current directory. \nAny changes made to the file in the current directory will NOT be reflected in the central rules repository.",
			state:       StateImportCopy,
			model:       nil, // Will be initialized when needed
		},
		item{
			title:       "🔗  Import rules (Link)",
			description: "Import a rule file from the central rules repository, to the current directory.\nThis will create a symlink to the file in the current directory. \nAny changes made to the file in the current directory WILL be reflected in the central rules repository, and vice versa.",
			state:       StateImportLink,
			model:       nil, // Will be initialized when needed
		},
		item{
			title:       "⬇️  Fetch rules from Github",
			description: "Download a new rules file from a Github repository or gist.\nThis will fetch the file and save it to the central rules repository.",
			state:       StateFetchGithub,
			model:       nil, // Will be initialized when needed
		},
		item{
			title:       "⚙️  Update settings",
			description: "Change the configuration of Rulem, such as storage directory.",
			state:       StateSettings,
			model:       nil, // Will be initialized when needed
		},
	}

	// Create list with items
	menuList := list.New(items, list.NewDefaultDelegate(), 0, 0)
	menuList.Title = "" // We'll use the layout for titles
	menuList.SetShowTitle(false)
	menuList.SetShowStatusBar(false)
	menuList.SetFilteringEnabled(true)
	menuList.SetShowHelp(false) // We'll use the layout for help

	// Create layout
	layout := components.NewLayout(components.LayoutConfig{
		MarginX:  2,
		MarginY:  1,
		MaxWidth: 100,
	})

	return MainModel{
		config:    cfg,
		logger:    logger,
		state:     StateMenu,
		prevState: StateMenu,
		menu:      menuList,
		layout:    layout,
	}
}

func (m MainModel) Init() tea.Cmd {
	m.logger.Info("MainModel initialized")
	return nil
}
func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Log all messages for debugging
	m.logger.LogMessage(msg)

	// Update layout first for size changes
	m.layout, _ = m.layout.Update(msg)

	// Single comprehensive switch statement handling all message types and states
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Handle window resize
		v := 14 // footer margins
		m.menu.SetSize(msg.Width-4, msg.Height-v)

		// Propagate size to active model if present
		if m.activeModel != nil {
			if updatedModel, modelCmd := m.activeModel.Update(msg); modelCmd != nil {
				m.activeModel = updatedModel.(MenuItemModel)
				cmds = append(cmds, modelCmd)
			}
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			// Handle global quit commands
			m.state = StateQuitting
			return m, tea.Quit
		}

		// Handle keyboard input based on current state
		switch m.state {
		case StateMenu:
			switch msg.String() {
			case "enter":
				// Handle menu selection
				if selectedItem, ok := m.menu.SelectedItem().(item); ok {
					m.logger.LogUserAction("menu_selection", selectedItem.title)
					return m.handleMenuSelection(selectedItem)
				}
			default:
				// Update the menu list for navigation/filtering
				m.menu, cmd = m.menu.Update(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}

		case StateComingSoon:
			switch msg.String() {
			case "esc":
				m.logger.LogStateTransition("MainModel", "StateComingSoon", "StateMenu")
				m.state = StateMenu
				m.comingSoonFeature = ""
				m.layout = m.layout.ClearError()
				return m, nil
			}

		case StateError:
			switch msg.String() {
			case "esc":
				m.logger.LogStateTransition("MainModel", "StateError", string(rune(m.prevState)))
				m.state = m.prevState
				m.err = nil
				m.layout = m.layout.ClearError()
				return m, nil
			}

		case StateSettings, StateSaveRules, StateImportCopy, StateImportLink, StateFetchGithub:
			switch msg.String() {
			case "esc":
				// Return to menu from feature states
				m.logger.LogStateTransition("MainModel", "FeatureState", "StateMenu")
				m.state = StateMenu
				m.activeModel = nil
				m.err = nil
				m.comingSoonFeature = ""
				m.layout = m.layout.ClearError()
				return m, nil
			default:
				// Delegate to active model
				if m.activeModel != nil {
					if updatedModel, modelCmd := m.activeModel.Update(msg); modelCmd != nil {
						m.activeModel = updatedModel.(MenuItemModel)
						cmds = append(cmds, modelCmd)
					}
				}
			}
		}

	case NavigateMsg:
		// Handle navigation between states
		m.logger.LogStateTransition("MainModel", string(rune(m.state)), string(rune(msg.State)))
		m.prevState = m.state
		m.state = msg.State
		m.err = nil
		m.loading = false
		m.comingSoonFeature = ""
		m.layout = m.layout.ClearError()
		return m, nil

	case ErrorMsg:
		// Handle error display
		m.logger.Error("Application error occurred", "error", msg.Err)
		m.err = msg.Err
		m.prevState = m.state
		m.state = StateError
		m.loading = false
		m.layout = m.layout.SetError(msg.Err)
		return m, nil

	case ComingSoonMsg:
		// Handle coming soon display
		m.logger.Debug("Showing coming soon for feature", "feature", msg.Feature)
		m.comingSoonFeature = msg.Feature
		m.state = StateComingSoon
		return m, nil

	default:
		// Handle any unrecognized message types
		// Delegate to active model if present
		if m.activeModel != nil {
			if updatedModel, modelCmd := m.activeModel.Update(msg); modelCmd != nil {
				m.activeModel = updatedModel.(MenuItemModel)
				cmds = append(cmds, modelCmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// handleNavigation processes navigation messages
func (m MainModel) handleNavigation(msg NavigateMsg) (tea.Model, tea.Cmd) {
	m.prevState = m.state
	m.state = msg.State
	m.err = nil
	m.loading = false
	m.comingSoonFeature = ""
	m.layout = m.layout.ClearError()

	return m, nil
}

// handleMenuSelection processes menu item selections using model-based approach
func (m MainModel) handleMenuSelection(selectedItem item) (tea.Model, tea.Cmd) {
	// Get or initialize the model for this menu item
	model := m.getOrInitializeModel(selectedItem.state)

	if model == nil {
		// Show coming soon message if model is not implemented
		return m, ShowComingSoon(selectedItem.title)
	}

	// Set the active model and navigate to its state
	m.activeModel = model
	return m, NavigateTo(selectedItem.state)
}

// getOrInitializeModel returns the model for a given state, initializing it if needed
func (m *MainModel) getOrInitializeModel(state AppState) MenuItemModel {
	switch state {
	case StateSettings:
		if m.settingsModel == nil {
			// TODO: Initialize settings model when implemented
			// m.settingsModel = NewSettingsModel(m.config)
		}
		return m.settingsModel

	case StateSaveRules:
		if m.saveRulesModel == nil {
			// TODO: Initialize save rules model when implemented
			// m.saveRulesModel = NewSaveRulesModel(m.config)
		}
		return m.saveRulesModel

	case StateImportCopy:
		if m.importCopyModel == nil {
			// TODO: Initialize import copy model when implemented
			// m.importCopyModel = NewImportCopyModel(m.config)
		}
		return m.importCopyModel

	case StateImportLink:
		if m.importLinkModel == nil {
			// TODO: Initialize import link model when implemented
			// m.importLinkModel = NewImportLinkModel(m.config)
		}
		return m.importLinkModel

	case StateFetchGithub:
		if m.fetchGithubModel == nil {
			// TODO: Initialize fetch github model when implemented
			// m.fetchGithubModel = NewFetchGithubModel(m.config)
		}
		return m.fetchGithubModel

	default:
		return nil
	}
}

func (m MainModel) View() string {
	if m.state == StateQuitting {
		m.layout = m.layout.SetConfig(components.LayoutConfig{
			Title: "👋 Goodbye!",
		})
		return m.layout.Render("Thank you for using Rulem!")
	}

	// Configure layout based on current state
	switch m.state {
	case StateMenu:
		return m.viewMenu()
	case StateError:
		return m.viewError()
	case StateComingSoon:
		return m.viewComingSoon()
	default:
		// Use active model's view if available
		if m.activeModel != nil {
			return m.activeModel.View()
		}
		// Show coming soon for unimplemented states
		return m.viewComingSoon()
	}
}

func (m MainModel) viewMenu() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "🔧 Rulem - Rule Migration Tool",
		Subtitle: "Manage and organize your migration rules efficiently",
		HelpText: "↑/↓ to navigate • Enter to select • / to filter • q to quit",
	})

	// Get the menu content
	menuContent := m.menu.View()

	return m.layout.Render(menuContent)
}

func (m MainModel) viewError() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "❌ Error",
		Subtitle: "Something went wrong",
		HelpText: "Press Esc to return • q to quit",
	})

	errorContent := ""
	if m.err != nil {
		errorContent = m.err.Error()
	}

	return m.layout.Render(errorContent)
}

func (m MainModel) viewComingSoon() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "🚧 Coming Soon",
		Subtitle: "This feature is under development",
		HelpText: "Press Esc to return to menu • q to quit",
	})

	content := "This feature is not yet implemented but will be available in a future version."
	if m.comingSoonFeature != "" {
		content = "The feature '" + m.comingSoonFeature + "' is not yet implemented but will be available in a future version."
	}

	return m.layout.Render(content)
}

// Helper functions for creating navigation commands
func NavigateTo(state AppState) tea.Cmd {
	return func() tea.Msg {
		return NavigateMsg{State: state}
	}
}

// func ShowError(err error) tea.Cmd {
// 	return func() tea.Msg {
// 		return ErrorMsg{Err: err}
// 	}
// }

func ShowComingSoon(feature string) tea.Cmd {
	return func() tea.Msg {
		return ComingSoonMsg{Feature: feature}
	}
}
