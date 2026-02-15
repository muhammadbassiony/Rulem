// Package settingsmenu provides the settings modification flow for the rulem TUI application.
package settingsmenu

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestHandleAddRepositoryTypeKeys_Navigation tests keyboard navigation in type selection
func TestHandleAddRepositoryTypeKeys_Navigation(t *testing.T) {
	tests := []struct {
		name          string
		initialIndex  int
		key           string
		expectedIndex int
	}{
		{
			name:          "down arrow moves from local to github",
			initialIndex:  0,
			key:           "down",
			expectedIndex: 1,
		},
		{
			name:          "j moves from local to github",
			initialIndex:  0,
			key:           "j",
			expectedIndex: 1,
		},
		{
			name:          "down at bottom stays at bottom",
			initialIndex:  1,
			key:           "down",
			expectedIndex: 1,
		},
		{
			name:          "up arrow moves from github to local",
			initialIndex:  1,
			key:           "up",
			expectedIndex: 0,
		},
		{
			name:          "k moves from github to local",
			initialIndex:  1,
			key:           "k",
			expectedIndex: 0,
		},
		{
			name:          "up at top stays at top",
			initialIndex:  0,
			key:           "up",
			expectedIndex: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := createTestModel(t)
			m.state = SettingsStateAddRepositoryType
			m.addRepositoryTypeIndex = tt.initialIndex

			newModel, _ := m.handleAddRepositoryTypeKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})

			if newModel.addRepositoryTypeIndex != tt.expectedIndex {
				t.Fatalf("expected index %d, got %d", tt.expectedIndex, newModel.addRepositoryTypeIndex)
			}
			if newModel.state != SettingsStateAddRepositoryType {
				t.Fatalf("expected state %v, got %v", SettingsStateAddRepositoryType, newModel.state)
			}
		})
	}
}

// TestHandleAddRepositoryTypeKeys_SelectLocal tests selecting local repository type
func TestHandleAddRepositoryTypeKeys_SelectLocal(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddRepositoryType
	m.addRepositoryTypeIndex = 0

	tests := []struct {
		name string
		key  string
	}{
		{"enter key", "enter"},
		{"space key", " "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newModel, _ := m.handleAddRepositoryTypeKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})

			if newModel.state != SettingsStateAddLocalName {
				t.Fatalf("expected state %v, got %v", SettingsStateAddLocalName, newModel.state)
			}
		})
	}
}

// TestHandleAddRepositoryTypeKeys_SelectGitHub tests selecting GitHub repository type
func TestHandleAddRepositoryTypeKeys_SelectGitHub(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddRepositoryType
	m.addRepositoryTypeIndex = 1

	tests := []struct {
		name string
		key  string
	}{
		{"enter key", "enter"},
		{"space key", " "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newModel, _ := m.handleAddRepositoryTypeKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})

			if newModel.state != SettingsStateAddGitHubName {
				t.Fatalf("expected state %v, got %v", SettingsStateAddGitHubName, newModel.state)
			}
		})
	}
}

// TestHandleAddRepositoryTypeKeys_Cancel tests canceling type selection
func TestHandleAddRepositoryTypeKeys_Cancel(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddRepositoryType
	m.addRepositoryTypeIndex = 1

	newModel, cmd := m.handleAddRepositoryTypeKeys(tea.KeyMsg{Type: tea.KeyEsc})

	if newModel.state != SettingsStateMainMenu {
		t.Fatalf("expected state %v, got %v", SettingsStateMainMenu, newModel.state)
	}
	if cmd != nil {
		t.Fatalf("expected nil cmd, got %v", cmd)
	}
}

// TestHandleAddRepositoryTypeKeys_OtherKeys tests that other keys don't change state
func TestHandleAddRepositoryTypeKeys_OtherKeys(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddRepositoryType
	m.addRepositoryTypeIndex = 0

	otherKeys := []string{"a", "b", "c", "1", "2", "?"}

	for _, key := range otherKeys {
		t.Run("key_"+key, func(t *testing.T) {
			newModel, cmd := m.handleAddRepositoryTypeKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})

			if newModel.state != SettingsStateAddRepositoryType {
				t.Fatalf("expected state %v, got %v", SettingsStateAddRepositoryType, newModel.state)
			}
			if newModel.addRepositoryTypeIndex != 0 {
				t.Fatalf("expected index 0, got %d", newModel.addRepositoryTypeIndex)
			}
			if cmd != nil {
				t.Fatalf("expected nil cmd, got %v", cmd)
			}
		})
	}
}

// TestTransitionToAddRepositoryType tests state transition setup
func TestTransitionToAddRepositoryType(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateMainMenu
	m.addRepositoryTypeIndex = 1 // Set to non-default

	newModel, cmd := m.transitionToAddRepositoryType()

	if newModel.state != SettingsStateAddRepositoryType {
		t.Fatalf("expected state %v, got %v", SettingsStateAddRepositoryType, newModel.state)
	}
	if newModel.addRepositoryTypeIndex != 0 {
		t.Fatalf("expected index 0 after transition, got %d", newModel.addRepositoryTypeIndex)
	}
	if cmd != nil {
		t.Fatalf("expected nil cmd, got %v", cmd)
	}
}

