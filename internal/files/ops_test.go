package files

import (
	"os"
	"strings"
	"testing"
)

func TestListSortsDirsFirst(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(dir+"/b_dir", 0755)
	os.WriteFile(dir+"/a_file", []byte("x"), 0644)
	os.WriteFile(dir+"/.hidden", []byte("x"), 0644)

	entries, err := List(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("hidden file leaked: %d entries", len(entries))
	}
	if !entries[0].IsDir || entries[0].Name != "b_dir" {
		t.Fatalf("dirs not first: %+v", entries[0])
	}

	all, err := List(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("showHidden broken: %d", len(all))
	}
}

func TestCopyMoveDelete(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	os.WriteFile(src+"/f.txt", []byte("hello"), 0644)

	if err := Copy(src+"/f.txt", dst); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(dst + "/f.txt")
	if string(data) != "hello" {
		t.Fatalf("copy content = %q", data)
	}

	os.Remove(src + "/f.txt") // move back must not overwrite
	if err := Move(dst+"/f.txt", src); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(src + "/f.txt"); err != nil {
		t.Fatal("move failed")
	}

	if err := Delete(src + "/f.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(src + "/f.txt"); !os.IsNotExist(err) {
		t.Fatal("delete failed")
	}
}

func TestCopyOntoItselfRefused(t *testing.T) {
	dir := t.TempDir()
	src := dir + "/keep.txt"
	os.WriteFile(src, []byte("precious"), 0644)

	if err := Copy(src, dir); err == nil {
		t.Fatal("same-path paste must be refused")
	}
	data, err := os.ReadFile(src)
	if err != nil || string(data) != "precious" {
		t.Fatalf("source damaged: %q %v", data, err)
	}
}

// BACKLOG-H5: overwrites must be refused, not silent.
func TestOverwritesRefused(t *testing.T) {
	srcDir, dstDir := t.TempDir(), t.TempDir()
	os.WriteFile(srcDir+"/f", []byte("new"), 0644)
	os.WriteFile(dstDir+"/f", []byte("old"), 0644)

	if err := Copy(srcDir+"/f", dstDir); err == nil {
		t.Fatal("copy overwrite must fail")
	}
	if err := Move(srcDir+"/f", dstDir); err == nil {
		t.Fatal("move overwrite must fail")
	}
	if string(mustRead(t, dstDir+"/f")) != "old" {
		t.Fatal("destination clobbered")
	}
	if err := Rename(srcDir+"/f", "f"); err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("rename onto itself should be caught: %v", err)
	}
}

// BACKLOG-H6: a directory may never be copied into its own subtree.
func TestCopyIntoOwnSubtreeRefused(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(root+"/sub", 0755); err != nil {
		t.Fatal(err)
	}
	if err := Copy(root, root+"/sub"); err == nil {
		t.Fatal("dir into own subtree must be refused")
	}
	if entries, _ := os.ReadDir(root + "/sub"); len(entries) != 0 {
		t.Fatal("subtree polluted by refused copy")
	}
}

func mustRead(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestChmod(t *testing.T) {
	dir := t.TempDir()
	p := dir + "/f"
	os.WriteFile(p, []byte("x"), 0600)
	if err := Chmod(p, 0755); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(p)
	if info.Mode().Perm() != 0755 {
		t.Fatalf("perm = %v", info.Mode().Perm())
	}
}
