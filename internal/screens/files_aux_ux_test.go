package screens

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"lcc/internal/files"
	"lcc/internal/ui"
)

// enter on a find hit must land the cursor ON the entry in the target
// listing, not at the top of it - hunting the file again would undo
// the search.
func TestAuxRevealLandsOnEntry(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "deep", "sub")
	os.MkdirAll(sub, 0o755)
	hit := filepath.Join(sub, "main.txt")
	os.WriteFile(hit, []byte("content"), 0o644)

	f := NewFiles()
	f = feed(f, ui.SizeMsg{Width: 100, Height: 30},
		dirListMsg{dir: root, list: mustList(t, root)}).(Files)
	f.mode = "find"
	f.findRes = []files.Entry{{Name: "main.txt", Path: hit}}
	f.syncAuxTables()

	sc, _ := f.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // aux enter
	f = sc.(Files)
	if f.mode != "list" || f.reveal != hit {
		t.Fatalf("reveal not staged: mode=%q reveal=%q", f.mode, f.reveal)
	}

	f = feed(f, dirListMsg{dir: sub, list: mustList(t, sub)}).(Files)
	if f.reveal != "" {
		t.Fatal("reveal not consumed by the listing")
	}
	if k, ok := f.tbl.SelectedKey(); !ok || k != hit {
		t.Fatalf("cursor on %q (ok=%v), want the hit %q", k, ok, hit)
	}
}

// Grep hits reveal their file the same way.
func TestAuxGrepRevealLandsOnHit(t *testing.T) {
	root := t.TempDir()
	hit := filepath.Join(root, "code.txt")
	os.WriteFile(hit, []byte("x"), 0o644)

	f := NewFiles()
	f = feed(f, ui.SizeMsg{Width: 100, Height: 30},
		dirListMsg{dir: root, list: mustList(t, root)}).(Files)
	f.mode = "grep"
	f.grepRes = []files.Match{{Path: hit, Line: 3, Col: 1, Text: "needle"}}
	f.syncAuxTables()

	sc, _ := f.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	f = sc.(Files)
	f = feed(f, dirListMsg{dir: root, list: mustList(t, root)}).(Files)
	if k, ok := f.tbl.SelectedKey(); !ok || k != hit {
		t.Fatalf("cursor on %q (ok=%v), want the hit %q", k, ok, hit)
	}
}

// At the result cap the tally must say so - "1000 results" reading as
// "everything" hides a truncated search.
func TestAuxStatusMarksTruncation(t *testing.T) {
	f := NewFiles()
	f.mode = "find"
	f.findRes = make([]files.Entry, findLimit)
	if got := f.auxStatus(); !strings.Contains(got, "1000+") {
		t.Fatalf("find status = %q, want 1000+ marker", got)
	}
	f.findRes = f.findRes[:5]
	if got := f.auxStatus(); strings.Contains(got, "+") {
		t.Fatalf("short find status = %q, want no marker", got)
	}

	f.mode = "grep"
	f.grepRes = make([]files.Match, grepLimit)
	if got := f.auxStatus(); !strings.Contains(got, "500+") {
		t.Fatalf("grep status = %q, want 500+ marker", got)
	}
}
