// Package filters lists the user's saved/favorite JQL filters; selecting one
// opens the issue list scoped to that filter's JQL.
package filters

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/hashkrish/jira-tui/internal/jiraclient"
	"github.com/hashkrish/jira-tui/internal/model"
	"github.com/hashkrish/jira-tui/internal/ui/components/issuelist"
	"github.com/hashkrish/jira-tui/internal/ui/keys"
	"github.com/hashkrish/jira-tui/internal/ui/screen"
	"github.com/hashkrish/jira-tui/internal/ui/styles"
)

type item struct {
	filter model.Filter
}

func (i item) Title() string       { return i.filter.Name }
func (i item) Description() string { return i.filter.JQL }
func (i item) FilterValue() string { return i.filter.Name }

type loadedMsg struct {
	filters []model.Filter
	err     error
}

type Model struct {
	client  *jiraclient.Client
	list    list.Model
	spinner spinner.Model
	loading bool
	empty   bool
}

func New(client *jiraclient.Client) Model {
	l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Favorite Filters"
	l.SetShowHelp(false)

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return Model{client: client, list: l, spinner: sp, loading: true}
}

func (m Model) Init() tea.Cmd {
	client := m.client
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		filters, err := client.ListFavoriteFilters()
		return loadedMsg{filters: filters, err: err}
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
		m.empty = len(msg.filters) == 0
		items := make([]list.Item, 0, len(msg.filters))
		for _, f := range msg.filters {
			items = append(items, item{filter: f})
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
			return m, screen.Push(issuelist.New(m.client, selected.filter.JQL))
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.loading && len(m.list.Items()) == 0 {
		return m.spinner.View() + " loading filters..."
	}
	if m.empty {
		return styles.Faint.Render("No favorite filters. Star a filter in Jira to see it here.")
	}
	return m.list.View()
}

func (m Model) Title() string {
	return "Filters"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
