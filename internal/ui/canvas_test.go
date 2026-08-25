package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"

	"charm.land/lipgloss/v2"
)

func forceTrueColor(t *testing.T) {
	t.Helper()
	restore := SetProfileOverride(colorprofile.TrueColor)
	t.Cleanup(restore)
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
	styled := lipgloss.NewStyle().Foreground(C("#FF0000")).
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
	restore := SetProfileOverride(colorprofile.ASCII)
	defer restore()

	out := Canvas("hello", 8, 2)
	lines := strings.Split(out, "\n")
	if len(lines) != 2 {
		t.Fatalf("%d lines, want 2", len(lines))
	}
	if strings.Contains(out, "\x1b[48") {
		t.Fatal("background escape emitted under Ascii profile")
	}
}

// Nested styling: an inner foreground-only span inside a background
// span must not punch a hole in the outer background — the painter
// re-synthesizes the enclosing style after every inner reset.
func TestCanvasPreservesOuterBackground(t *testing.T) {
	forceTrueColor(t)
	row := SelectedRow().
		Render("plain " + lipgloss.NewStyle().Foreground(Palette.Red).Render("red") + " tail")
	out := PaintBlock(row, 30, BG())
	// The inner span's reset must be replaced by a re-synthesis of the
	// outer style (bold + fg + bg riding in one combined sequence), so
	// the literal bare reset before " tail" disappears...
	if strings.Contains(out, "\x1b[0m tail") {
		t.Fatalf("inner reset leaked, outer bg lost: %q", out)
	}
	// ...the restored style still carries a background parameter...
	rest := out[strings.Index(out, "red\x1b[")+4:]
	if !strings.HasPrefix(rest, "[1;") || !strings.Contains(rest[:40], ";48;2;") {
		t.Fatalf("post-reset synthesis missing bg: %q", rest)
	}
	// ...and the line ends on the canvas fill after the row closes.
	if !strings.Contains(out, "\x1b[0m"+bgSeqFor(BG())) {
		t.Fatalf("canvas fill missing after outer close: %q", out)
	}
}

// Two adjacent chips with different backgrounds: each falls back to
// the canvas when it closes, never bleeding into the next segment.
func TestCanvasAdjacentBackgroundsClose(t *testing.T) {
	forceTrueColor(t)
	a := lipgloss.NewStyle().Background(Palette.Red).Render(" AA ")
	b := lipgloss.NewStyle().Background(Palette.Green).Render(" BB ")
	line := a + " mid " + b + " end"
	out := PaintBlock(line, 40, BG())
	if n := strings.Count(out, bgSeqFor(Palette.Red)); n != 1 {
		t.Errorf("red bg appears %d times, want 1", n)
	}
	if n := strings.Count(out, bgSeqFor(Palette.Green)); n != 1 {
		t.Errorf("green bg appears %d times, want 1", n)
	}
	if !strings.Contains(out, "\x1b[0m"+bgSeqFor(BG())) {
		t.Error("canvas fill not restored after spans closed")
	}
}

// Truncated escape fragments (the resize-corruption signature) are
// dropped instead of passed through to the terminal.
func TestCanvasDropsTruncatedEscapes(t *testing.T) {
	forceTrueColor(t)
	out := PaintBlock("ok \x1b[1;3", 10, BG())
	if strings.Contains(out, "\x1b[1;3") && !strings.HasSuffix(stripSeq(out), "k") {
		t.Errorf("truncated escape leaked: %q", out)
	}
	if lipgloss.Width(out) != 10 {
		t.Errorf("width %d, want 10", lipgloss.Width(out))
	}
}
