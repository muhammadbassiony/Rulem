package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSetupModelStateTransitions(t *testing.T) {
	tests := []struct {
		name          string
		initialState  SetupState
		input         string
		expectedState SetupState
		expectError   bool
	}{
		{
			name:          "welcome to storage input",
			initialState:  SetupStateWelcome,
			input:         "enter",
			expectedState: SetupStateStorageInput,
		},
		{
			name:          "welcome cancel",
			initialState:  SetupStateWelcome,
			input:         "esc",
			expectedState: SetupStateCancelled,
		},
		{
			name:          "storage input cancel",
			initialState:  SetupStateStorageInput,
			input:         "esc",
			expectedState: SetupStateCancelled,
		},
		{
			name:          "confirmation back to input",
			initialState:  SetupStateConfirmation,
			input:         "n",
			expectedState: SetupStateStorageInput,
		},
		{
			name:          "confirmation accept",
			initialState:  SetupStateConfirmation,
			input:         "y",
			expectedState: SetupStateComplete,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewSetupModel()
			model.state = tt.initialState

			if tt.initialState == SetupStateConfirmation {
				model.StorageDir = "/tmp/test"
			}

			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.input)}
			if tt.input == "enter" {
				msg = tea.KeyMsg{Type: tea.KeyEnter}
			} else if tt.input == "esc" {
				msg = tea.KeyMsg{Type: tea.KeyEsc}
			}

			updatedModel, _ := model.Update(msg)
			setupModel := updatedModel.(SetupModel)

			if setupModel.state != tt.expectedState {
				t.Errorf("Expected state %v, got %v", tt.expectedState, setupModel.state)
			}
		})
	}
}

func TestSetupModelTextInput(t *testing.T) {
	model := NewSetupModel()
	model.state = SetupStateStorageInput

	// Test character input
	chars := "test"
	for _, char := range chars {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		updatedModel, _ := model.Update(msg)
		model = updatedModel.(SetupModel)
	}

	if !strings.Contains(model.textInput.Value(), "test") {
		t.Error("Text input did not capture characters")
	}

	// Test backspace
	msg := tea.KeyMsg{Type: tea.KeyBackspace}
	updatedModel, _ := model.Update(msg)
	model = updatedModel.(SetupModel)

	if strings.HasSuffix(model.textInput.Value(), "test") {
		t.Error("Backspace did not work")
	}
}

func TestSetupModelWindowResize(t *testing.T) {
	model := NewSetupModel()

	// Test window resize
	resizeMsg := tea.WindowSizeMsg{Width: 120, Height: 30}
	updatedModel, _ := model.Update(resizeMsg)
	setupModel := updatedModel.(SetupModel)

	if setupModel.width != 120 || setupModel.height != 30 {
		t.Error("Window size not updated correctly")
	}

	// Test that input width is updated
	expectedWidth := setupModel.getInputWidth()
	if setupModel.textInput.Width != expectedWidth {
		t.Errorf("Expected input width %d, got %d", expectedWidth, setupModel.textInput.Width)
	}
}

func TestSetupModelValidation(t *testing.T) {
	model := NewSetupModel()
	model.state = SetupStateStorageInput

	// Test invalid input
	model.textInput.SetValue("../invalid/path")

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(msg)
	setupModel := updatedModel.(SetupModel)

	if setupModel.err == nil {
		t.Error("Expected validation error for invalid path")
	}

	if setupModel.state != SetupStateStorageInput {
		t.Error("Should remain in input state on validation error")
	}
}

func TestSetupModelRenderSections(t *testing.T) {
	model := NewSetupModel()

	tests := []struct {
		name     string
		state    SetupState
		checkFor []string
	}{
		{
			name:     "welcome view",
			state:    SetupStateWelcome,
			checkFor: []string{"Welcome to RuleMig", "Press Enter"},
		},
		{
			name:     "storage input view",
			state:    SetupStateStorageInput,
			checkFor: []string{"Storage Directory", "Enter storage"},
		},
		{
			name:     "confirmation view",
			state:    SetupStateConfirmation,
			checkFor: []string{"Confirm Configuration", "Is this correct"},
		},
		{
			name:     "complete view",
			state:    SetupStateComplete,
			checkFor: []string{"Setup Complete", "successfully"},
		},
		{
			name:     "cancelled view",
			state:    SetupStateCancelled,
			checkFor: []string{"cancelled"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model.state = tt.state
			if tt.state == SetupStateConfirmation {
				model.StorageDir = "/tmp/test"
			}

			view := model.View()

			for _, check := range tt.checkFor {
				if !strings.Contains(view, check) {
					t.Errorf("View should contain '%s', got: %s", check, view)
				}
			}
		})
	}
}
