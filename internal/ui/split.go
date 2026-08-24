package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Split was removed with the pane-divider redesign (ADR-0010):
// screens assemble main|preview via the shared scaffold in
// internal/screens/pane.go, which draws a divider column. ClipBlock
// remains the exact-width primitive.

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
