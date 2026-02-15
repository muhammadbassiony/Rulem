package settingsmenu

import (
	"fmt"
	"strings"
	"testing"

	"rulem/internal/repository"

	tea "github.com/charmbracelet/bubbletea"
)

// TestHandleManualRefreshKeys_Confirm tests confirming manual refresh
func TestHandleManualRefreshKeys_Confirm(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateManualRefresh
	m.selectedRepositoryID = "test-repo-id"

	confirmKeys := []string{"y", "Y", "enter"}

	for _, key := range confirmKeys {
		t.Run("confirm_with_"+key, func(t *testing.T) {
			newModel, cmd := m.handleManualRefreshKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})

			if cmd == nil {
				t.Fatalf("expected non-nil command for key %q", key)
			}
			if newModel.state != SettingsStateManualRefresh {
				t.Fatalf("expected state %v, got %v", SettingsStateManualRefresh, newModel.state)
			}
		})
	}
}

// TestHandleManualRefreshKeys_Cancel tests canceling manual refresh
func TestHandleManualRefreshKeys_Cancel(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateManualRefresh
	m.selectedRepositoryID = "test-repo-id"

	cancelKeys := []string{"n", "N", "esc"}

	for _, key := range cancelKeys {
		t.Run("cancel_with_"+key, func(t *testing.T) {
			var keyMsg tea.KeyMsg
			if key == "esc" {
				keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
			} else {
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
			}

			newModel, cmd := m.handleManualRefreshKeys(keyMsg)

			if newModel.state != SettingsStateRepositoryActions {
				t.Fatalf("expected state %v, got %v", SettingsStateRepositoryActions, newModel.state)
			}
			if cmd != nil {
				t.Fatalf("expected nil command on cancel")
			}
		})
	}
}

// TestHandleManualRefreshKeys_OtherKeys tests that other keys don't change state
func TestHandleManualRefreshKeys_OtherKeys(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateManualRefresh

	otherKeys := []string{"a", "b", "c", "1", "2", "?"}

	for _, key := range otherKeys {
		t.Run("key_"+key, func(t *testing.T) {
			newModel, cmd := m.handleManualRefreshKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})

			if newModel.state != SettingsStateManualRefresh {
				t.Fatalf("expected state %v, got %v", SettingsStateManualRefresh, newModel.state)
			}
			if cmd != nil {
				t.Fatalf("expected nil command for other keys")
			}
		})
	}
}

// TestHandleRefreshInProgressKeys_BlocksInput tests that input is blocked during refresh
func TestHandleRefreshInProgressKeys_BlocksInput(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateRefreshInProgress
	m.refreshInProgress = true

	testKeys := []string{"y", "n", "enter", " ", "a", "b"}

	for _, key := range testKeys {
		t.Run("blocks_"+key, func(t *testing.T) {
			newModel, cmd := m.handleRefreshInProgressKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})

			if newModel.state != SettingsStateRefreshInProgress {
				t.Fatalf("expected state %v, got %v", SettingsStateRefreshInProgress, newModel.state)
			}
			if cmd != nil {
				t.Fatalf("expected nil command during refresh")
			}
		})
	}
}

// TestHandleRefreshInProgressKeys_Escape tests escape key during refresh
func TestHandleRefreshInProgressKeys_Escape(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateRefreshInProgress

	newModel, cmd := m.handleRefreshInProgressKeys(tea.KeyMsg{Type: tea.KeyEsc})

	// Should stay in progress state (escape doesn't cancel)
	if newModel.state != SettingsStateRefreshInProgress {
		t.Fatalf("expected state %v, got %v", SettingsStateRefreshInProgress, newModel.state)
	}
	if cmd != nil {
		t.Fatalf("expected nil command on escape during refresh")
	}
}

// TestHandleRefreshErrorKeys_Dismiss tests dismissing refresh error
func TestHandleRefreshErrorKeys_Dismiss(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateRefreshError
	m.lastRefreshError = fmt.Errorf("test error")

	testKeys := []string{"enter", " ", "a", "esc"}

	for _, key := range testKeys {
		t.Run("dismiss_with_"+key, func(t *testing.T) {
			m.lastRefreshError = fmt.Errorf("test error")

			var keyMsg tea.KeyMsg
			if key == "esc" {
				keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
			} else {
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
			}

			newModel, cmd := m.handleRefreshErrorKeys(keyMsg)

			if newModel.state != SettingsStateRepositoryActions {
				t.Fatalf("expected state %v, got %v", SettingsStateRepositoryActions, newModel.state)
			}
			if newModel.lastRefreshError != nil {
				t.Fatalf("expected lastRefreshError to be cleared")
			}
			if cmd != nil {
				t.Fatalf("expected nil command on dismiss")
			}
		})
	}
}

