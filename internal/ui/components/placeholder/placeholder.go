// Package placeholder provides a minimal Screen implementation used to
// exercise the app shell (navigation, status bar, help, errors) before the
// real screens (issue list, detail, etc.) exist.
package placeholder

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/hashkrish/jira-tui/internal/ui/screen"
	"github.com/hashkrish/jira-tui/internal/ui/styles"
)

type Model struct {
	title   string
	body    string
	onEnter tea.Cmd
}

// New creates a placeholder screen. If onEnter is non-nil, pressing enter
// runs it (e.g. to push a child placeholder and prove navigation works).
func New(title, body string, onEnter tea.Cmd) Model {
	return Model{title: title, body: body, onEnter: onEnter}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (screen.Screen, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		if key.String() == "enter" && m.onEnter != nil {
			return m, m.onEnter
		}
	}
	return m, nil
}

func (m Model) View() string {
	return styles.Title.Render(m.title) + "\n\n" + m.body
}

func (m Model) Title() string {
	return m.title
}
