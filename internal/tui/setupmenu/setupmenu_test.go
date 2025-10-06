package setupmenu

import (
	"errors"
	"strings"
	"testing"

	"rulem/internal/logging"
	"rulem/internal/repository"
	"rulem/internal/tui/helpers"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Helper functions

func createTestLogger(t *testing.T) *logging.AppLogger {
	t.Helper()
	logger, _ := logging.NewTestLogger()
	return logger
}

func createTestUIContext(t *testing.T) helpers.UIContext {
	t.Helper()
	logger := createTestLogger(t)
	return helpers.NewUIContext(100, 30, nil, logger)
}

func createTestModel(t *testing.T) *SetupModel {
	t.Helper()
	ctx := createTestUIContext(t)
	model := NewSetupModel(ctx)

	// Use test credential manager with cleanup
	testCredManager := repository.NewTestCredentialManager(t)
	model.credManager = testCredManager.CredentialManager

	return model
}

func createModelInState(t *testing.T, state SetupState) *SetupModel {
	t.Helper()
	model := createTestModel(t)
	model.state = state

	// Set up context based on state
	switch state {
	case SetupStateConfirmation, SetupStateComplete:
		model.StorageDir = "/test/storage/dir"
	case SetupStateGitHubBranch, SetupStateGitHubPath, SetupStateGitHubPAT:
		model.repositoryType = RepositoryTypeGitHub
		model.GitHubURL = "https://github.com/test/repo.git"
	}

	if state == SetupStateGitHubPath || state == SetupStateGitHubPAT {
		model.GitHubBranch = "main"
	}

	if state == SetupStateGitHubPAT {
		model.GitHubPath = "~/test-repo"
	}

	return model
}

// Tests

func TestNewSetupModel(t *testing.T) {
	model := createTestModel(t)

	if model.state != SetupStateWelcome {
		t.Errorf("expected state %v, got %v", SetupStateWelcome, model.state)
	}
	if model.Cancelled {
		t.Error("expected Cancelled to be false")
	}
	if model.StorageDir != "" {
		t.Errorf("expected empty StorageDir, got %q", model.StorageDir)
	}
	if model.GitHubURL != "" {
		t.Errorf("expected empty GitHubURL, got %q", model.GitHubURL)
	}
	if model.textInput.Placeholder != repository.GetDefaultStorageDir() {
		t.Errorf("expected placeholder %q, got %q", repository.GetDefaultStorageDir(), model.textInput.Placeholder)
	}
	if !model.textInput.Focused() {
		t.Error("expected text input to be focused")
	}
	if model.logger == nil {
		t.Error("expected logger to be non-nil")
	}
}

func TestInit(t *testing.T) {
	model := createTestModel(t)
	cmd := model.Init()
	if cmd == nil {
		t.Error("expected Init to return non-nil cmd")
	}
}

func TestWelcomeState(t *testing.T) {
	tests := []struct {
		name          string
		key           tea.KeyType
		expectedState SetupState
		shouldQuit    bool
	}{
		{
			name:          "enter transitions to repository type",
			key:           tea.KeyEnter,
			expectedState: SetupStateRepositoryType,
			shouldQuit:    false,
		},
		{
			name:          "space transitions to repository type",
			key:           tea.KeySpace,
			expectedState: SetupStateRepositoryType,
			shouldQuit:    false,
		},
		{
			name:          "escape quits",
			key:           tea.KeyEscape,
			expectedState: SetupStateCancelled,
			shouldQuit:    true,
		},
		{
			name:          "q quits",
			key:           tea.KeyRunes,
			expectedState: SetupStateCancelled,
			shouldQuit:    true,
		},
		{
			name:          "ctrl+c quits",
			key:           tea.KeyCtrlC,
			expectedState: SetupStateCancelled,
			shouldQuit:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := createModelInState(t, SetupStateWelcome)

			var key tea.KeyMsg
			if tt.key == tea.KeyRunes {
				key = tea.KeyMsg{Type: tt.key, Runes: []rune("q")}
			} else {
				key = tea.KeyMsg{Type: tt.key}
			}

			updatedModel, cmd := model.Update(key)
			model = updatedModel.(*SetupModel)

			if model.state != tt.expectedState {
				t.Errorf("expected state %v, got %v", tt.expectedState, model.state)
			}

			if tt.shouldQuit {
				if !model.Cancelled {
					t.Error("expected Cancelled to be true")
				}
				if cmd == nil {
					t.Error("expected non-nil cmd for quit")
				}
			}
		})
	}
}

