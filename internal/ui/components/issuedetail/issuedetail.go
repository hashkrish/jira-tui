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
	"github.com/charmbracelet/lipgloss"

	"github.com/hashkrish/jira-tui/internal/jiraclient"
	"github.com/hashkrish/jira-tui/internal/jiraclient/adf"
	"github.com/hashkrish/jira-tui/internal/model"
	"github.com/hashkrish/jira-tui/internal/ui/screen"
	"github.com/hashkrish/jira-tui/internal/ui/styles"
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

	// renderer is cached across renders and only rebuilt when the wrap
	// width actually changes — width changes on every single event during
	// a live terminal resize, and glamour.WithAutoStyle() queries the
	// terminal for its background color (an OSC escape sequence round
	// trip) every time it's used to build a renderer, which is NOT cached
	// on glamour's side. Rebuilding per resize event froze the whole app
	// for the duration of the drag. See markdownRenderer for how the
	// style itself is resolved without repeating that query.
	renderer      *glamour.TermRenderer
	rendererWidth int
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

	if tabNames[m.activeTab] == "Details" {
		m.viewport.SetContent(m.renderDetailsTab())
		return
	}

	var raw string
	switch tabNames[m.activeTab] {
	case "Comments":
		raw = renderComments(m.comments)
	case "Worklog":
		raw = renderWorklogs(m.worklogs)
	case "History":
		raw = renderHistory(m.issue.Changelog)
	}

	renderer, err := m.markdownRenderer()
	if err != nil {
		m.viewport.SetContent(raw) // fall back to plain markdown source
		return
	}

	if raw == "" {
		raw = "_(nothing here yet)_"
	}
	rendered, err := renderer.Render(raw)
	if err != nil {
		m.viewport.SetContent(raw)
		return
	}
	m.viewport.SetContent(rendered)
}

// renderDetailsTab renders the heading and metadata as plain Lipgloss text
// (bypassing Glamour), then the subtasks/links/description through Glamour
// as Markdown. The heading skips Glamour because its H1 style renders a
// filled background block behind the text, which read as jarring for a
// one-line issue title; renderMetadata skips it because Glamour has no
// per-value color hook for the styling it needs (see its own doc comment).
func (m *Model) renderDetailsTab() string {
	issue := m.issue
	heading := styles.Title.Render(fmt.Sprintf("%s: %s", issue.Key, issue.Summary))
	bodyMD := renderDetailsBody(issue)

	renderer, err := m.markdownRenderer()
	if err != nil {
		return heading + "\n" + renderMetadata(issue) + "\n\n" + bodyMD
	}

	body, bErr := renderer.Render(bodyMD)
	if bErr != nil {
		body = bodyMD
	}
	return heading + "\n" + renderMetadata(issue) + "\n" + body
}

// markdownRenderer returns the cached Glamour renderer for the current
// width, building a new one only if the width changed (or none exists yet).
func (m *Model) markdownRenderer() (*glamour.TermRenderer, error) {
	width := max(m.width, 40)
	if m.renderer != nil && m.rendererWidth == width {
		return m.renderer, nil
	}

	// glamour.WithAutoStyle() re-runs its own uncached terminal background
	// query on every call, which is exactly what we're rebuilding here on
	// every resize; lipgloss.HasDarkBackground() answers the same
	// question but caches the query process-wide (sync.Once), so resolve
	// dark-vs-light through it instead and hand glamour a fixed style.
	style := "light"
	if lipgloss.HasDarkBackground() {
		style = "dark"
	}

	r, err := glamour.NewTermRenderer(glamour.WithStandardStyle(style), glamour.WithWordWrap(width))
	if err != nil {
		return nil, err
	}
	m.renderer = r
	m.rendererWidth = width
	return r, nil
}

// renderMetadata renders the issue's key fields as a couple of compact,
// bare-value rows (no field labels) instead of one field per line: type/
// status/priority/key on one row, assignee/reporter/due date on the next,
// and labels/components/fix versions/parent as a trailing tag row if any
// are set. Status and priority are colored by what they actually mean
// (status category, priority severity) rather than a single flat accent,
// so the row reads at a glance without needing labels to explain it. This
// is plain Lipgloss-styled text, not Markdown — assembled outside the
// Glamour pipeline so the color survives (see renderDetailsTab).
func renderMetadata(issue *model.Issue) string {
	sep := styles.Faint.Render("  ·  ")

	row1 := []string{styles.Value.Render(issue.IssueType)}
	row1 = append(row1, styles.StatusStyle(issue.StatusCategory).Render(issue.Status))
	row1 = append(row1, styles.PriorityStyle(issue.Priority).Render(issue.Priority))
	row1 = append(row1, styles.Faint.Render(issue.Key))

	row2 := []string{styles.Value.Render(orNone(issue.Assignee))}
	row2 = append(row2, styles.Faint.Render("reported by "+orNone(issue.Reporter)))
	if issue.DueDate != "" {
		row2 = append(row2, styles.Faint.Render("due "+issue.DueDate))
	}

	rows := []string{strings.Join(row1, sep), strings.Join(row2, sep)}

	var tags []string
	tags = append(tags, issue.Labels...)
	tags = append(tags, issue.Components...)
	tags = append(tags, issue.FixVersions...)
	if issue.ParentKey != "" {
		tags = append(tags, "parent "+issue.ParentKey)
	}
	if len(tags) > 0 {
		styled := make([]string, len(tags))
		for i, t := range tags {
			styled[i] = styles.Faint.Render("#" + t)
		}
		rows = append(rows, strings.Join(styled, "  "))
	}

	return strings.Join(rows, "\n")
}

func renderDetailsBody(issue *model.Issue) string {
	descMD, _ := adf.Render(issue.Description)

	var b strings.Builder
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

// tabCount returns the item count to show next to a tab name (e.g.
// "Comments (3)"), or 0 to leave the tab bare. Details/History aren't
// counted here: Details isn't a list, and the changelog's entry count
// isn't a meaningful summary the way a comment/worklog count is.
func (m Model) tabCount(name string) int {
	switch name {
	case "Comments":
		return len(m.comments)
	case "Worklog":
		return len(m.worklogs)
	}
	return 0
}

func (m Model) View() string {
	var b strings.Builder

	var tabParts []string
	for i, name := range tabNames {
		label := name
		if n := m.tabCount(name); n > 0 {
			label = fmt.Sprintf("%s (%d)", name, n)
		}
		if i == m.activeTab {
			tabParts = append(tabParts, styles.Selected.Render("["+label+"]"))
		} else {
			tabParts = append(tabParts, styles.Faint.Render(label))
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
