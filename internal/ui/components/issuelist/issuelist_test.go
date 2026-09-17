// External test package: internal/ui transitively imports issuelist (via
// filters), so this test cannot live in package issuelist without a cycle.
package issuelist_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
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
