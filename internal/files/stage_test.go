package files

import (
	"os"
	"path/filepath"
	"testing"
)

func tmpTree(t *testing.T) (dir, file string) {
	t.Helper()
	root := t.TempDir()
	dir = filepath.Join(root, "sub")
	file = filepath.Join(root, "a.txt")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	return dir, file
}

func TestStageValidation(t *testing.T) {
	d, f := tmpTree(t)

	var s Stager
	// deleting something that exists is fine
	if err := s.Stage(Op{Kind: OpDelete, Path: f}); err != nil {
		t.Fatal(err)
	}
	// deleting a missing path must fail at stage time
	if err := s.Stage(Op{Kind: OpDelete, Path: d + "/nope"}); err == nil {
		t.Fatal("expected error staging delete of missing path")
	}
	// mkdir onto existing path fails
	if err := s.Stage(Op{Kind: OpMkdir, Path: d}); err == nil {
		t.Fatal("expected error staging duplicate mkdir")
	}
	// rename with separators refused
	if err := s.Stage(Op{Kind: OpRename, Path: f, Arg: "x/y"}); err == nil {
		t.Fatal("separator in rename target must be refused")
	}
	// copying a dir onto itself refused
	if err := s.Stage(Op{Kind: OpCopy, Path: d, Arg: d}); err == nil {
		t.Fatal("copy onto itself must be refused")
	}
}

func TestStagerUndoClearDropFirst(t *testing.T) {
	_, f := tmpTree(t)
	var s Stager
	s.Stage(Op{Kind: OpDelete, Path: f})
	s.Stage(Op{Kind: OpDelete, Path: f}) // duplicate allowed at stage level? no—same op twice fine here since validation only lstat's disk
	op, ok := s.Undo()
	if !ok || op.Path != f {
		t.Fatalf("undo returned %v %v", op, ok)
	}
	if s.Len() != 1 {
		t.Fatalf("len = %d after undo", s.Len())
	}
	s.DropFirst(1)
	if s.Len() != 0 {
		t.Fatalf("len = %d after drop", s.Len())
	}
}

func TestApplyOpRoundTrip(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "x.txt")
	os.WriteFile(src, []byte("data"), 0644)
	dstDir := filepath.Join(root, "out")
	os.Mkdir(dstDir, 0755)

	if err := ApplyOp(Op{Kind: OpCopy, Path: src, Arg: dstDir}); err != nil {
		t.Fatal(err)
	}
	copied := filepath.Join(dstDir, "x.txt")
	if _, err := os.Stat(copied); err != nil {
		t.Fatal("copy not applied")
	}
	// move the copy elsewhere under a fresh dir to avoid clobbering src
	other := filepath.Join(root, "other")
	os.Mkdir(other, 0755)
	if err := ApplyOp(Op{Kind: OpMove, Path: copied, Arg: other}); err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(other, "x.txt")
	if _, err := os.Stat(moved); err != nil {
		t.Fatal("move not applied")
	}
	if err := ApplyOp(Op{Kind: OpChmod, Path: moved, Mode: 0o600}); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(moved)
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("chmod not applied: %v", info.Mode().Perm())
	}
	if err := ApplyOp(Op{Kind: OpDelete, Path: moved}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(moved); !os.IsNotExist(err) {
		t.Fatal("delete not applied")
	}
	if err := ApplyOp(Op{Kind: OpMkdir, Path: filepath.Join(root, "made")}); err != nil {
		t.Fatal(err)
	}
}
