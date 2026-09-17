// Package screen defines the interface every navigable view implements, and
// the navigation/error messages used to communicate with the root app
// model. It has no dependencies on internal/ui itself so that individual
// screen components can implement Screen without an import cycle.
package screen

import tea "github.com/charmbracelet/bubbletea"

// Screen is implemented by every navigable view (issue list, issue detail,
// project list, board view, ...).
type Screen interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (Screen, tea.Cmd)
	View() string
	Title() string
}

// PushMsg navigates forward to a new screen, pushing it onto the stack.
type PushMsg struct {
	Screen Screen
}

// PopMsg navigates back to the previous screen on the stack.
type PopMsg struct{}

// ErrMsg surfaces a non-fatal error to the status/error banner without
// crashing the program.
type ErrMsg struct {
	Err error
}

func Push(s Screen) tea.Cmd {
	return func() tea.Msg { return PushMsg{Screen: s} }
}

func Pop() tea.Cmd {
	return func() tea.Msg { return PopMsg{} }
}

func Error(err error) tea.Cmd {
	return func() tea.Msg { return ErrMsg{Err: err} }
}
