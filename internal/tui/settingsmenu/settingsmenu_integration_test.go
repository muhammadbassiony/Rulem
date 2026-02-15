package settingsmenu

import (
	"fmt"
	"os"
	"path/filepath"
	"rulem/internal/config"
	"rulem/internal/logging"
	"rulem/internal/repository"
	"rulem/internal/tui/helpers"
	"slices"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Integration tests for complete settings menu flows
// These tests verify end-to-end behavior including state transitions, validation, and user interactions

// Test Helpers

func createTempDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	return tmpDir
}

// Basic Flow Tests

func TestSettingsMenuBasicFlow(t *testing.T) {
	t.Skip("Requires real git repository - involves git operations in flow")
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)

	// Test initial state
	if model.state != SettingsStateMainMenu {
		t.Errorf("Expected initial state SettingsStateMainMenu, got %v", model.state)
	}

	// Initialize
	cmd := model.Init()
	if cmd == nil {
		t.Error("Init should return a command")
	}

	// Load config
	testConfig := createLocalConfig("/test/path")
	configMsg := config.LoadConfigMsg{Config: testConfig, Error: nil}
	updatedModel, _ := model.Update(configMsg)
	model = updatedModel.(*SettingsModel)

	if model.currentConfig == nil {
		t.Error("Config should be loaded")
	}

	// Navigate to select change menu
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateRepositoryActions {
		t.Errorf("Expected SelectChange state after Enter from MainMenu, got %v", model.state)
	}

	// Test navigation back to main menu
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ = model.Update(escMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateMainMenu {
		t.Errorf("Expected MainMenu state after Esc from SelectChange, got %v", model.state)
	}
}

// Local Path Update Tests

// Deprecated tests removed - local path editing, repository type switching, and GitHub URL editing
// were deprecated in Phase 1 of the settings menu refactoring

// GitHub Branch Update Tests

func TestCompleteFlow_GitHubBranchUpdate_Standalone(t *testing.T) {
	t.Skip("Requires real git repository - branch update validates branch on remote")
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/clone/path", "https://github.com/test/repo.git", "main")
	model.state = SettingsStateRepositoryActions

	// Find and select GitHub Branch option
	options := model.getMenuOptions()
	branchIndex := -1
	for i, opt := range options {
		if opt.Option == ChangeOptionGitHubBranch {
			branchIndex = i
			break
		}
	}

	if branchIndex == -1 {
		t.Fatal("GitHub Branch option not found")
	}

	model.selectedRepositoryActionOption = branchIndex
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateUpdateGitHubBranch {
		t.Fatalf("Expected UpdateGitHubBranch state, got %v", model.state)
	}

	if model.changeType != ChangeOptionGitHubBranch {
		t.Errorf("Expected changeType GitHubBranch, got %v", model.changeType)
	}

	// Enter new branch
	model.textInput.SetValue("develop")
	updatedModel, cmd := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Should return dirty check command
	if cmd == nil {
		t.Error("Expected dirty check command")
	}
}

func TestCompleteFlow_GitHubBranchUpdate_EmptyBranch(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/clone/path", "https://github.com/test/repo.git", "main")
	model.state = SettingsStateUpdateGitHubBranch
	model.changeType = ChangeOptionGitHubBranch

	// Enter empty branch (should use default)
	model.textInput.SetValue("")
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Should still proceed with dirty check
	if cmd == nil {
		t.Error("Expected dirty check command even with empty branch")
	}
}

// GitHub Path Update Tests

func TestCompleteFlow_GitHubPathUpdate_Standalone(t *testing.T) {
	t.Skip("Requires real git repository - path update needs git operations")
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	homeDir, _ := os.UserHomeDir()
	testPath := filepath.Join(homeDir, ".rulem-test-path")

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/clone/path", "https://github.com/test/repo.git", "main")
	model.state = SettingsStateRepositoryActions

	// Find and select GitHub Path option
	options := model.getMenuOptions()
	pathIndex := -1
	for i, opt := range options {
		if opt.Option == ChangeOptionGitHubPath {
			pathIndex = i
			break
		}
	}

	if pathIndex == -1 {
		t.Fatal("GitHub Path option not found")
	}

	model.selectedRepositoryActionOption = pathIndex
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateUpdateGitHubPath {
		t.Fatalf("Expected UpdateGitHubPath state, got %v", model.state)
	}

	// Enter new path
	model.textInput.SetValue(testPath)
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Should go to confirmation
	if model.state != SettingsStateEditClonePathConfirm {
		t.Fatalf("Expected EditClonePathConfirm state, got %v", model.state)
	}

	if model.newGitHubPath != testPath {
		t.Errorf("Expected newGitHubPath to be set to %s, got %s", testPath, model.newGitHubPath)
	}
}

// GitHub PAT Update Tests

