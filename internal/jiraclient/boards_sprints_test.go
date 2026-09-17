package jiraclient

import (
	"net/http/httptest"
	"testing"
)

func TestListBoards(t *testing.T) {
	srv := httptest.NewServer(serveFixture(t, "testdata/boards.json"))
	defer srv.Close()

	client := newTestClient(srv.URL)
	boards, err := client.ListBoards("PROJ")
	if err != nil {
		t.Fatalf("ListBoards() error = %v", err)
	}
	if len(boards) != 2 || boards[0].Name != "Alpha Board" || boards[1].Type != "kanban" {
		t.Errorf("unexpected boards: %+v", boards)
	}
}

func TestListSprints(t *testing.T) {
	srv := httptest.NewServer(serveFixture(t, "testdata/sprints.json"))
	defer srv.Close()

	client := newTestClient(srv.URL)
	sprints, err := client.ListSprints(1)
	if err != nil {
		t.Fatalf("ListSprints() error = %v", err)
	}
	if len(sprints) != 2 || sprints[1].State != "active" {
		t.Errorf("unexpected sprints: %+v", sprints)
	}
}

func TestGetSprintIssues(t *testing.T) {
	srv := httptest.NewServer(serveFixture(t, "testdata/sprint_issues.json"))
	defer srv.Close()

	client := newTestClient(srv.URL)
	result, err := client.GetSprintIssues(11, 0, 50)
	if err != nil {
		t.Fatalf("GetSprintIssues() error = %v", err)
	}
	if len(result.Issues) != 1 || result.Issues[0].Key != "PROJ-5" {
		t.Errorf("unexpected result: %+v", result)
	}
}
