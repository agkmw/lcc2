package app

import (
	"os"
	"path/filepath"
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

// The Files screen's prefs must survive saves made while another
// section is active (clock tick, quit from anywhere).
func TestSnapshotCapturesFilesFromAnySection(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	os.Setenv("HOME", os.TempDir())

	cwd := filepath.Join(t.TempDir(), "probe")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	filesScr := screens.NewFiles()
	filesScr.Hydrate(cwd, true, "mtime", true)

	r := NewStartingAt(0, screens.NewOverview(), screens.NewProcesses(),
		screens.NewDisks(), filesScr,
		screens.NewServices(), screens.NewUsersGroups())
	m, _ := r.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	root := m.(Root) // Overview is active; snapshot anyway

	session.Save(root.snapshot())
	st := session.Load()
	if st.Screen != 0 || st.Cwd != cwd || !st.Hidden ||
		st.SortKey != "mtime" || !st.SortDesc {
		t.Fatalf("snapshot lost Files prefs: %+v", st)
	}
}
