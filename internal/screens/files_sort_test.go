package screens

import (
	"testing"
	"time"

	"lcc2/internal/files"
)

func mkEntry(name string, size int64, mod time.Time, dir bool) files.Entry {
	return files.Entry{Name: name, Size: size, ModTime: mod, IsDir: dir}
}

// Dirs always lead; keys and direction apply within each group.
func TestSortEntries(t *testing.T) {
	now := time.Now()
	es := []files.Entry{
		mkEntry("b.txt", 5, now.Add(-time.Hour), false),
		mkEntry("zdir", 0, now.Add(-2*time.Hour), true),
		mkEntry("A.txt", 10, now.Add(-3*time.Hour), false),
		mkEntry("adir", 0, now.Add(-4*time.Hour), true),
	}

	sortEntries(es, fileSort{key: "name"})
	if es[0].Name != "adir" || es[1].Name != "zdir" || es[2].Name != "A.txt" || es[3].Name != "b.txt" {
		t.Fatalf("name sort: %v", names(es))
	}

	sortEntries(es, fileSort{key: "size"})
	if es[2].Name != "b.txt" || es[3].Name != "A.txt" {
		t.Fatalf("size asc: %v", names(es))
	}

	sortEntries(es, fileSort{key: "size", desc: true})
	if es[2].Name != "A.txt" || es[3].Name != "b.txt" {
		t.Fatalf("size desc: %v", names(es))
	}

	sortEntries(es, fileSort{key: "mtime"})
	if es[2].Name != "A.txt" { // oldest file first within group
		t.Fatalf("mtime asc: %v", names(es))
	}
}

// s cycles the key, S flips direction.
func TestFileSortCycle(t *testing.T) {
	s := fileSort{key: "name"}
	if s.next().key != "size" {
		t.Fatal("cycle skipped size")
	}
	s = fileSort{key: "mtime"}
	if s.next().key != "name" {
		t.Fatal("cycle did not wrap")
	}
	s = fileSort{key: "name", desc: true}.next()
	if !s.desc {
		t.Fatal("next lost direction")
	}
}

func names(es []files.Entry) []string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.Name
	}
	return out
}
