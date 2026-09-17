// Package helpbar renders the persistent single-line keybinding hint at the
// bottom of the screen. The full keybinding reference is a separate popup
// overlay (see internal/ui/app.go), not an expanded form of this bar, so
// this always stays one line regardless of terminal width.
package helpbar

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/hashkrish/jira-tui/internal/ui/keys"
)

type Model struct {
	help help.Model
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

func (m Model) View() string {
	return m.help.ShortHelpView(keys.Global.ShortHelp())
}
