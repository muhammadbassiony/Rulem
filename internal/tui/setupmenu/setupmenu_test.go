package setupmenu

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rulem/internal/filemanager"
	"rulem/internal/logging"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// TestNewSetupModel tests model initialization
func TestNewSetupModel(t *testing.T) {
	tests := []struct {
		name string
		want SetupState
	}{
		{
			name: "initializes with welcome state",
			want: SetupStateWelcome,
		},
	}

	logger, _ := logging.NewTestLogger()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			model := NewSetupModel(logger)

			assert.Equal(t, tt.want, model.state)
			assert.False(t, model.Cancelled)
			assert.Empty(t, model.StorageDir)
			assert.Equal(t, filemanager.GetDefaultStorageDir(), model.textInput.Placeholder)
			assert.True(t, model.textInput.Focused())
		})
	}
}

// TestStateTransitions tests all state transitions
func TestStateTransitions(t *testing.T) {
	// Create a temporary directory for testing valid paths
	tempDir, err := os.MkdirTemp("", "setupmenu-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	validTestPath := filepath.Join(tempDir, "test-storage")

	tests := []struct {
		name          string
		initialState  SetupState
		keyPress      string
		expectedState SetupState
		setupInput    string
		shouldQuit    bool
	}{
		// Welcome state transitions
		{
			name:          "welcome_to_storage_on_enter",
			initialState:  SetupStateWelcome,
			keyPress:      "enter",
			expectedState: SetupStateStorageInput,
		},
		{
			name:          "welcome_to_storage_on_space",
			initialState:  SetupStateWelcome,
			keyPress:      " ",
			expectedState: SetupStateStorageInput,
		},
		{
			name:          "welcome_quit_on_esc",
			initialState:  SetupStateWelcome,
			keyPress:      "esc",
			expectedState: SetupStateCancelled,
			shouldQuit:    true,
		},
		{
			name:          "welcome_quit_on_ctrl_c",
			initialState:  SetupStateWelcome,
			keyPress:      "ctrl+c",
			expectedState: SetupStateCancelled,
			shouldQuit:    true,
		},
		{
			name:          "welcome_quit_on_q",
			initialState:  SetupStateWelcome,
			keyPress:      "q",
			expectedState: SetupStateCancelled,
			shouldQuit:    true,
		},
		// Storage input state transitions
		{
			name:          "storage_to_confirmation_valid_input",
			initialState:  SetupStateStorageInput,
			keyPress:      "enter",
			setupInput:    validTestPath,
			expectedState: SetupStateConfirmation,
		},
		{
			name:          "storage_quit_on_esc",
			initialState:  SetupStateStorageInput,
			keyPress:      "esc",
			expectedState: SetupStateCancelled,
			shouldQuit:    true,
		},
		// Confirmation state transitions
		{
			name:          "confirmation_back_to_storage_on_n",
			initialState:  SetupStateConfirmation,
			keyPress:      "n",
			expectedState: SetupStateStorageInput,
		},
		{
			name:          "confirmation_back_to_storage_on_N",
			initialState:  SetupStateConfirmation,
			keyPress:      "N",
			expectedState: SetupStateStorageInput,
		},
		{
			name:          "confirmation_quit_on_esc",
			initialState:  SetupStateConfirmation,
			keyPress:      "esc",
			expectedState: SetupStateCancelled,
			shouldQuit:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := createModelInState(tt.initialState, tt.setupInput)

			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.keyPress)}
			if tt.keyPress == "enter" {
				keyMsg = tea.KeyMsg{Type: tea.KeyEnter}
			} else if tt.keyPress == "esc" {
				keyMsg = tea.KeyMsg{Type: tea.KeyEscape}
			} else if tt.keyPress == "ctrl+c" {
				keyMsg = tea.KeyMsg{Type: tea.KeyCtrlC}
			}

			updatedModel, cmd := model.Update(keyMsg)
			setupModel := updatedModel.(SetupModel)

			// If a command is returned, execute it and update the model
			if cmd != nil {
				msg := cmd()
				setupModel = updatedModel.(SetupModel)
				updatedModel, _ = setupModel.Update(msg)
				setupModel = updatedModel.(SetupModel)
			}

			assert.Equal(t, tt.expectedState, setupModel.state)

			if tt.shouldQuit {
				assert.True(t, setupModel.Cancelled)
				assert.NotNil(t, cmd)
			}
		})
	}
}

// TestInputValidation tests storage directory validation
func TestInputValidation(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		shouldError   bool
		expectedState SetupState
	}{
		{
			name:          "valid_absolute_path",
			input:         "/tmp/rulem",
			shouldError:   false,
			expectedState: SetupStateConfirmation,
		},
		{
			name:          "valid_home_path",
			input:         "~/rulem",
			shouldError:   false,
			expectedState: SetupStateConfirmation,
		},
		{
			name:          "empty_input",
			input:         "",
			shouldError:   true,
			expectedState: SetupStateStorageInput,
		},
		{
			name:          "whitespace_only",
			input:         "   ",
			shouldError:   true,
			expectedState: SetupStateStorageInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := createModelInState(SetupStateStorageInput, tt.input)

			keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
			updatedModel, cmd := model.Update(keyMsg)
			setupModel := updatedModel.(SetupModel)

			if tt.shouldError {
				// Should return a command that sends error message
				assert.NotNil(t, cmd)
				assert.Equal(t, tt.expectedState, setupModel.state)

				// Execute the command to get the error message
				msg := cmd()
				errorMsg, ok := msg.(setupErrorMsg)
				assert.True(t, ok)
				assert.Error(t, errorMsg.err)
			} else {
				assert.Equal(t, tt.expectedState, setupModel.state)
				assert.NotEmpty(t, setupModel.StorageDir)
			}
		})
	}
}

