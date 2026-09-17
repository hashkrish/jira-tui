package issuelist_test

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
	"github.com/krishnan/jira-tui/internal/ui/components/issuelist"
)

func TestTinyTerminalDoesNotPanic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"issues":[]}`))
	}))
	defer srv.Close()

	cfg := &config.Config{BaseURL: srv.URL, Email: "a@b.com", APIToken: "t", AuthMode: "basic"}
	client := jiraclient.New(cfg)
	scr := issuelist.New(client, "assignee = currentUser()")
	app := ui.New(client, scr, "Test User", srv.URL)

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(5, 3))
	time.Sleep(300 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}
