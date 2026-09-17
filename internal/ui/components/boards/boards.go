// Package boards lists the Agile boards for a project; selecting one opens
// its current sprint as a kanban board.
package boards

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/hashkrish/jira-tui/internal/jiraclient"
	"github.com/hashkrish/jira-tui/internal/model"
	"github.com/hashkrish/jira-tui/internal/ui/components/sprintboard"
	"github.com/hashkrish/jira-tui/internal/ui/keys"
	"github.com/hashkrish/jira-tui/internal/ui/screen"
	"github.com/hashkrish/jira-tui/internal/ui/styles"
)

type item struct {
	board model.Board
}

func (i item) Title() string       { return i.board.Name }
func (i item) Description() string { return i.board.Type }
func (i item) FilterValue() string { return i.board.Name }

type loadedMsg struct {
	boards []model.Board
	err    error
}

type Model struct {
	client     *jiraclient.Client
	projectKey string
	list       list.Model
	spinner    spinner.Model
	loading    bool
}

func New(client *jiraclient.Client, projectKey string) Model {
	l := list.New(nil, styles.ListDelegate(), 0, 0)
	l.Title = "Boards: " + projectKey
	l.SetShowHelp(false)

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return Model{client: client, projectKey: projectKey, list: l, spinner: sp, loading: true}
}

func (m Model) Init() tea.Cmd {
	client, projectKey := m.client, m.projectKey
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		boards, err := client.ListBoards(projectKey)
		return loadedMsg{boards: boards, err: err}
	})
}

func (m Model) Update(msg tea.Msg) (screen.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, max(msg.Height-8, 3))
		return m, nil

	case loadedMsg:
		m.loading = false
		if msg.err != nil {
			return m, screen.Error(msg.err)
		}
		items := make([]list.Item, 0, len(msg.boards))
		for _, b := range msg.boards {
			items = append(items, item{board: b})
		}
		return m, m.list.SetItems(items)

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		if key.Matches(msg, keys.Global.Enter) {
			selected, ok := m.list.SelectedItem().(item)
			if !ok {
				return m, nil
			}
			return m, screen.Push(sprintboard.New(m.client, selected.board.ID, selected.board.Name))
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.loading && len(m.list.Items()) == 0 {
		return m.spinner.View() + " loading boards..."
	}
	return styles.TrimTrailingBlankLines(m.list.View())
}

func (m Model) Title() string {
	return "Boards"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
