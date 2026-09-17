package jiraclient

import (
	"net/http/httptest"
	"testing"
)

func TestListFavoriteFilters(t *testing.T) {
	srv := httptest.NewServer(serveFixture(t, "testdata/filters.json"))
	defer srv.Close()

	client := newTestClient(srv.URL)
	filters, err := client.ListFavoriteFilters()
	if err != nil {
		t.Fatalf("ListFavoriteFilters() error = %v", err)
	}
	if len(filters) != 2 || filters[0].Name != "My Open Issues" || filters[1].JQL != "project = PROJ order by updated desc" {
		t.Errorf("unexpected filters: %+v", filters)
	}
}
