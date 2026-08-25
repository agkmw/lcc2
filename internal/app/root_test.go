package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"lcc2/internal/screens"
	"lcc2/internal/ui"
)

// The footer must sit on the very last terminal line and the header
// on the first, at every window size — short screens must clip the
// body, tall ones must pad it (regression: the footer floated up on
// sparse screens and slipped off the bottom on crowded ones).
func TestFooterPinnedToLastLine(t *testing.T) {
	for _, tc := range []struct{ w, h int }{
		{100, 16}, {100, 24}, {120, 40}, {80, 17}, {200, 50},
	} {
		r := New(screens.NewOverview(), screens.NewProcesses())
		m, _ := r.Update(tea.WindowSizeMsg{Width: tc.w, Height: tc.h})
		root := m.(Root)
		view := root.View().Content
		if strings.TrimSpace(view) == "" {
			t.Fatalf("w=%d h=%d: empty view", tc.w, tc.h)
		}
		lines := strings.Split(view, "\n")
		if got := len(lines); got != tc.h {
			t.Errorf("w=%d h=%d: rendered %d lines, want exactly %d",
				tc.w, tc.h, got, tc.h)
			continue
		}
		if !strings.Contains(lines[0], "lcc2") {
			t.Errorf("w=%d h=%d: header not on line 0: %q", tc.w, tc.h, lines[0])
		}
		if !strings.Contains(lines[tc.h-1], "quit") {
			t.Errorf("w=%d h=%d: footer not on last line: %q",
				tc.w, tc.h, lines[tc.h-1])
		}
	}
}

// Below the minimum geometry the app shows a friendly notice sized to
// the full frame instead of broken layouts.
func TestTooSmallNotice(t *testing.T) {
	for _, sz := range [][2]int{{40, 12}, {63, 15}, {20, 8}, {100, 9}} {
		w, h := sz[0], sz[1]
		r := New(screens.NewOverview(), screens.NewProcesses())
		m, _ := r.Update(tea.WindowSizeMsg{Width: w, Height: h})
		root := m.(Root)
		lines := strings.Split(root.View().Content, "\n")
		if len(lines) != h {
			t.Fatalf("%dx%d: %d lines, want %d", w, h, len(lines), h)
		}
		body := stripANSI(strings.Join(lines, "\n"))
		if !strings.Contains(body, "terminal too small") ||
			!strings.Contains(body, "have ") {
			t.Errorf("%dx%d: notice missing:\n%s", w, h, body)
		}
	}
}

// Notifications render as floating windows without shifting or washing
// out the rest of the frame.
func TestRootViewWithNotificationsKeepsFrame(t *testing.T) {
	r := New(screens.NewOverview(), screens.NewProcesses())
	m, _ := r.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	root := m.(Root)
	clean := root.View().Content

	m, _ = root.Update(ui.OkToast("saved now")())
	root = m.(Root)
	withNote := root.View().Content

	cleanLines := strings.Split(clean, "\n")
	noteLines := strings.Split(withNote, "\n")
	if len(noteLines) != len(cleanLines) {
		t.Fatalf("line count changed with note: %d vs %d", len(noteLines), len(cleanLines))
	}
	if noteLines[0] != cleanLines[0] {
		t.Fatal("header shifted by notification")
	}
	if !strings.Contains(noteLines[len(noteLines)-1], "quit") {
		t.Fatal("footer lost")
	}
	found := false
	for _, l := range noteLines[1:] {
		if strings.Contains(l, "saved now") {
			found = true
		}
	}
	if !found {
		t.Fatal("notification text not rendered")
	}

	// expiry removes only its own notification
	m, _ = root.Update(noteExpiryMsg{id: 999})
	root = m.(Root)
	if !strings.Contains(root.View().Content, "saved now") {
		t.Fatal("unknown expiry must not dismiss")
	}
	id := root.notes.Items()[0].ID
	m, _ = root.Update(noteExpiryMsg{id: id})
	root = m.(Root)
	if root.View().Content != clean {
		t.Fatal("view did not restore after expiry")
	}
}
