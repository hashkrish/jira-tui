package jiraclient

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/krishnan/jira-tui/internal/config"
)

func serveFixture(t *testing.T, path string) http.HandlerFunc {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading fixture %s: %v", path, err)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}
}

func newTestClient(srvURL string) *Client {
	cfg := &config.Config{BaseURL: srvURL, Email: "test@example.com", APIToken: "tok", AuthMode: "basic"}
	return New(cfg)
}

func TestSearchIssues(t *testing.T) {
	srv := httptest.NewServer(serveFixture(t, "testdata/search_result.json"))
	defer srv.Close()

	client := newTestClient(srv.URL)
	result, err := client.SearchIssues("project = PROJ", "", 50, nil)
	if err != nil {
		t.Fatalf("SearchIssues() error = %v", err)
	}
	if result.NextPageToken != "" {
		t.Errorf("NextPageToken = %q, want empty (last page)", result.NextPageToken)
	}
	if len(result.Issues) != 2 {
		t.Fatalf("len(Issues) = %d, want 2", len(result.Issues))
	}
	if result.Issues[0].Key != "PROJ-1" || result.Issues[0].Assignee != "Ada Lovelace" {
		t.Errorf("unexpected first issue: %+v", result.Issues[0])
	}
	if result.Issues[1].Assignee != "" {
		t.Errorf("expected unassigned issue to have empty Assignee, got %q", result.Issues[1].Assignee)
	}
}

func TestGetIssue(t *testing.T) {
	srv := httptest.NewServer(serveFixture(t, "testdata/issue_detail.json"))
	defer srv.Close()

	client := newTestClient(srv.URL)
	issue, err := client.GetIssue("PROJ-1")
	if err != nil {
		t.Fatalf("GetIssue() error = %v", err)
	}
	if issue.ParentKey != "PROJ-0" {
		t.Errorf("ParentKey = %q, want PROJ-0", issue.ParentKey)
	}
	if len(issue.Subtasks) != 1 || issue.Subtasks[0].Key != "PROJ-3" {
		t.Errorf("unexpected subtasks: %+v", issue.Subtasks)
	}
	if len(issue.Links) != 1 || issue.Links[0].Key != "PROJ-4" || issue.Links[0].Type != "blocks" {
		t.Errorf("unexpected links: %+v", issue.Links)
	}
	if len(issue.Description) == 0 {
		t.Error("expected non-empty raw ADF description")
	}
}

func TestGetComments(t *testing.T) {
	srv := httptest.NewServer(serveFixture(t, "testdata/comments.json"))
	defer srv.Close()

	client := newTestClient(srv.URL)
	comments, err := client.GetComments("PROJ-1")
	if err != nil {
		t.Fatalf("GetComments() error = %v", err)
	}
	if len(comments) != 1 || comments[0].Author != "Ada Lovelace" {
		t.Errorf("unexpected comments: %+v", comments)
	}
}

func TestGetWorklogs(t *testing.T) {
	srv := httptest.NewServer(serveFixture(t, "testdata/worklogs.json"))
	defer srv.Close()

	client := newTestClient(srv.URL)
	logs, err := client.GetWorklogs("PROJ-1")
	if err != nil {
		t.Fatalf("GetWorklogs() error = %v", err)
	}
	if len(logs) != 1 || logs[0].TimeSpent != "2h" {
		t.Errorf("unexpected worklogs: %+v", logs)
	}
}

func TestListProjectsPaginates(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			data, _ := os.ReadFile("testdata/projects_page1.json")
			_, _ = w.Write(data)
		} else {
			data, _ := os.ReadFile("testdata/projects_page2.json")
			_, _ = w.Write(data)
		}
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)
	projects, err := client.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 paginated calls, got %d", calls)
	}
	if len(projects) != 2 || projects[0].Key != "PROJ" || projects[1].Key != "BETA" {
		t.Errorf("unexpected projects: %+v", projects)
	}
}
