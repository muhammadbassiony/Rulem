package settingsmenu

import (
	"testing"

	"rulem/internal/config"
	"rulem/internal/logging"
	"rulem/internal/tui/helpers"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSettingsMenuBasicFlow(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)

	// Test initial state
	if model.state != SettingsStateStorageInput {
		t.Errorf("Expected initial state SettingsStateStorageInput, got %v", model.state)
	}

	// Test view rendering
	view := model.View()
	if view == "" {
		t.Error("View should not be empty")
	}

	// Test input handling
	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("test")})
	if cmd == nil {
		t.Error("Text input should return a command")
	}

	settingsModel := updatedModel.(*SettingsModel)
	if settingsModel.textInput.Value() != "test" {
		t.Errorf("Expected text input value 'test', got '%s'", settingsModel.textInput.Value())
	}
}

func TestSettingsMenuStateTransitions(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)

	// Test transition to confirmation state
	model.StorageDir = "/tmp/test"
	model.hasChanges = true
	model.state = SettingsStateConfirmation

	view := model.View()
	if view == "" {
		t.Error("Confirmation view should not be empty")
	}

	// Test transition to complete state
	model.state = SettingsStateComplete
	view = model.View()
	if view == "" {
		t.Error("Complete view should not be empty")
	}

	// Test transition to error state
	model.state = SettingsStateError
	view = model.View()
	if view == "" {
		t.Error("Error view should not be empty")
	}
}

func TestSettingsMenuKeyHandling(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)

	// Test escape key without changes
	model.hasChanges = false
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Error("Escape without changes should return navigation command")
	}

	// Test global quit
	_, cmd = model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("Ctrl+C should return quit command")
	}

	// Test enter key in input state
	model.textInput.SetValue("/tmp/test")
	_, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("Enter should return validation command")
	}
}

func TestSettingsMenuChangeDetection(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createTestConfigWithPath("/original")
	model.textInput.SetValue("/original")

	// No changes initially
	if model.hasChanges {
		t.Error("Should not have changes initially")
	}

	// Modify input to detect changes
	model.textInput.SetValue("/modified")
	model, _ = model.updateTextInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	if !model.hasChanges {
		t.Error("Should detect changes after modification")
	}
}

func TestSettingsMenuConfigLoading(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)

	// Test successful config loading
	testConfig := createTestConfigWithPath("/test/path")
	configMsg := config.LoadConfigMsg{Config: testConfig, Error: nil}

	updatedModel, cmd := model.Update(configMsg)
	if cmd != nil {
		t.Error("Config loading should not return a command")
	}

	settingsModel := updatedModel.(*SettingsModel)
	if settingsModel.currentConfig != testConfig {
		t.Error("Current config should be set")
	}

	if settingsModel.textInput.Value() != "/test/path" {
		t.Errorf("Text input should be set to config storage dir, got %s", settingsModel.textInput.Value())
	}
}
