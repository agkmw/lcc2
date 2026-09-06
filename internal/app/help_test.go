package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func openHelp(t *testing.T) Root {
	t.Helper()
	r, m := newTestRoot(t)
	r = m.(Root)
	r.active = 3 // files: the longest hint list, so scrolling matters
	m, _ = r.Update(keyMsg("?"))
	return m.(Root)
}

// The help panel is the same height on every screen: fixed viewport,
// not content-sized.
func TestHelpPanelSameHeightAcrossScreens(t *testing.T) {
	r := openHelp(t)
	want := -1
	for i := range r.order {
		r.active = i
		n := strings.Count(r.helpPanel(), "\n") + 1
		if want < 0 {
			want = n
			continue
		}
		if n != want {
			t.Fatalf("help panel on %s is %d lines, want %d", r.order[i], n, want)
		}
	}
	if want < 10 {
		t.Fatalf("help panel suspiciously short: %d lines", want)
	}
}

// ctrl+d / ctrl+u scroll half a viewport and clamp to the list.
func TestHelpCtrlDUClamp(t *testing.T) {
	r := openHelp(t)
	max := len(r.helpContent()) - r.helpViewport()
	if max <= 0 {
		t.Fatal("test needs content taller than the viewport")
	}

	for i := 0; i < 20; i++ {
		m, _ := r.Update(keyMsg("ctrl+d"))
		r = m.(Root)
	}
	if r.helpScroll != max {
		t.Fatalf("scroll = %d, want clamped %d", r.helpScroll, max)
	}
	for i := 0; i < 40; i++ {
		m, _ := r.Update(keyMsg("ctrl+u"))
		r = m.(Root)
	}
	if r.helpScroll != 0 {
		t.Fatalf("scroll = %d, want 0 after ctrl+u", r.helpScroll)
	}
}

// Wheel events scroll the help list, never the screen behind it.
func TestHelpWheelScrollsHelpOnly(t *testing.T) {
	r := openHelp(t)
	before := r.helpScroll

	m, _ := r.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	r = m.(Root)
	if r.helpScroll != before+3 {
		t.Fatalf("wheel down: scroll = %d, want %d", r.helpScroll, before+3)
	}
	m, _ = r.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	r = m.(Root)
	if r.helpScroll != before {
		t.Fatalf("wheel up: scroll = %d, want %d", r.helpScroll, before)
	}
}

// Reopening help resets the scroll to the top.
func TestHelpScrollResetsOnReopen(t *testing.T) {
	r := openHelp(t)
	m, _ := r.Update(keyMsg("ctrl+d"))
	r = m.(Root)
	if r.helpScroll == 0 {
		t.Fatal("scroll did not move")
	}
	m, _ = r.Update(keyMsg("esc"))
	r = m.(Root)
	m, _ = r.Update(keyMsg("?"))
	r = m.(Root)
	if r.helpScroll != 0 {
		t.Fatalf("scroll = %d after reopen, want 0", r.helpScroll)
	}
}

// The panel shows the visible range and the scroll hint.
func TestHelpPanelShowsPosition(t *testing.T) {
	r := openHelp(t)
	panel := stripANSIHelp(r.helpPanel())
	if !strings.Contains(panel, "1-") || !strings.Contains(panel, " of ") {
		t.Fatalf("position line missing: %q", panel)
	}
	if !strings.Contains(panel, "ctrl+d/u scroll") {
		t.Fatal("scroll hint missing")
	}
}

func stripANSIHelp(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
