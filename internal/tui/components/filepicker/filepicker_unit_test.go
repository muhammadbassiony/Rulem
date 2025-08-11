package filepicker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"rulem/internal/filemanager"
	"rulem/internal/logging"
	"rulem/internal/tui/helpers"
	"rulem/internal/tui/styles"
)

func newTestPicker(t *testing.T, title, subtitle string, files []filemanager.FileItem, w, h int) *FilePicker {
	t.Helper()
	logger, _ := logging.NewTestLogger()
	fp := NewFilePicker(title, subtitle, files, helpers.UIContext{
		Width:  w,
		Height: h,
		Logger: logger,
	})
	// speed up tests that rely on debounce
	fp.debounceDuration = 1 * time.Millisecond
	return &fp
}

func TestLRUCache_AddGetEvictUpdateClear(t *testing.T) {
	c := newLRU(10) // 10 bytes cap

	// Add two 5-byte entries
	c.Add("a", "12345")
	c.Add("b", "12345")

	if v, ok := c.Get("a"); !ok || v != "12345" {
		t.Fatalf("expected to get key 'a'")
	}
	if v, ok := c.Get("b"); !ok || v != "12345" {
		t.Fatalf("expected to get key 'b'")
	}

	// Add 3-byte entry, should evict LRU ("a")
	c.Add("c", "123")
	if _, ok := c.Get("a"); ok {
		t.Fatalf("expected 'a' to be evicted")
	}
	if _, ok := c.Get("b"); !ok {
		t.Fatalf("expected 'b' to still exist")
	}
	if _, ok := c.Get("c"); !ok {
		t.Fatalf("expected 'c' to exist")
	}

	// Update existing key size should reflect and maintain capacity
	c.Add("b", "12") // shrink
	if v, ok := c.Get("b"); !ok || v != "12" {
		t.Fatalf("expected updated 'b' value")
	}

	// Clear resets state
	c.Clear()
	if _, ok := c.Get("b"); ok {
		t.Fatalf("expected cache to be cleared")
	}
	if _, ok := c.Get("c"); ok {
		t.Fatalf("expected cache to be cleared")
	}
}

func TestLRUCache_SkipTooLarge(t *testing.T) {
	c := newLRU(5)
	c.Add("big", "0123456") // > cap, should be skipped
	if _, ok := c.Get("big"); ok {
		t.Fatalf("expected oversize entry to be skipped")
	}
	c.Add("ok", "0123")
	if _, ok := c.Get("ok"); !ok {
		t.Fatalf("expected smaller entry to be cached")
	}
}

func TestDetectGlamourStyle_EnvOverride(t *testing.T) {
	old := os.Getenv("GLAMOUR_STYLE")
	t.Cleanup(func() { _ = os.Setenv("GLAMOUR_STYLE", old) })

	_ = os.Setenv("GLAMOUR_STYLE", "dracula")
	style := detectGlamourStyle(1 * time.Millisecond)
	if style != "dracula" {
		t.Fatalf("expected env override style 'dracula', got %q", style)
	}
}

func TestNewFilePicker_InitialSizingAndState(t *testing.T) {
	files := []filemanager.FileItem{
		{Name: "a.md", Path: filepath.Join(t.TempDir(), "a.md")},
	}
	fp := newTestPicker(t, "Unit Title", "Sub", files, 120, 40)
	if fp.title != "Unit Title" || fp.subtitle != "Sub" {
		t.Fatalf("title/subtitle not set correctly")
	}
	if fp.fileList.Width() != 120 || fp.fileList.Height() != 40 {
		t.Fatalf("fileList initial size mismatch got %dx%d", fp.fileList.Width(), fp.fileList.Height())
	}
	if fp.viewport.Width != 120 || fp.viewport.Height != 40 {
		t.Fatalf("viewport initial size mismatch got %dx%d", fp.viewport.Width, fp.viewport.Height)
	}
}

func TestWindowResize_HeightsMatchAndWidthsSum(t *testing.T) {
	files := []filemanager.FileItem{
		{Name: "a.md", Path: filepath.Join(t.TempDir(), "a.md")},
	}
	fp := newTestPicker(t, "Unit Title", "Subtitle here", files, 100, 30)

	// Simulate a resize
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	_, _ = fp.Update(msg)

	// Heights should match
	if fp.fileList.Height() != fp.viewport.Height {
		t.Fatalf("list and viewport heights do not match: %d vs %d", fp.fileList.Height(), fp.viewport.Height)
	}

	// Widths should approximately sum to available width minus frames and left margin
	frameW, _ := styles.PaneStyle.GetFrameSize()
	const mainLeftMargin = 1
	avail := msg.Width - (2 * frameW) - mainLeftMargin
	sum := fp.fileList.Width() + fp.viewport.Width
	if sum != avail {
		t.Fatalf("expected widths sum %d, got %d (list=%d, vp=%d, frameW=%d)", avail, sum, fp.fileList.Width(), fp.viewport.Width, frameW)
	}

	// Header should still render
	out := fp.View()
	if !strings.Contains(out, "Unit Title") {
		t.Fatalf("expected title to be present in view")
	}
}

