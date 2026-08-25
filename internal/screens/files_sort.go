package screens

import (
	"sort"
	"strings"

	"lcc2/internal/files"
)

// fileSort is the listing order of the Files screen. Dirs always come
// first regardless of key or direction.
type fileSort struct {
	key  string // "name", "size", "mtime"
	desc bool
}

var fileSortCycle = []string{"name", "size", "mtime"}

func (s fileSort) next() fileSort {
	for i, k := range fileSortCycle {
		if k == s.key {
			return fileSort{key: fileSortCycle[(i+1)%len(fileSortCycle)], desc: s.desc}
		}
	}
	return fileSort{key: "name"}
}

func (s fileSort) label() string {
	l := s.key
	if s.desc {
		l += " ↓"
	}
	return l
}

// sortEntries orders entries in place: dirs first, then by the chosen
// key. name is case-insensitive.
func sortEntries(es []files.Entry, m fileSort) {
	less := func(a, b files.Entry) bool { return false }
	switch m.key {
	case "size":
		less = func(a, b files.Entry) bool { return a.Size < b.Size }
	case "mtime":
		less = func(a, b files.Entry) bool { return a.ModTime.Before(b.ModTime) }
	default:
		less = func(a, b files.Entry) bool {
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}
	}
	if m.desc {
		old := less
		less = func(a, b files.Entry) bool { return old(b, a) }
	}
	// Stable dirs-first wrap.
	sort.SliceStable(es, func(i, j int) bool {
		if es[i].IsDir != es[j].IsDir {
			return es[i].IsDir
		}
		return less(es[i], es[j])
	})
}
