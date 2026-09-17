// External test package: internal/ui transitively imports issuedetail (via
// projectlist -> boards -> sprintboard), so this test cannot live in
// package issuedetail itself without an import cycle.
package issuedetail_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/hashkrish/jira-tui/internal/config"
	"github.com/hashkrish/jira-tui/internal/jiraclient"
	"github.com/hashkrish/jira-tui/internal/ui"
	"github.com/hashkrish/jira-tui/internal/ui/components/issuedetail"
)

func newFixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var path string
		switch {
		case r.URL.Path == "/rest/api/3/issue/PROJ-1/comment":
			path = "../../../jiraclient/testdata/comments.json"
		case r.URL.Path == "/rest/api/3/issue/PROJ-1/worklog":
			path = "../../../jiraclient/testdata/worklogs.json"
		default:
			path = "../../../jiraclient/testdata/issue_detail.json"
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("reading fixture %s: %v", path, err)
			return
		}
		_, _ = w.Write(data)
	}))
}

func newTestModel(srv *httptest.Server) tea.Model {
	cfg := &config.Config{BaseURL: srv.URL, Email: "test@example.com", APIToken: "tok", AuthMode: "basic"}
	client := jiraclient.New(cfg)
	d := issuedetail.New(client, "PROJ-1")
	return ui.New(client, d, srv.URL)
}

func TestIssueDetailLoadsAllTabs(t *testing.T) {
	srv := newFixtureServer(t)
	defer srv.Close()

	tm := teatest.NewTestModel(t, newTestModel(srv), teatest.WithInitialTermSize(120, 60))

	// Details tab (default): description text rendered from ADF.
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Wire up GitHub Actions"))
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Looks good to me"))
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Debugged flaky test"))
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}
