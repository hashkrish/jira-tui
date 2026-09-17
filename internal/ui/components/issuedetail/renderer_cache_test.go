package issuedetail

import "testing"

// glamour.WithAutoStyle() queries the terminal for its background color
// (an OSC escape sequence round-trip) every time a renderer is built, which
// can take seconds on a slow/unresponsive terminal (SSH, tmux, ...).
// Rebuilding it on every tab switch made switching tabs feel like it hung.
// This locks in that the renderer is reused across calls at the same width.
func TestMarkdownRendererIsCachedAtSameWidth(t *testing.T) {
	m := New(nil, "PROJ-1")
	m.width = 80

	r1, err := m.markdownRenderer()
	if err != nil {
		t.Fatalf("markdownRenderer() error = %v", err)
	}
	r2, err := m.markdownRenderer()
	if err != nil {
		t.Fatalf("markdownRenderer() error = %v", err)
	}
	if r1 != r2 {
		t.Error("markdownRenderer() built a new renderer on the second call at the same width — it should be cached")
	}

	m.width = 120
	r3, err := m.markdownRenderer()
	if err != nil {
		t.Fatalf("markdownRenderer() error = %v", err)
	}
	if r3 == r1 {
		t.Error("markdownRenderer() reused the renderer after the width changed — word wrap would be stale")
	}
}
