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
	return ui.New(client, scr, srv.URL)
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
// with blank rows. This drives the issuelist.Model directly (not wrapped in
// ui.App, which separately pins its footer to the terminal's last row —
// that's a deliberate, unrelated concern this test isn't checking) so it
// can assert the screen's own rendered line count precisely.
func TestIssueListTableDoesNotPadWithBlankRows(t *testing.T) {
	srv := newFixtureServer(t)
	defer srv.Close()

	cfg := &config.Config{BaseURL: srv.URL, Email: "test@example.com", APIToken: "tok", AuthMode: "basic"}
	client := jiraclient.New(cfg)
	scr := issuelist.New(client, "order by updated desc")

	next, cmd := scr.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m := next.(issuelist.Model)
	m = driveScreenOnce(m, cmd)
	m = driveScreenOnce(m, m.Init())

	view := m.View()
	if !strings.Contains(view, "PROJ-1") {
		t.Fatalf("data row missing from rendered view entirely:\n%s", view)
	}

	lines := strings.Split(view, "\n")
	const wantLines = 3 // JQL line, table header, 1 data row (page info now lives in the status bar)
	if len(lines) != wantLines {
		t.Errorf("view has %d lines, want %d (no padded blank rows):\n%s", len(lines), wantLines, view)
	}
}

// driveScreenOnce applies a single Cmd's message (and, if it's a tea.Batch,
// each of its sub-Cmds' messages) to m, without chasing anything those
// messages themselves return. Mirrors driveOnce but for a bare
// issuelist.Model instead of one wrapped in ui.App.
func driveScreenOnce(m issuelist.Model, cmd tea.Cmd) issuelist.Model {
	if cmd == nil {
		return m
	}
	msg := cmd()
	if msg == nil {
		return m
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c == nil {
				continue
			}
			next, _ := m.Update(c())
			m = next.(issuelist.Model)
		}
		return m
	}
	next, _ := m.Update(msg)
	return next.(issuelist.Model)
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
