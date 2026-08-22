package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"lcc2/internal/screens"
)

// The footer must sit on the very last terminal line and the header
// on the first, at every window size — short screens must clip the
// body, tall ones must pad it (regression: the footer floated up on
// sparse screens and slipped off the bottom on crowded ones).
func TestFooterPinnedToLastLine(t *testing.T) {
	for _, tc := range []struct{ w, h int }{
		{100, 10}, {100, 16}, {100, 24}, {120, 40}, {80, 14}, {200, 50},
	} {
		r := New(screens.NewOverview(), screens.NewProcesses())
		m, _ := r.Update(tea.WindowSizeMsg{Width: tc.w, Height: tc.h})
		root := m.(Root)
		view := root.View()
		if view == "" {
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
		if !strings.Contains(lines[tc.h-1], "q quit") {
			t.Errorf("w=%d h=%d: footer not on last line: %q",
				tc.w, tc.h, lines[tc.h-1])
		}
	}
}
