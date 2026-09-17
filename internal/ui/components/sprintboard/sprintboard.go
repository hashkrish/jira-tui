// Package sprintboard renders a board's current sprint as a kanban-style
// board, with one column per status category (To Do / In Progress / Done).
package sprintboard

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/hashkrish/jira-tui/internal/jiraclient"
	"github.com/hashkrish/jira-tui/internal/model"
	"github.com/hashkrish/jira-tui/internal/ui/components/issuedetail"
	"github.com/hashkrish/jira-tui/internal/ui/keys"
	"github.com/hashkrish/jira-tui/internal/ui/screen"
	"github.com/hashkrish/jira-tui/internal/ui/styles"
)

// preferredColumnOrder puts the common status categories first; any other
// category encountered is appended after these, in first-seen order.
var preferredColumnOrder = []string{"To Do", "In Progress", "Done"}

type sprintsLoadedMsg struct {
	sprints []model.Sprint
	err     error
}

type issuesLoadedMsg struct {
	result *model.SearchResult
	err    error
}

type column struct {
	name   string
	issues []model.Issue
}

type Model struct {
	client    *jiraclient.Client
	boardID   int
	boardName string

	sprint    *model.Sprint
	columns   []column
	activeCol int
	cursors   []int // per-column selected row index

	loading  bool
	spinner  spinner.Model
	noSprint bool

	width, height int
}

func New(client *jiraclient.Client, boardID int, boardName string) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	return Model{client: client, boardID: boardID, boardName: boardName, spinner: sp, loading: true}
}

func (m Model) Init() tea.Cmd {
	client, boardID := m.client, m.boardID
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		sprints, err := client.ListSprints(boardID)
		return sprintsLoadedMsg{sprints: sprints, err: err}
	})
}

func selectSprint(sprints []model.Sprint) *model.Sprint {
	for _, s := range sprints {
		if s.State == "active" {
			sp := s
			return &sp
		}
	}
	for _, s := range sprints {
		if s.State == "future" {
			sp := s
			return &sp
		}
	}
	if len(sprints) > 0 {
		sp := sprints[len(sprints)-1]
		return &sp
	}
	return nil
}

func (m Model) fetchIssues() tea.Cmd {
	client, sprintID := m.client, m.sprint.ID
	return func() tea.Msg {
		result, err := client.GetSprintIssues(sprintID, 0, 200)
		return issuesLoadedMsg{result: result, err: err}
	}
}

func groupByStatusCategory(issues []model.Issue) []column {
	byCategory := map[string][]model.Issue{}
	var order []string
	for _, is := range issues {
		cat := is.StatusCategory
		if cat == "" {
			cat = "Other"
		}
		if _, ok := byCategory[cat]; !ok {
			order = append(order, cat)
		}
		byCategory[cat] = append(byCategory[cat], is)
	}

	var cols []column
	seen := map[string]bool{}
	for _, name := range preferredColumnOrder {
		if issues, ok := byCategory[name]; ok {
			cols = append(cols, column{name: name, issues: issues})
			seen[name] = true
		}
	}
	for _, name := range order {
		if !seen[name] {
			cols = append(cols, column{name: name, issues: byCategory[name]})
		}
	}
	return cols
}

func (m Model) Update(msg tea.Msg) (screen.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case sprintsLoadedMsg:
		if msg.err != nil {
			m.loading = false
			return m, screen.Error(msg.err)
		}
		m.sprint = selectSprint(msg.sprints)
		if m.sprint == nil {
			m.loading = false
			m.noSprint = true
			return m, nil
		}
		return m, m.fetchIssues()

	case issuesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			return m, screen.Error(msg.err)
		}
		m.columns = groupByStatusCategory(msg.result.Issues)
		m.cursors = make([]int, len(m.columns))
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
	if len(m.columns) == 0 {
		return m, nil
	}

	switch {
	case msg.String() == "left" || msg.String() == "h":
		if m.activeCol > 0 {
			m.activeCol--
		}
		return m, nil
	case msg.String() == "right" || msg.String() == "l":
		if m.activeCol < len(m.columns)-1 {
			m.activeCol++
		}
		return m, nil
	case key.Matches(msg, keys.Global.Up):
		if m.cursors[m.activeCol] > 0 {
			m.cursors[m.activeCol]--
		}
		return m, nil
	case key.Matches(msg, keys.Global.Down):
		if m.cursors[m.activeCol] < len(m.columns[m.activeCol].issues)-1 {
			m.cursors[m.activeCol]++
		}
		return m, nil
	case key.Matches(msg, keys.Global.Enter):
		col := m.columns[m.activeCol]
		idx := m.cursors[m.activeCol]
		if idx < 0 || idx >= len(col.issues) {
			return m, nil
		}
		issue := col.issues[idx]
		return m, screen.Push(issuedetail.New(m.client, issue.Key))
	}

	return m, nil
}

// cardHeight is the number of terminal lines each rendered card box occupies
// (rounded border top/bottom + 2 lines of content, no gap between cards).
const cardHeight = 4

func (m Model) View() string {
	if m.loading {
		return m.spinner.View() + " loading sprint board..."
	}
	if m.noSprint {
		return styles.Faint.Render("No active or future sprint found for this board.")
	}
	if len(m.columns) == 0 {
		return styles.Faint.Render("Sprint has no issues.")
	}

	colWidth := max((m.width-len(m.columns)*2)/len(m.columns), 20)

	// The board runs inside a fixed-size alt-screen terminal with no
	// scrollback, so a column taller than the available height would
	// otherwise render its top (including the header above) permanently
	// off-screen with no way to see it. Window each column instead, keeping
	// its selection in view.
	maxCards := max((max(m.height-9, 4))/cardHeight, 1)

	rendered := make([]string, len(m.columns))
	for i, col := range m.columns {
		rendered[i] = m.renderColumn(col, i, colWidth, maxCards)
	}

	header := styles.Title.Render(fmt.Sprintf("%s — %s", m.boardName, m.sprint.Name))
	return header + "\n\n" + lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

func windowStart(cursor, total, maxVisible int) int {
	if total <= maxVisible {
		return 0
	}
	start := cursor - maxVisible/2
	if start < 0 {
		start = 0
	}
	if start > total-maxVisible {
		start = total - maxVisible
	}
	return start
}

func (m Model) renderColumn(col column, colIndex, width, maxCards int) string {
	title := fmt.Sprintf("%s (%d)", col.name, len(col.issues))
	if colIndex == m.activeCol {
		title = styles.Selected.Render(title)
	} else {
		title = styles.Faint.Render(title)
	}

	cursor := m.cursors[colIndex]
	start := windowStart(cursor, len(col.issues), maxCards)
	end := min(start+maxCards, len(col.issues))

	var cards string
	if start > 0 {
		cards += styles.Faint.Render(fmt.Sprintf("↑ %d more above", start)) + "\n"
	}
	for i := start; i < end; i++ {
		is := col.issues[i]
		card := is.Key + "\n" + truncate(is.Summary, width-4)
		style := styles.Border.Width(width - 2)
		if colIndex == m.activeCol && i == cursor {
			style = style.BorderForeground(styles.ColorPrimary)
		}
		cards += style.Render(card) + "\n"
	}
	if remaining := len(col.issues) - end; remaining > 0 {
		cards += styles.Faint.Render(fmt.Sprintf("↓ %d more below", remaining)) + "\n"
	}

	return lipgloss.NewStyle().Width(width).Render(title + "\n" + cards)
}

func truncate(s string, max int) string {
	if max <= 1 || len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func (m Model) Title() string {
	return "Sprint Board"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
