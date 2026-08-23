package files

import (
	"os"
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
