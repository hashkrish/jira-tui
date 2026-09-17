// Package statusbar renders the persistent bottom-of-screen status line:
// connected user, Jira host, current breadcrumb, and a loading spinner.
package statusbar

import (
	"net/url"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/hashkrish/jira-tui/internal/ui/styles"
)

type Model struct {
	Host       string
	Breadcrumb string
	// PageInfo is a screen-provided right-aligned summary (e.g. paging
	// state) refreshed every render from the active screen; it isn't
	// persisted across screens the way Host/Breadcrumb are.
	PageInfo string
	Loading  bool
	width    int
	spinner  spinner.Model
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
	left := companyFromHost(m.Host)
	if m.Breadcrumb != "" {
		left += "  ›  " + m.Breadcrumb
	}
	right := m.PageInfo
	if m.Loading {
		spin := m.spinner.View() + " loading"
		if right != "" {
			right = spin + "  " + right
		} else {
			right = spin
		}
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

// companyFromHost extracts the company subdomain from a Jira base URL
// (e.g. "https://nptel-hq.atlassian.net" -> "nptel-hq"), keeping the status
// bar short and free of unrelated URL/domain noise.
func companyFromHost(host string) string {
	u, err := url.Parse(host)
	if err != nil || u.Host == "" {
		return host
	}
	return strings.SplitN(u.Host, ".", 2)[0]
}
