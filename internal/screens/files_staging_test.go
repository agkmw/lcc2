package screens

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"lcc2/internal/files"
	"lcc2/internal/ui"
)

func seedDir(t *testing.T) (string, []files.Entry) {
	t.Helper()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("alpha"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("beta"), 0644)
	os.Mkdir(filepath.Join(dir, "cdir"), 0755)
	lst, err := files.List(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	return dir, lst
}

func filesScreen(t *testing.T) (Files, string) {
	t.Helper()
	dir, lst := seedDir(t)
	f := NewFiles()
	f = feed(f, ui.SizeMsg{Width: 100, Height: 30}, dirListMsg{dir: dir, list: lst}).(Files)
	return f, dir
}

// The oil flow: delete is staged instantly (no confirm), shows in the
// badge, and only lands on disk after `w`.
func TestStagedDeleteSavesOnW(t *testing.T) {
	f, dir := seedScreenForFlow(t)

	target := filepath.Join(dir, "a.txt")
	f.tbl.SetCursor(indexOf(f.entries, target))

	f, _ = keyFeed(f, "d")
	if f.stager.Len() != 1 {
		t.Fatalf("delete not staged: len=%d", f.stager.Len())
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatal("delete hit disk before save")
	}
	if f.Badge() == "" {
		t.Fatal("badge empty despite pending op")
	}

	sc, cmd := f.Update(keyRunes("w"))
	f = sc.(Files)
	if cmd == nil {
		t.Fatal("w produced no save command")
	}

	// run the whole save chain synchronously
	for i := 0; i < 10 && f.saving; i++ {
		m := cmd()
		var c tea.Cmd
		sc, c = f.Update(m)
		f = sc.(Files)
		cmd = c
		if cmd == nil {
			break
		}
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("file still on disk after save")
	}
	if f.stager.Len() != 0 || f.Badge() != "" {
		t.Fatalf("queue/badge not cleared: %d %q", f.stager.Len(), f.Badge())
	}
}

func seedScreenForFlow(t *testing.T) (Files, string) {
	return filesScreen(t)
}

func indexOf(entries []files.Entry, path string) int {
	for i, e := range entries {
		if e.Path == path {
			return i
		}
	}
	return -1
}

// Multi-select: mark two entries with space, one delete stages both.
func TestMarkedDeleteStagesAll(t *testing.T) {
	f, dir := filesScreen(t)

	a := indexOf(f.entries, filepath.Join(dir, "a.txt"))
	b := indexOf(f.entries, filepath.Join(dir, "b.txt"))
	f.tbl.SetCursor(a)
	f, _ = keyFeed(f, " ")
	f.tbl.SetCursor(b)
	f, _ = keyFeed(f, " ")

	f, _ = keyFeed(f, "d")
	if got := f.stager.Len(); got != 2 {
		t.Fatalf("staged %d ops, want 2", got)
	}
}

// u undoes the most recent staged op; U discards everything.
func TestUndoAndDiscard(t *testing.T) {
	f, _ := filesScreen(t)
	f, _ = keyFeed(f, "d") // cursor on first entry
	f, _ = keyFeed(f, "m")
	f, _ = keyFeed(f, "enter") // commit mkdir prompt with empty name → no-op

	// stage a mkdir properly through the prompt action
	sc, _ := f.Update(keyRunes("m"))
	f = sc.(Files)
	in := *f.prompt
	in.SetValue("newdir")
	f.prompt = &in
	sc, _ = f.Update(tea.KeyMsg{Type: tea.KeyEnter})
	f = sc.(Files)

	before := f.stager.Len()
	if before < 2 {
		t.Fatalf("expected >=2 staged ops, have %d", before)
	}
	sc, _ = f.Update(keyRunes("u"))
	f = sc.(Files)
	if f.stager.Len() != before-1 {
		t.Fatalf("undo did not pop: %d -> %d", before, f.stager.Len())
	}
	sc, _ = f.Update(keyRunes("U"))
	f = sc.(Files)
	if f.stager.Len() != 0 {
		t.Fatal("discard left ops queued")
	}
}

// Preview follows the cursor: a text file yields its content lines.
func TestPreviewFollowsSelection(t *testing.T) {
	f, dir := filesScreen(t)
	path := filepath.Join(dir, "a.txt")
	f.tbl.SetCursor(indexOf(f.entries, path))

	content, err := files.ReadPreview(path, 10, 1<<10)
	if err != nil {
		t.Fatal(err)
	}
	f = feed(f, filePreviewMsg{path: path, p: content}).(Files)
	if !strings.Contains(f.previewContent(), "alpha") {
		t.Fatalf("preview missing content: %q", f.previewContent())
	}
	if !strings.Contains(f.View(), "alpha") {
		t.Fatal("preview content not rendered in view")
	}
}

func keyFeed(f Files, keys ...string) (Files, tea.Cmd) {
	var sc ui.Screen = f
	var last tea.Cmd
	for _, k := range keys {
		sc, last = sc.Update(keyRunes(k))
	}
	return sc.(Files), last
}
