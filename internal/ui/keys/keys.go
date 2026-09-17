// Package keys defines the global keymap shared across screens.
package keys

import "github.com/charmbracelet/bubbles/key"

// GlobalKeyMap holds bindings handled by the root app model regardless of
// which screen is active.
type GlobalKeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Enter     key.Binding
	Back      key.Binding
	Search    key.Binding
	Refresh   key.Binding
	Help      key.Binding
	Quit      key.Binding
	Projects  key.Binding
	Filters   key.Binding
	GotoIssue key.Binding
}

// Global is the default keymap instance.
var Global = GlobalKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "backspace"),
		key.WithHelp("esc", "back"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "more shortcuts"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Projects: key.NewBinding(
		key.WithKeys("P"),
		key.WithHelp("P", "projects"),
	),
	Filters: key.NewBinding(
		key.WithKeys("F"),
		key.WithHelp("F", "filters"),
	),
	// GotoIssue is handled as a "g"-prefixed chord in app.go (not via
	// key.Matches on this binding — WithKeys is unused for that reason)
	// since bubbles/key can't match a two-keystroke sequence directly.
	// "G" alone is left free for a future "go to last" binding (vim-style).
	GotoIssue: key.NewBinding(
		key.WithKeys("g", "t"),
		key.WithHelp("gt", "go to issue key"),
	),
}

// ShortHelp implements help.KeyMap.
func (k GlobalKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Back, k.Search, k.Help, k.Quit}
}

// FullHelp implements help.KeyMap. It backs the full keybinding reference
// popup (see internal/ui/app.go's helpPopupView), which renders each group
// as its own labeled block, so groups are organized by purpose rather than
// constrained to a fixed size.
func (k GlobalKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter, k.Back},
		{k.Search, k.Refresh},
		{k.Projects, k.Filters, k.GotoIssue},
		{k.Help, k.Quit},
	}
}
