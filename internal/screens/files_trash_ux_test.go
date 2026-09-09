package screens

import (
	"os"
	"path/filepath"
	"testing"

	"lcc/internal/files"
	"lcc/internal/ui"
)

// trashScreen builds the explorer parked inside a sandboxed home trash:
// one trashed file with a .trashinfo record pointing at a not-yet
// existing origin (home/A/foobar).
func trashScreen(t *testing.T) (f Files, home, trashed, origin string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	filesDir := filepath.Join(home, ".local", "share", "Trash", "files")
	infoDir := filepath.Join(home, ".local", "share", "Trash", "info")
	if err := os.MkdirAll(filesDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(infoDir, 0o700); err != nil {
		t.Fatal(err)
	}
	trashed = filepath.Join(filesDir, "foobar")
	if err := os.WriteFile(trashed, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	origin = filepath.Join(home, "A", "foobar") // do NOT create A
	info := "[Trash Info]\nPath=" + origin +
		"\nDeletionDate=2026-09-09T10:00:00\n"
	if err := os.WriteFile(filepath.Join(infoDir, "foobar.trashinfo"),
		[]byte(info), 0o600); err != nil {
		t.Fatal(err)
	}

	f = NewFiles()
	f = feed(f, ui.SizeMsg{Width: 100, Height: 30},
		dirListMsg{dir: filesDir, list: mustList(t, filesDir)}).(Files)
	f.trashFrom = home // remembered by `t` in the real flow
	return f, home, trashed, origin
}

// Inside the trash R stages restores (never rename) and the hints
// advertise the way back; applying the op puts the file back at its
// recorded origin.
func TestTrashHintsAndRestoreStage(t *testing.T) {
	f, _, trashed, origin := trashScreen(t)

	var restore, back, rename bool
	for _, b := range f.Hints() {
		k, d := b.Help().Key, b.Help().Desc
		switch {
		case k == "R" && d == "restore":
			restore = true
		case k == "esc" && d == "back to explorer":
			back = true
		case k == "R" && d == "rename":
			rename = true
		}
	}
	if !restore || !back || rename {
		t.Fatalf("trash hints wrong: restore=%v back=%v rename-listed=%v",
			restore, back, rename)
	}

	sc, _ := f.Update(keyRunes("R"))
	f = sc.(Files)
	if f.stager.Len() != 1 {
		t.Fatalf("restore not staged: %d", f.stager.Len())
	}
	op := f.stager.Ops()[0]
	if op.Kind != files.OpRestore || op.Path != trashed || op.Arg != origin {
		t.Fatalf("staged op = %v %v -> %v", op.Kind, op.Path, op.Arg)
	}
	if err := files.ApplyOp(op); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := os.Stat(origin); err != nil {
		t.Fatal("file not restored to its origin")
	}
}

// esc inside the trash returns to the remembered explorer dir; once
// outside, esc must not navigate or touch the staged queue.
func TestTrashEscReturnsToExplorer(t *testing.T) {
	f, home, _, _ := trashScreen(t)

	sc, _ := f.Update(keyRunes("esc"))
	f = sc.(Files)
	if f.trashFrom != "" {
		t.Fatalf("trashFrom not cleared: %q", f.trashFrom)
	}
	// The returned command re-lists home; feed its arrival message.
	f = feed(f, dirListMsg{dir: home, list: mustList(t, home)}).(Files)
	if f.cwd != home {
		t.Fatalf("esc did not return to the explorer: cwd=%q", f.cwd)
	}

	f = feed(f, keyRunes("esc")).(Files) // already outside the trash
	if f.cwd != home {
		t.Fatalf("esc navigated outside the trash: cwd=%q", f.cwd)
	}
	if f.stager.Len() != 0 || f.trashFrom != "" {
		t.Fatalf("esc mutated state: ops=%d trashFrom=%q",
			f.stager.Len(), f.trashFrom)
	}
}