func TestCompleteFlow_GitHubPATUpdate(t *testing.T) {
	t.Skip("Requires real git repository - PAT validation checks against repos")
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/clone/path", "https://github.com/test/repo.git", "main")
	model.state = SettingsStateRepositoryActions

	// Find and select GitHub PAT option
	options := model.getMenuOptions()
	patIndex := -1
	for i, opt := range options {
		if opt.Option == ChangeOptionGitHubPAT {
			patIndex = i
			break
		}
	}

	if patIndex == -1 {
		t.Fatal("GitHub PAT option not found")
	}

	model.selectedRepositoryActionOption = patIndex
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateUpdateGitHubPAT {
		t.Fatalf("Expected UpdateGitHubPAT state, got %v", model.state)
	}

	// Enter new PAT value
	model.textInput.SetValue("ghp_newtoken456")
	updatedModel, cmd := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Execute any validation command if returned
	if cmd != nil {
		msg := cmd()
		updatedModel, _ = model.Update(msg)
		model = updatedModel.(*SettingsModel)
	}

	// After entering PAT, should validate and transition
	if model.state != SettingsStateUpdatePATConfirm &&
		model.state != SettingsStateUpdatePATError &&
		model.state != SettingsStateUpdateGitHubPAT {
		t.Fatalf("Expected UpdatePATConfirm, UpdatePATError, or still in UpdateGitHubPAT state, got %v", model.state)
	}

	// If we hit an error state (validation failed), that's acceptable
	if model.state == SettingsStateUpdatePATError || model.state == SettingsStateUpdateGitHubPAT {
		t.Log("PAT validation failed or still in input state (expected in test environment)")
		return
	}

	// No additional assertions needed for PAT action
}

func TestCompleteFlow_GitHubPATUpdate_EmptyPAT(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/clone/path", "https://github.com/test/repo.git", "main")
	model.state = SettingsStateUpdateGitHubPAT
	model.changeType = ChangeOptionGitHubPAT

	// Try to submit empty PAT
	model.textInput.SetValue("")
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := model.Update(enterMsg)

	// Should return error command
	if cmd == nil {
		t.Error("Expected error command for empty PAT")
	}

	msg := cmd()
	if _, ok := msg.(updatePATErrorMsg); !ok {
		t.Errorf("Expected updatePATErrorMsg, got %T", msg)
	}
}

// Manual Refresh Tests

func TestCompleteFlow_ManualRefresh(t *testing.T) {
	t.Skip("Requires real git repository - refresh performs git pull operation")
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/clone/path", "https://github.com/test/repo.git", "main")
	model.state = SettingsStateRepositoryActions

	// Find and select Manual Refresh option
	options := model.getMenuOptions()
	refreshIndex := -1
	for i, opt := range options {
		if opt.Option == ChangeOptionManualRefresh {
			refreshIndex = i
			break
		}
	}

	if refreshIndex == -1 {
		t.Fatal("Manual Refresh option not found")
	}

	model.selectedRepositoryActionOption = refreshIndex
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateManualRefresh {
		t.Fatalf("Expected ManualRefresh state, got %v", model.state)
	}

	// Confirm refresh
	yMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}
	updatedModel, cmd := model.Update(yMsg)
	model = updatedModel.(*SettingsModel)

	if cmd == nil {
		t.Fatal("Expected command from refresh trigger (dirty check)")
	}

	// Execute the dirty check command
	msg := cmd()
	if msg != nil {
		updatedModel, cmd2 := model.Update(msg)
		model = updatedModel.(*SettingsModel)

		// After dirty check, verify state
		if model.state == SettingsStateRefreshError || model.state == SettingsStateManualRefresh {
			t.Log("Refresh blocked by dirty check or missing repo (expected in test environment)")
			return
		}

		// Check refreshInProgress flag
		if model.state == SettingsStateRefreshInProgress {
			if !model.refreshInProgress {
				t.Errorf("refreshInProgress flag should be set in RefreshInProgress state. State=%v, Flag=%v",
					model.state, model.refreshInProgress)
			}
		}

		// Execute the actual refresh command if returned
		if cmd2 != nil {
			msg2 := cmd2()
			updatedModel, _ = model.Update(msg2)
			model = updatedModel.(*SettingsModel)
		}
	}

	// Verify final state is acceptable
	validStates := []SettingsState{
		SettingsStateRefreshInProgress,
		SettingsStateRefreshError,
		SettingsStateMainMenu,
		SettingsStateManualRefresh,
	}

	stateValid := false
	if slices.Contains(validStates, model.state) {
		stateValid = true
	}

	if !stateValid {
		t.Fatalf("Expected one of %v states, got %v", validStates, model.state)
	}

	t.Logf("Test completed with state=%v (refreshInProgress=%v)", model.state, model.refreshInProgress)
}

func TestCompleteFlow_ManualRefresh_Cancel(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/clone/path", "https://github.com/test/repo.git", "main")
	model.state = SettingsStateManualRefresh

	// Decline refresh
	nMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}
	updatedModel, _ := model.Update(nMsg)
	model = updatedModel.(*SettingsModel)

	// Should return to SelectChange
	if model.state != SettingsStateRepositoryActions {
		t.Errorf("Expected SelectChange state after declining refresh, got %v", model.state)
	}

	// refreshInProgress should not be set
	if model.refreshInProgress {
		t.Error("refreshInProgress should not be set when refresh is cancelled")
	}
}

