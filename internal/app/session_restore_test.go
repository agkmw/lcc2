package app

import (
	"testing"

	"lcc2/internal/session"
)

// Switching 4 -> 2 -> 6 then quitting must persist screen 6 (the one
// the user was on), not the last screen they left. Regression: the
// save ran before the active flip, recording the outgoing index, and
// the q path never saved at all.
func TestQuitPersistsCurrentSection(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir()) // UserConfigDir falls back to HOME

	r, m := newTestRoot(t)
	r = m.(Root)
	for _, i := range []int{3, 1, 5} {
		m, _ = r.Update(keyMsg(string(rune('1' + i))))
		r = m.(Root)
	}
	if _, cmd := r.Update(keyMsg("q")); cmd == nil {
		t.Fatal("q did not quit")
	}
	if got := session.Load().Screen; got != 5 {
		t.Fatalf("persisted screen = %d, want 5", got)
	}
}

// Every switchTo lands with the destination persisted, even if the
// process dies before the next clock tick.
func TestSwitchPersistsDestination(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	r, m := newTestRoot(t)
	r = m.(Root)
	m, _ = r.Update(keyMsg("4")) // files (index 3)
	r = m.(Root)
	m, _ = r.Update(keyMsg("5")) // services (index 4)
	r = m.(Root)
	if got := session.Load().Screen; got != 4 {
		t.Fatalf("persisted screen = %d, want 4", got)
	}
}
