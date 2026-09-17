package sprintboard

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/krishnan/jira-tui/internal/config"
	"github.com/krishnan/jira-tui/internal/jiraclient"
)

// drive runs a Cmd (which may be a tea.Batch of several Cmds, as Init/handleKey
// return) to completion and applies every resulting message to m in turn,
// deterministically, without a real Bubble Tea program or terminal. It
// applies a spinner.TickMsg once but does not chase the follow-up tick Cmd
// spinner.Update returns while animating — that chain never terminates on
// its own (a real program only stops it by pausing the ticker or by the
// screen no longer being the active one).
func drive(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	for cmd != nil {
		msg := cmd()
		if msg == nil {
			return m
		}
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, c := range batch {
				if c == nil {
					continue
				}
				m = applyOnce(m, c())
			}
			return m
		}
		m, cmd = applyAndContinue(m, msg)
	}
	return m
}

func applyOnce(m Model, msg tea.Msg) Model {
	if msg == nil {
		return m
	}
	next, cmd := m.Update(msg)
	m = next.(Model)
	if _, isTick := msg.(spinner.TickMsg); isTick {
		return m // don't chase the animation's next tick
	}
	if cmd != nil {
		return applyOnce(m, cmd())
	}
	return m
}

func applyAndContinue(m Model, msg tea.Msg) (Model, tea.Cmd) {
	next, cmd := m.Update(msg)
	m = next.(Model)
	if _, isTick := msg.(spinner.TickMsg); isTick {
		return m, nil // don't chase the animation's next tick
	}
	return m, cmd
}

// newManyIssuesServer serves a board with one active sprint whose "To Do"
// column has far more issues than fit in a small terminal, reproducing the
// overflow bug: without windowing, the board's header and any shorter
// columns scroll permanently off the top of the fixed-size alt-screen
// terminal (which has no scrollback), and the user is left staring at
// nothing but the tail of the tallest column.
func newManyIssuesServer(t *testing.T, count int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/agile/1.0/board/1/sprint":
			fmt.Fprint(w, `{"isLast":true,"values":[{"id":11,"name":"Sprint 1","state":"active"}]}`)
		case "/rest/agile/1.0/sprint/11/issue":
			var b strings.Builder
			b.WriteString(`{"startAt":0,"maxResults":200,"total":0,"issues":[`)
			for i := 0; i < count; i++ {
				if i > 0 {
					b.WriteString(",")
				}
				fmt.Fprintf(&b, `{"key":"WOR-%d","fields":{"summary":"Issue number %d","status":{"name":"To Do","statusCategory":{"name":"To Do"}},"issuetype":{"name":"Task"},"priority":{"name":"Low"}}}`, i, i)
			}
			b.WriteString(`]}`)
			_, _ = w.Write([]byte(b.String()))
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
		}
	}))
}

func newTestModel(t *testing.T, srv *httptest.Server, width, height int) Model {
	t.Helper()
	cfg := &config.Config{BaseURL: srv.URL, Email: "test@example.com", APIToken: "tok", AuthMode: "basic"}
	client := jiraclient.New(cfg)
	m := New(client, 1, "Alpha Board")

	sized, sizeCmd := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	m = sized.(Model)
	if sizeCmd != nil {
		m = drive(t, m, sizeCmd)
	}
	m = drive(t, m, m.Init())
	return m
}

func TestOverflowKeepsHeaderVisibleAndWindowsToSelection(t *testing.T) {
	srv := newManyIssuesServer(t, 20)
	defer srv.Close()

	m := newTestModel(t, srv, 100, 15)

	view := m.View()
	if !strings.Contains(view, "Alpha Board") {
		t.Fatalf("initial view is missing the board header entirely:\n%s", view)
	}
	if !strings.Contains(view, "WOR-0") {
		t.Errorf("initial view should show the first card:\n%s", view)
	}

	lineCount := strings.Count(view, "\n") + 1
	if lineCount > 15 {
		t.Errorf("rendered view is %d lines, taller than the 15-line terminal — the header/other content would scroll off-screen with no scrollback:\n%s", lineCount, view)
	}

	// Move the cursor down past whatever window is currently visible.
	for i := 0; i < 15; i++ {
		next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = next.(Model)
		if cmd != nil {
			m = drive(t, m, cmd)
		}
	}

	view = m.View()
	if !strings.Contains(view, "Alpha Board") {
		t.Errorf("board header disappeared after scrolling down — it must stay visible since the whole board is windowed to fit:\n%s", view)
	}
	if !strings.Contains(view, "WOR-15") {
		t.Errorf("selected card (WOR-15, cursor at index 15) is not visible after scrolling down — windowing did not follow the cursor:\n%s", view)
	}

	lineCount = strings.Count(view, "\n") + 1
	if lineCount > 15 {
		t.Errorf("rendered view after scrolling is %d lines, taller than the 15-line terminal:\n%s", lineCount, view)
	}
}

func TestWindowStart(t *testing.T) {
	tests := []struct {
		name                  string
		cursor, total, maxVis int
		want                  int
	}{
		{"fits entirely", 2, 5, 10, 0},
		{"cursor at start", 0, 20, 5, 0},
		{"cursor centered", 10, 20, 5, 8},
		{"cursor near end clamps to last window", 19, 20, 5, 15},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := windowStart(tt.cursor, tt.total, tt.maxVis)
			if got != tt.want {
				t.Errorf("windowStart(%d, %d, %d) = %d, want %d", tt.cursor, tt.total, tt.maxVis, got, tt.want)
			}
		})
	}
}
