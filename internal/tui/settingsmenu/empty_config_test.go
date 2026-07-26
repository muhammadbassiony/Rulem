package settingsmenu

import (
	"rulem/internal/config"
	"rulem/internal/repository"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// An empty repository list is a normal state — it is what remains after the user
// deletes every repository. These tests pin the behaviour the settings menu must
// keep in that state: no panics, a usable main menu, and a way back to a working
// configuration.

func TestSettingsMenu_EmptyConfig_MainMenuIsUsable(t *testing.T) {
	model := createTestModelWithConfig(t, &config.Config{
		Repositories: []repository.RepositoryEntry{},
	})

	view := model.viewMainMenu()
	if view == "" {
		t.Fatal("main menu should render with no repositories configured")
	}
	if !strings.Contains(view, "no repositories configured") {
		t.Errorf("expected an empty-state explanation, got:\n%s", view)
	}
	if !strings.Contains(view, "Add New Repository") {
		t.Errorf("expected the Add New Repository action to remain reachable, got:\n%s", view)
	}
}

func TestSettingsMenu_EmptyConfig_ListKeepsActionItems(t *testing.T) {
	items := BuildSettingsMainMenuItems(nil)

	if len(items) != 2 {
		t.Fatalf("expected the 2 action items with no repositories, got %d", len(items))
	}
	for _, item := range items {
		if _, ok := item.(SettingsActionListItem); !ok {
			t.Errorf("expected only action items, got %T", item)
		}
	}
}

func TestSettingsMenu_EmptyConfig_AddRepositoryFlowStarts(t *testing.T) {
	model := createTestModelWithConfig(t, &config.Config{
		Repositories: []repository.RepositoryEntry{},
	})
	model.state = SettingsStateMainMenu

	// With no repositories, "Add New Repository" is the first list entry.
	model.repoList.Select(0)
	updated, _ := model.handleMainMenuKeys(tea.KeyMsg{Type: tea.KeyEnter})

	if updated.state != SettingsStateAddRepositoryType {
		t.Fatalf("expected Add Repository flow to start, got state %v", updated.state)
	}
}

func TestSettingsMenu_EmptyConfig_ViewsDoNotPanic(t *testing.T) {
	model := createTestModelWithConfig(t, &config.Config{
		Repositories: []repository.RepositoryEntry{},
	})

	// selectedRepositoryID intentionally points at nothing: this is the state
	// left behind right after the last repository is deleted.
	model.selectedRepositoryID = ""

	states := []SettingsState{
		SettingsStateMainMenu,
		SettingsStateRepositoryActions,
		SettingsStateConfirmDelete,
		SettingsStateDeleteError,
		SettingsStateUpdateGitHubPAT,
		SettingsStateUpdatePATConfirm,
		SettingsStateComplete,
	}

	for _, state := range states {
		model.state = state
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("View() panicked in state %v with an empty config: %v", state, r)
				}
			}()
			_ = model.View()
		}()
	}
}

func TestSettingsMenu_NilConfig_ViewsDoNotPanic(t *testing.T) {
	model := createTestModel(t)
	model.currentConfig = nil

	states := []SettingsState{
		SettingsStateMainMenu,
		SettingsStateRepositoryActions,
		SettingsStateConfirmDelete,
		SettingsStateUpdateGitHubPAT,
	}

	for _, state := range states {
		model.state = state
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("View() panicked in state %v with a nil config: %v", state, r)
				}
			}()
			_ = model.View()
		}()
	}
}

func TestGetMenuOptions_NilConfig(t *testing.T) {
	model := createTestModel(t)
	model.currentConfig = nil

	options := model.getMenuOptions()
	if len(options) == 0 {
		t.Fatal("expected menu options even with a nil config")
	}
	if options[len(options)-1].Option != ChangeOptionBack {
		t.Error("last option should be ChangeOptionBack")
	}
}
