package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"lcc2/internal/screens"
)

// Resize gauntlet: shrink -> grow -> shrink cycles across sections and
// modal states. After every step the frame must be exactly h lines,
// never wider than w, and glyph-clean. This is the automated
// replacement for eyeballing live terminals — it pins the resize
// corruption class of bugs (bubbles rune-truncation slicing escapes).
func TestResizeCyclesKeepFrame(t *testing.T) {
	cycle := [][2]int{
		{120, 36}, {64, 16}, {90, 28}, {200, 50}, {66, 17}, {120, 36}, {80, 24},
	}
	scenarios := []struct {
		name  string
		sec   byte
		modal []tea.Msg
	}{
		{"overview", '1', nil},
		{"files", '4', nil},
		{"files-prompt", '4', []tea.Msg{keyRune("m")}},
		{"svc-confirm", '5', []tea.Msg{keyRune("s")}},
		{"help-open", '2', []tea.Msg{keyRune("?")}},
	}
	for _, sc := range scenarios {
		r := New(screens.NewOverview(), screens.NewProcesses(),
			screens.NewDisks(), screens.NewFiles(),
			screens.NewServices(), screens.NewUsersGroups())
		m, _ := r.Update(tea.WindowSizeMsg{Width: cycle[0][0], Height: cycle[0][1]})
		m, _ = m.Update(keyRune(string(sc.sec)))
		for _, k := range sc.modal {
			m, _ = m.Update(k)
		}
		for _, sz := range cycle {
			w, h := sz[0], sz[1]
			m, _ = m.Update(tea.WindowSizeMsg{Width: w, Height: h})
			view := viewString(m)
			lines := strings.Split(view, "\n")
			if len(lines) != h {
				t.Errorf("%s @%dx%d: %d lines (want %d)", sc.name, w, h, len(lines), h)
				continue
			}
			for i, l := range lines {
				if lw := lipgloss.Width(l); lw > w {
					t.Errorf("%s @%dx%d: line %d is %d cells", sc.name, w, h, i, lw)
					break
				}
			}
			if off := offending(view); off != "" {
				t.Errorf("%s @%dx%d: ambiguous glyph %s", sc.name, w, h, off)
			}
		}
	}
}

// Below-floor sizes must show the notice frame, not section content.
func TestResizeBelowFloorShowsNotice(t *testing.T) {
	r := New(screens.NewOverview(), screens.NewProcesses(),
		screens.NewDisks(), screens.NewFiles(),
		screens.NewServices(), screens.NewUsersGroups())
	m, _ := r.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	for _, sz := range [][2]int{{50, 20}, {63, 15}, {100, 12}} {
		w, h := sz[0], sz[1]
		m, _ = m.Update(tea.WindowSizeMsg{Width: w, Height: h})
		lines := strings.Split(viewString(m), "\n")
		if len(lines) != h {
			t.Fatalf("%dx%d: %d lines, want %d", w, h, len(lines), h)
		}
		if !strings.Contains(stripANSI(viewString(m)), "terminal too small") {
			t.Errorf("%dx%d: notice not rendered", w, h)
		}
	}
	// Recovery: growing back must restore the real UI instantly.
	m, _ = m.Update(tea.WindowSizeMsg{Width: 110, Height: 30})
	if first := strings.Split(viewString(m), "\n")[0]; !strings.Contains(first, "lcc2") {
		t.Errorf("no recovery after grow: %q", stripANSI(first))
	}
}
