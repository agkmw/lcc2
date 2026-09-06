package screens

import (
	"strings"
	"testing"
)

// Regression: overlayCenter spliced panels (perm editor, chown form,
// review) onto rows whose SGR style could still be open at the cut
// point, bleeding bold/bright into the panel text.
func TestOverlayCenterTerminatesBaseStyleAtCut(t *testing.T) {
	base := strings.Repeat("\n", 4) + "\x1b[1m" + strings.Repeat("x", 60)
	out := overlayCenter(base, "line one\nline two", 40)

	i := strings.Index(out, "\x1b[0m")
	if i < 0 {
		t.Fatal("overlayCenter emitted no style reset")
	}
	after := out[i+len("\x1b[0m"):]
	if !strings.HasPrefix(after, "line one") {
		t.Fatalf("panel text does not directly follow the reset: %q", out[i:i+40])
	}
}