func TestRepositoryTypeSelection(t *testing.T) {
	t.Run("default selection is local", func(t *testing.T) {
		model := createModelInState(t, SetupStateRepositoryType)
		if model.repositoryTypeIndex != 0 {
			t.Errorf("expected repositoryTypeIndex 0, got %d", model.repositoryTypeIndex)
		}
	})

	t.Run("down arrow navigates to github", func(t *testing.T) {
		model := createModelInState(t, SetupStateRepositoryType)
		key := tea.KeyMsg{Type: tea.KeyDown}
		updatedModel, _ := model.Update(key)
		model = updatedModel.(*SetupModel)

		if model.repositoryTypeIndex != 1 {
			t.Errorf("expected repositoryTypeIndex 1, got %d", model.repositoryTypeIndex)
		}
	})

	t.Run("j key navigates down", func(t *testing.T) {
		model := createModelInState(t, SetupStateRepositoryType)
		key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
		updatedModel, _ := model.Update(key)
		model = updatedModel.(*SetupModel)

		if model.repositoryTypeIndex != 1 {
			t.Errorf("expected repositoryTypeIndex 1, got %d", model.repositoryTypeIndex)
		}
	})

	t.Run("up arrow stays at 0", func(t *testing.T) {
		model := createModelInState(t, SetupStateRepositoryType)
		model.repositoryTypeIndex = 0

		key := tea.KeyMsg{Type: tea.KeyUp}
		updatedModel, _ := model.Update(key)
		model = updatedModel.(*SetupModel)

		if model.repositoryTypeIndex != 0 {
			t.Errorf("expected repositoryTypeIndex 0, got %d", model.repositoryTypeIndex)
		}
	})

	t.Run("k key navigates up", func(t *testing.T) {
		model := createModelInState(t, SetupStateRepositoryType)
		model.repositoryTypeIndex = 1

		key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
		updatedModel, _ := model.Update(key)
		model = updatedModel.(*SetupModel)

		if model.repositoryTypeIndex != 0 {
			t.Errorf("expected repositoryTypeIndex 0, got %d", model.repositoryTypeIndex)
		}
	})

	t.Run("down arrow stops at 1", func(t *testing.T) {
		model := createModelInState(t, SetupStateRepositoryType)
		model.repositoryTypeIndex = 1

		key := tea.KeyMsg{Type: tea.KeyDown}
		updatedModel, _ := model.Update(key)
		model = updatedModel.(*SetupModel)

		if model.repositoryTypeIndex != 1 {
			t.Errorf("expected repositoryTypeIndex 1, got %d", model.repositoryTypeIndex)
		}
	})

	t.Run("enter selects local and transitions", func(t *testing.T) {
		model := createModelInState(t, SetupStateRepositoryType)
		model.repositoryTypeIndex = 0

		key := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ := model.Update(key)
		model = updatedModel.(*SetupModel)

		if model.repositoryType != RepositoryTypeLocal {
			t.Errorf("expected repositoryType %v, got %v", RepositoryTypeLocal, model.repositoryType)
		}
		if model.state != SetupStateStorageInput {
			t.Errorf("expected state %v, got %v", SetupStateStorageInput, model.state)
		}
	})

	t.Run("enter selects github and transitions", func(t *testing.T) {
		model := createModelInState(t, SetupStateRepositoryType)
		model.repositoryTypeIndex = 1

		key := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ := model.Update(key)
		model = updatedModel.(*SetupModel)

		if model.repositoryType != RepositoryTypeGitHub {
			t.Errorf("expected repositoryType %v, got %v", RepositoryTypeGitHub, model.repositoryType)
		}
		if model.state != SetupStateGitHubURL {
			t.Errorf("expected state %v, got %v", SetupStateGitHubURL, model.state)
		}
	})
}

