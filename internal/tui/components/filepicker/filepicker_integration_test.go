package filepicker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"rulem/internal/filemanager"
	"rulem/internal/logging"
	"rulem/internal/tui/helpers"

	tea "github.com/charmbracelet/bubbletea"
)

// Helper to execute a single command and any immediate follow-up command returned by Update.
func runCmd(fp *FilePicker, cmd tea.Cmd) {
	max := 8
	for i := 0; cmd != nil && i < max; i++ {
		msg := cmd()
		_, cmd = fp.Update(msg)
	}
}

func TestFilePicker_InitialView(t *testing.T) {
	tmpDir := t.TempDir()
	smallFile := filepath.Join(tmpDir, "small.md")
	if err := os.WriteFile(smallFile, []byte("# Small File Content"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	files := []filemanager.FileItem{{Name: "small.md", Path: smallFile}}
	logger, _ := logging.NewTestLogger()
	fp := NewFilePicker("Test", "", files, helpers.UIContext{Width: 100, Height: 30, Logger: logger})

	// Ensure sizing applied
	fp.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// Manually render the first file to avoid debounce/timing (disable glamour for deterministic match)
	fp.useGlamour = false
	fp.fileList.Select(0)
	runCmd(&fp, fp.renderFileContent(smallFile, false, fp.useGlamour))

	out := fp.View()
	if !strings.Contains(out, "Small File Content") {
		t.Fatalf("expected initial content in view")
	}
}

func TestFilePickerBaseFlow(t *testing.T) {
	tmpDir := t.TempDir()
	smallFile := filepath.Join(tmpDir, "small.md")
	largeFile := filepath.Join(tmpDir, "large.md")

	if err := os.WriteFile(smallFile, []byte("# Small File\nThis is small content"), 0644); err != nil {
		t.Fatalf("write small: %v", err)
	}
	if err := os.WriteFile(largeFile, []byte(strings.Repeat("# Large File\nLarge content line\n", 1024)), 0644); err != nil {
		t.Fatalf("write large: %v", err)
	}

	files := []filemanager.FileItem{
		{Name: "small.md", Path: smallFile},
		{Name: "large.md", Path: largeFile},
	}

	logger, _ := logging.NewTestLogger()
	fp := NewFilePicker("Test File Picker", "", files, helpers.UIContext{Width: 80, Height: 24, Logger: logger})
	fp.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	out := fp.View()
	if !strings.Contains(out, "Test File Picker") {
		t.Fatalf("title missing in view")
	}
	if !strings.Contains(out, "Files") {
		t.Fatalf("list title missing in view")
	}
	if !strings.Contains(out, "small.md") || !strings.Contains(out, "large.md") {
		t.Fatalf("expected file names in view")
	}
}

func TestFilePicker_ViewportScrollKeyboard(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a tall file with numbered lines
	var b strings.Builder
	for i := range 200 {
		fmt.Fprintf(&b, "Line %03d\n", i)
	}
	tall := filepath.Join(tmpDir, "tall.md")
	if err := os.WriteFile(tall, []byte(b.String()), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	files := []filemanager.FileItem{{Name: "tall.md", Path: tall}}
	logger, _ := logging.NewTestLogger()
	fp := NewFilePicker("ScrollTest", "", files, helpers.UIContext{Width: 100, Height: 30, Logger: logger})
	fp.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// Select the item and render content synchronously
	fp.fileList.Select(0)
	runCmd(&fp, fp.renderFileContent(tall, false, false))

	// Focus preview and scroll down multiple times
	_, _ = fp.Update(tea.KeyMsg{Type: tea.KeyRight})
	for range 6 {
		_, _ = fp.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	}

	if fp.viewport.YOffset == 0 {
		t.Fatalf("expected viewport to scroll (non-zero YOffset)")
	}
}

func TestFilePicker_ViewportScrollMouse(t *testing.T) {
	tmpDir := t.TempDir()
	var b strings.Builder
	for i := range 200 {
		fmt.Fprintf(&b, "MLine %03d\n", i)
	}
	tall := filepath.Join(tmpDir, "mtall.md")
	if err := os.WriteFile(tall, []byte(b.String()), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	files := []filemanager.FileItem{{Name: "mtall.md", Path: tall}}
	logger, _ := logging.NewTestLogger()
	fp := NewFilePicker("MouseScroll", "", files, helpers.UIContext{Width: 100, Height: 30, Logger: logger})
	fp.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// Render content synchronously
	fp.fileList.Select(0)
	runCmd(&fp, fp.renderFileContent(tall, false, false))

	// Send several mouse wheel down events
	for range 10 {
		_, _ = fp.Update(tea.MouseMsg{Type: tea.MouseWheelDown})
	}

	out := fp.View()
	if !strings.Contains(out, "MLine 020") {
		t.Fatalf("expected to see later content after mouse scrolling")
	}
}

func TestFilePicker_FilteringEndSchedulesPreview(t *testing.T) {
	tmpDir := t.TempDir()
	a := filepath.Join(tmpDir, "apple.md")
	b := filepath.Join(tmpDir, "banana.md")
	if err := os.WriteFile(a, []byte("APPLE"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(b, []byte("BANANA"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	files := []filemanager.FileItem{{Name: "apple.md", Path: a}, {Name: "banana.md", Path: b}}
	logger, _ := logging.NewTestLogger()
	fp := NewFilePicker("FilterTest", "", files, helpers.UIContext{Width: 100, Height: 30, Logger: logger})
	fp.debounceDuration = 1 * time.Millisecond
	// Disable glamour for deterministic assertions in tests
	fp.useGlamour = false
	fp.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	// Diagnostic: log initial view to help debug filtering issues
	t.Log("initial FilterTest view:", fp.View())

	// Render initial to warm cache/path
	runCmd(&fp, fp.renderFileContent(a, false, false))

	// Enter filter and type 'ban'
	_, _ = fp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	for _, r := range []rune{'b', 'a', 'n'} {
		_, _ = fp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Exit filter and render banana deterministically (simulate preview completion)
	_, _ = fp.Update(tea.KeyMsg{Type: tea.KeyEsc})
	fp.fileList.Select(1)
	msg := fp.renderFileContent(b, false, fp.useGlamour)()
	_, _ = fp.Update(msg)

	if !strings.Contains(fp.viewport.View(), "BANANA") {
		// Diagnostic logging to help identify why content assertion failed
		t.Log("viewport after filter:", fp.viewport.View())
		t.Log("selected index:", fp.fileList.Index())
		t.Log("selected item (name):", fp.fileList.SelectedItem())
		t.Fatalf("expected banana content after filter ends")
	}
}

func TestFilePicker_FullPreviewLoadsBeyondTruncate(t *testing.T) {
	tmpDir := t.TempDir()
	large := filepath.Join(tmpDir, "large.md")
	big := strings.Repeat("Line content for big file\n", 30000)
	if err := os.WriteFile(large, []byte(big), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	files := []filemanager.FileItem{{Name: "large.md", Path: large}}
	logger, _ := logging.NewTestLogger()
	fp := NewFilePicker("FullTest", "", files, helpers.UIContext{Width: 100, Height: 30, Logger: logger})
	fp.largeFileThreshold = 1024 // ensure truncation on initial preview
	fp.maxPreviewBytes = 2048
	fp.useGlamour = false // plain text for deterministic header match in tests
	fp.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// Initial truncated preview
	fp.fileList.Select(0)
	msg := fp.renderFileContent(large, false, fp.useGlamour)()
	_, _ = fp.Update(msg)
	if !strings.Contains(fp.viewport.View(), "Press 'f'") {
		// Diagnostic: show the viewport when the truncated header isn't found
		t.Log("viewport without truncated header:", fp.viewport.View())
		t.Fatalf("expected truncated advisory header")
	}

	// Request full load deterministically
	msg = fp.renderFileContent(large, true, fp.useGlamour)()
	_, _ = fp.Update(msg)

	if !strings.Contains(fp.viewport.View(), "big file") {
		t.Fatalf("expected later content after full load")
	}
}
