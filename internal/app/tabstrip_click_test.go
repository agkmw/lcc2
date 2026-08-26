package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"lcc2/internal/screens"
)

// Clicking a tab chip must switch sections: the hit spans recorded
// during View have to survive until the next Update (value receivers
// used to discard them, making stripHit always miss).
func TestTabStripClickSwitchesSection(t *testing.T) {
	forceTrueColorApp(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // switchTo saves the session

	r := New(screens.NewOverview(), screens.NewProcesses(),
		screens.NewDisks(), screens.NewFiles(),
		screens.NewServices(), screens.NewUsersGroups())
	m, _ := r.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	root := m.(Root)
	if root.View().Content == "" {
		t.Fatal("empty render")
	}

	x := -1
	for i := 0; i < 120; i++ {
		if idx, ok := root.stripHit(i); ok && idx == 2 {
			x = i
			break
		}
	}
	if x < 0 {
		t.Fatal("no hit span for section 2 after render")
	}

	m, cmd := root.Update(tea.MouseClickMsg(tea.Mouse{
		X: x, Y: 0, Button: tea.MouseLeft}))
	root = m.(Root)
	if root.active != 2 {
		t.Fatalf("click on tab 2: active=%d", root.active)
	}
	if cmd == nil {
		t.Fatal("switch produced no cmd (screen never init'd)")
	}
}