// TestTextInputBehavior tests text input functionality
func TestTextInputBehavior(t *testing.T) {
	tests := []struct {
		name             string
		initialInput     string
		keySequence      []string
		expectedValue    string
		shouldClearError bool
	}{
		{
			name:          "typing_updates_input",
			initialInput:  "",
			keySequence:   []string{"a", "b", "c"},
			expectedValue: "abc",
		},
		{
			name:             "typing_clears_error",
			initialInput:     "",
			keySequence:      []string{"a"},
			expectedValue:    "a",
			shouldClearError: true,
		},
		{
			name:          "backspace_removes_characters",
			initialInput:  "test",
			keySequence:   []string{"backspace"},
			expectedValue: "tes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := createModelInState(SetupStateStorageInput, tt.initialInput)

			// Add an error if we're testing error clearing
			if tt.shouldClearError {
				model.layout = model.layout.SetError(errors.New("test error"))
			}

			for _, key := range tt.keySequence {
				var keyMsg tea.KeyMsg
				switch key {
				case "backspace":
					keyMsg = tea.KeyMsg{Type: tea.KeyBackspace}
				default:
					keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
				}

				updatedModel, _ := model.Update(keyMsg)
				model = updatedModel.(SetupModel)
			}

			assert.Equal(t, tt.expectedValue, model.textInput.Value())

			if tt.shouldClearError {
				assert.Nil(t, model.layout.GetError())
			}
		})
	}
}

// TestErrorHandling tests error scenarios
func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name         string
		errorMsg     error
		expectsError bool
	}{
		{
			name:         "handles_setup_error",
			errorMsg:     errors.New("validation failed"),
			expectsError: true,
		},
		{
			name:         "clears_error_on_complete",
			errorMsg:     nil,
			expectsError: false,
		},
	}

	logger, _ := logging.NewTestLogger()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewSetupModel(logger)

			if tt.errorMsg != nil {
				errorMessage := setupErrorMsg{err: tt.errorMsg}
				updatedModel, _ := model.Update(errorMessage)
				model = updatedModel.(SetupModel)

				assert.Equal(t, tt.errorMsg, model.layout.GetError())
			}

			if !tt.expectsError {
				completeMsg := setupCompleteMsg{}
				updatedModel, _ := model.Update(completeMsg)
				model = updatedModel.(SetupModel)

				assert.Nil(t, model.layout.GetError())
				assert.Equal(t, SetupStateComplete, model.state)
			}
		})
	}
}

// TestViewRendering tests view output
func TestViewRendering(t *testing.T) {
	tests := []struct {
		name          string
		state         SetupState
		storageDir    string
		hasError      bool
		expectedTitle string
		expectedHelp  string
	}{
		{
			name:          "welcome_view",
			state:         SetupStateWelcome,
			expectedTitle: "🔧 Welcome to Rulem!",
			expectedHelp:  "Press Enter to continue, or Esc to cancel",
		},
		{
			name:          "storage_input_view",
			state:         SetupStateStorageInput,
			expectedTitle: "📁 Storage Directory",
			expectedHelp:  "Press Enter to continue • Esc to cancel • Use ~ for home directory",
		},
		{
			name:          "confirmation_view",
			state:         SetupStateConfirmation,
			storageDir:    "/test/path",
			expectedTitle: "✅ Confirm Configuration",
			expectedHelp:  "Press y to confirm • n to go back • Esc to cancel",
		},
		{
			name:          "complete_view",
			state:         SetupStateComplete,
			storageDir:    "/test/path",
			expectedTitle: "🎉 Setup Complete!",
		},
		{
			name:          "cancelled_view",
			state:         SetupStateCancelled,
			expectedTitle: "❌ Setup Cancelled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := createModelInState(tt.state, "")
			if tt.storageDir != "" {
				model.StorageDir = tt.storageDir
			}

			view := model.View()

			assert.Contains(t, view, tt.expectedTitle)
			if tt.expectedHelp != "" {
				for _, part := range strings.Fields(tt.expectedHelp) {
					assert.Contains(t, view, part)
				}
			}
			if tt.storageDir != "" && (tt.state == SetupStateConfirmation || tt.state == SetupStateComplete) {
				assert.Contains(t, view, tt.storageDir)
			}
		})
	}
}

// Helper function to create model in specific state
func createModelInState(state SetupState, input string) SetupModel {
	logger, _ := logging.NewTestLogger()

	model := NewSetupModel(logger)
	model.state = state

	if input != "" {
		model.textInput.SetValue(input)
	}

	if state == SetupStateConfirmation || state == SetupStateComplete {
		model.StorageDir = "/test/storage/dir"
	}

	return model
}

// Benchmark tests
func BenchmarkUpdate(b *testing.B) {
	logger, _ := logging.NewTestLogger()

	model := NewSetupModel(logger)
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.Update(keyMsg)
	}
}

func BenchmarkView(b *testing.B) {
	logger, _ := logging.NewTestLogger()

	model := NewSetupModel(logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.View()
	}
}
