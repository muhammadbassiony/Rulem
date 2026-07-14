package filepicker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"rulem/internal/filemanager"
)

// H1/H2 regression: while the list filter is active, action keybindings
// (f/g/q/enter/left/right) must NOT fire. Every key must be forwarded to the
// filter input instead.
func TestFilePicker_FilterTypingDoesNotTriggerActions(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "alpha.md")
	b := filepath.Join(dir, "beta.md")
	_ = os.WriteFile(a, []byte("ALPHA"), 0644)
	_ = os.WriteFile(b, []byte("BETA"), 0644)

	files := []filemanager.FileItem{{Name: "alpha.md", Path: a}, {Name: "beta.md", Path: b}}
	fp := newTestPicker(t, "t", "", files, 100, 30)
	fp.useGlamour = true
	glamourBefore := fp.useGlamour

	// Enter filtering.
	_, _ = fp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if fp.fileList.FilterState() != list.Filtering {
		t.Fatalf("expected to be in filtering state after '/'")
	}

	// Type action-binding runes; they must become filter text, not fire actions.
	for _, r := range []rune{'f', 'g', 'q'} {
		_, cmd := fp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		// No key while filtering may ask to cancel the picker or select a file.
		if cmd != nil {
			switch cmd().(type) {
			case CancelledMsg:
				t.Fatalf("filter key %q produced CancelledMsg", string(r))
			case FileSelectedMsg:
				t.Fatalf("filter key %q produced FileSelectedMsg", string(r))
			}
		}
	}

	// The filter query should have accumulated exactly the typed runes.
	if got := fp.fileList.FilterValue(); got != "fgq" {
		t.Fatalf("expected filter value %q, got %q", "fgq", got)
	}
	// 'g' must not have toggled the glamour format.
	if fp.useGlamour != glamourBefore {
		t.Fatalf("glamour format toggled while filtering (before=%v after=%v)", glamourBefore, fp.useGlamour)
	}
	// Still filtering (no action key ended it).
	if fp.fileList.FilterState() != list.Filtering {
		t.Fatalf("expected to remain in filtering state")
	}

	// Esc exits filtering (must not cancel the picker).
	_, cmd := fp.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		if _, ok := cmd().(CancelledMsg); ok {
			t.Fatalf("esc during filtering must not cancel the picker")
		}
	}
	if fp.fileList.FilterState() == list.Filtering {
		t.Fatalf("expected esc to exit filtering")
	}

	// Now (not filtering) a plain 'q' cancels the picker via CancelledMsg.
	_, cmd = fp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatalf("expected a command from 'q' after filtering ended")
	}
	if _, ok := cmd().(CancelledMsg); !ok {
		t.Fatalf("expected CancelledMsg from 'q', got %T", cmd())
	}
}

// H2: the quit binding must emit CancelledMsg (not tea.Quit) when not filtering,
// for both 'q' and 'esc'.
func TestFilePicker_QuitEmitsCancelledMsg(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.md")
	_ = os.WriteFile(a, []byte("A"), 0644)
	files := []filemanager.FileItem{{Name: "a.md", Path: a}}
	fp := newTestPicker(t, "t", "", files, 80, 20)

	for _, k := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'q'}},
		{Type: tea.KeyEsc},
	} {
		_, cmd := fp.Update(k)
		if cmd == nil {
			t.Fatalf("expected command for key %v", k)
		}
		if _, ok := cmd().(CancelledMsg); !ok {
			t.Fatalf("expected CancelledMsg for key %v, got %T", k, cmd())
		}
	}
}

// H3: renderFileContent's command closure must not read mutable picker state
// from its background goroutine while the Update goroutine mutates it via
// SetSize. Run with -race to detect the data race this guards against.
func TestFilePicker_RenderSetSizeNoRace(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.md")
	_ = os.WriteFile(a, []byte(strings.Repeat("# Heading\nsome text\n", 500)), 0644)
	files := []filemanager.FileItem{{Name: "a.md", Path: a}}
	fp := newTestPicker(t, "t", "", files, 100, 30)
	fp.glamourStyle = "dark"
	fp.fileList.Select(0)

	// Kick off a render on a background goroutine, exactly as Bubble Tea runs
	// commands, while the "Update" goroutine concurrently resizes the picker.
	cmd := fp.renderFileContent(a, false, true)
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()

	for i := 0; i < 50; i++ {
		fp.SetSize(80+i, 24)
	}

	<-done
}

// M1: a render that was scheduled before a resize (which clears the cache and
// bumps its generation) is word-wrapped to a now-stale width. When it completes
// it must be neither applied to the viewport nor cached.
func TestFilePicker_StaleGenerationRenderDropped(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.md")
	_ = os.WriteFile(a, []byte("OLDWIDTHCONTENT"), 0644)
	files := []filemanager.FileItem{{Name: "a.md", Path: a}}
	fp := newTestPicker(t, "t", "", files, 100, 30)
	fp.useGlamour = false
	fp.fileList.Select(0)

	// A preview render is already in flight (captures the current generation).
	inflight := fp.renderFileContent(a, false, false)
	genBefore := fp.contentCache.generation

	// A resize arrives before that render completes: clears the cache, bumps the
	// generation and schedules a fresh preview at the new width.
	_, _ = fp.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	if fp.contentCache.generation == genBefore {
		t.Fatalf("expected resize to bump the cache generation")
	}

	// The stale in-flight render now completes and is delivered.
	staleMsg, ok := inflight().(FileRenderedMsg)
	if !ok {
		t.Fatalf("expected FileRenderedMsg from in-flight render")
	}
	if staleMsg.generation != genBefore {
		t.Fatalf("expected stale render to carry pre-resize generation %d, got %d", genBefore, staleMsg.generation)
	}
	_, _ = fp.Update(staleMsg)

	// It must be neither applied to the viewport nor cached.
	if strings.Contains(fp.viewport.View(), "OLDWIDTHCONTENT") {
		t.Fatalf("stale-width render was applied to the viewport")
	}
	if _, ok := fp.contentCache.Get(staleMsg.cacheKey); ok {
		t.Fatalf("stale-width render was cached")
	}

	// A fresh render (current generation) is applied and cached normally.
	freshMsg, ok := fp.renderFileContent(a, false, false)().(FileRenderedMsg)
	if !ok {
		t.Fatalf("expected FileRenderedMsg from fresh render")
	}
	if freshMsg.generation != fp.contentCache.generation {
		t.Fatalf("fresh render should carry the current generation")
	}
	_, _ = fp.Update(freshMsg)
	if !strings.Contains(fp.viewport.View(), "OLDWIDTHCONTENT") {
		t.Fatalf("fresh render should have been applied to the viewport")
	}
	if _, ok := fp.contentCache.Get(freshMsg.cacheKey); !ok {
		t.Fatalf("fresh render should have been cached")
	}
}
