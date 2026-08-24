package app

import (
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"lcc2/internal/screens"
)

var clockRe = regexp.MustCompile(`[0-9]{2}:[0-9]{2}`)

func normalizeClock(s string) string { return clockRe.ReplaceAllString(s, "HH:MM") }

func freshAt(w, h int, active byte) string {
	r := New(screens.NewOverview(), screens.NewProcesses(),
		screens.NewDisks(), screens.NewFiles(),
		screens.NewServices(), screens.NewUsersGroups())
	m, _ := r.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m, _ = m.Update(keyMsg(string(active)))
	return normalizeClock(m.View())
}

// The resize/tab-switch gauntlet: after every step the frame must be
// well-formed AND byte-identical to a freshly constructed model in the
// same logical state. Anything else is stale-cell ghosting — the
// "other pages rendering through" bug seen under tmux split/zoom.
func TestResizeTabSwitchNoGhosting(t *testing.T) {
	type step struct {
		w, h   int
		active string
	}
	steps := []step{
		{100, 30, "1"},
		{140, 40, "4"}, // grow + switch
		{80, 24, "2"},  // shrink (tmux zoom out)
		{200, 50, "5"},
		{70, 20, "1"}, // back to start size, different content
	}

	r := New(screens.NewOverview(), screens.NewProcesses(),
		screens.NewDisks(), screens.NewFiles(),
		screens.NewServices(), screens.NewUsersGroups())
	for i, st := range steps {
		if i > 0 {
			m, _ := r.Update(keyMsg(st.active))
			r = m.(Root)
		}
		m, _ := r.Update(tea.WindowSizeMsg{Width: st.w, Height: st.h})
		r = m.(Root)

		view := normalizeClock(r.View())
		want := freshAt(st.w, st.h, st.active[len(st.active)-1])

		if view != want {
			t.Errorf("step %d (%dx%d sec=%s): frame diverges from fresh render",
				i, st.w, st.h, st.active)
		}
		lines := strings.Split(view, "\n")
		if len(lines) != st.h {
			t.Errorf("step %d: %d lines, want %d", i, len(lines), st.h)
		}
	}
}