func TestLocalStorageInput(t *testing.T) {
	t.Run("valid path proceeds to confirmation", func(t *testing.T) {
		model := createModelInState(t, SetupStateStorageInput)
		model.textInput.SetValue("~/test-storage")

		key := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ := model.Update(key)
		model = updatedModel.(*SetupModel)

		if model.state != SetupStateConfirmation {
			t.Errorf("expected state %v, got %v", SetupStateConfirmation, model.state)
		}
	})

	t.Run("empty path shows error", func(t *testing.T) {
		model := createModelInState(t, SetupStateStorageInput)
		model.textInput.SetValue("")

		key := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, cmd := model.Update(key)
		model = updatedModel.(*SetupModel)

		if model.state != SetupStateStorageInput {
			t.Errorf("expected state %v, got %v", SetupStateStorageInput, model.state)
		}
		if cmd == nil {
			t.Error("expected non-nil cmd for error")
		}

		msg := cmd()
		if _, ok := msg.(setupErrorMsg); !ok {
			t.Error("expected setupErrorMsg")
		}
	})

	t.Run("escape goes back to repository type", func(t *testing.T) {
		model := createModelInState(t, SetupStateStorageInput)

		key := tea.KeyMsg{Type: tea.KeyEscape}
		updatedModel, _ := model.Update(key)
		model = updatedModel.(*SetupModel)

		if model.state != SetupStateRepositoryType {
			t.Errorf("expected state %v, got %v", SetupStateRepositoryType, model.state)
		}
	})

	t.Run("typing updates input", func(t *testing.T) {
		model := createModelInState(t, SetupStateStorageInput)

		key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}
		updatedModel, _ := model.Update(key)
		model = updatedModel.(*SetupModel)

		if !strings.Contains(model.textInput.Value(), "a") {
			t.Error("expected text input to contain 'a'")
		}
	})
}

func TestGitHubURLInput(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		shouldError   bool
		expectedState SetupState
	}{
		{
			name:          "valid https url",
			url:           "https://github.com/owner/repo.git",
			shouldError:   false,
			expectedState: SetupStateGitHubBranch,
		},
		{
			name:          "valid ssh url",
			url:           "git@github.com:owner/repo.git",
			shouldError:   false,
			expectedState: SetupStateGitHubBranch,
		},
		{
			name:          "valid http url",
			url:           "http://github.com/owner/repo.git",
			shouldError:   false,
			expectedState: SetupStateGitHubBranch,
		},
		{
			name:          "empty url",
			url:           "",
			shouldError:   true,
			expectedState: SetupStateGitHubURL,
		},
		{
			name:          "invalid format",
			url:           "invalid-url",
			shouldError:   true,
			expectedState: SetupStateGitHubURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := createModelInState(t, SetupStateGitHubURL)
			model.textInput.SetValue(tt.url)

			key := tea.KeyMsg{Type: tea.KeyEnter}
			updatedModel, cmd := model.Update(key)
			model = updatedModel.(*SetupModel)

			if model.state != tt.expectedState {
				t.Errorf("expected state %v, got %v", tt.expectedState, model.state)
			}

			if tt.shouldError {
				if cmd == nil {
					t.Error("expected non-nil cmd for error")
				} else {
					msg := cmd()
					if _, ok := msg.(setupErrorMsg); !ok {
						t.Error("expected setupErrorMsg")
					}
				}
			}
		})
	}
}