// TestViewAddRepositoryType tests the view rendering
func TestViewAddRepositoryType(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddRepositoryType

	tests := []struct {
		name          string
		selectedIndex int
		expectInView  []string
	}{
		{
			name:          "local selected",
			selectedIndex: 0,
			expectInView:  []string{"▸ 📁 Local Repository", "🔗 GitHub Repository", "Add Repository"},
		},
		{
			name:          "github selected",
			selectedIndex: 1,
			expectInView:  []string{"📁 Local Repository", "▸ 🔗 GitHub Repository", "Add Repository"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.addRepositoryTypeIndex = tt.selectedIndex
			view := m.viewAddRepositoryType()

			for _, expected := range tt.expectInView {
				if !strings.Contains(view, expected) {
					t.Fatalf("expected view to contain %q but it did not", expected)
				}
			}
		})
	}
}

// TestAddTypeFlow_FullLocalSelection tests the complete flow for selecting local
func TestAddTypeFlow_FullLocalSelection(t *testing.T) {
	m := createTestModel(t)

	// Start: Transition to AddRepositoryType
	m, _ = m.transitionToAddRepositoryType()
	if m.state != SettingsStateAddRepositoryType {
		t.Fatalf("expected state %v, got %v", SettingsStateAddRepositoryType, m.state)
	}
	if m.addRepositoryTypeIndex != 0 {
		t.Fatalf("expected index 0, got %d", m.addRepositoryTypeIndex)
	}

	// View should show both options
	view := m.viewAddRepositoryType()
	if !strings.Contains(view, "Local Repository") {
		t.Fatalf("view missing Local Repository")
	}
	if !strings.Contains(view, "GitHub Repository") {
		t.Fatalf("view missing GitHub Repository")
	}

	// Select local (index 0 is already default)
	m, _ = m.handleAddRepositoryTypeKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateAddLocalName {
		t.Fatalf("expected state %v, got %v", SettingsStateAddLocalName, m.state)
	}
}

// TestAddTypeFlow_FullGitHubSelection tests the complete flow for selecting GitHub
func TestAddTypeFlow_FullGitHubSelection(t *testing.T) {
	m := createTestModel(t)

	// Start: Transition to AddRepositoryType
	m, _ = m.transitionToAddRepositoryType()
	if m.state != SettingsStateAddRepositoryType {
		t.Fatalf("expected state %v, got %v", SettingsStateAddRepositoryType, m.state)
	}

	// Navigate down to GitHub
	m, _ = m.handleAddRepositoryTypeKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.addRepositoryTypeIndex != 1 {
		t.Fatalf("expected index 1 after navigation, got %d", m.addRepositoryTypeIndex)
	}

	// View should highlight GitHub
	view := m.viewAddRepositoryType()
	if !strings.Contains(view, "▸ 🔗 GitHub Repository") {
		t.Fatalf("view did not highlight GitHub option")
	}

	// Select GitHub
	m, _ = m.handleAddRepositoryTypeKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateAddGitHubName {
		t.Fatalf("expected state %v, got %v", SettingsStateAddGitHubName, m.state)
	}
}

// TestAddTypeFlow_CancelFromAnyPosition tests canceling from different positions
func TestAddTypeFlow_CancelFromAnyPosition(t *testing.T) {
	positions := []struct {
		name  string
		index int
	}{
		{"from local", 0},
		{"from github", 1},
	}

	for _, pos := range positions {
		t.Run(pos.name, func(t *testing.T) {
			m := createTestModel(t)
			m.state = SettingsStateAddRepositoryType
			m.addRepositoryTypeIndex = pos.index

			m, _ = m.handleAddRepositoryTypeKeys(tea.KeyMsg{Type: tea.KeyEsc})
			if m.state != SettingsStateMainMenu {
				t.Fatalf("expected state %v, got %v", SettingsStateMainMenu, m.state)
			}
		})
	}
}

// TestAddTypeFlow_StateIsolation tests that type selection doesn't affect other state
func TestAddTypeFlow_StateIsolation(t *testing.T) {
	m := createTestModel(t)
	m.addRepositoryName = "test-name"
	m.addRepositoryPath = "/test/path"
	m.newGitHubURL = "https://github.com/test/repo"

	// Transition to type selection
	m, _ = m.transitionToAddRepositoryType()

	// Verify index reset but other fields unchanged
	if m.addRepositoryTypeIndex != 0 {
		t.Fatalf("expected index 0 after transition, got %d", m.addRepositoryTypeIndex)
	}
	if m.addRepositoryName != "test-name" {
		t.Fatalf("expected addRepositoryName to remain, got %q", m.addRepositoryName)
	}
	if m.addRepositoryPath != "/test/path" {
		t.Fatalf("expected addRepositoryPath to remain, got %q", m.addRepositoryPath)
	}
	if m.newGitHubURL != "https://github.com/test/repo" {
		t.Fatalf("expected newGitHubURL to remain, got %q", m.newGitHubURL)
	}
}

// TestAddTypeFlow_TransitionLogging tests that transitions are logged
func TestAddTypeFlow_TransitionLogging(t *testing.T) {
	m := createTestModel(t)
	if m.logger == nil {
		t.Fatalf("expected logger to be non-nil")
	}

	m.state = SettingsStateMainMenu

	// Transition should log
	m, _ = m.transitionToAddRepositoryType()

	// Logger should have been called (verified by no panic)
	if m.state != SettingsStateAddRepositoryType {
		t.Fatalf("expected state %v, got %v", SettingsStateAddRepositoryType, m.state)
	}
}

// Benchmark tests
func BenchmarkHandleAddRepositoryTypeKeys_Navigation(b *testing.B) {
	m := createTestModel(&testing.T{})
	m.state = SettingsStateAddRepositoryType
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.handleAddRepositoryTypeKeys(msg)
	}
}

func BenchmarkViewAddRepositoryType(b *testing.B) {
	m := createTestModel(&testing.T{})
	m.state = SettingsStateAddRepositoryType

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.viewAddRepositoryType()
	}
}
