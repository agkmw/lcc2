package files

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
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

	if err := Trash(src); err != nil {
		t.Fatalf("err=%v", err)
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
	if err := Trash(src1); err != nil {
		t.Fatalf("first: %v", err)
	}
	os.WriteFile(src1, []byte("two"), 0o644)
	if err := Trash(src1); err != nil {
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

// A source the home trash cannot take (cross-device) must be refused
// with an explicit error — never deleted permanently, never half-moved.
func TestTrashRefusesCrossDevice(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", "")

	prev := osRename
	osRename = func(string, string) error { return syscall.EXDEV }
	t.Cleanup(func() { osRename = prev })

	src := filepath.Join(t.TempDir(), "keep.txt")
	os.WriteFile(src, []byte("precious"), 0o644)

	err := Trash(src)
	if err == nil || !strings.Contains(err.Error(), "different filesystem") {
		t.Fatalf("err = %v, want cross-device refusal", err)
	}
	if _, serr := os.Lstat(src); serr != nil {
		t.Fatal("source did not survive the refusal")
	}
	ents, rerr := os.ReadDir(filepath.Join(home, ".local", "share", "Trash", "files"))
	if rerr == nil && len(ents) != 0 {
		t.Fatalf("trash received data on refusal: %v", ents)
	}
}