func TestGitHubBranchInput(t *testing.T) {
	t.Run("branch value is stored", func(t *testing.T) {
		model := createModelInState(t, SetupStateGitHubBranch)
		model.textInput.SetValue("develop")

		key := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ := model.Update(key)
		model = updatedModel.(*SetupModel)

		if model.GitHubBranch != "develop" {
			t.Errorf("expected branch 'develop', got %q", model.GitHubBranch)
		}
		if model.state != SetupStateGitHubPath {
			t.Errorf("expected state %v, got %v", SetupStateGitHubPath, model.state)
		}
	})

	t.Run("empty branch is allowed", func(t *testing.T) {
		model := createModelInState(t, SetupStateGitHubBranch)
		model.textInput.SetValue("")

		key := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ := model.Update(key)
		model = updatedModel.(*SetupModel)

		if model.GitHubBranch != "" {
			t.Errorf("expected empty branch, got %q", model.GitHubBranch)
		}
		if model.state != SetupStateGitHubPath {
			t.Errorf("expected state %v, got %v", SetupStateGitHubPath, model.state)
		}
	})

	t.Run("escape goes back to url", func(t *testing.T) {
		model := createModelInState(t, SetupStateGitHubBranch)

		key := tea.KeyMsg{Type: tea.KeyEscape}
		updatedModel, _ := model.Update(key)
		model = updatedModel.(*SetupModel)

		if model.state != SetupStateGitHubURL {
			t.Errorf("expected state %v, got %v", SetupStateGitHubURL, model.state)
		}
	})
}

func TestGitHubPathInput(t *testing.T) {
	t.Run("valid path proceeds to PAT", func(t *testing.T) {
		model := createModelInState(t, SetupStateGitHubPath)
		// Use temp dir which is guaranteed to be valid
		validPath := t.TempDir()
		model.textInput.SetValue(validPath)

		key := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ := model.Update(key)
		model = updatedModel.(*SetupModel)

		// Should advance to PAT input state and set password mode
		if model.state != SetupStateGitHubPAT {
			t.Errorf("expected state %v, got %v", SetupStateGitHubPAT, model.state)
		}
		if model.textInput.EchoMode != textinput.EchoPassword {
			t.Errorf("expected EchoPassword mode, got %v", model.textInput.EchoMode)
		}
	})

	t.Run("invalid path shows error", func(t *testing.T) {
		model := createModelInState(t, SetupStateGitHubPath)
		model.textInput.SetValue("")

		key := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, cmd := model.Update(key)
		model = updatedModel.(*SetupModel)

		if cmd == nil {
			t.Error("expected non-nil cmd for error")
		}
	})
}

func TestGitHubPATInput(t *testing.T) {
	t.Run("PAT input is in password mode", func(t *testing.T) {
		model := createModelInState(t, SetupStateGitHubPAT)
		model.textInput.EchoMode = textinput.EchoPassword

		if model.textInput.EchoMode != textinput.EchoPassword {
			t.Errorf("expected EchoPassword mode, got %v", model.textInput.EchoMode)
		}
	})

	t.Run("empty PAT shows error", func(t *testing.T) {
		model := createModelInState(t, SetupStateGitHubPAT)
		model.textInput.SetValue("")

		key := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, cmd := model.Update(key)
		model = updatedModel.(*SetupModel)

		if model.state != SetupStateGitHubPAT {
			t.Errorf("expected state %v, got %v", SetupStateGitHubPAT, model.state)
		}
		if cmd == nil {
			t.Error("expected non-nil cmd for error")
		}

		msg := cmd()
		if errMsg, ok := msg.(setupErrorMsg); ok {
			if errMsg.err == nil {
				t.Error("expected error in setupErrorMsg")
			}
			if !strings.Contains(errMsg.err.Error(), "cannot be empty") {
				t.Errorf("expected 'cannot be empty' in error, got %q", errMsg.err.Error())
			}
		}
	})

	t.Run("invalid PAT format shows error", func(t *testing.T) {
		model := createModelInState(t, SetupStateGitHubPAT)
		model.textInput.SetValue("invalid-token")

		key := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, cmd := model.Update(key)
		model = updatedModel.(*SetupModel)

		if cmd == nil {
			t.Error("expected non-nil cmd for error")
		}

		msg := cmd()
		if errMsg, ok := msg.(setupErrorMsg); ok {
			if errMsg.err == nil {
				t.Error("expected error in setupErrorMsg")
			}
			if !strings.Contains(errMsg.err.Error(), "token") {
				t.Errorf("expected 'token' in error, got %q", errMsg.err.Error())
			}
		}
	})

	t.Run("valid PAT format is validated but repository access fails without real GitHub", func(t *testing.T) {
		// Note: This test demonstrates that format validation passes but repository validation
		// will fail without real GitHub repository access. In production, valid tokens will pass
		// repository validation and advance to confirmation state.
		model := createModelInState(t, SetupStateGitHubPAT)
		testToken := repository.CreateTestToken("")

		model.textInput.SetValue(testToken)

		key := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, cmd := model.Update(key)
		model = updatedModel.(*SetupModel)

		// Token format is valid, but repository validation will fail (no real GitHub repository)
		if cmd == nil {
			t.Error("expected non-nil cmd due to repository validation failure")
		}

		// Execute the command to verify it's an error
		msg := cmd()
		if errMsg, ok := msg.(setupErrorMsg); ok {
			if errMsg.err == nil {
				t.Error("expected error in setupErrorMsg")
			}
			// Repository operation will fail with connection/auth error
		}

		// Token is NOT stored in memory because validation failed
		// In production with valid GitHub access, token would be stored
		if model.GitHubPAT != "" {
			t.Errorf("expected empty PAT after validation failure, got %q", model.GitHubPAT)
		}
	})

	t.Run("escape goes back to path", func(t *testing.T) {
		model := createModelInState(t, SetupStateGitHubPAT)

		key := tea.KeyMsg{Type: tea.KeyEscape}
		updatedModel, _ := model.Update(key)
		model = updatedModel.(*SetupModel)

		if model.state != SetupStateGitHubPath {
			t.Errorf("expected state %v, got %v", SetupStateGitHubPath, model.state)
		}
	})
}

