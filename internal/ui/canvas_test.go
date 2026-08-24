package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func forceTrueColor(t *testing.T) {
	t.Helper()
	old := lipgloss.DefaultRenderer().ColorProfile()
	lipgloss.DefaultRenderer().SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.DefaultRenderer().SetColorProfile(old) })
}

// Canvas must return exactly h lines of exactly w visible cells each,
// every one carrying the app background — that is what keeps the app
// opaque on transparent terminals.
func TestCanvasDimensionsAndFill(t *testing.T) {
	forceTrueColor(t)
	frame := "short\n" + strings.Repeat("x", 30)
	out := Canvas(frame, 10, 4)
	lines := strings.Split(out, "\n")
	if len(lines) != 4 {
		t.Fatalf("%d lines, want 4", len(lines))
	}
	seq := bgSeqFor(BG())
	if seq == "" {
		t.Fatal("no bg sequence for BG()")
	}
	for i, l := range lines {
		if w := lipgloss.Width(l); w != 10 {
			t.Errorf("line %d: %d cells, want 10", i, w)
		}
		if !strings.Contains(l, seq) {
			t.Errorf("line %d: missing app background fill", i)
		}
	}
}

// Inner styled spans end with a reset; the canvas must re-open its
// background after each reset so no transparent holes appear.
func TestCanvasSurvivesInnerResets(t *testing.T) {
	forceTrueColor(t)
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).
		Render("red") + " plain"
	out := PaintBlock(styled, 20, Palette.Surface)
	seq := bgSeqFor(Palette.Surface)
	if n := strings.Count(out, seq); n < 2 {
		t.Fatalf("bg sequence appears %d times, want >=2 (open + post-reset)", n)
	}
}

// On color-less profiles the painter must degrade to plain clipping —
// raw escapes would corrupt NO_COLOR terminals.
func TestCanvasAsciiProfileSkipsEscapes(t *testing.T) {
	old := lipgloss.DefaultRenderer().ColorProfile()
	lipgloss.DefaultRenderer().SetColorProfile(termenv.Ascii)
	defer lipgloss.DefaultRenderer().SetColorProfile(old)

	out := Canvas("hello", 8, 2)
	lines := strings.Split(out, "\n")
	if len(lines) != 2 {
		t.Fatalf("%d lines, want 2", len(lines))
	}
	if strings.Contains(out, "\x1b[48") {
		t.Fatal("background escape emitted under Ascii profile")
	}
}
