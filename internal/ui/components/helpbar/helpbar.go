// Package helpbar wraps bubbles/help to render the keybinding hint line,
// toggling between short (single-line) and full (multi-column) help.
package helpbar

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/krishnan/jira-tui/internal/ui/keys"
)

type Model struct {
	help help.Model
	Full bool
}

func New() Model {
	return Model{help: help.New()}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		m.help.Width = wsm.Width
	}
	return m, nil
}

// Toggle switches between short and full help.
func (m Model) Toggle() Model {
	m.Full = !m.Full
	m.help.ShowAll = m.Full
	return m
}

func (m Model) View() string {
	return m.help.View(keys.Global)
}
