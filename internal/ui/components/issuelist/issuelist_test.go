// External test package: internal/ui transitively imports issuelist (via
// filters), so this test cannot live in package issuelist without a cycle.
package issuelist_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/hashkrish/jira-tui/internal/config"
	"github.com/hashkrish/jira-tui/internal/jiraclient"
	"github.com/hashkrish/jira-tui/internal/ui"
	"github.com/hashkrish/jira-tui/internal/ui/components/issuelist"
)

// driveOnce applies a single Cmd's message (and, if it's a tea.Batch, each
// of its sub-Cmds' messages) to app, without chasing anything those
// messages themselves return. That's enough here since none of these
// fixtures trigger further async work beyond the initial load.
func driveOnce(app ui.App, cmd tea.Cmd) ui.App {
	if cmd == nil {
		return app
	}
	msg := cmd()
	if msg == nil {
		return app
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c == nil {
				continue
			}
			m, _ := app.Update(c())
			app = m.(ui.App)
		}
		return app
	}
	m, _ := app.Update(msg)
	return m.(ui.App)
}

// newFixtureServer serves a single-issue page from /rest/api/3/search/jql
// with no nextPageToken (i.e. the only/last page).
func newFixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"issues": [
				{"key": "PROJ-1", "fields": {"summary": "First issue", "status": {"name": "To Do"}, "issuetype": {"name": "Task"}, "priority": {"name": "High"}, "assignee": {"displayName": "Ada Lovelace"}}}
			]
		}`)
	}))
}

// newPaginatedFixtureServer serves two pages: page one (no nextPageToken
// query param) returns a token; page two (nextPageToken=abc) returns none.
func newPaginatedFixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("nextPageToken") == "page2token" {
			fmt.Fprint(w, `{
				"issues": [
					{"key": "PROJ-2", "fields": {"summary": "Second page issue", "status": {"name": "To Do"}, "issuetype": {"name": "Task"}, "priority": {"name": "Low"}, "assignee": null}}
				]
			}`)
			return
		}
		fmt.Fprint(w, `{
			"issues": [
				{"key": "PROJ-1", "fields": {"summary": "First page issue", "status": {"name": "To Do"}, "issuetype": {"name": "Task"}, "priority": {"name": "High"}, "assignee": {"displayName": "Ada Lovelace"}}}
			],
			"nextPageToken": "page2token"
		}`)
	}))
}

func newTestModel(t *testing.T, srv *httptest.Server) tea.Model {
	cfg := &config.Config{BaseURL: srv.URL, Email: "test@example.com", APIToken: "tok", AuthMode: "basic"}
	client := jiraclient.New(cfg)
	scr := issuelist.New(client, "order by updated desc")
	return ui.New(client, scr, "Test User", srv.URL)
}

func TestIssueListLoadsAndRenders(t *testing.T) {
	srv := newFixtureServer(t)
	defer srv.Close()

	tm := teatest.NewTestModel(t, newTestModel(t, srv), teatest.WithInitialTermSize(100, 30))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("PROJ-1")) && bytes.Contains(b, []byte("First issue"))
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestIssueListSearchToggle(t *testing.T) {
	srv := newFixtureServer(t)
	defer srv.Close()

	tm := teatest.NewTestModel(t, newTestModel(t, srv), teatest.WithInitialTermSize(100, 30))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("PROJ-1"))
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		// Active search input is prefilled with the current JQL and uses the
		// "/ " prompt, which does not appear in the inactive "press / to edit" label.
		return bytes.Contains(b, []byte("/ order by updated desc"))
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("press / to edit"))
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// TestIssueListTableDoesNotPadWithBlankRows locks in two related fixes:
// table.SetHeight(h) treats h as the TOTAL height including the header row
// (viewport height = h - header height), so sizing it to just the data row
// count previously zeroed out the viewport when there was only one row
// (hiding the row entirely); and separately, sizing it to the full
// available budget regardless of row count made bubbles/table pad the rest
// with blank rows. This drives the screen directly (no teatest/ANSI stream)
// so it can assert the exact rendered line count.
func TestIssueListTableDoesNotPadWithBlankRows(t *testing.T) {
	srv := newFixtureServer(t)
	defer srv.Close()

	cfg := &config.Config{BaseURL: srv.URL, Email: "test@example.com", APIToken: "tok", AuthMode: "basic"}
	client := jiraclient.New(cfg)
	scr := issuelist.New(client, "order by updated desc")
	app := ui.New(client, scr, "Test User", srv.URL)

	m, cmd := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app = m.(ui.App)
	app = driveOnce(app, cmd)
	app = driveOnce(app, app.Init())

	view := app.View()
	if !strings.Contains(view, "PROJ-1") {
		t.Fatalf("data row missing from rendered view entirely:\n%s", view)
	}

	lines := strings.Split(view, "\n")
	const wantLines = 6 // JQL line, table header, 1 data row, page footer, status bar, help bar
	if len(lines) != wantLines {
		t.Errorf("view has %d lines, want %d (no padded blank rows):\n%s", len(lines), wantLines, view)
	}
}

func TestIssueListPagination(t *testing.T) {
	srv := newPaginatedFixtureServer(t)
	defer srv.Close()

	tm := teatest.NewTestModel(t, newTestModel(t, srv), teatest.WithInitialTermSize(100, 30))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("First page issue"))
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Second page issue"))
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("First page issue"))
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}