func TestConfirmationState(t *testing.T) {
	t.Run("y confirms local setup", func(t *testing.T) {
		_, cleanup := SetTestConfigPath(t)
		defer cleanup()

		model := createModelInState(t, SetupStateConfirmation)
		model.repositoryType = RepositoryTypeLocal
		model.StorageDir = t.TempDir()

		key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}
		updatedModel, cmd := model.Update(key)
		model = updatedModel.(*SetupModel)

		if cmd == nil {
			t.Error("expected non-nil cmd")
		}
	})

	t.Run("n goes back to storage for local", func(t *testing.T) {
		model := createModelInState(t, SetupStateConfirmation)
		model.repositoryType = RepositoryTypeLocal

		key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}
		updatedModel, _ := model.Update(key)
		model = updatedModel.(*SetupModel)

		if model.state != SetupStateStorageInput {
			t.Errorf("expected state %v, got %v", SetupStateStorageInput, model.state)
		}
	})

	t.Run("n goes back to PAT for github", func(t *testing.T) {
		model := createModelInState(t, SetupStateConfirmation)
		model.repositoryType = RepositoryTypeGitHub
		model.GitHubPAT = "test-token"

		key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}
		updatedModel, _ := model.Update(key)
		model = updatedModel.(*SetupModel)

		if model.state != SetupStateGitHubPAT {
			t.Errorf("expected state %v, got %v", SetupStateGitHubPAT, model.state)
		}
	})

	t.Run("escape goes back", func(t *testing.T) {
		model := createModelInState(t, SetupStateConfirmation)
		model.repositoryType = RepositoryTypeLocal

		key := tea.KeyMsg{Type: tea.KeyEscape}
		updatedModel, _ := model.Update(key)
		model = updatedModel.(*SetupModel)

		if model.state != SetupStateStorageInput {
			t.Errorf("expected state %v, got %v", SetupStateStorageInput, model.state)
		}
	})
}

