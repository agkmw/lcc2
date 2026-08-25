package app

import (
	"github.com/charmbracelet/colorprofile"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"lcc2/internal/screens"
	"lcc2/internal/ui"
)

func forceTrueColorApp(t *testing.T) {
	t.Helper()
	restore := ui.SetProfileOverride(colorprofile.TrueColor)
	t.Cleanup(restore)
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
		_ = m.(Root)
		line := strings.Split(viewString(m), "\n")[0]
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

// viewString exposes the frame string to tests (View() wraps it in a
// tea.View).
func viewString(m tea.Model) string {
	return m.(interface{ View() tea.View }).View().Content
}