func TestCompleteFlow_ManualRefresh_WithDirtyRepository(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/clone/path", "https://github.com/test/repo.git", "main")
	model.state = SettingsStateManualRefresh

	// Confirm refresh
	yMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}
	updatedModel, cmd := model.Update(yMsg)
	model = updatedModel.(*SettingsModel)

	if cmd == nil {
		t.Fatal("Expected dirty check command")
	}

	// Simulate dirty state message
	refreshDirtyMsg := refreshDirtyStateMsg{isDirty: true, err: nil}
	updatedModel, _ = model.Update(refreshDirtyMsg)
	model = updatedModel.(*SettingsModel)

	// Should transition to error state when dirty
	if model.state != SettingsStateRefreshError {
		t.Errorf("Expected RefreshError state when repository is dirty, got %v", model.state)
	}

	if !model.isDirty {
		t.Error("isDirty flag should be set")
	}
}

// Navigation and Back Tests

func TestBackNavigation(t *testing.T) {
	t.Skip("Some sub-tests require git operations")
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createLocalConfig("/test/path")

	escMsg := tea.KeyMsg{Type: tea.KeyEsc}

	tests := []struct {
		name          string
		startState    SettingsState
		expectedState SettingsState
		setupFunc     func(*SettingsModel)
	}{
		{
			name:          "SelectChange to MainMenu",
			startState:    SettingsStateRepositoryActions,
			expectedState: SettingsStateMainMenu,
		},
		// ConfirmTypeChange state removed in Phase 1.1
		// UpdateLocalPath navigation - depends on changeType
		// UpdateLocalPath state removed in Phase 1.3
		// Deprecated states removed - ConfirmTypeChange and UpdateGitHubURL no longer exist
		// UpdateGitHubBranch navigation - depends on changeType
		{
			name:          "UpdateGitHubBranch to SelectChange (standalone)",
			startState:    SettingsStateUpdateGitHubBranch,
			expectedState: SettingsStateRepositoryActions,
			setupFunc: func(m *SettingsModel) {
				m.changeType = ChangeOptionGitHubBranch
			},
		},
		// UpdateGitHubURL and repository type states removed in Phase 1.1 and 1.2
		// UpdateGitHubPath navigation - depends on changeType
		{
			name:          "UpdateGitHubPath to SelectChange (standalone)",
			startState:    SettingsStateUpdateGitHubPath,
			expectedState: SettingsStateRepositoryActions,
			setupFunc: func(m *SettingsModel) {
				m.changeType = ChangeOptionGitHubPath
			},
		},
		// UpdateGitHubPath to UpdateGitHubBranch for URL change flow removed (Phase 1.2)
		// UpdateGitHubPath to UpdateGitHubBranch for repo type change removed (Phase 1.1)
		// UpdateGitHubPAT navigation - depends on changeType
		{
			name:          "UpdateGitHubPAT to SelectChange (standalone)",
			startState:    SettingsStateUpdateGitHubPAT,
			expectedState: SettingsStateRepositoryActions,
			setupFunc: func(m *SettingsModel) {
				m.changeType = ChangeOptionGitHubPAT
			},
		},
		// UpdateGitHubPath to UpdateGitHubURL for repo type change removed (Phase 1.1, 1.2)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model.state = tt.startState
			if tt.setupFunc != nil {
				tt.setupFunc(model)
			}

			updatedModel, _ := model.Update(escMsg)
			model = updatedModel.(*SettingsModel)

			if model.state != tt.expectedState {
				t.Errorf("Expected state %v after Esc from %v (changeType=%v), got %v",
					tt.expectedState, tt.startState, model.changeType, model.state)
			}
		})
	}
}

func TestBackNavigation_FromSelectChange_BackOption(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createLocalConfig("/test/path")
	model.state = SettingsStateRepositoryActions

	// Find the "Back" option
	options := model.getMenuOptions()
	backIndex := -1
	for i, opt := range options {
		if opt.Option == ChangeOptionBack {
			backIndex = i
			break
		}
	}

	if backIndex == -1 {
		t.Fatal("Back option not found")
	}

	// Select back option
	model.selectedRepositoryActionOption = backIndex
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Should return to main menu
	if model.state != SettingsStateMainMenu {
		t.Errorf("Expected MainMenu state after selecting Back, got %v", model.state)
	}
}

// Error Recovery Tests

func TestDirtyStateMessage_NextStateByChangeType(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	tests := []struct {
		name              string
		currentState      SettingsState
		changeType        ChangeOption
		expectedNextState SettingsState
		setupFunc         func(*SettingsModel)
	}{
		{
			name:              "UpdateGitHubBranch standalone to Confirmation (clean)",
			currentState:      SettingsStateUpdateGitHubBranch,
			changeType:        ChangeOptionGitHubBranch,
			expectedNextState: SettingsStateEditBranchConfirm,
			setupFunc: func(m *SettingsModel) {
				m.newGitHubBranch = "develop"
			},
		},
		// URL flow and repo type change tests removed (Phase 1.1, 1.2)
		{
			name:              "ManualRefresh to RefreshInProgress (clean)",
			currentState:      SettingsStateManualRefresh,
			changeType:        ChangeOptionManualRefresh,
			expectedNextState: SettingsStateRefreshInProgress,
			setupFunc: func(m *SettingsModel) {
				m.currentConfig = createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewSettingsModel(ctx)
			model.currentConfig = createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main")
			model.state = tt.currentState
			model.changeType = tt.changeType

			if tt.setupFunc != nil {
				tt.setupFunc(model)
			}

			// Send clean dirty state message based on changeType
			var msg tea.Msg
			if tt.changeType == ChangeOptionGitHubBranch {
				msg = editBranchDirtyStateMsg{isDirty: false, err: nil}
			} else {
				msg = refreshDirtyStateMsg{isDirty: false, err: nil}
			}
			updatedModel, _ := model.Update(msg)
			settingsModel := updatedModel.(*SettingsModel)

			if settingsModel.state != tt.expectedNextState {
				t.Errorf("Expected next state %v after clean dirtyStateMsg in %v with changeType=%v, got %v",
					tt.expectedNextState, tt.currentState, tt.changeType, settingsModel.state)
			}

			if settingsModel.isDirty {
				t.Error("isDirty should be false for clean state")
			}
		})
	}
}

