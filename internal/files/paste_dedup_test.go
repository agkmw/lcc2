package files

import (
	"os"
	"path/filepath"
	"testing"
)

// Pasting into the same directory must resolve to a fresh name, not
// error: plain base when free, name.2 / name.3 on collision.
func TestUniqueDstDeduplicates(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644)
	srcDir := filepath.Join(dir, "other")
	os.Mkdir(srcDir, 0o755)
	src := filepath.Join(srcDir, "a.txt")
	os.WriteFile(src, []byte("x"), 0o644)

	s := NewStager()

	got := s.UniqueDst(src, dir)
	if got != filepath.Join(dir, "a.2.txt") {
		t.Fatalf("first dedupe = %q, want a.2.txt", got)
	}

	// A queued copy claims its destination even before it exists.
	if err := s.Stage(Op{Kind: OpCopy, Path: src, Arg: got}); err != nil {
		t.Fatal(err)
	}
	if got := s.UniqueDst(src, dir); got != filepath.Join(dir, "a.3.txt") {
		t.Fatalf("second dedupe = %q, want a.3.txt (claim ignored)", got)
	}
}

// Free names are used as-is: pasting into an empty directory keeps
// the original name.
func TestUniqueDstKeepsFreeName(t *testing.T) {
	dir := t.TempDir()
	s := NewStager()
	src := filepath.Join(t.TempDir(), "report.pdf")
	if got := s.UniqueDst(src, dir); got != filepath.Join(dir, "report.pdf") {
		t.Fatalf("got %q", got)
	}
}

// Double-pasting the same source stages two copies that both apply.
func TestDoublePasteAppliesBoth(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "doc")
	os.WriteFile(src, []byte("v"), 0o644)

	s := NewStager()
	for i := 0; i < 2; i++ {
		dst := s.UniqueDst(src, dir)
		if err := s.Stage(Op{Kind: OpCopy, Path: src, Arg: dst}); err != nil {
			t.Fatalf("stage %d: %v", i, err)
		}
	}
	for i, op := range s.Ops() {
		if err := ApplyOp(op); err != nil {
			t.Fatalf("apply %d (%s): %v", i, op.Label(), err)
		}
	}
	for _, name := range []string{"doc.2", "doc.3"} { // no ext: stem.2
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("%s not created: %v", name, err)
		}
	}
}

// The create-file op stages with clobber protection and applies to an
// empty regular file.
func TestCreateFileOp(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "notes.txt")

	s := NewStager()
	if err := s.Stage(Op{Kind: OpCreate, Path: p}); err != nil {
		t.Fatal(err)
	}
	if err := s.Stage(Op{Kind: OpCreate, Path: filepath.Join(dir, "x/y")}); err == nil {
		t.Fatal("bad create name must fail")
	}

	if err := ApplyOp(s.Ops()[0]); err != nil {
		t.Fatal(err)
	}
	if err := s.Stage(Op{Kind: OpCreate, Path: p}); err == nil {
		t.Fatal("staging a create over an existing path must fail")
	}
	info, err := os.Stat(p)
	if err != nil || info.IsDir() {
		t.Fatalf("created entry wrong: %v %v", info, err)
	}
	if data, _ := os.ReadFile(p); len(data) != 0 {
		t.Fatalf("created file not empty: %q", data)
	}
}
