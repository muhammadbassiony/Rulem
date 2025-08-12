package filepicker

import (
	clist "container/list"
	"fmt"
	"io"
	"os"
	filemanager "rulem/internal/filemanager"
	"rulem/internal/logging"
	"rulem/internal/tui/helpers"
	"rulem/internal/tui/styles"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

type KeyMap struct {
	Select       key.Binding
	Up           key.Binding
	Down         key.Binding
	Quit         key.Binding
	Filter       key.Binding
	Full         key.Binding
	ToggleFormat key.Binding
	FocusLeft    key.Binding
	FocusRight   key.Binding
}

// focusedPane identifies which pane (list or preview) has keyboard focus
type focusedPane int

const (
	focusList focusedPane = iota
	focusPreview
)

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Select:       key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Up:           key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:         key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Quit:         key.NewBinding(key.WithKeys("q", "esc"), key.WithHelp("q/esc", "quit")),
		Filter:       key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Full:         key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "load full")),
		ToggleFormat: key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "toggle format")),
		FocusLeft:    key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "focus list")),
		FocusRight:   key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "focus preview")),
	}
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Select, k.Filter, k.Full, k.ToggleFormat, k.FocusRight, k.FocusLeft, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Select, k.Filter, k.Full, k.ToggleFormat, k.FocusRight, k.FocusLeft, k.Quit},
	}
}

type FilePicker struct {
	logger *logging.AppLogger

	title        string
	subtitle     string
	files        []filemanager.FileItem
	fileList     list.Model
	selectedFile filemanager.FileItem
	keys         KeyMap
	viewport     viewport.Model
	help         help.Model

	windowWidth  int
	windowHeight int

	// Loading state management
	isLoading            bool
	loadingPath          string
	currentRenderID      uint64    // Tracks the latest render request
	renderCounter        *uint64   // Atomic counter for render requests
	contentCache         *lruCache // LRU cache with a byte cap
	contentCacheCapBytes int       // maximum bytes allowed in contentCache

	// Debounce and preview controls
	debounceDuration  time.Duration
	pendingDebounceID uint64

	// Preview options
	largeFileThreshold int // bytes
	maxPreviewBytes    int // bytes for truncated previews
	useGlamour         bool
	glamourStyle       string

	// focus management
	focusPane focusedPane
}

type (
	// This is sent from the main program to the file picker to let it know
	// that files are ready to be displayed and to update the file list
	FilesReadyMsg struct {
		Files []filemanager.FileItem
	}

	// To be sent from the file picker to the main program when a file is selected
	FileSelectedMsg struct {
		File filemanager.FileItem
	}

	// internal: sent after a debounce period to trigger preview
	debouncedPreviewMsg struct {
		path string
		seq  uint64
	}

	// File render message with request ID
	// this will be rendered after the preview debounce period
	FileRenderedMsg struct {
		content  string
		path     string
		renderID uint64
		cacheKey string
	}

	FileReadErrorMsg struct {
		err      error
		path     string
		renderID uint64
	}
)

// lruEntry represents a single cache item
type lruEntry struct {
	key     string
	content string
	size    int
}

// lruCache is a simple LRU cache with a byte capacity cap.
// It evicts least-recently-used entries until under capacity.
type lruCache struct {
	capacityBytes int
	currentBytes  int
	ll            *clist.List
	items         map[string]*clist.Element
}

func newLRU(capacity int) *lruCache {
	return &lruCache{
		capacityBytes: capacity,
		currentBytes:  0,
		ll:            clist.New(),
		items:         make(map[string]*clist.Element),
	}
}

func (c *lruCache) Get(key string) (string, bool) {
	if el, ok := c.items[key]; ok {
		c.ll.MoveToFront(el)
		return el.Value.(*lruEntry).content, true
	}
	return "", false
}

func (c *lruCache) Add(key string, content string) {
	size := len(content)
	// Skip caching entries larger than total capacity
	if size > c.capacityBytes {
		return
	}
	// Update if exists
	if el, ok := c.items[key]; ok {
		ent := el.Value.(*lruEntry)
		c.currentBytes += size - ent.size
		ent.content = content
		ent.size = size
		c.ll.MoveToFront(el)
	} else {
		el := c.ll.PushFront(&lruEntry{key: key, content: content, size: size})
		c.items[key] = el
		c.currentBytes += size
	}
	// Evict until under capacity
	for c.currentBytes > c.capacityBytes && c.ll.Len() > 0 {
		tail := c.ll.Back()
		ent := tail.Value.(*lruEntry)
		delete(c.items, ent.key)
		c.ll.Remove(tail)
		c.currentBytes -= ent.size
	}
}