// TestTriggerRefresh_NonGitHubRepository tests refresh with non-GitHub repo
func TestTriggerRefresh_NonGitHubRepository(t *testing.T) {
	m := createTestModel(t)

	// Add a local repository
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:   "local-repo",
			Name: "Local Test",
			Type: repository.RepositoryTypeLocal,
			Path: "/test/path",
		},
	}
	m.selectedRepositoryID = "local-repo"

	cmd := m.triggerRefresh()
	if cmd == nil {
		t.Fatalf("expected non-nil command from triggerRefresh")
	}

	// Execute command
	msg := cmd()

	// Should return error message
	refreshMsg, ok := msg.(refreshCompleteMsg)
	if !ok {
		t.Fatalf("expected refreshCompleteMsg, got %T", msg)
	}
	if refreshMsg.success {
		t.Fatalf("expected refresh to fail for local repo")
	}
	if refreshMsg.err == nil {
		t.Fatalf("expected an error for local repo refresh")
	}
	if !strings.Contains(refreshMsg.err.Error(), "not a GitHub repository") {
		t.Fatalf("expected error to mention not a GitHub repository")
	}
}

// TestTriggerRefresh_RepositoryNotFound tests refresh with invalid repository ID
func TestTriggerRefresh_RepositoryNotFound(t *testing.T) {
	m := createTestModel(t)
	m.selectedRepositoryID = "nonexistent-repo"

	cmd := m.triggerRefresh()
	if cmd == nil {
		t.Fatalf("expected command from triggerRefresh")
	}

	msg := cmd()

	refreshMsg, ok := msg.(refreshCompleteMsg)
	if !ok {
		t.Fatalf("expected refreshCompleteMsg, got %T", msg)
	}
	if refreshMsg.success {
		t.Fatalf("expected success to be false")
	}
	if refreshMsg.err == nil {
		t.Fatalf("expected an error")
	}
}

// TestTriggerRefresh_MissingRemoteURL tests refresh with GitHub repo missing URL
func TestTriggerRefresh_MissingRemoteURL(t *testing.T) {
	m := createTestModel(t)

	// Add GitHub repository without remote URL
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:   "github-repo",
			Name: "GitHub Test",
			Type: repository.RepositoryTypeGitHub,
			Path: "/test/path",
			// RemoteURL is nil
		},
	}
	m.selectedRepositoryID = "github-repo"

	cmd := m.triggerRefresh()
	if cmd == nil {
		t.Fatalf("expected command from triggerRefresh")
	}

	msg := cmd()

	refreshMsg, ok := msg.(refreshCompleteMsg)
	if !ok {
		t.Fatalf("expected refreshCompleteMsg, got %T", msg)
	}
	if refreshMsg.success {
		t.Fatalf("expected success false")
	}
	if refreshMsg.err == nil {
		t.Fatalf("expected an error")
	}
	if !strings.Contains(refreshMsg.err.Error(), "missing remote URL") {
		t.Fatalf("expected error to mention missing remote URL")
	}
}

// TestTransitionToManualRefresh tests state transition
func TestTransitionToManualRefresh(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateRepositoryActions

	newModel, cmd := m.transitionToManualRefresh()

	if newModel.state != SettingsStateManualRefresh {
		t.Fatalf("expected %v, got %v", SettingsStateManualRefresh, newModel.state)
	}
	if cmd != nil {
		t.Fatalf("expected nil command")
	}
}

// TestViewManualRefresh tests the refresh confirmation view
func TestViewManualRefresh(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateManualRefresh

	view := m.viewManualRefresh()

	expectedContent := []string{
		"Manual Refresh",
		"pull",
		"latest changes",
		"GitHub",
	}

	for _, content := range expectedContent {
		if !strings.Contains(view, content) {
			t.Fatalf("expected view to contain %q", content)
		}
	}
}

// TestViewRefreshInProgress tests the in-progress view
func TestViewRefreshInProgress(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateRefreshInProgress

	view := m.viewRefreshInProgress()

	if view == "" {
		t.Fatalf("expected non-empty view")
	}
	if !strings.Contains(view, "Syncing") {
		t.Fatalf("expected view to mention Syncing")
	}
}

