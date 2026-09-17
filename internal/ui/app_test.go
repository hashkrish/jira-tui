package ui

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/krishnan/jira-tui/internal/ui/components/placeholder"
	"github.com/krishnan/jira-tui/internal/ui/screen"
)

func newTestApp() App {
	home := placeholder.New(
		"Home",
		"press enter to push",
		screen.Push(placeholder.New("Child", "child body", nil)),
	)
	return New(nil, home, "Ada Lovelace", "example.atlassian.net")
}

func TestNavigationPushAndBack(t *testing.T) {
	tm := teatest.NewTestModel(t, newTestApp(), teatest.WithInitialTermSize(80, 24))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Home"))
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("child body"))
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("press enter to push"))
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestQuit(t *testing.T) {
	tm := teatest.NewTestModel(t, newTestApp(), teatest.WithInitialTermSize(80, 24))
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}