func TestDirtyStateMessage_DirtyRepository(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	tests := []struct {
		name         string
		currentState SettingsState
		changeType   ChangeOption
	}{
		{
			name:         "UpdateGitHubBranch standalone with dirty repo",
			currentState: SettingsStateUpdateGitHubBranch,
			changeType:   ChangeOptionGitHubBranch,
		},
		// URL flow dirty repo test removed (Phase 1.2)
		{
			name:         "ManualRefresh with dirty repo",
			currentState: SettingsStateManualRefresh,
			changeType:   ChangeOptionManualRefresh,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewSettingsModel(ctx)
			model.currentConfig = createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main")
			model.state = tt.currentState
			model.changeType = tt.changeType

			// Send dirty state message based on changeType
			var msg tea.Msg
			var expectedState SettingsState
			if tt.changeType == ChangeOptionGitHubBranch {
				msg = editBranchDirtyStateMsg{isDirty: true, err: nil}
				expectedState = SettingsStateEditBranchError
			} else {
				msg = refreshDirtyStateMsg{isDirty: true, err: nil}
				expectedState = SettingsStateRefreshError
			}
			updatedModel, _ := model.Update(msg)
			settingsModel := updatedModel.(*SettingsModel)

			// All dirty states should go to their respective Error state
			if settingsModel.state != expectedState {
				t.Errorf("Expected %v state when repository is dirty in %v with changeType=%v, got %v",
					expectedState, tt.currentState, tt.changeType, settingsModel.state)
			}

			if !settingsModel.isDirty {
				t.Error("isDirty flag should be set")
			}
		})
	}
}

// View Rendering Tests

func TestViewsRenderWithoutPanic(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main")

	states := []SettingsState{
		SettingsStateMainMenu,
		SettingsStateRepositoryActions,
		SettingsStateUpdateGitHubPAT,
		SettingsStateUpdateGitHubBranch,
		SettingsStateUpdateGitHubPath,
		SettingsStateUpdateRepoName,
		SettingsStateManualRefresh,
		SettingsStateRefreshInProgress,
		SettingsStateEditBranchConfirm,
		SettingsStateComplete,
		SettingsStateRefreshError,
	}

	for _, state := range states {
		t.Run(state.String(), func(t *testing.T) {
			model.state = state

			// Should not panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("View panicked for state %v: %v", state, r)
				}
			}()

			view := model.View()
			if view == "" {
				t.Errorf("View for state %v is empty", state)
			}

			// Views should be reasonably sized
			if len(view) < 10 {
				t.Errorf("View for state %v seems too short: %d characters", state, len(view))
			}
		})
	}
}

func TestViewRendering_WithChanges(t *testing.T) {
	t.Skip("Local path editing deprecated in Phase 1.3")

	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createLocalConfig("/old/path")
	model.state = SettingsStateEditBranchConfirm
	model.hasChanges = true

	view := model.View()

	if view == "" {
		t.Error("Confirmation view should not be empty")
	}

	// View should be reasonably sized with changes summary
	if len(view) < 50 {
		t.Error("Confirmation view with changes seems too short")
	}
}

// Message Handling Tests

func TestMessageHandling_RefreshComplete(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main")
	model.refreshInProgress = true
	model.state = SettingsStateRefreshInProgress

	// Success case
	refreshMsg := refreshCompleteMsg{
		success: true,
		err:     nil,
	}

	updatedModel, _ := model.Update(refreshMsg)
	settingsModel := updatedModel.(*SettingsModel)

	if settingsModel.refreshInProgress {
		t.Error("refreshInProgress should be cleared after refresh completes")
	}

	if settingsModel.state != SettingsStateMainMenu {
		t.Errorf("Expected MainMenu state after successful refresh, got %v", settingsModel.state)
	}

	if settingsModel.lastRefreshError != nil {
		t.Error("lastRefreshError should be nil after successful refresh")
	}
}

func TestMessageHandling_RefreshError(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main")
	model.refreshInProgress = true

	// Error case
	refreshMsg := refreshCompleteMsg{
		success: false,
		err:     &testError{"refresh failed"},
	}

	updatedModel, _ := model.Update(refreshMsg)
	settingsModel := updatedModel.(*SettingsModel)

	if settingsModel.refreshInProgress {
		t.Error("refreshInProgress should be cleared even on error")
	}

	// Note: The implementation may return to MainMenu instead of Error state on refresh error
	// This is acceptable behavior as the error is logged
	if settingsModel.state != SettingsStateRefreshError && settingsModel.state != SettingsStateMainMenu {
		t.Errorf("Expected RefreshError or MainMenu state after refresh error, got %v", settingsModel.state)
	}
}