func (c *lruCache) Clear() {
	c.ll.Init()
	c.items = make(map[string]*clist.Element)
	c.currentBytes = 0
}

// detectGlamourStyle attempts to detect terminal background using termenv,
// but will respect GLAMOUR_STYLE if set to a concrete value (not "auto").
// A timeout ensures we never hang on terminals that don't respond.
func detectGlamourStyle(timeout time.Duration) string {
	// Default fallback if detection doesn't finish in time
	defaultStyle := "dark"

	style := os.Getenv("GLAMOUR_STYLE")
	if style != "" && style != "auto" {
		return style
	}

	type result struct{ style string }
	ch := make(chan result, 1)

	go func() {
		out := termenv.NewOutput(os.Stdout)
		if out.HasDarkBackground() {
			ch <- result{style: "dark"}
			return
		}
		ch <- result{style: "light"}
	}()

	select {
	case r := <-ch:
		return r.style
	case <-time.After(timeout):
		return defaultStyle
	}
}

func NewFilePicker(title, subtitle string, files []filemanager.FileItem, ctx helpers.UIContext) FilePicker {
	// convert files to list Items
	items := make([]list.Item, len(files))
	for i, f := range files {
		items[i] = f
	}

	fileList := list.New(items, list.NewDefaultDelegate(), ctx.Width, ctx.Height)
	fileList.Title = "Files"
	fileList.SetShowStatusBar(false)
	fileList.SetFilteringEnabled(true)
	fileList.SetShowHelp(false)

	viewport := viewport.New(ctx.Width, ctx.Height)
	viewport.MouseWheelEnabled = true

	keys := DefaultKeyMap()
	help := help.New()

	renderCounter := uint64(0)

	return FilePicker{
		logger:               ctx.Logger,
		title:                title,
		subtitle:             subtitle,
		files:                files,
		fileList:             fileList,
		selectedFile:         filemanager.FileItem{},
		viewport:             viewport,
		keys:                 keys,
		help:                 help,
		windowWidth:          ctx.Width,
		windowHeight:         ctx.Height,
		renderCounter:        &renderCounter,
		currentRenderID:      0,
		contentCacheCapBytes: 1 << 20, // 1 MiB cap
		contentCache:         newLRU(1 << 20),
		debounceDuration:     200 * time.Millisecond,
		largeFileThreshold:   7 * 1024, // 7KB
		maxPreviewBytes:      2 * 1024, // 2KB
		useGlamour:           true,
		focusPane:            focusList,
	}
}

func (fp *FilePicker) Init() tea.Cmd {
	// Detect glamour style once (with timeout and env override) to avoid repeated
	// OSC/CSI queries during rendering and ensure stable behavior across terminals.
	if fp.glamourStyle == "" {
		fp.glamourStyle = detectGlamourStyle(50 * time.Millisecond)
		fp.logger.Debug("Glamour style selected", "style", fp.glamourStyle)
	}

	// If there are files, set initial loading state and trigger preview
	if len(fp.fileList.Items()) > 0 {
		selectedItem := fp.fileList.Items()[0].(filemanager.FileItem)

		return fp.scheduleDebouncedPreview(selectedItem.Path)
	}
	return nil
}

