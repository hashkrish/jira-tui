package ui

import (
	"bytes"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/hashkrish/jira-tui/internal/ui/components/placeholder"
	"github.com/hashkrish/jira-tui/internal/ui/screen"
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

// TestHelpPopup drives App.Update directly (rather than through teatest's
// output stream, which only proves text appeared *at some point* and can't
// prove it later stopped being shown) to verify: "?" replaces the screen
// with the keybinding popup rather than expanding the footer, any other key
// closes it and restores the underlying screen, and the navigation stack is
// left untouched by the whole exchange.
func TestHelpPopup(t *testing.T) {
	app := newTestApp()
	sized, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = sized.(App)

	view := app.View()
	if strings.Contains(view, "Keybindings") {
		t.Fatalf("help popup shown before '?' was pressed:\n%s", view)
	}
	if !strings.Contains(view, "press enter to push") {
		t.Fatalf("expected the home screen's body before opening help:\n%s", view)
	}

	next, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	app = next.(App)

	view = app.View()
	if !strings.Contains(view, "Keybindings") {
		t.Fatalf("'?' did not open the help popup:\n%s", view)
	}
	if strings.Contains(view, "press enter to push") {
		t.Fatalf("help popup should fully replace the screen, not sit alongside it:\n%s", view)
	}
	if len(app.stack) != 1 {
		t.Errorf("opening help changed the navigation stack: len = %d, want 1", len(app.stack))
	}

	next, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	app = next.(App)

	view = app.View()
	if strings.Contains(view, "Keybindings") {
		t.Fatalf("help popup still shown after a dismiss key:\n%s", view)
	}
	if !strings.Contains(view, "press enter to push") {
		t.Fatalf("home screen was not restored after closing help:\n%s", view)
	}
	if len(app.stack) != 1 {
		t.Errorf("closing help changed the navigation stack: len = %d, want 1", len(app.stack))
	}
}

func TestQuit(t *testing.T) {
	tm := teatest.NewTestModel(t, newTestApp(), teatest.WithInitialTermSize(80, 24))
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}
