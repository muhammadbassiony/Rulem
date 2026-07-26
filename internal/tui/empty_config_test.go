package tui

import (
	"testing"

	"rulem/internal/config"
	"rulem/internal/logging"
	"rulem/internal/repository"

	tea "github.com/charmbracelet/bubbletea"
)

// After deleting every repository the user lands back on the main menu with an
// empty configuration. Every entry must still be reachable and render an
// explanation rather than crashing the program.

func emptyConfig() *config.Config {
	return &config.Config{Repositories: []repository.RepositoryEntry{}}
}

func TestMainModel_EmptyConfig_AllMenuEntriesAreSafe(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	states := []AppState{StateSettings, StateSaveRules, StateImportCopy, StateRepoStatus}

	for _, state := range states {
		model := NewMainModel(emptyConfig(), logger)
		model.windowWidth = 80
		model.windowHeight = 24

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("entering state %v with an empty config panicked: %v", state, r)
				}
			}()

			sub := model.getOrInitializeModel(state)
			if sub == nil {
				t.Errorf("expected a model for state %v, got nil", state)
				return
			}
			model.activeModel = sub
			model.state = state

			// Init and a first render are what the user actually triggers.
			_ = sub.Init()
			if view := model.View(); view == "" {
				t.Errorf("state %v rendered an empty view", state)
			}
		}()
	}
}

func TestMainModel_EmptyConfig_MenuRenders(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	model := NewMainModel(emptyConfig(), logger)

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	main := updated.(*MainModel)

	if view := main.View(); view == "" {
		t.Error("expected the main menu to render with an empty config")
	}
}

func TestMainModel_NilRepositories_MenuRenders(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	// A config decoded from `repositories:` with no value yields a nil slice,
	// which must behave the same as an empty one.
	model := NewMainModel(&config.Config{}, logger)

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	main := updated.(*MainModel)

	if view := main.View(); view == "" {
		t.Error("expected the main menu to render with nil repositories")
	}
}
