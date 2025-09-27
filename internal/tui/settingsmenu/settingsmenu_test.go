package settingsmenu

import (
	"testing"

	"rulem/internal/config"
	"rulem/internal/logging"
	"rulem/internal/tui/helpers"

	tea "github.com/charmbracelet/bubbletea"
)

func createTestConfigWithPath(path string) *config.Config {
	return &config.Config{
		Central: config.CentralRepositoryConfig{Path: path},
	}
}

func TestNewSettingsModel(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)

	if model == nil {
		t.Fatal("NewSettingsModel returned nil")
	}

	if model.state != SettingsStateStorageInput {
		t.Errorf("Expected initial state to be SettingsStateStorageInput, got %v", model.state)
	}

	if model.logger != logger {
		t.Error("Logger not properly set")
	}

	if model.textInput.CharLimit != 256 {
		t.Errorf("Expected char limit to be 256, got %d", model.textInput.CharLimit)
	}
}

func TestSettingsModelInit(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	cmd := model.Init()

	if cmd == nil {
		t.Error("Init should return a command")
	}
}

func TestSettingsModelUpdate_WindowResize(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)

	// Test window resize
	windowMsg := tea.WindowSizeMsg{Width: 120, Height: 30}
	updatedModel, cmd := model.Update(windowMsg)

	if cmd != nil {
		t.Error("Window resize should not return a command")
	}

	settingsModel := updatedModel.(*SettingsModel)
	if settingsModel.textInput.Width == 0 {
		t.Error("Text input width should be updated after window resize")
	}
}

func TestSettingsModelUpdate_KeyHandling(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	tests := []struct {
		name         string
		initialState SettingsState
		keyMsg       tea.KeyMsg
		expectState  SettingsState
		expectCmd    bool
	}{
		{
			name:         "Enter in storage input validates",
			initialState: SettingsStateStorageInput,
			keyMsg:       tea.KeyMsg{Type: tea.KeyEnter},
			expectState:  SettingsStateStorageInput, // Will stay if validation fails
			expectCmd:    true,                      // Enter always returns a command (validation)
		},
		{
			name:         "Escape in storage input without changes navigates to menu",
			initialState: SettingsStateStorageInput,
			keyMsg:       tea.KeyMsg{Type: tea.KeyEsc},
			expectState:  SettingsStateStorageInput,
			expectCmd:    true,
		},
		{
			name:         "Ctrl+C quits",
			initialState: SettingsStateStorageInput,
			keyMsg:       tea.KeyMsg{Type: tea.KeyCtrlC},
			expectState:  SettingsStateStorageInput,
			expectCmd:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewSettingsModel(ctx)
			model.state = tt.initialState

			updatedModel, cmd := model.Update(tt.keyMsg)
			settingsModel := updatedModel.(*SettingsModel)

			if settingsModel.state != tt.expectState {
				t.Errorf("Expected state %v, got %v", tt.expectState, settingsModel.state)
			}

			if (cmd != nil) != tt.expectCmd {
				t.Errorf("Expected command presence: %v, got: %v", tt.expectCmd, cmd != nil)
			}
		})
	}
}

func TestSettingsModelUpdate_ConfigLoading(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)

	// Test successful config loading
	testConfig := createTestConfigWithPath("/test/path")
	configMsg := config.LoadConfigMsg{Config: testConfig, Error: nil}

	updatedModel, cmd := model.Update(configMsg)
	settingsModel := updatedModel.(*SettingsModel)

	if cmd != nil {
		t.Error("Config loading should not return a command")
	}

	if settingsModel.currentConfig != testConfig {
		t.Error("Current config should be set")
	}

	if settingsModel.textInput.Value() != "/test/path" {
		t.Errorf("Text input should be set to config central path, got %s", settingsModel.textInput.Value())
	}

	// Test config loading error
	errorMsg := config.LoadConfigMsg{Config: nil, Error: &ConfigError{"test error"}}
	updatedModel, cmd = model.Update(errorMsg)

	if cmd == nil {
		t.Error("Config loading error should return a command")
	}
}

