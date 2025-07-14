package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func createTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	return path
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestFileViewerMessagesAndNavigation(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "a.md", "desc A")
	m := NewFileViewerModel(dir)
	// Set and check error message
	m.message = "This is an error"
	m.messageType = "error"
	out := m.View()
	if want := "This is an error"; !strings.Contains(out, want) {
		t.Errorf("expected error message in view output")
	}
	// Set and check success message
	m.message = "Success!"
	m.messageType = "success"
	out = m.View()
	if want := "Success!"; !strings.Contains(out, want) {
		t.Errorf("expected success message in view output")
	}
	// Simulate navigation: select file, go to next step, then 'q' to quit
	m.selected[0] = struct{}{}
	m.mode = modeChooseAction
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = m2.(fileViewerModel)
	if m.mode != modeSelectFiles {
		t.Errorf("expected modeSelectFiles after 'q', got %v", m.mode)
	}
}

func TestListInstructionFiles(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "a.md", "desc A\nrest")
	createTestFile(t, dir, "b.md", "desc B\nrest")
	createTestFile(t, dir, "empty.md", "")
	files, err := listInstructionFiles(dir)
	if err != nil {
		t.Fatalf("listInstructionFiles error: %v", err)
	}
	if len(files) != 3 {
		t.Errorf("expected 3 files, got %d", len(files))
	}
	m := map[string]string{}
	for _, f := range files {
		m[f.Name] = f.Description
	}
	if m["a.md"] != "desc A" {
		t.Errorf("expected 'desc A', got '%s'", m["a.md"])
	}
	if m["b.md"] != "desc B" {
		t.Errorf("expected 'desc B', got '%s'", m["b.md"])
	}
	if m["empty.md"] != "(no description)" && m["empty.md"] != "" {
		t.Errorf("expected '(no description)' or '', got '%s'", m["empty.md"])
	}
}

func TestFileViewerSelection(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "a.md", "desc A")
	createTestFile(t, dir, "b.md", "desc B")
	m := NewFileViewerModel(dir)
	if m.cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", m.cursor)
	}
	// Move down
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = m2.(fileViewerModel)
	if m.cursor != 1 {
		t.Errorf("expected cursor at 1, got %d", m.cursor)
	}
	// Select file
	m2, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m = m2.(fileViewerModel)
	if _, ok := m.selected[1]; !ok {
		t.Error("expected file at cursor 1 to be selected")
	}
	// Deselect file
	m2, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m = m2.(fileViewerModel)
	if _, ok := m.selected[1]; ok {
		t.Error("expected file at cursor 1 to be deselected")
	}
}