func (fp *FilePicker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	fp.logger.Debug("FilePicker received message", "type", fmt.Sprintf("%T", msg))

	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Get the currently selected file path before any updates
	var oldSelectedPath string
	if item := fp.fileList.SelectedItem(); item != nil {
		oldSelectedPath = item.(filemanager.FileItem).Path
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		fp.windowWidth = msg.Width
		fp.windowHeight = msg.Height
		fp.help.Width = msg.Width

		// Compute frame sizes from the final pane style to avoid clipping.
		frameW, frameH := styles.PaneStyle.GetFrameSize()
		// Total horizontal extras for two panes (frame only) = 2 * frameW
		totalExtras := frameW * 2

		// Account for a small left margin for the main panes container
		// to align with help and title/subtitle
		const mainLeftMargin = 1
		avail := max(msg.Width-totalExtras-mainLeftMargin, 0)

		// Content widths
		listWidth := avail / 3
		vpWidth := avail - listWidth

		if listWidth < 20 {
			listWidth = 20
		}
		if vpWidth < 30 {
			vpWidth = 30
		}

		// Calculate dynamic heights: header (title+subtitle) and help.
		// Use the same styles that View() uses so measured height matches rendered height.
		headerView := styles.TitleStyle.Render(fp.title)
		if fp.subtitle != "" {
			headerView = lipgloss.JoinVertical(lipgloss.Left, headerView, styles.SubtitleStyle.Render(fp.subtitle))
		}
		headerView = styles.HeaderContainerStyle.Render(headerView)

		helpView := styles.HelpContainerStyle.Render(styles.HelpStyle.Render(fp.help.View(fp.keys)))

		headerH := lipgloss.Height(headerView)
		helpH := lipgloss.Height(helpView)

		contentHeight := max(msg.Height-headerH-helpH-frameH, 5)
		fp.logger.Debug("Resizing heights", "Reported height", msg.Height, "help height", helpH, "header height", headerH, "content height", contentHeight)

		fp.fileList.SetSize(listWidth, contentHeight)
		fp.viewport.Width = vpWidth
		fp.viewport.Height = contentHeight

		fp.logger.Debug("Window resized", "width", msg.Width, "height", msg.Height, "list_width", listWidth, "viewport_width", vpWidth)
		return fp, nil

	case tea.MouseMsg:
		// Always let the viewport handle mouse events (for wheel scrolling)
		var vpcmd tea.Cmd
		fp.viewport, vpcmd = fp.viewport.Update(msg)
		if vpcmd != nil {
			cmds = append(cmds, vpcmd)
		}
		return fp, tea.Batch(cmds...)

	case list.FilterMatchesMsg:
		// Forward filter matches to the list; avoid changing preview here
		fp.fileList, cmd = fp.fileList.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return fp, tea.Batch(cmds...)

	case FileRenderedMsg:
		// File already rendered using glamour, now we can apply it to the viewport
		fp.logger.Debug("File rendered message received",
			"path", msg.path,
			"renderID", msg.renderID,
			"currentRenderID", fp.currentRenderID,
			"content_length", len(msg.content))

		// Always cache the content regardless of staleness
		key := msg.cacheKey
		if key == "" {
			key = fp.cacheKey(msg.path, false, fp.useGlamour)
		}
		fp.contentCache.Add(key, msg.content)
		fp.logger.Debug("Cached rendered content", "path", msg.path, "cacheKey", key)

		// Only display if this render is for the current selection and not stale
		currentSelectedPath := ""
		if item := fp.fileList.SelectedItem(); item != nil {
			currentSelectedPath = item.(filemanager.FileItem).Path
		}

		// Display if: it's for the current selection AND it's not stale
		if msg.path == currentSelectedPath && msg.renderID >= fp.currentRenderID {
			fp.currentRenderID = msg.renderID
			fp.viewport.SetContent(msg.content)
			fp.isLoading = false
			fp.loadingPath = ""
			fp.logger.Debug("Applied rendered content to viewport", "path", msg.path, "renderID", msg.renderID)
		} else {
			fp.logger.Debug("Content cached but not displayed",
				"path", msg.path,
				"renderID", msg.renderID,
				"currentSelectedPath", currentSelectedPath,
				"isStale", msg.renderID < fp.currentRenderID)
		}
		return fp, cmd

	case FileReadErrorMsg:
		currentSelectedPath := ""
		if item := fp.fileList.SelectedItem(); item != nil {
			currentSelectedPath = item.(filemanager.FileItem).Path
		}

		// only display error if it's for the current selection and not stale
		if msg.path == currentSelectedPath && msg.renderID >= fp.currentRenderID {
			fp.currentRenderID = msg.renderID
			fp.logger.Error("Error reading file", "error", msg.err, "path", msg.path)
			errorContent := fmt.Sprintf("Error reading file %s: %v", msg.path, msg.err)
			fp.viewport.SetContent(errorContent)
			fp.isLoading = false
			fp.loadingPath = ""
		} else {
			fp.logger.Debug("Ignoring stale error result", "path", msg.path, "renderID", msg.renderID)
		}
		return fp, nil

	case debouncedPreviewMsg:
		// Only render if this is the latest scheduled preview and selection still matches
		if msg.seq != fp.pendingDebounceID {
			fp.logger.Debug("Ignoring stale debounced preview", "path", msg.path)
			return fp, nil
		}
		currentSelectedPath := ""
		if item := fp.fileList.SelectedItem(); item != nil {
			currentSelectedPath = item.(filemanager.FileItem).Path
		}
		if currentSelectedPath != msg.path {
			fp.logger.Debug("Debounced path no longer selected; skipping", "path", msg.path, "selected", currentSelectedPath)
			return fp, nil
		}

		// Use cache first (try trunc then full) for immediate responsiveness
		// else trigger render (auto mode: full or truncated based on size)
		cacheKeyTrunc := fp.cacheKey(msg.path, false, fp.useGlamour)
		cacheKeyFull := fp.cacheKey(msg.path, true, fp.useGlamour)
		if cached, ok := fp.contentCache.Get(cacheKeyTrunc); ok {
			fp.viewport.SetContent(cached)
			fp.isLoading = false
			fp.loadingPath = ""
			return fp, nil
		} else if cached, ok := fp.contentCache.Get(cacheKeyFull); ok {
			fp.viewport.SetContent(cached)
			fp.isLoading = false
			fp.loadingPath = ""
			return fp, nil
		}

		// If not cached, render the file content
		return fp, fp.renderFileContent(msg.path, false, fp.useGlamour)

	case FilesReadyMsg:
		fp.logger.Debug("Files ready message received", "files", msg.Files)
		fp.files = msg.Files
		items := make([]list.Item, len(fp.files))
		for i, f := range fp.files {
			items[i] = f
		}
		fp.fileList.SetItems(items)
		fp.fileList.ResetSelected()
		fp.logger.Debug("File list updated with new files", "count", len(fp.files))
		fp.viewport.GotoTop()

		fp.contentCache.Clear()

		if len(fp.fileList.Items()) > 0 {
			selectedItem := fp.fileList.Items()[0].(filemanager.FileItem)

			cmds = append(cmds, fp.scheduleDebouncedPreview(selectedItem.Path))
		}
		return fp, tea.Batch(cmds...)

	case tea.KeyMsg:
		// If filtering is active, ESC should only exit the filter (not quit the app)
		if msg.String() == "esc" && fp.fileList.FilterState() == list.Filtering {
			var lcmd tea.Cmd
			fp.fileList, lcmd = fp.fileList.Update(msg)
			if lcmd != nil {
				cmds = append(cmds, lcmd)
			}
			return fp, tea.Batch(cmds...)
		}

		// Focus switching
		if key.Matches(msg, fp.keys.FocusRight) {
			fp.focusPane = focusPreview
			return fp, nil
		}
		if key.Matches(msg, fp.keys.FocusLeft) {
			fp.focusPane = focusList
			return fp, nil
		}

		// When preview has focus, route scroll keys to viewport
		if fp.focusPane == focusPreview {
			switch msg.String() {
			case "up", "down", "pgup", "pgdown", "ctrl+u", "ctrl+d", "home", "end", "k", "j":
				var vcmd tea.Cmd
				fp.viewport, vcmd = fp.viewport.Update(msg)
				if vcmd != nil {
					cmds = append(cmds, vcmd)
				}
				return fp, tea.Batch(cmds...)
			}
		}

		// Handle key bindings
		switch {
		case key.Matches(msg, fp.keys.Select):
			selectedItem, ok := fp.fileList.SelectedItem().(filemanager.FileItem)
			if ok {
				fp.logger.Debug("File selected via Enter", "path", selectedItem.Path)
				return fp, func() tea.Msg {
					return FileSelectedMsg{File: selectedItem}
				}
			}

		case key.Matches(msg, fp.keys.Quit):
			return fp, tea.Quit

		case key.Matches(msg, fp.keys.Full):
			// Load full preview for current selection
			if item := fp.fileList.SelectedItem(); item != nil {
				p := item.(filemanager.FileItem).Path
				fp.logger.Debug("User requested full preview", "path", p)
				fp.isLoading = true
				fp.loadingPath = p
				fp.viewport.SetContent("📄 Loading full preview for " + p + "...")
				return fp, fp.renderFileContent(p, true, fp.useGlamour)
			}

		case key.Matches(msg, fp.keys.ToggleFormat):
			// Toggle glamour formatting and re-render current selection (use cache immediately if available)
			fp.useGlamour = !fp.useGlamour
			if item := fp.fileList.SelectedItem(); item != nil {
				p := item.(filemanager.FileItem).Path
				fp.logger.Debug("Toggled formatting", "useGlamour", fp.useGlamour, "path", p)

				// Try cache (trunc then full) first, for the sake of responsiveness
				if cached, ok := fp.contentCache.Get(fp.cacheKey(p, false, fp.useGlamour)); ok {
					fp.viewport.SetContent(cached)
					fp.isLoading = false
					fp.loadingPath = ""
					return fp, nil
				}
				if cached, ok := fp.contentCache.Get(fp.cacheKey(p, true, fp.useGlamour)); ok {
					fp.viewport.SetContent(cached)
					fp.isLoading = false
					fp.loadingPath = ""
					return fp, nil
				}
				// Fallback to render
				fp.isLoading = true
				fp.loadingPath = p
				fp.viewport.SetContent("📄 Rendering with formatting: " + fmt.Sprintf("%v", fp.useGlamour))
				return fp, fp.renderFileContent(p, false /*full-auto*/, fp.useGlamour)
			}

		default:
			// Forward all other keys to the list (including filtering)
			prev := fp.fileList.FilterState()
			fp.fileList, cmd = fp.fileList.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			// If filtering just ended, schedule preview for the selected item
			if prev == list.Filtering && fp.fileList.FilterState() != list.Filtering {
				if sel := fp.fileList.SelectedItem(); sel != nil {
					p := sel.(filemanager.FileItem).Path
					fp.logger.Debug("Filtering ended; scheduling preview", "path", p)
					cmds = append(cmds, fp.scheduleDebouncedPreview(p))
				}
			}

			// Normal list change handling
			var newSelectedPath string
			if item := fp.fileList.SelectedItem(); item != nil {
				newSelectedPath = item.(filemanager.FileItem).Path
			}

			if newSelectedPath != "" && newSelectedPath != oldSelectedPath {
				fp.logger.Debug("Selection changed, updating preview", "new_path", newSelectedPath)
				// If currently filtering, don't preview yet
				if fp.fileList.FilterState() == list.Filtering {
					fp.logger.Debug("Selection changed during filtering; skipping preview")
				} else {
					// If cached content exists, apply immediately (prefer trunc, then full)
					if cached, ok := fp.contentCache.Get(fp.cacheKey(newSelectedPath, false, fp.useGlamour)); ok {
						fp.viewport.SetContent(cached)
						fp.isLoading = false
						fp.loadingPath = ""
					} else if cached, ok := fp.contentCache.Get(fp.cacheKey(newSelectedPath, true, fp.useGlamour)); ok {
						fp.viewport.SetContent(cached)
						fp.isLoading = false
						fp.loadingPath = ""
					} else {
						// Debounce new preview to avoid spamming renders while scrolling
						cmds = append(cmds, fp.scheduleDebouncedPreview(newSelectedPath))
					}
				}
			}

			return fp, tea.Batch(cmds...)
		}
	}

	return fp, tea.Batch(cmds...)
}

