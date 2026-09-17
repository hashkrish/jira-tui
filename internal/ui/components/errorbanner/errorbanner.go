// Package errorbanner renders a dismissible non-fatal error message.
package errorbanner

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/krishnan/jira-tui/internal/ui/styles"
)

type Model struct {
	message string
	width   int
}

// DismissMsg tells the banner to clear itself (e.g. sent when the user
// takes any other action after seeing an error).
type DismissMsg struct{}

func New() Model {
	return Model{}
}

// Show sets the current error message.
func (m Model) Show(err error) Model {
	if err == nil {
		m.message = ""
		return m
	}
	m.message = err.Error()
	return m
}

func (m Model) Visible() bool {
	return m.message != ""
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case DismissMsg:
		m.message = ""
	}
	return m, nil
}

func (m Model) View() string {
	if m.message == "" {
		return ""
	}
	return styles.ErrorBanner.Width(m.width).Render("⚠ " + m.message)
}
