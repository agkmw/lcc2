package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// The box must be a perfect rectangle: top and bottom borders exactly
// w cells, corners in the same column (regression: the top-right ┐
// sat one column short of every body row's right edge).
func TestSectionRectangle(t *testing.T) {
	for _, w := range []int{20, 40, 64, 106} {
		box := Section("proc", "cpu", "load 1.0 - 4 cores", w, 5, "row1\nrow2")
		lines := strings.Split(box, "\n")
		if len(lines) != 5 {
			t.Fatalf("w=%d: %d lines", w, len(lines))
		}
		for i, l := range lines {
			if lw := lipglossWidth(l); lw != w {
				t.Errorf("w=%d: line %d is %d cells (want %d)", w, i, lw, w)
			}
		}
		top, bottom := stripSeq(lines[0]), stripSeq(lines[len(lines)-1])
		// Corner column = display width of everything before the glyph.
		topCol := lipgloss.Width(top[:strings.LastIndex(top, "┐")])
		botCol := lipgloss.Width(bottom[:strings.LastIndex(bottom, "┘")])
		if topCol != botCol {
			t.Errorf("w=%d: corners misaligned: ┐@%d ┘@%d\n%s\n%s",
				w, topCol, botCol, top, bottom)
		}
	}
}

func lipglossWidth(s string) int { return lipgloss.Width(s) }
