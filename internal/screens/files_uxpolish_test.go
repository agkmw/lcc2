package screens

import (
	"path/filepath"
	"strings"
	"testing"
)

// The active sort is shown in the pane header, not only in the status
// bar hint.
func TestSortInPaneHeader(t *testing.T) {
	f, _ := filesScreen(t)
	if !strings.Contains(stripANSI(f.View()), "name - 3 items") {
		t.Fatal("pane header missing default sort")
	}
	s, _ := f.Update(keyRunes("S")) // reverse
	f = s.(Files)
	if !strings.Contains(stripANSI(f.View()), "name desc") {
		t.Fatal("pane header missing desc direction")
	}
}

// Clearing marks with esc reports the loss; esc without marks stays
// silent.
func TestMarksClearedToast(t *testing.T) {
	f, dir := filesScreen(t)
	path := filepath.Join(dir, "a.txt")

	f.tbl.SetCursor(indexOf(f.entries, path))
	f, cmd := keyFeed(f, " ") // mark one
	if cmd != nil {
		t.Fatal("marking produced a cmd")
	}
	sc, cmd := f.Update(keyRunes("esc"))
	f = sc.(Files)
	if len(f.marked) != 0 {
		t.Fatal("esc did not clear marks")
	}
	msg := toastOf(t, cmd)
	if !strings.Contains(msg.Text, "marks cleared (1)") {
		t.Fatalf("toast = %+v", msg)
	}

	// esc again: no marks, no toast
	sc, cmd = f.Update(keyRunes("esc"))
	f = sc.(Files)
	if cmd != nil {
		t.Fatal("esc without marks produced a toast")
	}
}
