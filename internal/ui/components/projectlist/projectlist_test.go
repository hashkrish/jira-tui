// External test package: internal/ui transitively imports projectlist, so
// this test cannot live in package projectlist itself without a cycle.
package projectlist_test

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
	"github.com/hashkrish/jira-tui/internal/ui/components/projectlist"
)

// newFixtureServer serves canned responses for the whole
// project -> boards -> sprint -> issues navigation chain.
func newFixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/rest/api/3/project/search":
			fmt.Fprint(w, `{"isLast": true, "values": [{"key": "PROJ", "name": "Project Alpha", "lead": {"displayName": "Grace Hopper"}}]}`)
		case r.URL.Path == "/rest/agile/1.0/board":
			fmt.Fprint(w, `{"isLast": true, "values": [{"id": 1, "name": "Alpha Board", "type": "scrum"}]}`)
		case r.URL.Path == "/rest/agile/1.0/board/1/sprint":
			fmt.Fprint(w, `{"isLast": true, "values": [{"id": 11, "name": "Sprint 2", "state": "active"}]}`)
		case r.URL.Path == "/rest/agile/1.0/sprint/11/issue":
			fmt.Fprint(w, `{"startAt":0,"maxResults":50,"total":1,"issues":[{"key":"PROJ-5","fields":{"summary":"Sprint task","status":{"name":"In Progress","statusCategory":{"name":"In Progress"}},"issuetype":{"name":"Task"},"priority":{"name":"Medium"},"assignee":{"displayName":"Grace Hopper"}}}]}`)
		case r.URL.Path == "/rest/api/3/issue/PROJ-5":
			fmt.Fprint(w, `{"key":"PROJ-5","fields":{"summary":"Sprint task","status":{"name":"In Progress","statusCategory":{"name":"In Progress"}},"issuetype":{"name":"Task"},"priority":{"name":"Medium"},"assignee":{"displayName":"Grace Hopper"}}}`)
		case r.URL.Path == "/rest/api/3/issue/PROJ-5/comment":
			fmt.Fprint(w, `{"comments":[]}`)
		case r.URL.Path == "/rest/api/3/issue/PROJ-5/worklog":
			fmt.Fprint(w, `{"worklogs":[]}`)
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
		}
	}))
}

func newTestModel(srv *httptest.Server) tea.Model {
	cfg := &config.Config{BaseURL: srv.URL, Email: "test@example.com", APIToken: "tok", AuthMode: "basic"}
	client := jiraclient.New(cfg)
	return ui.New(client, projectlist.New(client), srv.URL)
}

func TestProjectToBoardToSprintNavigation(t *testing.T) {
	srv := newFixtureServer(t)
	defer srv.Close()

	tm := teatest.NewTestModel(t, newTestModel(srv), teatest.WithInitialTermSize(120, 40))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Project Alpha"))
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // project -> boards

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Alpha Board"))
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // board -> sprint board

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Sprint 2")) && bytes.Contains(b, []byte("PROJ-5"))
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // card -> issue detail

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Sprint task"))
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}