func (fp *FilePicker) View() string {
	// Header (title + optional subtitle) at the top
	title := styles.TitleStyle.Render(fp.title)
	var header string
	if fp.subtitle != "" {
		sub := styles.SubtitleStyle.Render(fp.subtitle)
		header = lipgloss.JoinVertical(lipgloss.Left, title, sub)
	} else {
		header = title
	}

	// Add some margins around header using container style
	header = styles.HeaderContainerStyle.Render(header)

	// Focus-aware pane styles using centralized styles
	listStyle := styles.PaneStyle
	vpStyle := styles.PaneStyle
	switch fp.focusPane {
	case focusList:
		listStyle = styles.PaneFocusedStyle
	case focusPreview:
		vpStyle = styles.PaneFocusedStyle
	}

	// Respect sizes computed in WindowSizeMsg (SetSize + Width)
	// Apply explicit widths based on list/viewport sizes and keep header at the top.
	listStyle = listStyle.Width(fp.fileList.Width()).Height(fp.fileList.Height())
	vpStyle = vpStyle.Width(fp.viewport.Width).Height(fp.viewport.Height)

	panes := lipgloss.JoinHorizontal(
		lipgloss.Top,
		listStyle.Render(fp.fileList.View()),
		vpStyle.Render(fp.viewport.View()),
	)
	// Left padding for the main panes using container style
	panes = styles.MainContainerStyle.Render(panes)

	// Help with padding and margin using container style
	helpView := styles.HelpContainerStyle.Render(styles.HelpStyle.Render(fp.help.View(fp.keys)))

	return lipgloss.JoinVertical(lipgloss.Left, header, panes, helpView)
}

