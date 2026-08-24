package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/lipgloss"

	"lcc2/internal/screens"
)

// Narrow terminals must degrade the tab strip by priority (badges ->
// numbers -> label width), never clip mid-tab and never lose the
// active section name (backlog L9). Below MinW the too-small notice
// renders instead, so only >=64 widths exercise the strip.
func TestTabStripDegradesWithinWidth(t *testing.T) {
	for _, w := range []int{64, 72, 90, 120} {
		r := New(screens.NewOverview(), screens.NewProcesses(),
			screens.NewDisks(), screens.NewFiles(),
			screens.NewServices(), screens.NewUsersGroups())
		m, _ := r.Update(tea.WindowSizeMsg{Width: w, Height: 24})
		root := m.(Root)
		line := strings.Split(root.View(), "\n")[0]
		if lw := lipgloss.Width(line); lw > w {
			t.Errorf("w=%d: strip is %d cells wide", w, lw)
		}
		if !strings.Contains(stripANSI(line), "Overview") {
			t.Errorf("w=%d: active label lost in %q", w, line)
		}
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	esc := false
	for _, r := range s {
		switch {
		case r == '\x1b':
			esc = true
		case esc:
			if r == 'm' {
				esc = false
			}
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
