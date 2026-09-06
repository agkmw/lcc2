package app

import (
	"strings"
	"testing"
)

// Regression: splicing a panel onto a base row whose style is still
// open at the cut point let that SGR state bleed into the panel text
// - help text randomly rendered bold/bright depending on what sat
// underneath. The splice must terminate the base row's style first.
func TestOverlayTerminatesBaseStyleAtCut(t *testing.T) {
	r := NewStartingAt(0)
	r.width = 40 // panel centers at x > 0, cutting the run mid-style

	// One long bold run, style never closed; the panel lands mid-run.
	base := strings.Repeat("\n", 4) + "\x1b[1m"+strings.Repeat("x", 60)
	panel := "line one\nline two"

	out := r.overlay(base, panel)

	i := strings.Index(out, "\x1b[0m")
	if i < 0 {
		t.Fatal("overlay emitted no style reset")
	}
	after := out[i+len("\x1b[0m"):] // everything past the first reset
	if !strings.HasPrefix(after, "line one") {
		t.Fatalf("panel text does not directly follow the reset: %q", out[i:i+40])
	}
}

