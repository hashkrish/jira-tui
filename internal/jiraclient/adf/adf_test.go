package adf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderGolden(t *testing.T) {
	cases := []string{"basic", "extended"}

	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("testdata", name+".json"))
			if err != nil {
				t.Fatalf("reading fixture: %v", err)
			}
			want, err := os.ReadFile(filepath.Join("testdata", name+".md"))
			if err != nil {
				t.Fatalf("reading golden file: %v", err)
			}

			got, err := Render(raw)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			gotTrimmed := strings.TrimRight(got, "\n")
			wantTrimmed := strings.TrimRight(string(want), "\n")
			if gotTrimmed != wantTrimmed {
				t.Errorf("Render() mismatch\n--- got ---\n%s\n--- want ---\n%s", gotTrimmed, wantTrimmed)
			}
		})
	}
}

func TestRenderEmpty(t *testing.T) {
	got, err := Render(nil)
	if err != nil {
		t.Fatalf("Render(nil) error = %v", err)
	}
	if got != "" {
		t.Errorf("Render(nil) = %q, want empty string", got)
	}
}

func TestRenderInvalidJSON(t *testing.T) {
	_, err := Render([]byte("{not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}
