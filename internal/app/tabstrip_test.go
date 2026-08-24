package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"lcc2/internal/screens"
)

func forceTrueColorApp(t *testing.T) {
	t.Helper()
	old := lipgloss.DefaultRenderer().ColorProfile()
	lipgloss.DefaultRenderer().SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.DefaultRenderer().SetColorProfile(old) })
}

// Narrow terminals must degrade the tab strip by priority (badges ->
// numbers -> label width), never clip mid-tab and never lose the
// active section name (backlog L9). Below MinW the too-small notice
// renders instead, so only >=64 widths exercise the strip.
func TestTabStripDegradesWithinWidth(t *testing.T) {
	forceTrueColorApp(t)
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
		// The active section renders as an inverted chip: at least one
		// background-setting sequence on the strip line (a second can
		// legitimately appear when the painter re-synthesizes the chip
		// after the inner number span's reset).
		if n := strings.Count(line, ";48;2;"); n < 1 {
			t.Errorf("w=%d: no chip background on the strip", w)
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
