package sprintboard_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/krishnan/jira-tui/internal/config"
	"github.com/krishnan/jira-tui/internal/jiraclient"
	"github.com/krishnan/jira-tui/internal/ui"
	"github.com/krishnan/jira-tui/internal/ui/components/sprintboard"
)

func TestTinyTerminalDoesNotPanic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/agile/1.0/board/1/sprint":
			_, _ = w.Write([]byte(`{"isLast":true,"values":[{"id":1,"name":"S1","state":"active"}]}`))
		default:
			_, _ = w.Write([]byte(`{"startAt":0,"maxResults":50,"total":1,"issues":[{"key":"P-1","fields":{"summary":"x","status":{"name":"To Do","statusCategory":{"name":"To Do"}},"issuetype":{"name":"Task"},"priority":{"name":"Low"}}}]}`))
		}
	}))
	defer srv.Close()

	cfg := &config.Config{BaseURL: srv.URL, Email: "a@b.com", APIToken: "t", AuthMode: "basic"}
	client := jiraclient.New(cfg)
	scr := sprintboard.New(client, 1, "Board")
	app := ui.New(client, scr, "Test User", srv.URL)

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(5, 3))
	time.Sleep(300 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}
