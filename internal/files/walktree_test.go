package files

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestWalkTree(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	files := []string{
		filepath.Join(dir, "a.txt"),
		filepath.Join(sub, "b.txt"),
	}
	for _, f := range files {
		if err := os.WriteFile(f, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(files[0], filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}

	paths, err := WalkTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)
	var got []string
	for _, p := range paths {
		got = append(got, strings.TrimPrefix(p, dir))
	}
	want := []string{"", "/a.txt", "/sub", "/sub/b.txt"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("WalkTree = %v, want %v (symlink excluded)", got, want)
	}
}
