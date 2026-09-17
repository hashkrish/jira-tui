// Package issuedetail implements the issue detail screen: metadata
// sidebar-style header plus tabbed Description/Comments/Worklog/History,
// each rendered as scrollable Markdown (ADF converted via internal/jiraclient/adf,
// styled via Glamour).
package issuedetail

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"

	"github.com/krishnan/jira-tui/internal/jiraclient"
	"github.com/krishnan/jira-tui/internal/jiraclient/adf"
	"github.com/krishnan/jira-tui/internal/model"
	"github.com/krishnan/jira-tui/internal/ui/screen"
	"github.com/krishnan/jira-tui/internal/ui/styles"
)

var tabNames = []string{"Details", "Comments", "Worklog", "History"}

type issueLoadedMsg struct {
	issue *model.Issue
	err   error
}

type commentsLoadedMsg struct {
	comments []model.Comment
	err      error
}

type worklogsLoadedMsg struct {
	worklogs []model.Worklog
	err      error
}

type Model struct {
	client *jiraclient.Client
	key    string

	issue    *model.Issue
	comments []model.Comment
	worklogs []model.Worklog

	activeTab int
	viewport  viewport.Model
	spinner   spinner.Model
	loading   bool

	width, height int
}

// New creates the issue detail screen for the given issue key. summary is
// shown immediately (from the row the user selected) while full detail loads.
func New(client *jiraclient.Client, key string) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return Model{
		client:   client,
		key:      key,
		viewport: viewport.New(80, 20),
		spinner:  sp,
		loading:  true,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchIssue(), m.fetchComments(), m.fetchWorklogs())
}

func (m Model) fetchIssue() tea.Cmd {
	client, key := m.client, m.key
	return func() tea.Msg {
		issue, err := client.GetIssue(key)
		return issueLoadedMsg{issue: issue, err: err}
	}
}

func (m Model) fetchComments() tea.Cmd {
	client, key := m.client, m.key
	return func() tea.Msg {
		comments, err := client.GetComments(key)
		return commentsLoadedMsg{comments: comments, err: err}
	}
}