// TestViewRefreshError tests the error view
func TestViewRefreshError(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateRefreshError
	m.lastRefreshError = fmt.Errorf("network timeout")

	view := m.viewRefreshError()

	if !strings.Contains(view, "Failed") {
		t.Fatalf("expected view to contain Failed")
	}
	if !strings.Contains(view, "network timeout") {
		t.Fatalf("expected view to contain network timeout")
	}
	if !strings.Contains(view, "Press any key") {
		t.Fatalf("expected view to contain prompt")
	}
}

// TestViewRefreshError_NoError tests error view with nil error
func TestViewRefreshError_NoError(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateRefreshError
	m.lastRefreshError = nil

	view := m.viewRefreshError()

	if !strings.Contains(view, "Failed") {
		t.Fatalf("expected view to contain Failed")
	}
	if !strings.Contains(view, "Unknown error") {
		t.Fatalf("expected view to contain Unknown error")
	}
}

// TestRefreshFlow_CompleteSuccess tests successful refresh flow
func TestRefreshFlow_CompleteSuccess(t *testing.T) {
	m := createTestModel(t)

	// Setup GitHub repository
	url := "https://github.com/test/repo"
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:        "github-repo",
			Name:      "Test Repo",
			Type:      repository.RepositoryTypeGitHub,
			Path:      t.TempDir(), // Use temp dir for test
			RemoteURL: &url,
		},
	}
	m.selectedRepositoryID = "github-repo"

	// Step 1: Transition to refresh confirmation
	m, _ = m.transitionToManualRefresh()
	if m.state != SettingsStateManualRefresh {
		t.Fatalf("expected %v, got %v", SettingsStateManualRefresh, m.state)
	}

	// Step 2: View should show confirmation
	view := m.viewManualRefresh()
	if !strings.Contains(view, "Manual Refresh") {
		t.Fatalf("expected view to contain Manual Refresh")
	}

	// Step 3: Confirm refresh (triggers dirty check)
	m, cmd := m.handleManualRefreshKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("should trigger dirty check")
	}
}

// TestRefreshFlow_Cancel tests canceling refresh
func TestRefreshFlow_Cancel(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateManualRefresh

	// Cancel with 'n'
	m, cmd := m.handleManualRefreshKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})

	if m.state != SettingsStateRepositoryActions {
		t.Fatalf("expected %v, got %v", SettingsStateRepositoryActions, m.state)
	}
	if cmd != nil {
		t.Fatalf("expected nil command on cancel")
	}
}

// TestRefreshFlow_ErrorHandling tests error state flow
func TestRefreshFlow_ErrorHandling(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateRefreshError
	m.lastRefreshError = fmt.Errorf("test error")

	// View should show error
	view := m.viewRefreshError()
	if !strings.Contains(view, "test error") {
		t.Fatalf("expected view to contain test error")
	}

	// Dismiss error
	m, _ = m.handleRefreshErrorKeys(tea.KeyMsg{Type: tea.KeyEnter})

	if m.state != SettingsStateRepositoryActions {
		t.Fatalf("expected %v, got %v", SettingsStateRepositoryActions, m.state)
	}
	if m.lastRefreshError != nil {
		t.Fatalf("expected lastRefreshError to be cleared")
	}
}

// TestRefreshFlow_StateIsolation tests that refresh doesn't affect other state
func TestRefreshFlow_StateIsolation(t *testing.T) {
	m := createTestModel(t)
	m.selectedRepositoryID = "test-repo"
	m.addRepositoryName = "test-name"
	m.newGitHubBranch = "main"

	// Transition to refresh
	m, _ = m.transitionToManualRefresh()

	// Other state should be unchanged
	if m.selectedRepositoryID != "test-repo" {
		t.Fatalf("expected selected repo preserved")
	}
	if m.addRepositoryName != "test-name" {
		t.Fatalf("expected addRepositoryName preserved")
	}
	if m.newGitHubBranch != "main" {
		t.Fatalf("expected newGitHubBranch preserved")
	}
}

// Benchmarks

func BenchmarkHandleManualRefreshKeys(b *testing.B) {
	m := createTestModel(&testing.T{})
	m.state = SettingsStateManualRefresh
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.handleManualRefreshKeys(msg)
	}
}

func BenchmarkViewManualRefresh(b *testing.B) {
	m := createTestModel(&testing.T{})
	m.state = SettingsStateManualRefresh

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.viewManualRefresh()
	}
}

func BenchmarkTriggerRefresh(b *testing.B) {
	m := createTestModel(&testing.T{})
	url := "https://github.com/test/repo"
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:        "test",
			Type:      repository.RepositoryTypeGitHub,
			RemoteURL: &url,
			Path:      "/tmp/test",
		},
	}
	m.selectedRepositoryID = "test"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.triggerRefresh()
	}
}
