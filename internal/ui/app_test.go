package ui

import (
	"bytes"
	"errors"
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

// TestFooterHasNoExtraBlankLine locks in that the chrome below the screen
// body is exactly two lines (status bar, then help bar) directly after the
// body's own last line, with no blank filler in between. An earlier attempt
// pinned the footer to the terminal's last row by inserting padding between
// the body and the footer, but that just relocated the "stray blank lines"
// complaint (from below the footer to above it) rather than resolving it —
// the fix is to not insert any filler at all and accept that a screen
// shorter than the terminal simply leaves natural blank rows at the very
// bottom of the alt-screen, not visible as a gap between rendered content.
func TestFooterHasNoExtraBlankLine(t *testing.T) {
	app := newTestApp()
	sized, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = sized.(App)

	view := app.View()
	bodyLines := strings.Count(placeholder.New("Home", "press enter to push", nil).View(), "\n") + 1
	wantLines := bodyLines + 2 // + status bar + help bar, no blank filler
	gotLines := strings.Count(view, "\n") + 1
	if gotLines != wantLines {
		t.Errorf("footer view has %d lines, want %d (body=%d + status + help, no filler):\n%s",
			gotLines, wantLines, bodyLines, view)
	}
}

// TestErrorBannerDismissedByGlobalShortcuts locks in the fix for a bug where
// pressing Back/Projects/Filters/Help returned early from the key handler,
// bypassing the "any key dismisses a shown error" logic — so a stale error
// banner could linger on screen indefinitely (adding a persistent extra
// line) until the user happened to press some other, unhandled key.
func TestErrorBannerDismissedByGlobalShortcuts(t *testing.T) {
	app := newTestApp()
	sized, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = sized.(App)

	next, _ := app.Update(screen.ErrMsg{Err: errors.New("boom")})
	app = next.(App)

	if !strings.Contains(app.View(), "boom") {
		t.Fatalf("error banner not shown after ErrMsg:\n%s", app.View())
	}

	// esc matches the Back binding, which used to return early and skip
	// dismissing the error.
	next, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = next.(App)

	if strings.Contains(app.View(), "boom") {
		t.Errorf("error banner still shown after a global-shortcut key (esc):\n%s", app.View())
	}
}

func TestQuit(t *testing.T) {
	tm := teatest.NewTestModel(t, newTestApp(), teatest.WithInitialTermSize(80, 24))
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}
