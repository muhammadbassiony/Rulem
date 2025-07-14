package ui

import (
	"bufio"
	"os"
	"path/filepath"
	"rulemig/internal/validation"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// fileInfo holds the name and description of an instruction file.
type fileInfo struct {
	Name        string
	Path        string
	Description string // First line of file, or fallback
}

// fileViewerModel is the Bubble Tea model for the file viewer UI.
type migrationMode int

const (
	modeSelectFiles migrationMode = iota
	modeChooseAction
	modeChooseAssistant
	modeCustomLocation
)

type fileViewerModel struct {
	files           []fileInfo
	selected        map[int]struct{} // selected file indices
	cursor          int              // current cursor position
	err             error
	mode            migrationMode // current UI mode
	action          string        // "copy" or "symlink"
	assistant       string        // selected assistant
	assistantCursor int           // cursor for assistant selection
	customLocation  string        // user-entered custom location
	inputBuffer     string        // buffer for text input
	message         string        // error/success message to display
	messageType     string        // "error" or "success"
}

// NewFileViewerModel creates a new file viewer model and loads files from dir.
func NewFileViewerModel(dir string) fileViewerModel {
	files, err := listInstructionFiles(dir)
	return fileViewerModel{
		files:    files,
		selected: make(map[int]struct{}),
		cursor:   0,
		err:      err,
		mode:     modeSelectFiles,
	}
}

// Update is part of the Bubble Tea Model interface.
func (m fileViewerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeSelectFiles:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.files)-1 {
					m.cursor++
				}
			case " ":
				if _, ok := m.selected[m.cursor]; ok {
					delete(m.selected, m.cursor)
				} else {
					m.selected[m.cursor] = struct{}{}
				}
			case "enter":
				if len(m.selected) > 0 {
					m.mode = modeChooseAction
				}
			case "q":
				// Return to main menu or exit
				return m, tea.Quit
			}
		}
	case modeChooseAction:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "c":
				m.action = "copy"
			case "s":
				m.action = "symlink"
			case "enter":
				if m.action != "" {
					m.mode = modeChooseAssistant
				}
			case "q":
				// Go back to file selection
				m.mode = modeSelectFiles
				m.action = ""
			}
		}
	case modeChooseAssistant:
		assistants := []string{"copilot", "cursor", "claude", "gemini-cli", "opencode"}
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "up", "k":
				if m.assistantCursor > 0 {
					m.assistantCursor--
				}
			case "down", "j":
				if m.assistantCursor < len(assistants)-1 {
					m.assistantCursor++
				}
			case "enter":
				m.assistant = assistants[m.assistantCursor]
				m.mode = modeCustomLocation
				m.inputBuffer = ""
			case "q":
				// Go back to migration method selection
				m.mode = modeChooseAction
				m.assistant = ""
			}
		}
	case modeCustomLocation:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyRunes:
				m.inputBuffer += string(msg.Runes)
			case tea.KeyBackspace:
				if len(m.inputBuffer) > 0 {
					m.inputBuffer = m.inputBuffer[:len(m.inputBuffer)-1]
				}
			case tea.KeyEnter:
				m.customLocation = strings.TrimSpace(m.inputBuffer)
				// Next: trigger migration or next step
			}
			if msg.String() == "q" {
				// Go back to assistant selection
				m.mode = modeChooseAssistant
				m.customLocation = ""
				m.inputBuffer = ""
			}
		}
	}
	return m, nil
}

// listMarkdownFiles returns a slice of fileInfo for all .md files in dir.
func listInstructionFiles(dir string) ([]fileInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []fileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		desc := "(no description)"
		f, err := os.Open(path)
		if err == nil {
			scanner := bufio.NewScanner(f)
			if scanner.Scan() {
				first := strings.TrimSpace(scanner.Text())
				if first != "" {
					desc = first
				}
			}
			f.Close()
		}
		files = append(files, fileInfo{
			Name:        entry.Name(),
			Path:        path,
			Description: desc,
		})
	}
	return files, nil
}

// Init is part of the Bubble Tea Model interface.
func (m fileViewerModel) Init() tea.Cmd {
	return nil
}

func (m fileViewerModel) View() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render("[rulemig] Instruction Files")
	var b strings.Builder
	b.WriteString(title + "\n\n")
	if m.err != nil {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("Error: "+m.err.Error()) + "\n\n")
	}
	if m.message != "" {
		color := "10"
		if m.messageType == "error" {
			color = "9"
		}
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Bold(true).Render(m.message) + "\n\n")
	}
	if len(m.files) == 0 {
		b.WriteString("(No markdown instruction files found)")
		return b.String()
	}
	switch m.mode {
	case modeSelectFiles:
		b.WriteString("Select one or more instruction files to migrate:\n")
		for i, file := range m.files {
			cursor := "  "
			if m.cursor == i {
				cursor = "> "
			}
			selected := "[ ]"
			if _, ok := m.selected[i]; ok {
				selected = "[x]"
			}
			b.WriteString(cursor + selected + " " + file.Name)
			if file.Description != "" {
				b.WriteString(": " + file.Description)
			}
			b.WriteString("\n")
		}
		b.WriteString("\nUse up/down to move, space to select/deselect, enter to continue. Press 'q' to quit at any time.")
	case modeChooseAction:
		b.WriteString("You have selected the following files for migration:\n")
		for i := range m.selected {
			fname := m.files[i].Name
			path := filepath.Join("<storage-dir>", fname) // Replace with actual storage dir in integration
			if err := validation.ValidateMarkdownFile(path); err != nil {
				b.WriteString("- " + fname + " [INVALID]: " + err.Error() + "\n")
			} else {
				b.WriteString("- " + fname + "\n")
			}
		}
		b.WriteString("\nChoose migration method:\n")
		b.WriteString("  [c] Copy   [s] Symlink\n")
		if m.action == "" {
			b.WriteString("(Press 'c' for Copy or 's' for Symlink. Press 'q' to go back.)")
		} else {
			b.WriteString("\nYou chose: " + m.action + " (press enter to select target AI assistant)")
		}
	case modeChooseAssistant:
		assistants := []string{"copilot", "cursor", "claude", "gemini-cli", "opencode"}
		b.WriteString("Select the target AI assistant for migration:\n")
		for i, a := range assistants {
			cursor := "  "
			if m.assistantCursor == i {
				cursor = "> "
			}
			selected := ""
			if m.assistant == a {
				selected = "[selected] "
			}
			b.WriteString(cursor + selected + a + "\n")
		}
		b.WriteString("\nUse up/down to move, enter to select. Press 'q' to go back.")
		if m.assistant != "" {
			b.WriteString("\nYou chose: " + m.assistant)
		}
	case modeCustomLocation:
		b.WriteString("You may enter a custom location for the instruction files, or press enter to use the default location for your selected assistant.\n")
		b.WriteString("Current input: " + m.inputBuffer + "\n")
		b.WriteString("(Type a path and press enter, or just press enter to use default. Press 'q' to go back.)\n")
		if m.customLocation != "" {
			b.WriteString("\nCustom location set: " + m.customLocation)
		}
	}
	return b.String()
}
