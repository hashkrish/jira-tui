// Package keys defines the global keymap shared across screens.
package keys

import "github.com/charmbracelet/bubbles/key"

// GlobalKeyMap holds bindings handled by the root app model regardless of
// which screen is active.
type GlobalKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Back     key.Binding
	Search   key.Binding
	Refresh  key.Binding
	Help     key.Binding
	Quit     key.Binding
	Projects key.Binding
	Filters  key.Binding
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
		key.WithHelp("?", "help"),
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
}

// ShortHelp implements help.KeyMap.
func (k GlobalKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Back, k.Search, k.Help, k.Quit}
}

// FullHelp implements help.KeyMap. Groups are capped at 2 bindings each
// (rather than 4) so the rendered help stays 2 rows tall instead of 4 —
// bubbles/help lays each group out as a column, so more/shorter groups
// trade width (which terminals usually have plenty of) for height (which
// they don't).
func (k GlobalKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.Enter, k.Back},
		{k.Search, k.Refresh},
		{k.Projects, k.Filters},
		{k.Help, k.Quit},
	}
}
