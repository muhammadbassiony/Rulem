// Package tui provides the Terminal User Interface for the rulem application.
//
// This package implements a modern TUI using the Bubble Tea framework and Lipgloss styling.
// It provides an interactive menu-driven interface for managing AI assistant rule files,
// including features like:
//
// - Main navigation menu with filtering capabilities
// - Save rules functionality for storing rule files in a central repository
// - Import rules functionality for copying/linking rules to current directory
// - Settings management for configuring storage locations
// - GitHub integration for fetching rules from remote repositories
// - Error handling and user feedback through consistent UI patterns
//
// The TUI follows a state-based architecture where different application states
// (menu, settings, save rules, etc.) are handled by specialized models that
// implement the tea.Model interface. State transitions are managed through
// custom message types and a centralized navigation system.
//
// Key Components:
//   - MainModel: Root model that orchestrates the entire TUI application
//   - AppState: Enumeration of possible application states
//   - MenuItemModel: Interface for models that can be displayed in menus
//   - Navigation system: Message-based state transitions between different views
//
// The package is designed to be extensible, allowing new features to be added
// as separate models that integrate with the main navigation system.
package tui

import (
	"rulem/internal/config"
	"rulem/internal/logging"
	"rulem/internal/tui/components"
	"rulem/internal/tui/helpers"
	"rulem/internal/tui/importrulesmenu"
	saverulesmodel "rulem/internal/tui/saverulesmodel"
	settingsmenu "rulem/internal/tui/settingsmenu"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// AppState represents the current state of the TUI application.
// Each state corresponds to a different view or mode of operation.
type AppState int

const (
	// StateMenu represents the main navigation menu
	StateMenu AppState = iota
	StateError
	StateComingSoon
	StateQuitting

	StateSettings
	StateSaveRules
	StateImportCopy
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

// MenuItemModel interface for menu item models.
// Any model that can be displayed as a menu item must implement this interface,
// which requires implementing the tea.Model interface for Bubble Tea compatibility.
type MenuItemModel interface {
	tea.Model
}

// Enhanced item struct with model reference
type item struct {
	title       string
	description string
	state       AppState
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.description }
func (i item) FilterValue() string { return i.title }

// MainModel is the root model for the TUI application.
//
// This struct serves as the central coordinator for the entire application,
// managing state transitions, user input, and rendering. It implements the
// tea.Model interface and handles the main application loop.
//
// Key responsibilities:
//   - Managing application state and state transitions
//   - Coordinating between different feature-specific models
//   - Handling user input and delegating to appropriate handlers
//   - Managing window resizing and layout updates
//   - Providing consistent error handling and user feedback
//   - Orchestrating the main navigation menu
//
// The model uses a state-based architecture where different application states
// correspond to different views and behaviors. It maintains references to
// configuration, logging, and UI components to provide a cohesive user experience.
type MainModel struct {
	config    *config.Config
	logger    *logging.AppLogger
	state     AppState
	prevState AppState // For returning from error states

	// Main menu list
	menu list.Model

	// Current active model (always fresh, no caching)
	activeModel MenuItemModel

	// Layout for consistent UI
	layout components.LayoutModel

	// Window dimensions for creating submodels
	windowWidth  int
	windowHeight int

	// UI state
	err               error
	loading           bool
	loadingMessage    string
	comingSoonFeature string
}

func NewMainModel(cfg *config.Config, logger *logging.AppLogger) *MainModel {
	// Create menu items with model references
	items := []list.Item{
		item{
			title:       "💾  Save rules file",
			description: "Save a rules file from current directory to the central rules repository",
			state:       StateSaveRules,
		},
		item{
			title:       "📄  Import rules (Copy)",
			description: "Import a rule file from the central rules repository, to the current directory.\nYou will have the option to either copy or link the rules file. \nYou can also select your AI assistant or IDE or CLI coding tool so we can customize the file for you.",
			state:       StateImportCopy,
		},
		item{
			title:       "⬇️  Fetch rules from Github",
			description: "Download a new rules file from a Github repository or gist.\nThis will fetch the file and save it to the central rules repository.",
			state:       StateFetchGithub,
		},
		item{
			title:       "⚙️  Update settings",
			description: "Modify your Rulem configuration settings, such as storage directory.",
			state:       StateSettings,
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

	return &MainModel{
		config:    cfg,
		logger:    logger,
		state:     StateMenu,
		prevState: StateMenu,
		menu:      menuList,
		layout:    layout,
	}
}

func (m *MainModel) Init() tea.Cmd {
	m.logger.Info("MainModel initialized")
	return nil
}

func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// log only strategic events
	if wm, ok := msg.(tea.WindowSizeMsg); ok {
		m.logger.Debug("window resize", "width", wm.Width, "height", wm.Height)
	}

	// Update layout first for size changes
	m.layout, _ = m.layout.Update(msg)

	// Single comprehensive switch statement handling all message types and states
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Store window dimensions for creating submodels
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height

		// Handle window resize with validation
		if msg.Width > 0 && msg.Height > 0 {
			v := 14 // footer margins
			m.menu.SetSize(msg.Width-4, msg.Height-v)

			// Propagate size to active model if present
			if m.activeModel != nil {
				updatedModel, modelCmd := m.activeModel.Update(msg)
				m.activeModel = updatedModel.(MenuItemModel)
				if modelCmd != nil {
					cmds = append(cmds, modelCmd)
				}
			}
		} else {
			m.logger.Warn("Invalid window dimensions received", "width", msg.Width, "height", msg.Height)
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:

		if msg.String() == "ctrl+c" {
			// Handle global quit commands
			m.state = StateQuitting
			return m, tea.Quit
		}

		// Handle keyboard input based on current state
		switch m.state {
		case StateMenu:
			switch msg.String() {
			case "q":
				// Handle quit only when not filtering
				if m.menu.FilterState() != list.Filtering {
					m.state = StateQuitting
					return m, tea.Quit
				}
				// When filtering, pass "q" to the menu for filtering
				m.menu, cmd = m.menu.Update(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			case "enter":
				// Handle menu selection only when not filtering
				if m.menu.FilterState() != list.Filtering {
					if selectedItem, ok := m.menu.SelectedItem().(item); ok {
						m.logger.LogUserAction("menu_selection", selectedItem.title)
						return m.handleMenuSelection(selectedItem)
					}
				}
				// When filtering, pass enter to the menu
				m.menu, cmd = m.menu.Update(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
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

		case StateSettings, StateSaveRules, StateImportCopy, StateFetchGithub:
			// Delegate all messages to active model - they handle their own navigation
			if m.activeModel != nil {
				updatedModel, modelCmd := m.activeModel.Update(msg)
				m.activeModel = updatedModel.(MenuItemModel)
				if modelCmd != nil {
					cmds = append(cmds, modelCmd)
				}
			}
		}

	case list.FilterMatchesMsg:
		// update the menu with filter matches for menu state only
		switch m.state {
		case StateMenu:
			m.menu, cmd = m.menu.Update(msg)
			if cmd != nil {
				return m, nil
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

	case helpers.NavigateToMainMenuMsg:
		// Handle navigation back to main menu from any submodel
		m.logger.LogStateTransition("MainModel", "FeatureState", "StateMenu")
		return m.returnToMenu(), nil

	case config.ReloadConfigMsg:
		// Handle config reload after settings updates
		if msg.Error != nil {
			m.logger.Error("Failed to reload configuration", "error", msg.Error)
			return m, func() tea.Msg { return ErrorMsg{Err: msg.Error} }
		}
		if msg.Config != nil {
			m.logger.Info("Configuration reloaded successfully")
			m.config = msg.Config
		}
		return m, nil

	default:
		// Handle any unrecognized message types
		// Delegate to active model if present
		if m.activeModel != nil {
			updatedModel, modelCmd := m.activeModel.Update(msg)
			if menuModel, ok := updatedModel.(MenuItemModel); ok {
				m.activeModel = menuModel
				if modelCmd != nil {
					cmds = append(cmds, modelCmd)
				}
			} else {
				m.logger.Error("Active model returned invalid type, returning to menu")
				return m.returnToMenu(), nil
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// handleMenuSelection processes menu item selections using model-based approach
func (m *MainModel) handleMenuSelection(selectedItem item) (tea.Model, tea.Cmd) {
	// Get or initialize the model for this menu item
	model := m.getOrInitializeModel(selectedItem.state)

	if model == nil {
		// Show coming soon message if model is not implemented
		return m, ShowComingSoon(selectedItem.title)
	}

	// Set the active model and navigate to its state
	m.activeModel = model

	var cmds []tea.Cmd
	// Call the model's Init() method to start any commands
	modelInitCmd := model.Init()
	if modelInitCmd != nil {
		cmds = append(cmds, modelInitCmd)
	}

	// Send window size if layout has dimensions
	if m.layout.ContentWidth() > 0 && m.layout.ContentHeight() > 0 {
		windowMsg := tea.WindowSizeMsg{Width: m.layout.ContentWidth(), Height: m.layout.ContentHeight()}
		updatedModel, windowCmd := model.Update(windowMsg)
		m.activeModel = updatedModel.(MenuItemModel)
		if windowCmd != nil {
			cmds = append(cmds, windowCmd)
		}
	}

	cmds = append(cmds, NavigateTo(selectedItem.state))
	return m, tea.Batch(cmds...)
}

// GetUIContext creates a UI context with current dimensions and app state
func (m *MainModel) GetUIContext() helpers.UIContext {
	return helpers.NewUIContext(m.windowWidth, m.windowHeight, m.config, m.logger)
}

// getOrInitializeModel always creates a fresh model to ensure up-to-date settings
func (m *MainModel) getOrInitializeModel(state AppState) MenuItemModel {
	// Validate that we have valid dimensions before creating models
	if !m.hasValidDimensions() {
		m.logger.Warn("Cannot initialize model without valid window dimensions", "state", state)
		return nil
	}

	ctx := m.GetUIContext()

	switch state {
	case StateSettings:
		m.logger.Debug("Creating fresh settings model")
		return settingsmenu.NewSettingsModel(ctx)

	case StateSaveRules:
		m.logger.Debug("Creating fresh save rules model")
		return saverulesmodel.NewSaveRulesModel(ctx)

	case StateImportCopy:
		m.logger.Debug("Creating fresh import rules model")
		return importrulesmenu.NewImportRulesModel(ctx)

	case StateFetchGithub:
		// TODO: Initialize fetch github model when implemented
		// return NewFetchGithubModel(ctx)
		m.logger.Debug("Fetch GitHub model not yet implemented")
		return nil

	default:
		m.logger.Warn("Unknown state requested for model initialization", "state", state)
		return nil
	}
}

func (m *MainModel) View() string {
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

func (m *MainModel) viewMenu() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "🔧 Rulem - Rule Migration Tool",
		Subtitle: "Manage and organize your migration rules efficiently",
		HelpText: "↑/↓ to navigate • Enter to select • / to filter • q to quit • Ctrl+C to force quit",
	})

	// Get the menu content
	menuContent := m.menu.View()

	return m.layout.Render(menuContent)
}

// hasValidDimensions checks if window dimensions are valid for model creation
func (m *MainModel) hasValidDimensions() bool {
	return m.windowWidth > 0 && m.windowHeight > 0
}

// returnToMenu safely returns to the main menu and cleans up state
func (m *MainModel) returnToMenu() tea.Model {
	m.state = StateMenu
	m.activeModel = nil
	m.err = nil
	m.comingSoonFeature = ""
	m.layout = m.layout.ClearError()
	return m
}

func (m *MainModel) viewError() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "❌ Error",
		Subtitle: "Something went wrong",
		HelpText: "Press Esc to return • Ctrl+C to quit",
	})

	errorContent := ""
	if m.err != nil {
		errorContent = m.err.Error()
	}

	return m.layout.Render(errorContent)
}

func (m *MainModel) viewComingSoon() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "🚧 Coming Soon",
		Subtitle: "This feature is under development",
		HelpText: "Press Esc to return to menu • Ctrl+C to quit",
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

func ShowComingSoon(feature string) tea.Cmd {
	return func() tea.Msg {
		return ComingSoonMsg{Feature: feature}
	}
}