//  HELPERS / COMMANDS

// cacheKey composes a cache key based on path and render options
func (fp *FilePicker) cacheKey(path string, full bool, glamourOn bool) string {
	mode := "trunc"
	if full {
		mode = "full"
	}
	fmtMode := "plain"
	if glamourOn {
		fmtMode = "glamour"
	}
	return path + "|" + mode + "|" + fmtMode
}

func (fp *FilePicker) scheduleDebouncedPreview(p string) tea.Cmd {
	fp.isLoading = true
	fp.loadingPath = p
	fp.viewport.SetContent("📄 Loading " + p + "...")
	seq := atomic.AddUint64(&fp.pendingDebounceID, 1)
	return tea.Tick(fp.debounceDuration, func(time.Time) tea.Msg {
		return debouncedPreviewMsg{path: p, seq: seq}
	})
}

func (fp *FilePicker) renderFileContent(path string, full bool, glamourOn bool) tea.Cmd {
	renderID := atomic.AddUint64(fp.renderCounter, 1)

	return func() tea.Msg {
		fp.logger.Debug("Reading and rendering file", "path", path, "renderID", renderID)
		// detect file size
		fi, statErr := os.Stat(path)
		if statErr != nil {
			fp.logger.Error("Failed to stat file", "path", path, "error", statErr, "renderID", renderID)
			return FileReadErrorMsg{err: statErr, path: path, renderID: renderID}
		}

		// Decide how many bytes to read
		var toRead int64 = fi.Size()
		truncated := false
		fp.logger.Debug("Checking truncation", "path", path, "size", fi.Size(), "full", full, "largeFileThreshold", fp.largeFileThreshold, "bigger ? ", fi.Size() > int64(fp.largeFileThreshold), "renderID", renderID)
		if !full && fi.Size() > int64(fp.largeFileThreshold) {
			// truncated preview for large files
			if int64(fp.maxPreviewBytes) < toRead {
				toRead = int64(fp.maxPreviewBytes)
			}
			truncated = true
		}

		// Read content (possibly partial)
		f, err := os.Open(path)
		if err != nil {
			fp.logger.Error("Failed to open file", "path", path, "error", err, "renderID", renderID)
			return FileReadErrorMsg{err: err, path: path, renderID: renderID}
		}
		defer f.Close()

		buf := make([]byte, toRead)
		n, rerr := io.ReadFull(f, buf)
		if rerr != nil && rerr != io.ErrUnexpectedEOF && rerr != io.EOF {
			fp.logger.Error("Failed to read file contents", "path", path, "error", rerr, "renderID", renderID)
			return FileReadErrorMsg{err: rerr, path: path, renderID: renderID}
		}
		content := buf[:n]

		vpWidth := fp.viewport.Width - 2
		if vpWidth <= 0 {
			vpWidth = 80
			fp.logger.Debug("Using fallback viewport width", "width", vpWidth, "viewport_width", fp.viewport.Width)
		}

		// Build header if truncated or indicate formatting mode
		header := ""
		if truncated {
			header = fmt.Sprintf("[Preview truncated to %d KB of %d KB. Press 'f' to load full.]\n\n", n/1024, fi.Size()/1024)
		}

		var renderedContent string
		if glamourOn {
			renderer, err := glamour.NewTermRenderer(
				glamour.WithStandardStyle(fp.glamourStyle),
				glamour.WithWordWrap(vpWidth),
			)
			if err != nil {
				fp.logger.Error("Failed to create glamour renderer", "error", err, "renderID", renderID)
				return FileReadErrorMsg{err: err, path: path, renderID: renderID}
			}

			rc, err := renderer.Render(string(content))
			if err != nil {
				fp.logger.Error("Failed to render content with glamour", "error", err, "renderID", renderID)
				return FileReadErrorMsg{err: err, path: path, renderID: renderID}
			}
			renderedContent = header + rc + header
		} else {
			// plain text without markdown rendering
			renderedContent = header + string(content) + header
		}

		fp.logger.Debug("File rendered successfully", "path", path, "renderID", renderID, "content_length", len(renderedContent), "truncated", truncated, "glamour", glamourOn)
		return FileRenderedMsg{content: renderedContent, path: path, renderID: renderID, cacheKey: fp.cacheKey(path, !truncated, glamourOn)}
	}
}
