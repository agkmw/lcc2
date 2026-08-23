package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Split places two blocks side by side, hard-clipping each column to
// its exact cell budget so the joined result never exceeds total
// columns — panes cannot bleed through surrounding chrome. When total
// is too narrow for a sensible side-by-side (left would get fewer than
// minLeft cells), the blocks stack vertically instead.
func Split(left, right string, lw int, total int) string {
	const minLeft = 24
	if lw > total-minLeft-1 {
		return left + "\n" + right
	}
	rw := total - lw - 1
	ll := strings.Split(left, "\n")
	rl := strings.Split(right, "\n")
	n := len(ll)
	if len(rl) > n {
		n = len(rl)
	}
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = clipLine(at(ll, i), lw) + " " + clipLine(at(rl, i), rw)
	}
	return strings.Join(out, "\n")
}

func at(lines []string, i int) string {
	if i < len(lines) {
		return lines[i]
	}
	return ""
}

// clipLine force-fits a line into w display cells, truncating ANSI
// styled content safely and padding short lines with plain spaces.
func clipLine(line string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(line) > w {
		line = ansi.Truncate(line, w, "")
	}
	if gap := w - lipgloss.Width(line); gap > 0 {
		line += strings.Repeat(" ", gap)
	}
	return line
}

// ClipBlock fits every line of a block into w cells.
func ClipBlock(s string, w int) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = clipLine(lines[i], w)
	}
	return strings.Join(lines, "\n")
}
