// Package issuelist implements the main issue search/browse screen: a JQL
// input (toggled with "/") over a paginated results table.
package issuelist

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/hashkrish/jira-tui/internal/jiraclient"
	"github.com/hashkrish/jira-tui/internal/model"
	"github.com/hashkrish/jira-tui/internal/ui/components/issuedetail"
	"github.com/hashkrish/jira-tui/internal/ui/components/search"
	"github.com/hashkrish/jira-tui/internal/ui/keys"
	"github.com/hashkrish/jira-tui/internal/ui/screen"
	"github.com/hashkrish/jira-tui/internal/ui/styles"
)

const pageSize = 50

// DefaultJQL is used when the screen is first opened. /rest/api/3/search/jql
// rejects "unbounded" queries (no filter clause, e.g. bare "order by ..."),
// so this defaults to the current user's issues rather than everything.
const DefaultJQL = "assignee = currentUser() order by updated desc"

type searchResultMsg struct {
	result *model.SearchResult
	err    error
}

type Model struct {
	client *jiraclient.Client

	jql string

	// The /rest/api/3/search/jql endpoint only supports forward-only token
	// pagination (no offset/total). pageTokens[i] is the token used to fetch
	// page i (pageTokens[0] is always ""); pageIndex is the current page.
	// This lets "p" step back to an already-visited page without the API
	// supporting random access.
	pageTokens    []string
	pageIndex     int
	nextPageToken string

	issues  []model.Issue
	table   table.Model
	search  search.Model
	spinner spinner.Model
	loading bool

	width, height int
	// maxTableHeight is the row budget available for the table (derived from
	// the terminal height). The table's own height is set to whatever is
	// smaller of this and the actual row count — bubbles/table pads its
	// viewport with blank rows to fill whatever height it's given, so
	// setting it to the full budget regardless of content produced a wall
	// of empty rows below a short result set.
	maxTableHeight int
}

// New creates the issue list screen for the given JQL (DefaultJQL if empty).
func New(client *jiraclient.Client, jql string) Model {
	if jql == "" {
		jql = DefaultJQL
	}

	columns := []table.Column{
		{Title: "Key", Width: 10},
		{Title: "Type", Width: 10},
		{Title: "Status", Width: 14},
		{Title: "Assignee", Width: 18},
		{Title: "Priority", Width: 10},
		{Title: "Summary", Width: 40},
	}
	t := table.New(table.WithColumns(columns), table.WithFocused(true), table.WithStyles(styles.TableStyles()))

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return Model{
		client:     client,
		jql:        jql,
		pageTokens: []string{""},
		table:      t,
		search:     search.New(),
		spinner:    sp,
		loading:    true,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetch())
}

func (m Model) fetch() tea.Cmd {
	client, jql, token := m.client, m.jql, m.pageTokens[m.pageIndex]
	return func() tea.Msg {
		result, err := client.SearchIssues(jql, token, pageSize, nil)
		return searchResultMsg{result: result, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (screen.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.maxTableHeight = max(msg.Height-8, 3)
		m.applyTableHeight()
		m.resizeColumns()
		m.search = m.search.SetWidth(msg.Width - 4)
		return m, nil

	case searchResultMsg:
		m.loading = false
		if msg.err != nil {
			return m, screen.Error(msg.err)
		}
		m.issues = msg.result.Issues
		m.nextPageToken = msg.result.NextPageToken
		m.table.SetRows(toRows(msg.result.Issues))
		m.table.SetCursor(0)
		m.applyTableHeight()
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (screen.Screen, tea.Cmd) {
	if m.search.Active {
		switch msg.String() {
		case "enter":
			m.jql = m.search.Value()
			m.pageTokens = []string{""}
			m.pageIndex = 0
			m.search = m.search.Deactivate()
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.fetch())
		case "esc":
			m.search = m.search.Deactivate()
			return m, nil
		}
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		return m, cmd
	}

	switch {
	case key.Matches(msg, keys.Global.Search):
		m.search = m.search.Activate(m.jql)
		return m, nil

	case key.Matches(msg, keys.Global.Refresh):
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.fetch())

	case key.Matches(msg, keys.Global.Enter):
		row := m.table.Cursor()
		if row < 0 || row >= len(m.issues) {
			return m, nil
		}
		issue := m.issues[row]
		return m, screen.Push(issuedetail.New(m.client, issue.Key))

	case msg.String() == "n":
		if m.nextPageToken == "" {
			return m, nil
		}
		if m.pageIndex+1 >= len(m.pageTokens) {
			m.pageTokens = append(m.pageTokens, m.nextPageToken)
		}
		m.pageIndex++
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.fetch())

	case msg.String() == "p":
		if m.pageIndex == 0 {
			return m, nil
		}
		m.pageIndex--
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.fetch())
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// tableHeaderHeight is how many lines table.Model reserves for its header
// row: SetHeight(h) sets the *content* viewport to h minus this, so h must
// include the header, not just the data row count.
const tableHeaderHeight = 1

// applyTableHeight sizes the table to however many data rows it actually
// has (at least 1) plus the header, capped at maxTableHeight, instead of
// always claiming the full budget and letting bubbles/table pad the rest
// with blank rows.
func (m *Model) applyTableHeight() {
	rows := max(len(m.issues), 1)
	m.table.SetHeight(min(rows+tableHeaderHeight, m.maxTableHeight))
}

func (m *Model) resizeColumns() {
	fixed := 10 + 10 + 14 + 18 + 10 // key, type, status, assignee, priority
	summaryWidth := m.width - fixed - 12
	if summaryWidth < 20 {
		summaryWidth = 20
	}
	cols := m.table.Columns()
	if len(cols) == 6 {
		cols[5].Width = summaryWidth
		m.table.SetColumns(cols)
	}
}

func toRows(issues []model.Issue) []table.Row {
	rows := make([]table.Row, 0, len(issues))
	for _, is := range issues {
		rows = append(rows, table.Row{is.Key, is.IssueType, is.Status, is.Assignee, is.Priority, is.Summary})
	}
	return rows
}

func (m Model) View() string {
	var b string
	if m.search.Active {
		b += m.search.View() + "\n"
	} else {
		b += styles.Faint.Render(fmt.Sprintf("JQL: %s (press / to edit)", m.jql)) + "\n"
	}

	if !m.loading && len(m.issues) == 0 {
		b += styles.Faint.Render("No issues match this query.")
	} else {
		b += m.table.View()
	}

	footer := fmt.Sprintf("page %d", m.pageIndex+1)
	if m.nextPageToken != "" {
		footer += " (n: next)"
	}
	if m.pageIndex > 0 {
		footer += " (p: prev)"
	}
	if m.loading {
		footer = m.spinner.View() + " loading  " + footer
	}
	b += "\n" + styles.Faint.Render(footer)

	return b
}

func (m Model) Title() string {
	return "Issues"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
