package files

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Without gio reachable, Trash renames into ~/.local/share/Trash and
// writes a .trashinfo record. Empty PATH guarantees gio is absent
// even where it is installed.
func TestTrashHomeFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", "")

	src := filepath.Join(home, "doomed.txt")
	os.WriteFile(src, []byte("bye"), 0o644)

	permanent, err := Trash(src)
	if err != nil || permanent {
		t.Fatalf("permanent=%v err=%v", permanent, err)
	}
	if _, err := os.Lstat(src); !os.IsNotExist(err) {
		t.Fatal("source still exists")
	}
	filesDir := filepath.Join(home, ".local", "share", "Trash", "files")
	infoDir := filepath.Join(home, ".local", "share", "Trash", "info")

	trashed := filepath.Join(filesDir, "doomed.txt")
	data, err := os.ReadFile(trashed)
	if err != nil || string(data) != "bye" {
		t.Fatalf("trashed content missing: %v", err)
	}
	info, err := os.ReadFile(filepath.Join(infoDir, "doomed.txt.trashinfo"))
	if err != nil || !strings.HasPrefix(string(info), "[Trash Info]\nPath=") {
		t.Fatalf("trashinfo missing: %v %q", err, info)
	}
}

// Colliding names get numeric suffixes instead of clobbering.
func TestTrashCollisionSuffix(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	src1 := filepath.Join(home, "same.txt")
	os.WriteFile(src1, []byte("one"), 0o644)
	if permanent, err := Trash(src1); err != nil || permanent {
		t.Fatalf("first: %v %v", permanent, err)
	}
	os.WriteFile(src1, []byte("two"), 0o644)
	if _, err := Trash(src1); err != nil {
		t.Fatal(err)
	}

	filesDir := filepath.Join(home, ".local", "share", "Trash", "files")
	first, _ := os.ReadFile(filepath.Join(filesDir, "same.txt"))
	second, _ := os.ReadFile(filepath.Join(filesDir, "same.2.txt"))
	if string(first) != "one" || string(second) != "two" {
		t.Fatalf("collision handling broken: %q %q", first, second)
	}
}

// uniqueTarget keeps suffixing until the name is free.
func TestUniqueTarget(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "n.txt"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(dir, "n.2.txt"), []byte("x"), 0o644)
	got := uniqueTarget(dir, "n.txt")
	if filepath.Base(got) != "n.3.txt" {
		t.Fatalf("unique = %q", got)
	}
}
