package jiraclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashkrish/jira-tui/internal/config"
)

func TestMyself(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/myself" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got == "" {
			t.Error("expected Authorization header to be set")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Myself{
			AccountID:    "abc123",
			DisplayName:  "Test User",
			EmailAddress: "test@example.com",
		})
	}))
	defer srv.Close()

	cfg := &config.Config{BaseURL: srv.URL, Email: "test@example.com", APIToken: "tok", AuthMode: "basic"}
	client := New(cfg)

	me, err := client.Myself()
	if err != nil {
		t.Fatalf("Myself() error = %v", err)
	}
	if me.DisplayName != "Test User" {
		t.Errorf("DisplayName = %q, want %q", me.DisplayName, "Test User")
	}
}

func TestMyselfAuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"errorMessages":["Unauthorized"]}`))
	}))
	defer srv.Close()

	cfg := &config.Config{BaseURL: srv.URL, Email: "test@example.com", APIToken: "bad", AuthMode: "basic"}
	client := New(cfg)

	_, err := client.Myself()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusUnauthorized)
	}
}
