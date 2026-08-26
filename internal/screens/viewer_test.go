package screens

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lcc2/internal/ui"
)

// $EDITOR wins, then $VISUAL, then $PAGER, then less -R. Arguments in
// the env value (e.g. "code -w") must survive the split.
func TestResolveViewer(t *testing.T) {
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")
	t.Setenv("PAGER", "")
	if v := resolveViewer(); strings.Join(v, " ") != "less -R" {
		t.Fatalf("default = %v", v)
	}

	t.Setenv("PAGER", "moar")
	if v := resolveViewer(); strings.Join(v, " ") != "moar" {
		t.Fatalf("pager = %v", v)
	}

	t.Setenv("VISUAL", "code -w")
	if v := resolveViewer(); len(v) != 2 || v[0] != "code" || v[1] != "-w" {
		t.Fatalf("visual = %v", v)
	}

	t.Setenv("EDITOR", "nvim")
	if v := resolveViewer(); v[0] != "nvim" {
		t.Fatalf("editor = %v", v)
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