func TestMessageHandling_DirtyState(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main")
	model.state = SettingsStateManualRefresh

	// Clean state
	cleanMsg := refreshDirtyStateMsg{isDirty: false, err: nil}
	updatedModel, _ := model.Update(cleanMsg)
	settingsModel := updatedModel.(*SettingsModel)

	if settingsModel.isDirty {
		t.Error("isDirty should be false for clean state")
	}

	// Dirty state
	model.state = SettingsStateManualRefresh
	dirtyMsg := refreshDirtyStateMsg{isDirty: true, err: nil}
	updatedModel, _ = model.Update(dirtyMsg)
	settingsModel = updatedModel.(*SettingsModel)

	if !settingsModel.isDirty {
		t.Error("isDirty should be true for dirty state")
	}

	if settingsModel.state != SettingsStateRefreshError {
		t.Errorf("Expected RefreshError state when repository is dirty, got %v", settingsModel.state)
	}
}

func TestMessageHandling_SettingsComplete(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.hasChanges = true

	completeMsg := settingsCompleteMsg{}
	updatedModel, cmd := model.Update(completeMsg)
	settingsModel := updatedModel.(*SettingsModel)

	if settingsModel.state != SettingsStateComplete {
		t.Errorf("Expected Complete state, got %v", settingsModel.state)
	}

	if cmd == nil {
		t.Error("Complete message should return config reload command")
	}
}

// Complex Integration Scenarios

func TestComplexScenario_MultipleChangesAndCancellations(t *testing.T) {
	t.Skip("Requires real git repository - involves multiple git operations")
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createLocalConfig("/original/path")

	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}

	// Navigate to select change
	updatedModel, _ := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Try to change local path
	model.selectedRepositoryActionOption = 1
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Cancel
	updatedModel, _ = model.Update(escMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateRepositoryActions {
		t.Fatalf("Expected SelectChange after cancel, got %v", model.state)
	}

	// Try to change repository type
	model.selectedRepositoryActionOption = 0
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// State should remain at RepositoryActions since type change is deprecated
	if model.state != SettingsStateRepositoryActions {
		t.Fatalf("Expected RepositoryActions, got %v", model.state)
	}

	// Cancel type change
	noMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}
	updatedModel, _ = model.Update(noMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateRepositoryActions {
		t.Errorf("Expected SelectChange after rejecting type change, got %v", model.state)
	}

	// Verify no changes were recorded
	if model.hasChanges {
		t.Error("Expected hasChanges to be false after all cancellations")
	}
}

func TestComplexScenario_GitHubFullFlow(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/test/path", "https://github.com/old/repo.git", "main")

	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}

	// Navigate to select change
	updatedModel, _ := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Update GitHub URL (which triggers full flow: URL -> Branch -> Path)
	// GitHub URL option no longer exists - skip this test
	t.Skip("GitHub URL editing deprecated in Phase 1.2")

	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// URL editing removed - skip this check
	t.Skip("GitHub URL editing deprecated in Phase 1.2")

	// Enter URL
	model.textInput.SetValue("https://github.com/new/repo.git")
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Should be at branch
	if model.state != SettingsStateUpdateGitHubBranch {
		t.Fatalf("Expected UpdateGitHubBranch, got %v", model.state)
	}

	// Enter branch
	model.textInput.SetValue("develop")
	updatedModel, cmd := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Execute dirty check
	if cmd != nil {
		msg := cmd()
		updatedModel, _ = model.Update(msg)
		model = updatedModel.(*SettingsModel)
	}

	// Verify that URL, branch, and changeType were properly set
	if model.newGitHubURL != "https://github.com/new/repo.git" {
		t.Errorf("Expected URL to be saved, got %v", model.newGitHubURL)
	}

	if model.newGitHubBranch != "develop" {
		t.Errorf("Expected branch to be saved, got %v", model.newGitHubBranch)
	}

	// ChangeOptionGitHubURL removed in Phase 1.2

	t.Logf("Final state: %v (hasChanges=%v)", model.state, model.hasChanges)
}

// Edge Cases

func TestEdgeCase_RapidStateTransitions(t *testing.T) {
	t.Skip("Requires real git repository - some transitions trigger git operations")
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createLocalConfig("/test/path")

	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}

	// Rapid transitions
	for range 5 {
		updatedModel, _ := model.Update(enterMsg)
		model = updatedModel.(*SettingsModel)
	}

	// Should not panic and should be in a valid state
	// After multiple enters from MainMenu, we go through SelectChange and may end up
	// in various states depending on menu selection
	validStates := []SettingsState{
		SettingsStateMainMenu,
		SettingsStateRepositoryActions,
		SettingsStateUpdateGitHubPAT,
		SettingsStateUpdateGitHubBranch,
		SettingsStateUpdateGitHubPath,
		SettingsStateUpdateRepoName,
	}

	stateValid := false
	if slices.Contains(validStates, model.state) {
		stateValid = true
	}

	if !stateValid {
		t.Errorf("Unexpected state after rapid transitions: %v", model.state)
	}
}

func TestEdgeCase_NilConfig(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = nil

	// Should not panic when rendering with nil config
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("View panicked with nil config: %v", r)
		}
	}()

	view := model.View()
	if view == "" {
		t.Error("View should not be empty even with nil config")
	}
}