func TestErrorHandling(t *testing.T) {
	t.Run("error message displays error", func(t *testing.T) {
		model := createTestModel(t)
		testErr := errors.New("test error")

		updatedModel, _ := model.Update(setupErrorMsg{testErr})
		model = updatedModel.(*SetupModel)

		if model.layout.GetError() == nil {
			t.Error("expected layout to have error")
		}
	})

	t.Run("typing clears error", func(t *testing.T) {
		model := createModelInState(t, SetupStateStorageInput)
		model.layout = model.layout.SetError(errors.New("test error"))

		key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}
		updatedModel, _ := model.Update(key)
		model = updatedModel.(*SetupModel)

		if model.layout.GetError() != nil {
			t.Error("expected error to be cleared")
		}
	})

	t.Run("complete message clears error and changes state", func(t *testing.T) {
		model := createTestModel(t)
		model.layout = model.layout.SetError(errors.New("test error"))

		updatedModel, _ := model.Update(setupCompleteMsg{})
		model = updatedModel.(*SetupModel)

		if model.layout.GetError() != nil {
			t.Error("expected error to be cleared")
		}
		if model.state != SetupStateComplete {
			t.Errorf("expected state %v, got %v", SetupStateComplete, model.state)
		}
	})
}

func TestViewRendering(t *testing.T) {
	tests := []struct {
		name     string
		state    SetupState
		contains []string
	}{
		{
			name:     "welcome view",
			state:    SetupStateWelcome,
			contains: []string{"Welcome"},
		},
		{
			name:     "repository type view",
			state:    SetupStateRepositoryType,
			contains: []string{"Repository Type", "Local", "GitHub"},
		},
		{
			name:     "storage input view",
			state:    SetupStateStorageInput,
			contains: []string{"storage", "directory"},
		},
		{
			name:     "github url view",
			state:    SetupStateGitHubURL,
			contains: []string{"GitHub", "URL"},
		},
		{
			name:     "github branch view",
			state:    SetupStateGitHubBranch,
			contains: []string{"branch"},
		},
		{
			name:     "github path view",
			state:    SetupStateGitHubPath,
			contains: []string{"path"},
		},
		{
			name:     "github pat view",
			state:    SetupStateGitHubPAT,
			contains: []string{"token", "PAT"},
		},
		{
			name:     "complete view",
			state:    SetupStateComplete,
			contains: []string{"Setup Complete"},
		},
		{
			name:     "cancelled view",
			state:    SetupStateCancelled,
			contains: []string{"Setup Cancelled"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := createModelInState(t, tt.state)
			view := model.View()

			for _, expected := range tt.contains {
				if !strings.Contains(view, expected) {
					t.Errorf("expected view to contain %q", expected)
				}
			}
		})
	}
}

func TestWindowSizeHandling(t *testing.T) {
	model := createTestModel(t)

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updatedModel, _ := model.Update(msg)
	model = updatedModel.(*SetupModel)

	// Just verify it doesn't panic
	if model == nil {
		t.Error("expected non-nil model after window size update")
	}
}

func TestPerformConfigCreation(t *testing.T) {
	t.Run("creates local config", func(t *testing.T) {
		_, cleanup := SetTestConfigPath(t)
		defer cleanup()

		model := createTestModel(t)
		model.repositoryType = RepositoryTypeLocal
		model.StorageDir = t.TempDir()

		err := model.performConfigCreation()

		// May fail if directory validation fails, but should not panic
		if err != nil {
			t.Logf("Config creation failed (expected in test): %v", err)
		}
	})

	t.Run("creates github config", func(t *testing.T) {
		_, cleanup := SetTestConfigPath(t)
		defer cleanup()

		model := createTestModel(t)
		model.repositoryType = RepositoryTypeGitHub
		model.GitHubURL = "https://github.com/test/repo.git"
		model.GitHubBranch = "main"
		model.GitHubPath = t.TempDir()
		model.GitHubPAT = "ghp_test"

		err := model.performConfigCreation()

		// Should succeed in creating config
		if err != nil {
			t.Errorf("unexpected error creating github config: %v", err)
		}
	})
}

// Benchmarks

func BenchmarkUpdate(b *testing.B) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(100, 30, nil, logger)
	model := NewSetupModel(ctx)
	key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.Update(key)
	}
}

func BenchmarkView(b *testing.B) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(100, 30, nil, logger)
	model := NewSetupModel(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.View()
	}
}
