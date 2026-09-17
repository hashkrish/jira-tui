// Package search wraps bubbles/textinput as a small reusable inline JQL
// input, toggled on/off by the screen that embeds it (rather than being a
// separate pushed Screen), so search-mode keys don't fight with navigation.
package search

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	input  textinput.Model
	Active bool
}

func New() Model {
	ti := textinput.New()
	ti.Placeholder = "JQL query (e.g. project = PROJ order by updated desc)"
	ti.Prompt = "/ "
	return Model{input: ti}
}

// Activate turns on the input, pre-filled with the given JQL, and focuses it.
func (m Model) Activate(prefill string) Model {
	m.Active = true
	m.input.SetValue(prefill)
	m.input.CursorEnd()
	m.input.Focus()
	return m
}

func (m Model) Deactivate() Model {
	m.Active = false
	m.input.Blur()
	return m
}

func (m Model) SetWidth(w int) Model {
	m.input.Width = w
	return m
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) Value() string {
	return m.input.Value()
}

func (m Model) View() string {
	return m.input.View()
}