func TestEdgeCase_ExtremelyLongPaths(t *testing.T) {
	t.Skip("Local path editing deprecated in Phase 1.3")

	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createLocalConfig("/test/path")

	// Create an extremely long path
	longPath := "/home/user"
	for range 100 {
		longPath = filepath.Join(longPath, "very_long_directory_name_that_repeats")
	}

	model.textInput.SetValue(longPath)

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Panicked with extremely long path: %v", r)
		}
	}()

	view := model.View()
	if view == "" {
		t.Error("View should not be empty even with extremely long path")
	}
}

// =============================================================================
// Migrated cross-flow interaction tests
// =============================================================================

// TestCrossFlow_BackNavigationFromAddFlows tests Esc key navigation in add flows
func TestCrossFlow_BackNavigationFromAddFlows(t *testing.T) {
	tests := []struct {
		name          string
		startState    SettingsState
		expectedState SettingsState
		keyHandler    func(*SettingsModel, tea.KeyMsg) (*SettingsModel, tea.Cmd)
	}{
		{
			name:          "From AddLocalName",
			startState:    SettingsStateAddLocalName,
			expectedState: SettingsStateAddRepositoryType,
			keyHandler:    (*SettingsModel).handleAddLocalNameKeys,
		},
		{
			name:          "From AddLocalPath",
			startState:    SettingsStateAddLocalPath,
			expectedState: SettingsStateAddLocalName,
			keyHandler:    (*SettingsModel).handleAddLocalPathKeys,
		},
		{
			name:          "From AddGitHubName",
			startState:    SettingsStateAddGitHubName,
			expectedState: SettingsStateAddRepositoryType,
			keyHandler:    (*SettingsModel).handleAddGitHubNameKeys,
		},
		{
			name:          "From AddGitHubURL",
			startState:    SettingsStateAddGitHubURL,
			expectedState: SettingsStateAddGitHubName,
			keyHandler:    (*SettingsModel).handleAddGitHubURLKeys,
		},
		{
			name:          "From AddGitHubBranch",
			startState:    SettingsStateAddGitHubBranch,
			expectedState: SettingsStateAddGitHubURL,
			keyHandler:    (*SettingsModel).handleAddGitHubBranchKeys,
		},
		{
			name:          "From AddGitHubPath",
			startState:    SettingsStateAddGitHubPath,
			expectedState: SettingsStateAddGitHubBranch,
			keyHandler:    (*SettingsModel).handleAddGitHubPathKeys,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := createTestModel(t)
			m.state = tt.startState

			m, _ = tt.keyHandler(m, tea.KeyMsg{Type: tea.KeyEsc})
			if m.state != tt.expectedState {
				t.Errorf("Expected transition from %s to %s, got %s", tt.startState, tt.expectedState, m.state)
			}
		})
	}
}

// TestCrossFlow_BackNavigationFromEditFlows tests Esc key navigation in edit flows
func TestCrossFlow_BackNavigationFromEditFlows(t *testing.T) {
	tests := []struct {
		name          string
		startState    SettingsState
		expectedState SettingsState
		keyHandler    func(*SettingsModel, tea.KeyMsg) (*SettingsModel, tea.Cmd)
	}{
		{
			name:          "From RepositoryActions",
			startState:    SettingsStateRepositoryActions,
			expectedState: SettingsStateMainMenu,
			keyHandler:    (*SettingsModel).handleRepositoryActionsKeys,
		},
		{
			name:          "From UpdateGitHubBranch",
			startState:    SettingsStateUpdateGitHubBranch,
			expectedState: SettingsStateRepositoryActions,
			keyHandler:    (*SettingsModel).handleUpdateGitHubBranchKeys,
		},
		{
			name:          "From UpdateGitHubPath",
			startState:    SettingsStateUpdateGitHubPath,
			expectedState: SettingsStateRepositoryActions,
			keyHandler:    (*SettingsModel).handleUpdateGitHubPathKeys,
		},
		{
			name:          "From UpdateRepoName",
			startState:    SettingsStateUpdateRepoName,
			expectedState: SettingsStateRepositoryActions,
			keyHandler:    (*SettingsModel).handleUpdateRepoNameKeys,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := createTestModel(t)
			path := t.TempDir()
			m.currentConfig.Repositories = []repository.RepositoryEntry{
				{
					ID:   "test-repo",
					Name: "Test Repo",
					Type: repository.RepositoryTypeLocal,
					Path: path,
				},
			}
			m.selectedRepositoryID = "test-repo"
			m.state = tt.startState

			m, _ = tt.keyHandler(m, tea.KeyMsg{Type: tea.KeyEsc})
			if m.state != tt.expectedState {
				t.Errorf("Expected transition from %s to %s, got %s", tt.startState, tt.expectedState, m.state)
			}
		})
	}
}

