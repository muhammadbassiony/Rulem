package filepicker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"rulem/internal/filemanager"
)

// writeLayoutFixtures creates a long markdown file (preview much taller than the
// file list) and a tiny one (preview shorter than the list), returning both paths.
func writeLayoutFixtures(t *testing.T) (long, short string) {
	t.Helper()
	dir := t.TempDir()

	long = filepath.Join(dir, "long.md")
	var sb strings.Builder
	sb.WriteString("# Long document\n\n")
	for i := 0; i < 200; i++ {
		sb.WriteString("This is a fairly long paragraph line that is wide enough to fill the preview pane and wrap, producing content far taller than the file list.\n\n")
	}
	if err := os.WriteFile(long, []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("write long fixture: %v", err)
	}

	short = filepath.Join(dir, "short.md")
	if err := os.WriteFile(short, []byte("# Tiny\n\nok\n"), 0o644); err != nil {
		t.Fatalf("write short fixture: %v", err)
	}
	return long, short
}

// applyRender drives the async render pipeline synchronously: it executes the
// render command and feeds the resulting message back through Update, exactly
// as the Bubble Tea runtime would.
func applyRender(t *testing.T, fp *FilePicker, path string, glamour bool) {
	t.Helper()
	msg := fp.renderFileContent(path, false, glamour)()
	m, _ := fp.Update(msg)
	*fp = *m.(*FilePicker)
}

// assertPanesEqualHeight fails if the bordered list and preview panes differ in
// height. This is the core invariant: their bottom borders must always align.
func assertPanesEqualHeight(t *testing.T, fp *FilePicker, ctx string) {
	t.Helper()
	left, right := fp.paneViews()
	lh, rh := lipgloss.Height(left), lipgloss.Height(right)
	if lh != rh {
		t.Errorf("%s: pane heights differ: left(list)=%d right(preview)=%d", ctx, lh, rh)
	}
}

// TestPaneEqualHeight_ToggleFormat reproduces the reported bug: after toggling
// the preview format with 'g', the left (file list) pane rendered shorter than
// the right (preview) pane. The panes must stay the same height in every mode.
func TestPaneEqualHeight_ToggleFormat(t *testing.T) {
	long, _ := writeLayoutFixtures(t)
	files := []filemanager.FileItem{
		{Name: "long.md", Path: long, RepositoryName: "repo"},
	}

	fp := newTestPicker(t, "Save rules file", "Pick a destination", files, 120, 35)

	// Initial render (glamour on by default).
	applyRender(t, fp, long, fp.useGlamour)
	assertPanesEqualHeight(t, fp, "initial render (glamour on)")

	// Press 'g' to switch to plain text — the mode from the bug screenshot.
	m, cmd := fp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	*fp = *m.(*FilePicker)
	if fp.useGlamour {
		t.Fatalf("expected glamour to be toggled off after 'g'")
	}
	if cmd != nil {
		// Execute the re-render command it scheduled and apply the result.
		if msg := cmd(); msg != nil {
			m, _ = fp.Update(msg)
			*fp = *m.(*FilePicker)
		}
	}
	assertPanesEqualHeight(t, fp, "after 'g' (plain text)")

	// Toggle back to glamour.
	m, cmd = fp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	*fp = *m.(*FilePicker)
	if cmd != nil {
		if msg := cmd(); msg != nil {
			m, _ = fp.Update(msg)
			*fp = *m.(*FilePicker)
		}
	}
	assertPanesEqualHeight(t, fp, "after 'g' again (glamour on)")
}

// TestPaneEqualHeight_Matrix exercises the invariant across a range of terminal
// sizes, both format modes, and both a long preview (taller than the list) and
// a tiny preview (shorter than the list). Every combination must keep the two
// panes equal in height.
func TestPaneEqualHeight_Matrix(t *testing.T) {
	long, short := writeLayoutFixtures(t)
	files := []filemanager.FileItem{
		{Name: "long.md", Path: long, RepositoryName: "repo"},
		{Name: "short.md", Path: short, RepositoryName: "repo"},
	}

	fp := newTestPicker(t, "Title", "Subtitle", files, 120, 35)

	sizes := [][2]int{
		{60, 12}, {60, 24}, {80, 20}, {100, 30},
		{120, 35}, {160, 40}, {200, 50}, {73, 17},
	}
	for _, sz := range sizes {
		w, h := sz[0], sz[1]
		fp.SetSize(w, h)
		for _, glamour := range []bool{false, true} {
			for _, path := range []string{long, short} {
				applyRender(t, fp, path, glamour)
				ctx := fmt.Sprintf("size %dx%d glamour=%v %s", w, h, glamour, filepath.Base(path))
				assertPanesEqualHeight(t, fp, ctx)
			}
		}
	}
}
