// Package statusbar renders the persistent bottom-of-screen status line:
// connected user, Jira host, current breadcrumb, and a loading spinner.
package statusbar

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/hashkrish/jira-tui/internal/ui/styles"
)

type Model struct {
	UserDisplayName string
	Host            string
	Breadcrumb      string
	Loading         bool
	width           int
	spinner         spinner.Model
}

func New() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return Model{spinner: s}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case spinner.TickMsg:
		if m.Loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

// StartLoading begins the spinner animation.
func (m Model) StartLoading() (Model, tea.Cmd) {
	m.Loading = true
	return m, m.spinner.Tick
}

// StopLoading halts the spinner.
func (m Model) StopLoading() Model {
	m.Loading = false
	return m
}

func (m Model) View() string {
	left := fmt.Sprintf("%s @ %s", m.UserDisplayName, m.Host)
	if m.Breadcrumb != "" {
		left += "  ›  " + m.Breadcrumb
	}
	right := ""
	if m.Loading {
		right = m.spinner.View() + " loading"
	}

	content := left
	if right != "" {
		pad := m.width - len(left) - len(right) - 2
		if pad < 1 {
			pad = 1
		}
		content = left + repeat(" ", pad) + right
	}

	return styles.StatusBar.Width(max(m.width, 0)).Render(content)
}

func repeat(s string, n int) string {
	out := make([]byte, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, s[0])
	}
	return string(out)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