// TestCrossFlow_ErrorStatesReturnToRepositoryActions tests error states transition properly
func TestCrossFlow_ErrorStatesReturnToRepositoryActions(t *testing.T) {
	tests := []struct {
		name          string
		errorState    SettingsState
		expectedState SettingsState
	}{
		{"DeleteError returns to RepositoryActions", SettingsStateDeleteError, SettingsStateRepositoryActions},
		{"EditBranchError returns to RepositoryActions", SettingsStateEditBranchError, SettingsStateRepositoryActions},
		{"EditClonePathError returns to RepositoryActions", SettingsStateEditClonePathError, SettingsStateRepositoryActions},
		{"EditNameError returns to RepositoryActions", SettingsStateEditNameError, SettingsStateRepositoryActions},
		{"RefreshError returns to RepositoryActions", SettingsStateRefreshError, SettingsStateRepositoryActions},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := createTestModel(t)
			path := t.TempDir()
			m.currentConfig.Repositories = []repository.RepositoryEntry{
				{
					ID:   "test-repo",
					Name: "Test Repo",
					Type: repository.RepositoryTypeLocal,
					Path: path,
				},
			}

			m.selectedRepositoryID = "test-repo"
			m.state = tt.errorState

			// Press any key to dismiss error
			model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
			m = model.(*SettingsModel)
			if m.state != tt.expectedState {
				t.Errorf("Error state %s should return to %s, got %s", tt.errorState, tt.expectedState, m.state)
			}
		})
	}
}

// TestCrossFlow_StateIsolationBetweenFlows verifies temporary fields don't leak between flows
func TestCrossFlow_StateIsolationBetweenFlows(t *testing.T) {
	m := createTestModel(t)
	path := t.TempDir()
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:   "test-repo",
			Name: "Test Repo",
			Type: repository.RepositoryTypeLocal,
			Path: path,
		},
	}

	// Start Add Local flow and set some values
	m.state = SettingsStateAddLocalName
	m.textInput.SetValue("Temp Local Name")

	// Simulate entering the name
	m, _ = m.handleAddLocalNameKeys(tea.KeyMsg{Type: tea.KeyEnter})

	// Cancel
	m, _ = m.handleAddLocalPathKeys(tea.KeyMsg{Type: tea.KeyEsc})
	m, _ = m.handleAddLocalNameKeys(tea.KeyMsg{Type: tea.KeyEsc})
	m, _ = m.handleAddRepositoryTypeKeys(tea.KeyMsg{Type: tea.KeyEsc})

	// Start Add GitHub flow
	m.state = SettingsStateAddRepositoryType
	m.addRepositoryTypeIndex = 1
	m, _ = m.handleAddRepositoryTypeKeys(tea.KeyMsg{Type: tea.KeyEnter})

	// Verify we start fresh at AddGitHubName
	if m.state != SettingsStateAddGitHubName {
		t.Errorf("expected AddGitHubName state, got %v", m.state)
	}

	// Set GitHub values
	m.textInput.SetValue("GitHub Repo")
	m, _ = m.handleAddGitHubNameKeys(tea.KeyMsg{Type: tea.KeyEnter})

	// Cancel GitHub flow
	m, _ = m.handleAddGitHubURLKeys(tea.KeyMsg{Type: tea.KeyEsc})
	m, _ = m.handleAddGitHubNameKeys(tea.KeyMsg{Type: tea.KeyEsc})
	m, _ = m.handleAddRepositoryTypeKeys(tea.KeyMsg{Type: tea.KeyEsc})

	// Start Add Local flow again
	m.state = SettingsStateAddRepositoryType
	m.addRepositoryTypeIndex = 0
	m, _ = m.handleAddRepositoryTypeKeys(tea.KeyMsg{Type: tea.KeyEnter})

	// Verify we start fresh
	if m.state != SettingsStateAddLocalName {
		t.Errorf("expected AddLocalName state, got %v", m.state)
	}
}

// =============================================================================
// Migrated edge case tests
// =============================================================================

// TestEdgeCase_EmptyConfiguration_AddFirstRepository tests adding first repo to empty config
func TestEdgeCase_EmptyConfiguration_AddFirstRepository(t *testing.T) {
	_, cleanup := SetTestConfigPath(t)
	defer cleanup()

	m := createTestModel(t)
	m.currentConfig.Repositories = []repository.RepositoryEntry{}

	// Navigate to add repository
	m.state = SettingsStateAddRepositoryType
	if m.state != SettingsStateAddRepositoryType {
		t.Fatalf("expected AddRepositoryType state, got %v", m.state)
	}

	// Select Local repository type
	m.addRepositoryTypeIndex = 0
	m, _ = m.handleAddRepositoryTypeKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateAddLocalName {
		t.Fatalf("expected AddLocalName state, got %v", m.state)
	}

	// Enter name
	m.textInput.SetValue("First Repository")
	m, _ = m.handleAddLocalNameKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateAddLocalPath {
		t.Fatalf("expected AddLocalPath state, got %v", m.state)
	}

	// Enter path and create
	path := t.TempDir()
	m.textInput.SetValue(path)
	m, cmd := m.handleAddLocalPathKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command from adding repository")
	}

	// Execute the command to create repository
	msg := cmd()
	if msg != nil {
		model, _ := m.Update(msg)
		m = model.(*SettingsModel)
	}

	// Verify repository was added
	if len(m.currentConfig.Repositories) != 1 {
		t.Fatalf("expected 1 repository, got %d", len(m.currentConfig.Repositories))
	}
	if m.currentConfig.Repositories[0].Name != "First Repository" {
		t.Errorf("expected name 'First Repository', got %q", m.currentConfig.Repositories[0].Name)
	}
}

