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

// Help may only advertise keys the current screen answers to: no
// phantom globals on the list-less Overview, no duplicated or
// contradictory rows on table screens.
func TestHelpContentMatchesScreen(t *testing.T) {
	r, m := newTestRoot(t)
	r = m.(Root)
	open := func(i int) Root {
		r.active = i
		m, _ := r.Update(keyMsg("?"))
		return m.(Root)
	}

	pan := stripANSIHelp(open(0).helpPanel()) // overview
	for _, phantom := range []string{"move selection", "filter list",
		"select / open", "back / cancel"} {
		if strings.Contains(pan, phantom) {
			t.Errorf("overview help advertises %q, which overview does not bind", phantom)
		}
	}
	if !strings.Contains(pan, "graph") {
		t.Errorf("overview help missing its own g hint: %q", pan)
	}

	pan = stripANSIHelp(open(1).helpPanel()) // processes
	if !strings.Contains(pan, "move selection") {
		t.Error("process help missing the j/k list row")
	}
	if n := strings.Count(pan, "[/]"); n != 1 {
		t.Errorf("process help shows %d '/' rows, want 1 (screen hint only)", n)
	}

	pan = stripANSIHelp(open(2).helpPanel()) // disks
	if !strings.Contains(pan, "analyze") || strings.Contains(pan, "select / open") {
		t.Errorf("disks help enter verb wrong: %q", pan)
	}
}

// q closes the overlay; a second q then quits from the base screen.
func TestHelpQCloses(t *testing.T) {
	r := openHelp(t)
	m, _ := r.Update(keyMsg("q"))
	r = m.(Root)
	if r.helpOpen {
		t.Fatal("q did not close help")
	}
}

// tab and the digit keys switch screens while the overlay stays open,
// and the panel content follows the new active screen.
func TestHelpFollowsScreenSwitch(t *testing.T) {
	r := openHelp(t) // files
	if !strings.Contains(stripANSIHelp(r.helpPanel()), "Files") {
		t.Fatal("panel should start on Files")
	}
	if r.helpScroll != 0 {
		t.Fatal("test needs help opened at scroll 0")
	}
	m, _ := r.Update(keyMsg("ctrl+d"))
	r = m.(Root)
	if r.helpScroll == 0 {
		t.Fatal("scroll did not move")
	}

	m, _ = r.Update(keyMsg("tab"))
	r = m.(Root)
	if !r.helpOpen {
		t.Fatal("tab closed the overlay")
	}
	if !strings.Contains(stripANSIHelp(r.helpPanel()), "Services") {
		t.Fatalf("panel did not follow the switch: %q", stripANSIHelp(r.helpPanel()))
	}
	if r.helpScroll != 0 {
		t.Fatalf("scroll = %d after switch, want reset to 0", r.helpScroll)
	}
	if strings.Contains(strings.Join(r.helpContent(), "\n"), "open dir") {
		t.Error("content still lists files-only keys after switching")
	}

	m, _ = r.Update(keyMsg("1"))
	r = m.(Root)
	if !strings.Contains(stripANSIHelp(r.helpPanel()), "Overview") {
		t.Fatal("digit switch did not follow")
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
