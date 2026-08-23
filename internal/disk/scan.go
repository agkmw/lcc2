package disk

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Item is one row of a directory analysis: a top-level child of the
// scanned root (directory with recursive size, or plain file).
type Item struct {
	Path  string
	Name  string
	IsDir bool
	Size  int64 // subtree size for directories, file size otherwise
}

// ScanResult is the outcome of analyzing one directory.
type ScanResult struct {
	Root      string
	Items     []Item // children of Root, sorted by size desc
	TotalSize int64
	Duration  time.Duration
}

// ScanDir analyzes the direct children of root, accumulating recursive
// byte sizes per child (like `du`). Progress reports cumulative bytes.
// Symlinks are never followed; unreadable subtrees are skipped.
func ScanDir(ctx context.Context, root string, progress func(bytes int64)) (*ScanResult, error) {
	start := time.Now()
	clean := filepath.Clean(root)

	sizes := map[string]int64{}     // dir -> accumulated bytes (all levels)
	fileItems := map[string]int64{} // file path -> size

	var total int64
	var lastReport time.Time

	err := filepath.WalkDir(clean, func(path string, d fs.DirEntry, err error) error {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if err != nil {
			if path == clean {
				return err // root unreadable is fatal, not skippable
			}
			return fs.SkipDir // permission denied etc: skip silently
		}
		if d.IsDir() {
			sizes[path] = 0
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		var size int64
		if info, ierr := d.Info(); ierr == nil {
			size = info.Size()
		}
		sizes[path] = -size // marker; folded into parents below
		fileItems[path] = size
		total += size
		if progress != nil && time.Since(lastReport) > 150*time.Millisecond {
			progress(total)
			lastReport = time.Now()
		}
		return nil
	})
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr // never pass off partial data as complete
		}
		return nil, err
	}

	dirSizes := map[string]int64{}
	for path, v := range sizes {
		if v >= 0 { // a directory
			continue
		}
		size := -v
		for p := filepath.Dir(path); ; p = filepath.Dir(p) {
			if _, isDir := sizes[p]; !isDir {
				break // outside scanned tree or a file row
			}
			dirSizes[p] += size
			if p == clean || p == string(filepath.Separator) || p == "." {
				break
			}
		}
	}

	res := &ScanResult{Root: clean, TotalSize: total, Duration: time.Since(start)}
	children, _ := os.ReadDir(clean)
	for _, e := range children {
		child := filepath.Join(clean, e.Name())
		it := Item{Path: child, Name: e.Name(), IsDir: e.IsDir()}
		if it.IsDir {
			it.Size = dirSizes[child]
		} else {
			it.Size = fileItems[child]
		}
		res.Items = append(res.Items, it)
	}
	sort.SliceStable(res.Items, func(i, j int) bool {
		if res.Items[i].IsDir != res.Items[j].IsDir {
			return res.Items[i].IsDir
		}
		return res.Items[i].Size > res.Items[j].Size
	})
	return res, nil
}

// LargestFiles returns up to n biggest regular files among items.
func LargestFiles(items []Item, n int) []Item {
	out := make([]Item, 0, len(items))
	for _, it := range items {
		if !it.IsDir {
			out = append(out, it)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Size > out[j].Size })
	if len(out) > n {
		out = out[:n]
	}
	return out
}
