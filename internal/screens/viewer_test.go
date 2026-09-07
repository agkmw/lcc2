package screens

import (
	"os"
	"path/filepath"
	"testing"

	"lcc2/internal/ui"
)

// $EDITOR wins, then $VISUAL, then the first installed terminal
// editor. Arguments in the env value (e.g. "code -w") must survive
// the split. There is deliberately no pager fallback: less cannot
// save, and both callers are edit gestures.
func TestResolveViewer(t *testing.T) {
	fakeBin := t.TempDir()
	writeExe := func(name string) {
		p := filepath.Join(fakeBin, name)
		os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755)
	}
	writeExe("nvim")
	writeExe("nano")
	t.Setenv("PATH", fakeBin)

	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")
	if v, err := resolveViewer(); err != nil || v[0] != filepath.Join(fakeBin, "nvim") {
		t.Fatalf("first installed editor = %v, err %v", v, err)
	}

	t.Setenv("VISUAL", "code -w")
	if v, err := resolveViewer(); err != nil || len(v) != 2 || v[0] != "code" || v[1] != "-w" {
		t.Fatalf("visual = %v, err %v", v, err)
	}

	t.Setenv("EDITOR", "nvim")
	if v, err := resolveViewer(); err != nil || v[0] != "nvim" {
		t.Fatalf("editor = %v, err %v", v, err)
	}
}

// With no env and no editor binary on PATH, the viewer refuses with a
// hint instead of silently opening a pager that cannot edit.
func TestResolveViewerNoneFound(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")
	if _, err := resolveViewer(); err == nil {
		t.Fatal("expected an error with no editor anywhere")
	}
}

// Closing the viewer must refetch the preview too: the file may have
// been edited on disk, and only the listing was refreshed before.
func TestViewerDoneRefetchesPreview(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "note.txt"), []byte("v1"), 0644)
	f := NewFiles()
	f = feed(f, ui.SizeMsg{Width: 100, Height: 30},
		dirListMsg{dir: dir, list: mustList(t, dir)}).(Files)
	if e, ok := f.selected(); !ok || e.IsDir {
		t.Fatal("fixture: expected a selected file entry")
	}

	sc, cmd := f.Update(viewerDoneMsg{})
	nf := sc.(Files)
	if cmd == nil {
		t.Fatal("no refresh command after viewer closed")
	}
	if !nf.fetching {
		t.Fatal("preview fetch not kicked after viewer closed")
	}
}