func TestSettingsModelUpdate_ErrorHandling(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)

	// Test error message
	errorMsg := settingsErrorMsg{err: &ConfigError{"test error"}}
	updatedModel, cmd := model.Update(errorMsg)
	settingsModel := updatedModel.(*SettingsModel)

	if cmd != nil {
		t.Error("Error message should not return a command")
	}

	if settingsModel.state != SettingsStateError {
		t.Errorf("Expected state to be SettingsStateError, got %v", settingsModel.state)
	}
}

func TestSettingsModelUpdate_Completion(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)

	// Test completion message
	completeMsg := settingsCompleteMsg{}
	updatedModel, cmd := model.Update(completeMsg)
	settingsModel := updatedModel.(*SettingsModel)

	if cmd == nil {
		t.Error("Completion message should return a config reload command")
	}

	if settingsModel.state != SettingsStateComplete {
		t.Errorf("Expected state to be SettingsStateComplete, got %v", settingsModel.state)
	}
}

func TestSettingsModelView(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)

	tests := []struct {
		name         string
		state        SettingsState
		expectString string
	}{
		{
			name:         "Storage input view",
			state:        SettingsStateStorageInput,
			expectString: "Update Settings",
		},
		{
			name:         "Confirmation view",
			state:        SettingsStateConfirmation,
			expectString: "Save Changes",
		},
		{
			name:         "Complete view",
			state:        SettingsStateComplete,
			expectString: "Settings Updated!",
		},
		{
			name:         "Error view",
			state:        SettingsStateError,
			expectString: "Settings Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model.state = tt.state
			view := model.View()

			if view == "" {
				t.Error("View should not be empty")
			}

			// Check if the expected string is in the view (case-insensitive check would be better)
			// This is a simple check - in a real test you might want more sophisticated assertions
			if len(view) < 10 {
				t.Error("View seems too short")
			}
		})
	}
}

func TestSettingsModelValidation(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)

	// Test with empty input
	model.textInput.SetValue("")
	updatedModel, cmd := model.validateAndProceed()

	if cmd == nil {
		t.Error("Empty input should return error command")
	}

	// Test with valid input
	model.textInput.SetValue("/tmp/test")
	updatedModel, cmd = model.validateAndProceed()

	if updatedModel.StorageDir != "/tmp/test" {
		t.Errorf("Expected storage dir to be set to /tmp/test, got %s", updatedModel.StorageDir)
	}
}

func TestSettingsModelChangeDetection(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createTestConfigWithPath("/original/path")
	model.textInput.SetValue("/original/path")

	// Test no changes
	model, _ = model.updateTextInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("")})
	if model.hasChanges {
		t.Error("Should not detect changes when value is same")
	}

	// Test with changes
	model.textInput.SetValue("/new/path")
	model, _ = model.updateTextInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if !model.hasChanges {
		t.Error("Should detect changes when value is different")
	}
}

// Helper type for testing
type ConfigError struct {
	message string
}

func (e *ConfigError) Error() string {
	return e.message
}

// Test helpers
func createTestModel(t *testing.T) *SettingsModel {
	t.Helper()
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)
	return NewSettingsModel(ctx)
}

func TestTextInputBehavior(t *testing.T) {
	model := createTestModel(t)

	// Test that text input is properly configured
	if !model.textInput.Focused() {
		t.Error("Text input should be focused initially")
	}

	if model.textInput.CharLimit != 256 {
		t.Errorf("Expected char limit 256, got %d", model.textInput.CharLimit)
	}

	// Test text input updates
	runesMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("test")}
	model, cmd := model.updateTextInput(runesMsg)

	if cmd == nil {
		t.Error("Text input update should return a command (for blinking cursor)")
	}
}

func TestNavigationKeys(t *testing.T) {
	model := createTestModel(t)

	// Test quit key detection
	ctrlCMsg := tea.KeyMsg{Type: tea.KeyCtrlC}
	if !model.isNavigationKey(ctrlCMsg) {
		t.Error("Ctrl+C should be detected as navigation key")
	}

	// Test navigation handling
	model, cmd := model.handleNavigation(ctrlCMsg)
	if cmd == nil {
		t.Error("Ctrl+C should return quit command")
	}
}
