package files

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// trashFixture points HOME at a sandbox and materialises a home trash
// with one entry, returning its files dir.
func trashFixture(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	d := filepath.Join(home, ".local", "share", "Trash", "files")
	if err := os.MkdirAll(d, 0o700); err != nil {
		t.Fatal(err)
	}
	return d
}

// InTrash must match only real descendants: a sibling with a shared
// prefix ("Trash/files-x") is not inside the trash.
func TestInTrashBoundaries(t *testing.T) {
	d := trashFixture(t)

	inside := filepath.Join(d, "note.txt")
	if err := os.WriteFile(inside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !InTrash(inside) {
		t.Error("file inside trash not recognised")
	}
	if !InTrash(filepath.Join(d, "sub", "deep.txt")) {
		t.Error("nested file inside trash not recognised")
	}
	sibling := filepath.Clean(d) + "-x" // e.g. Trash/files-x
	if InTrash(filepath.Join(sibling, "evil.txt")) {
		t.Error("sibling prefix dir matched as inside trash")
	}
	if InTrash(filepath.Join(filepath.Dir(d), "info", "a.trashinfo")) {
		t.Error("trash info dir matched as inside trash")
	}
	if InTrash(t.TempDir()) {
		t.Error("unrelated dir matched as inside trash")
	}
}

// InTrash is false when no trash exists yet, and TrashDir must not
// create one as a side effect of looking.
func TestTrashDirLookupOnly(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if _, ok := TrashDir(); ok {
		t.Fatal("TrashDir reported a trash that does not exist")
	}
	if InTrash("/tmp/x") {
		t.Fatal("InTrash true without a trash dir")
	}
	base := filepath.Join(t.TempDir(), ".local", "share", "Trash")
	if _, err := os.Stat(base); !os.IsNotExist(err) {
		t.Fatal("lookup created the trash directory")
	}
}

// Applying a delete op to a path already inside the trash purges it
// permanently instead of re-trashing it into itself.
func TestApplyOpDeleteInsideTrashIsPermanent(t *testing.T) {
	d := trashFixture(t)
	victim := filepath.Join(d, "gone.txt")
	if err := os.WriteFile(victim, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ApplyOp(Op{Kind: OpDelete, Path: victim}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := os.Lstat(victim); !os.IsNotExist(err) {
		t.Fatal("trash-local file survived the delete op")
	}
	// And it must not have come back as a new trash entry.
	entries, _ := os.ReadDir(d)
	if len(entries) != 0 {
		t.Fatalf("trash dir not empty after purge: %v", entries)
	}
}

// findTool accepts the Debian spelling: with only fdfind on PATH, the
// alias resolves instead of failing.
func TestFindToolAlias(t *testing.T) {
	bin := t.TempDir()
	p := filepath.Join(bin, "fdfind")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	got, err := findTool("fd", "fdfind")
	if err != nil || got != p {
		t.Fatalf("findTool = %q, %v; want %q", got, err, p)
	}
	if _, err := findTool("definitely-missing"); !errors.Is(err, ErrMissingTool) {
		t.Fatalf("missing tool err = %v", err)
	}
}