func TestView_RendersTitleSubtitleAndHelp(t *testing.T) {
	files := []filemanager.FileItem{
		{Name: "a.md", Path: filepath.Join(t.TempDir(), "a.md")},
	}
	fp := newTestPicker(t, "My Title", "My Subtitle", files, 100, 30)

	// Trigger sizing so help width is set
	_, _ = fp.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	out := fp.View()
	if !strings.Contains(out, "My Title") {
		t.Fatalf("expected title in view")
	}
	if !strings.Contains(out, "My Subtitle") {
		t.Fatalf("expected subtitle in view")
	}
	// help should include some known bindings' help text
	if !strings.Contains(out, "filter") && !strings.Contains(out, "toggle format") {
		t.Fatalf("expected help hints to be present in view output")
	}

	// Also ensure the output has some left margin/padding effect (spaces) before panes/header
	// This is a heuristic check; we expect some leading spaces due to margins.
	lines := strings.Split(out, "\n")
	foundIndented := false
	for _, ln := range lines {
		if strings.HasPrefix(ln, " ") && strings.Contains(ln, "My Title") {
			foundIndented = true
			break
		}
	}
	if !foundIndented {
		t.Log("warning: could not confirm left margin for header heuristically")
	}
}

func TestScheduleDebouncedPreviewSetsLoading(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "f.md")
	if err := os.WriteFile(tmp, []byte("# Hi"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	files := []filemanager.FileItem{{Name: "f.md", Path: tmp}}
	fp := newTestPicker(t, "t", "", files, 80, 20)

	fp.scheduleDebouncedPreview(tmp)
	if !fp.isLoading || fp.loadingPath != tmp {
		t.Fatalf("expected loading state to be set")
	}
	if !strings.Contains(fp.viewport.View(), "📄 Loading") {
		t.Fatalf("expected loading content in viewport")
	}

	// Execute the command; it will tick after 1ms
	time.Sleep(2 * time.Millisecond)
}

func TestFileRenderedMsg_AppliesOnlyForCurrentSelectionAndNotStale(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.md")
	b := filepath.Join(dir, "b.md")
	_ = os.WriteFile(a, []byte("AAA"), 0644)
	_ = os.WriteFile(b, []byte("BBB"), 0644)

	files := []filemanager.FileItem{{Name: "a.md", Path: a}, {Name: "b.md", Path: b}}
	fp := newTestPicker(t, "t", "", files, 80, 20)

	// Select first item (a) explicitly
	fp.fileList.Select(0)

	// Set some initial content
	fp.viewport.SetContent("OLD")
	fp.currentRenderID = 5

	// Message for b (not selected) should not change viewport, but content is cached
	msgNotSelected := FileRenderedMsg{
		content:  "CONTENT_B",
		path:     b,
		renderID: 6,
		cacheKey: fp.cacheKey(b, false, fp.useGlamour),
	}
	_, _ = fp.Update(msgNotSelected)
	if fp.viewport.View() == "CONTENT_B" {
		t.Fatalf("viewport should not update for non-selected path")
	}
	if _, ok := fp.contentCache.Get(fp.cacheKey(b, false, fp.useGlamour)); !ok {
		t.Fatalf("expected content for b to be cached")
	}

	// Message for selected (a) but stale renderID should not change viewport
	msgStale := FileRenderedMsg{
		content:  "STALE_A",
		path:     a,
		renderID: 3, // stale
		cacheKey: fp.cacheKey(a, false, fp.useGlamour),
	}
	_, _ = fp.Update(msgStale)
	if fp.viewport.View() == "STALE_A" {
		t.Fatalf("viewport should not update with stale content")
	}

	// Fresh message for selected should apply
	msgFresh := FileRenderedMsg{
		content:  "FRESH_A",
		path:     a,
		renderID: 7,
		cacheKey: fp.cacheKey(a, false, fp.useGlamour),
	}
	_, _ = fp.Update(msgFresh)
	if !strings.Contains(fp.viewport.View(), "FRESH_A") {
		t.Fatalf("viewport should update with fresh content")
	}
}

func TestFileReadErrorMsg_AppliesOnlyForCurrentSelection(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.md")
	b := filepath.Join(dir, "b.md")
	_ = os.WriteFile(a, []byte("AAA"), 0644)
	_ = os.WriteFile(b, []byte("BBB"), 0644)

	files := []filemanager.FileItem{{Name: "a.md", Path: a}, {Name: "b.md", Path: b}}
	fp := newTestPicker(t, "t", "", files, 80, 20)
	fp.fileList.Select(0) // select a

	fp.viewport.SetContent("OLD_ERR")
	fp.currentRenderID = 1

	// Error for b (not selected) ignored
	_, _ = fp.Update(FileReadErrorMsg{err: os.ErrNotExist, path: b, renderID: 2})
	if !strings.Contains(fp.viewport.View(), "OLD_ERR") {
		t.Fatalf("viewport should not update for error on non-selected path")
	}

	// Error for selected with fresh id should update
	_, _ = fp.Update(FileReadErrorMsg{err: os.ErrNotExist, path: a, renderID: 3})
	if !strings.Contains(fp.viewport.View(), "Error reading file") {
		t.Fatalf("expected error message in viewport")
	}
}

func TestRenderFileContent_SmallAndLarge(t *testing.T) {
	dir := t.TempDir()
	small := filepath.Join(dir, "s.md")
	large := filepath.Join(dir, "l.md")
	if err := os.WriteFile(small, []byte("# Small\nHello"), 0644); err != nil {
		t.Fatalf("write small: %v", err)
	}
	if err := os.WriteFile(large, []byte(strings.Repeat("L\n", 5000)), 0644); err != nil {
		t.Fatalf("write large: %v", err)
	}

	fp := newTestPicker(t, "t", "", []filemanager.FileItem{{Name: "s.md", Path: small}}, 80, 20)
	// Make sure viewport has a sensible width for wrapping calculations inside renderer
	fp.viewport.Width = 80

	// Small (no truncation)
	msgSmall := fp.renderFileContent(small, false, false)()
	frSmall, ok := msgSmall.(FileRenderedMsg)
	if !ok {
		t.Fatalf("expected FileRenderedMsg for small file")
	}
	if !strings.Contains(frSmall.content, "Hello") {
		t.Fatalf("expected small content in rendered output")
	}
	if got := frSmall.cacheKey; got != fp.cacheKey(small, true, false) {
		t.Fatalf("unexpected cacheKey, got=%q", got)
	}

	// Large (truncated)
	fp.largeFileThreshold = 32
	fp.maxPreviewBytes = 16
	msgLarge := fp.renderFileContent(large, false, false)()
	frLarge, ok := msgLarge.(FileRenderedMsg)
	if !ok {
		t.Fatalf("expected FileRenderedMsg for large file")
	}
	if !strings.Contains(frLarge.content, "Press 'f' to load full.") {
		t.Fatalf("expected truncated header advisory")
	}
	if got := frLarge.cacheKey; got != fp.cacheKey(large, false, false) {
		t.Fatalf("unexpected cacheKey for large truncation, got=%q", got)
	}
}

func TestDebouncedPreviewMsg_SequenceMismatchIgnored(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.md")
	_ = os.WriteFile(a, []byte("AAA"), 0644)
	files := []filemanager.FileItem{{Name: "a.md", Path: a}}
	fp := newTestPicker(t, "t", "", files, 80, 20)
	fp.fileList.Select(0)

	// Pre-set viewport content
	fp.viewport.SetContent("UNCHANGED")
	fp.pendingDebounceID = 2 // latest seq is 2

	// Send older seq
	_, _ = fp.Update(debouncedPreviewMsg{path: a, seq: 1})

	if !strings.Contains(fp.viewport.View(), "UNCHANGED") {
		t.Fatalf("viewport should remain unchanged for stale debounced preview")
	}
}

func TestFocusKeys_ChangeFocusPane(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.md")
	_ = os.WriteFile(a, []byte("AAA"), 0644)
	files := []filemanager.FileItem{{Name: "a.md", Path: a}}
	fp := newTestPicker(t, "t", "", files, 80, 20)

	if fp.focusPane != focusList {
		t.Fatalf("expected initial focus to be list")
	}

	_, _ = fp.Update(tea.KeyMsg{Type: tea.KeyRight})
	if fp.focusPane != focusPreview {
		t.Fatalf("expected focus to move to preview")
	}

	_, _ = fp.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if fp.focusPane != focusList {
		t.Fatalf("expected focus to move back to list")
	}
}

func TestToggleFormat_UsesCacheWhenAvailable(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.md")
	_ = os.WriteFile(a, []byte("# A"), 0644)
	files := []filemanager.FileItem{{Name: "a.md", Path: a}}
	fp := newTestPicker(t, "t", "", files, 80, 20)
	fp.fileList.Select(0)

	// Prime cache with plain version of the content for path 'a'
	keyPlain := fp.cacheKey(a, false, false)
	const cachedPlain = "CACHED_PLAIN"
	fp.contentCache.Add(keyPlain, cachedPlain)

	// Start with glamour = true so toggling goes to false and should use cached plain
	fp.useGlamour = true
	fp.viewport.SetContent("OLD")

	_, _ = fp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if !strings.Contains(fp.viewport.View(), cachedPlain) {
		t.Fatalf("expected viewport to use cached content after toggling format")
	}
}

func TestKeyMap_HelpBindingsNonEmpty(t *testing.T) {
	km := DefaultKeyMap()
	short := km.ShortHelp()
	full := km.FullHelp()

	if len(short) == 0 {
		t.Fatalf("expected ShortHelp to have entries")
	}
	if len(full) == 0 || len(full[0]) == 0 {
		t.Fatalf("expected FullHelp to have entries")
	}
}

func TestCacheKey_Composition(t *testing.T) {
	fp := newTestPicker(t, "t", "", nil, 10, 10)
	got := fp.cacheKey("/p.md", true, true)
	want := "/p.md|full|glamour"
	if got != want {
		t.Fatalf("unexpected cacheKey: got=%q want=%q", got, want)
	}
	got = fp.cacheKey("/p.md", false, false)
	want = "/p.md|trunc|plain"
	if got != want {
		t.Fatalf("unexpected cacheKey: got=%q want=%q", got, want)
	}
}

func TestWindowResize_DoesNotProduceNegativeHeight(t *testing.T) {
	// Very small height to stress the layout math
	files := []filemanager.FileItem{
		{Name: "a.md", Path: filepath.Join(t.TempDir(), "a.md")},
	}
	fp := newTestPicker(t, "Tiny", "TinySub", files, 10, 5)
	_, _ = fp.Update(tea.WindowSizeMsg{Width: 20, Height: 5})
	if fp.fileList.Height() <= 0 || fp.viewport.Height <= 0 {
		t.Fatalf("content height should be clamped to positive value, got list=%d, vp=%d", fp.fileList.Height(), fp.viewport.Height)
	}
}

func TestHeaderAndHelpMeasurementMatchesView(t *testing.T) {
	// This test ensures that the computed header/help heights in Update use the same
	// styling as View, preventing mismatches that cause visual artifacts.
	files := []filemanager.FileItem{
		{Name: "a.md", Path: filepath.Join(t.TempDir(), "a.md")},
	}
	fp := newTestPicker(t, "MeasureTitle", "MeasureSub", files, 100, 30)

	// Compute expected header/help height using similar rendering to Update
	headerView := styles.TitleStyle.Render(fp.title)
	if fp.subtitle != "" {
		headerView = lipgloss.JoinVertical(lipgloss.Left, headerView, styles.SubtitleStyle.Render(fp.subtitle))
	}
	headerView = lipgloss.NewStyle().MarginLeft(1).MarginBottom(1).Render(headerView)
	headerH := lipgloss.Height(headerView)

	helpView := styles.HelpStyle.Render(fp.help.View(fp.keys))
	helpView = lipgloss.NewStyle().MarginLeft(1).MarginTop(1).Render(helpView)
	helpH := lipgloss.Height(helpView)

	// Now resize and ensure heights used by component match resulting content area split
	w, h := 100, 30
	_, _ = fp.Update(tea.WindowSizeMsg{Width: w, Height: h})

	frameW, frameH := styles.PaneStyle.GetFrameSize()
	const mainLeftMargin = 1
	availW := w - (2 * frameW) - mainLeftMargin
	if fp.fileList.Width()+fp.viewport.Width != availW {
		t.Fatalf("width allocation mismatch: list+vp=%d expected=%d", fp.fileList.Width()+fp.viewport.Width, availW)
	}

	contentH := h - headerH - helpH - frameH
	if fp.fileList.Height() != contentH || fp.viewport.Height != contentH {
		t.Fatalf("height allocation mismatch: list=%d vp=%d expected=%d (headerH=%d helpH=%d frameH=%d)",
			fp.fileList.Height(), fp.viewport.Height, contentH, headerH, helpH, frameH)
	}
}
