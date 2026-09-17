// External test package: internal/ui transitively imports filters.
package filters_test

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
	"github.com/hashkrish/jira-tui/internal/ui/components/filters"
)

func newFixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/api/3/filter/favourite":
			fmt.Fprint(w, `[{"id":"1","name":"My Open Issues","jql":"assignee = currentUser()"}]`)
		case "/rest/api/3/search/jql":
			fmt.Fprint(w, `{"issues":[{"key":"PROJ-9","fields":{"summary":"Filtered issue","status":{"name":"To Do"},"issuetype":{"name":"Task"},"priority":{"name":"Low"}}}]}`)
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
		}
	}))
}

func TestFiltersToIssueListNavigation(t *testing.T) {
	srv := newFixtureServer(t)
	defer srv.Close()

	cfg := &config.Config{BaseURL: srv.URL, Email: "test@example.com", APIToken: "tok", AuthMode: "basic"}
	client := jiraclient.New(cfg)
	app := ui.New(client, filters.New(client), srv.URL)

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(100, 30))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("My Open Issues"))
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Filtered issue"))
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}