func (m Model) fetchWorklogs() tea.Cmd {
	client, key := m.client, m.key
	return func() tea.Msg {
		worklogs, err := client.GetWorklogs(key)
		return worklogsLoadedMsg{worklogs: worklogs, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (screen.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = max(msg.Height-9, 3)
		m.refreshContent()
		return m, nil

	case issueLoadedMsg:
		if msg.err != nil {
			m.loading = false
			return m, screen.Error(msg.err)
		}
		m.issue = msg.issue
		m.loading = false
		m.refreshContent()
		return m, nil

	case commentsLoadedMsg:
		if msg.err != nil {
			return m, screen.Error(msg.err)
		}
		m.comments = msg.comments
		m.refreshContent()
		return m, nil

	case worklogsLoadedMsg:
		if msg.err != nil {
			return m, screen.Error(msg.err)
		}
		m.worklogs = msg.worklogs
		m.refreshContent()
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

var (
	nextTabKey = key.NewBinding(key.WithKeys("tab"))
	prevTabKey = key.NewBinding(key.WithKeys("shift+tab"))
)

func (m Model) handleKey(msg tea.KeyMsg) (screen.Screen, tea.Cmd) {
	switch {
	case key.Matches(msg, nextTabKey):
		m.activeTab = (m.activeTab + 1) % len(tabNames)
		m.refreshContent()
		return m, nil
	case key.Matches(msg, prevTabKey):
		m.activeTab = (m.activeTab - 1 + len(tabNames)) % len(tabNames)
		m.refreshContent()
		return m, nil
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *Model) refreshContent() {
	if m.issue == nil {
		m.viewport.SetContent("Loading issue details...")
		return
	}

	var raw string
	switch tabNames[m.activeTab] {
	case "Details":
		raw = renderDetails(m.issue)
	case "Comments":
		raw = renderComments(m.comments)
	case "Worklog":
		raw = renderWorklogs(m.worklogs)
	case "History":
		raw = renderHistory(m.issue.Changelog)
	}

	rendered, err := renderMarkdown(raw, max(m.width, 40))
	if err != nil {
		m.viewport.SetContent(raw) // fall back to plain markdown source
		return
	}
	m.viewport.SetContent(rendered)
}

func renderMarkdown(md string, width int) (string, error) {
	if md == "" {
		md = "_(nothing here yet)_"
	}
	r, err := glamour.NewTermRenderer(glamour.WithAutoStyle(), glamour.WithWordWrap(width))
	if err != nil {
		return "", err
	}
	return r.Render(md)
}

func renderDetails(issue *model.Issue) string {
	descMD, _ := adf.Render(issue.Description)

	var b strings.Builder
	fmt.Fprintf(&b, "# %s: %s\n\n", issue.Key, issue.Summary)
	fmt.Fprintf(&b, "| | |\n|---|---|\n")
	fmt.Fprintf(&b, "| Type | %s |\n", issue.IssueType)
	fmt.Fprintf(&b, "| Status | %s |\n", issue.Status)
	fmt.Fprintf(&b, "| Priority | %s |\n", issue.Priority)
	fmt.Fprintf(&b, "| Assignee | %s |\n", orNone(issue.Assignee))
	fmt.Fprintf(&b, "| Reporter | %s |\n", orNone(issue.Reporter))
	if len(issue.Labels) > 0 {
		fmt.Fprintf(&b, "| Labels | %s |\n", strings.Join(issue.Labels, ", "))
	}
	if len(issue.Components) > 0 {
		fmt.Fprintf(&b, "| Components | %s |\n", strings.Join(issue.Components, ", "))
	}
	if len(issue.FixVersions) > 0 {
		fmt.Fprintf(&b, "| Fix versions | %s |\n", strings.Join(issue.FixVersions, ", "))
	}
	if issue.DueDate != "" {
		fmt.Fprintf(&b, "| Due date | %s |\n", issue.DueDate)
	}
	if issue.ParentKey != "" {
		fmt.Fprintf(&b, "| Parent | %s |\n", issue.ParentKey)
	}
	b.WriteString("\n")

	if len(issue.Subtasks) > 0 {
		b.WriteString("**Subtasks**\n\n")
		for _, st := range issue.Subtasks {
			fmt.Fprintf(&b, "- %s: %s (%s)\n", st.Key, st.Summary, st.Status)
		}
		b.WriteString("\n")
	}

	if len(issue.Links) > 0 {
		b.WriteString("**Linked issues**\n\n")
		for _, l := range issue.Links {
			fmt.Fprintf(&b, "- %s %s: %s\n", l.Type, l.Key, l.Summary)
		}
		b.WriteString("\n")
	}

	b.WriteString("---\n\n")
	b.WriteString(descMD)
	return b.String()
}

func renderComments(comments []model.Comment) string {
	if len(comments) == 0 {
		return "_No comments._"
	}
	var b strings.Builder
	for i, c := range comments {
		if i > 0 {
			b.WriteString("\n\n---\n\n")
		}
		bodyMD, _ := adf.Render(c.Body)
		fmt.Fprintf(&b, "**%s** · %s\n\n%s", c.Author, c.Created, bodyMD)
	}
	return b.String()
}

func renderWorklogs(worklogs []model.Worklog) string {
	if len(worklogs) == 0 {
		return "_No work logged._"
	}
	var b strings.Builder
	for i, w := range worklogs {
		if i > 0 {
			b.WriteString("\n\n---\n\n")
		}
		commentMD, _ := adf.Render(w.Comment)
		fmt.Fprintf(&b, "**%s** · %s · logged %s\n\n%s", w.Author, w.Started, w.TimeSpent, commentMD)
	}
	return b.String()
}

func renderHistory(entries []model.HistoryEntry) string {
	if len(entries) == 0 {
		return "_No history available._"
	}
	var b strings.Builder
	for i, e := range entries {
		if i > 0 {
			b.WriteString("\n\n")
		}
		fmt.Fprintf(&b, "**%s** · %s\n", e.Author, e.Created)
		for _, c := range e.Changes {
			fmt.Fprintf(&b, "- %s: %s → %s\n", c.Field, orNone(c.From), orNone(c.To))
		}
	}
	return b.String()
}

func orNone(s string) string {
	if s == "" {
		return "_none_"
	}
	return s
}

func (m Model) View() string {
	var b strings.Builder

	var tabParts []string
	for i, name := range tabNames {
		if i == m.activeTab {
			tabParts = append(tabParts, styles.Selected.Render("["+name+"]"))
		} else {
			tabParts = append(tabParts, styles.Faint.Render(name))
		}
	}
	b.WriteString(strings.Join(tabParts, "  "))
	if m.loading {
		b.WriteString("  " + m.spinner.View() + " loading")
	}
	b.WriteString("\n\n")

	b.WriteString(m.viewport.View())
	return b.String()
}

func (m Model) Title() string {
	if m.issue != nil {
		return m.issue.Key + ": " + m.issue.Summary
	}
	return m.key
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