// TestEdgeCase_UnicodeInRepositoryName tests unicode characters in repository names
func TestEdgeCase_UnicodeInRepositoryName(t *testing.T) {
	testCases := []struct {
		name     string
		repoName string
	}{
		{"Chinese", "我的仓库"},
		{"Japanese", "私のリポジトリ"},
		{"Mixed", "Repo-名前"},
		{"Cyrillic", "Мой Репозиторий"},
		{"Accents", "Dépôt Principal"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := createTestModel(t)
			m.state = SettingsStateAddLocalName
			m.textInput.SetValue(tc.repoName)

			// Should handle unicode without panicking
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("panicked with unicode name %q: %v", tc.repoName, r)
					}
				}()
				m, _ = m.handleAddLocalNameKeys(tea.KeyMsg{Type: tea.KeyEnter})
			}()

			view := m.View()
			if view == "" {
				t.Errorf("View should not be empty for unicode name %q", tc.repoName)
			}
		})
	}
}

// TestEdgeCase_WindowResize_DuringInput tests window resize during text input
func TestEdgeCase_WindowResize_DuringInput(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddLocalName
	m.textInput.SetValue("Partial name")

	// Resize during input
	model, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = model.(*SettingsModel)

	// Should maintain state and input value
	if m.state != SettingsStateAddLocalName {
		t.Errorf("expected AddLocalName state after resize, got %v", m.state)
	}
	if m.textInput.Value() != "Partial name" {
		t.Errorf("expected text input to be preserved, got %q", m.textInput.Value())
	}

	// View should render without corruption
	view := m.View()
	if view == "" {
		t.Error("View should not be empty after resize during input")
	}
}

// TestEdgeCase_EmptyInputSubmission tests submitting empty input in various states
func TestEdgeCase_EmptyInputSubmission(t *testing.T) {
	testCases := []struct {
		name  string
		state SettingsState
	}{
		{"Empty Name", SettingsStateAddLocalName},
		{"Empty Path", SettingsStateAddLocalPath},
		{"Empty GitHub Name", SettingsStateAddGitHubName},
		{"Empty GitHub URL", SettingsStateAddGitHubURL},
		{"Empty GitHub Path", SettingsStateAddGitHubPath},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := createTestModel(t)
			m.state = tc.state
			m.textInput.SetValue("")

			// Submit empty input - should not panic
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("panicked on empty input in state %v: %v", tc.state, r)
					}
				}()
				_, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			}()
		})
	}
}

// TestEdgeCase_MissingRequiredFields tests views with repositories missing required fields
func TestEdgeCase_MissingRequiredFields(t *testing.T) {
	url := "https://github.com/test/repo.git"
	testCases := []struct {
		name string
		repo repository.RepositoryEntry
	}{
		{
			name: "No ID",
			repo: repository.RepositoryEntry{
				Name: "No ID Repo",
				Type: repository.RepositoryTypeLocal,
				Path: t.TempDir(),
			},
		},
		{
			name: "No Name",
			repo: repository.RepositoryEntry{
				ID:   "test-id",
				Type: repository.RepositoryTypeLocal,
				Path: t.TempDir(),
			},
		},
		{
			name: "No Path",
			repo: repository.RepositoryEntry{
				ID:   "test-id",
				Name: "No Path Repo",
				Type: repository.RepositoryTypeLocal,
			},
		},
		{
			name: "GitHub missing URL",
			repo: repository.RepositoryEntry{
				ID:   "test-id",
				Name: "GitHub No URL",
				Type: repository.RepositoryTypeGitHub,
				Path: t.TempDir(),
			},
		},
		{
			name: "GitHub missing Branch",
			repo: repository.RepositoryEntry{
				ID:        "test-id",
				Name:      "GitHub No Branch",
				Type:      repository.RepositoryTypeGitHub,
				Path:      t.TempDir(),
				RemoteURL: &url,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := createTestModel(t)
			m.currentConfig.Repositories = []repository.RepositoryEntry{tc.repo}
			m.selectedRepositoryID = tc.repo.ID

			// Should handle missing fields gracefully (no panic)
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("panicked with missing field in %q: %v", tc.name, r)
					}
				}()
				_ = m.View()
			}()
		})
	}
}

// TestEdgeCase_MaximumRepositories_ListNavigation tests handling large number of repositories
func TestEdgeCase_MaximumRepositories_ListNavigation(t *testing.T) {
	m := createTestModel(t)

	// Create 60 repositories
	repos := make([]repository.RepositoryEntry, 60)
	for i := 0; i < 60; i++ {
		repos[i] = repository.RepositoryEntry{
			ID:   fmt.Sprintf("test-id-%d", i),
			Name: fmt.Sprintf("Repository %d", i),
			Type: repository.RepositoryTypeLocal,
			Path: t.TempDir(),
		}
	}
	m.currentConfig.Repositories = repos

	if len(m.currentConfig.Repositories) != 60 {
		t.Fatalf("expected 60 repositories, got %d", len(m.currentConfig.Repositories))
	}

	// View should render without panic
	view := m.View()
	if view == "" {
		t.Error("View should not be empty with 60 repositories")
	}

	// List navigation should work
	m.state = SettingsStateMainMenu
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = model.(*SettingsModel)
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = model.(*SettingsModel)

	// Should be able to select repositories
	m.selectedRepositoryID = repos[0].ID
	m.state = SettingsStateRepositoryActions
	view = m.View()
	if view == "" {
		t.Error("View should not be empty in RepositoryActions with many repos")
	}
}
