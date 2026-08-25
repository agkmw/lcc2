package app

import (
	"os"
	"testing"

	tea "charm.land/bubbletea/v2"

	"lcc2/internal/screens"
	"lcc2/internal/session"
)

// Switching screens persists the index; the Files screen contributes
// its cwd/sort preferences to the same snapshot.
func TestSessionSnapshot(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	os.Setenv("HOME", os.TempDir()) // keep trash/home checks away from real home

	r := NewStartingAt(0, screens.NewOverview(), screens.NewProcesses(),
		screens.NewDisks(), screens.NewFiles(),
		screens.NewServices(), screens.NewUsersGroups())
	m, _ := r.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	root := m.(Root)

	m, _ = root.Update(keyMsg("4")) // switch to Files
	root = m.(Root)
	session.Save(root.snapshot()) // what switchTo does internally

	st := session.Load()
	if st.Screen != 3 {
		t.Errorf("screen = %d, want 3", st.Screen)
	}
	if st.Cwd == "" {
		t.Error("files cwd missing from snapshot")
	}
}
