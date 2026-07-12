// Package repostatusmenu implements the "Refresh GitHub repositories" screen.
//
// It shows the sync status of every configured repository and offers a single
// action: refetch all GitHub repositories. There is deliberately no per-repo
// selection — the screen stays a simple status board with one button.
//
// Status semantics:
//   - local repositories are listed for completeness but are never synced
//   - a GitHub repository with a missing clone directory is re-cloned on refresh
//   - a GitHub repository with uncommitted local changes is skipped on refresh
//     (rulem never discards local work)
package repostatusmenu

import (
	"context"
	"fmt"
	"os"
	"strings"

	"rulem/internal/config"
	"rulem/internal/logging"
	"rulem/internal/repository"
	"rulem/internal/tui/components"
	"rulem/internal/tui/helpers"
	"rulem/internal/tui/styles"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type menuState int

const (
	stateChecking menuState = iota
	stateReady
	stateRefreshing
)

// repoRow is one line of the status board.
type repoRow struct {
	Name   string
	Kind   string // "local" or "github (branch)"
	Path   string
	Status string
}

type (
	statusRowsMsg struct {
		rows []repoRow
	}

	refreshDoneMsg struct {
		prepared []repository.PreparedRepository
		err      error
	}
)

type RepoStatusModel struct {
	logger  *logging.AppLogger
	layout  components.LayoutModel
	spinner spinner.Model
	cfg     *config.Config

	state menuState
	rows  []repoRow

	// lastSync holds the most recent refresh outcome per repository ID and is
	// merged into the status rows after a refresh.
	lastSync map[string]string
}

func NewRepoStatusModel(ctx helpers.UIContext) *RepoStatusModel {
	layout := components.NewLayout(components.LayoutConfig{
		MarginX:  2,
		MarginY:  1,
		MaxWidth: 100,
	})
	if ctx.HasValidDimensions() {
		layout, _ = layout.Update(tea.WindowSizeMsg{Width: ctx.Width, Height: ctx.Height})
	}

	s := spinner.New()
	s.Style = styles.SpinnerStyle
	s.Spinner = spinner.Pulse

	return &RepoStatusModel{
		logger:   ctx.Logger,
		layout:   layout,
		spinner:  s,
		cfg:      ctx.Config,
		state:    stateChecking,
		lastSync: map[string]string{},
	}
}

func (m *RepoStatusModel) Init() tea.Cmd {
	return tea.Batch(m.checkStatusCmd(), m.spinner.Tick)
}

func (m *RepoStatusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.layout, _ = m.layout.Update(msg)

	switch msg := msg.(type) {
	case statusRowsMsg:
		m.rows = msg.rows
		m.state = stateReady
		return m, nil

	case refreshDoneMsg:
		if msg.err != nil {
			m.logger.Error("Repository refresh failed", "error", msg.err)
			m.layout = m.layout.SetError(msg.err)
		} else {
			m.layout = m.layout.ClearError()
		}
		for _, prep := range msg.prepared {
			if prep.IsRemote() {
				m.lastSync[prep.ID()] = prep.SyncResult.GetMessage()
			}
		}
		// Re-check the on-disk state so dirty/missing markers are current.
		return m, m.checkStatusCmd()

	case spinner.TickMsg:
		if m.state == stateChecking || m.state == stateRefreshing {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, func() tea.Msg { return helpers.NavigateToMainMenuMsg{} }
		case "r", "enter":
			if m.state == stateReady && m.hasGitHubRepos() {
				m.state = stateRefreshing
				return m, tea.Batch(m.refreshCmd(), m.spinner.Tick)
			}
		}
	}

	return m, nil
}

func (m *RepoStatusModel) View() string {
	help := "q/esc back"
	if m.hasGitHubRepos() {
		help = "r refresh all GitHub repositories • q/esc back"
	}
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "🔄 GitHub Repositories",
		Subtitle: m.subtitle(),
		HelpText: help,
	})

	switch m.state {
	case stateChecking:
		return m.layout.Render(fmt.Sprintf("%s Checking repository status...", m.spinner.View()))
	case stateRefreshing:
		return m.layout.Render(fmt.Sprintf("%s Refreshing repositories... (clones may take a moment)", m.spinner.View()))
	default:
		return m.layout.Render(m.renderRows())
	}
}

func (m *RepoStatusModel) subtitle() string {
	if m.cfg == nil || len(m.cfg.Repositories) == 0 {
		return "No repositories configured."
	}
	if !m.hasGitHubRepos() {
		return "Only local repositories are configured - there is nothing to refetch."
	}
	return "Sync status of your configured repositories. Repositories with local\nchanges are skipped during refresh so your edits are never lost."
}

func (m *RepoStatusModel) hasGitHubRepos() bool {
	if m.cfg == nil {
		return false
	}
	for _, r := range m.cfg.Repositories {
		if r.IsRemote() {
			return true
		}
	}
	return false
}

func (m *RepoStatusModel) renderRows() string {
	if len(m.rows) == 0 {
		return "No repositories configured - add one in Settings."
	}
	var b strings.Builder
	for _, row := range m.rows {
		b.WriteString(fmt.Sprintf("%s  (%s)\n", row.Name, row.Kind))
		b.WriteString(fmt.Sprintf("    %s\n", row.Path))
		b.WriteString(fmt.Sprintf("    %s\n\n", row.Status))
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m *RepoStatusModel) checkStatusCmd() tea.Cmd {
	cfg := m.cfg
	lastSync := m.lastSync
	return func() tea.Msg {
		if cfg == nil {
			return statusRowsMsg{}
		}
		return statusRowsMsg{rows: buildStatusRows(cfg.Repositories, lastSync)}
	}
}

func (m *RepoStatusModel) refreshCmd() tea.Cmd {
	cfg := m.cfg
	logger := m.logger
	return func() tea.Msg {
		prepared, err := repository.PrepareAllRepositories(context.Background(), cfg.Repositories, logger)
		return refreshDoneMsg{prepared: prepared, err: err}
	}
}

// buildStatusRows computes the status board from the configured repositories
// and the outcome of the most recent refresh (may be empty).
func buildStatusRows(repos []repository.RepositoryEntry, lastSync map[string]string) []repoRow {
	rows := make([]repoRow, 0, len(repos))
	for _, repo := range repos {
		row := repoRow{Name: repo.Name, Path: repo.Path}

		if repo.IsLocal() {
			row.Kind = "local"
			row.Status = "📁 local directory - not synced with any remote"
			rows = append(rows, row)
			continue
		}

		branch := "default branch"
		if repo.Branch != nil && *repo.Branch != "" {
			branch = *repo.Branch
		}
		row.Kind = "github • " + branch

		switch {
		case pathMissing(repo.Path):
			row.Status = "⬇️  clone missing - refresh will re-clone it"
		default:
			dirty, err := repository.CheckGithubRepositoryStatus(repo.Path)
			switch {
			case err != nil:
				row.Status = fmt.Sprintf("⚠️  cannot read status: %v", err)
			case dirty:
				row.Status = "✋ local changes - refresh will skip this repository"
			default:
				row.Status = "✅ clean - in sync with the last fetch"
			}
		}

		if msg, ok := lastSync[repo.ID]; ok && msg != "" {
			row.Status += "\n    last refresh: " + msg
		}
		rows = append(rows, row)
	}
	return rows
}

func pathMissing(path string) bool {
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}
