// Package projectlist shows all projects visible to the user; selecting one
// drills into its boards.
package projectlist

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/hashkrish/jira-tui/internal/jiraclient"
	"github.com/hashkrish/jira-tui/internal/model"
	"github.com/hashkrish/jira-tui/internal/ui/components/boards"
	"github.com/hashkrish/jira-tui/internal/ui/keys"
	"github.com/hashkrish/jira-tui/internal/ui/screen"
	"github.com/hashkrish/jira-tui/internal/ui/styles"
)

type loadedMsg struct {
	projects []model.Project
	err      error
}

type Model struct {
	client   *jiraclient.Client
	projects []model.Project
	table    table.Model
	spinner  spinner.Model
	loading  bool
}

func New(client *jiraclient.Client) Model {
	columns := []table.Column{
		{Title: "Key", Width: 12},
		{Title: "Name", Width: 40},
		{Title: "Lead", Width: 24},
	}
	t := table.New(table.WithColumns(columns), table.WithFocused(true), table.WithStyles(styles.TableStyles()))

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return Model{client: client, table: t, spinner: sp, loading: true}
}

func (m Model) Init() tea.Cmd {
	client := m.client
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		projects, err := client.ListProjects()
		return loadedMsg{projects: projects, err: err}
	})
}

func (m Model) Update(msg tea.Msg) (screen.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.table.SetHeight(max(msg.Height-8, 3))
		return m, nil

	case loadedMsg:
		m.loading = false
		if msg.err != nil {
			return m, screen.Error(msg.err)
		}
		m.projects = msg.projects
		rows := make([]table.Row, 0, len(msg.projects))
		for _, p := range msg.projects {
			rows = append(rows, table.Row{p.Key, p.Name, p.Lead})
		}
		m.table.SetRows(rows)
		m.table.SetCursor(0)
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		if key.Matches(msg, keys.Global.Enter) {
			row := m.table.Cursor()
			if row < 0 || row >= len(m.projects) {
				return m, nil
			}
			project := m.projects[row]
			return m, screen.Push(boards.New(m.client, project.Key))
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.loading && len(m.projects) == 0 {
		return m.spinner.View() + " loading projects..."
	}
	if !m.loading && len(m.projects) == 0 {
		return "No projects found."
	}
	return m.table.View()
}

func (m Model) Title() string {
	return "Projects"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
