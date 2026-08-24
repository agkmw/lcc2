package files

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ErrMissingTool reports that an optional external helper binary is
// not installed; screens surface it as a toast.
var ErrMissingTool = errors.New("required tool not found")

func findTool(name string) (string, error) {
	p, err := exec.LookPath(name)
	if err != nil {
		return "", ErrMissingTool
	}
	return p, nil
}

// Find locates paths under root whose name matches pattern using fd.
// Hidden entries follow the flag; results are capped and sorted
// dirs-first like List.
func Find(ctx context.Context, root, pattern string, hidden bool, limit int) ([]Entry, error) {
	if strings.TrimSpace(pattern) == "" {
		return []Entry{}, nil
	}
	bin, err := findTool("fd")
	if err != nil {
		return nil, err
	}
	args := []string{"--absolute-path", "--max-results", strconv.Itoa(limit)}
	if hidden {
		args = append(args, "--hidden")
	}
	args = append(args, "--", pattern, root)
	out, err := exec.CommandContext(ctx, bin, args...).Output()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil && len(out) == 0 {
		return nil, err
	}
	var entries []Entry
	for _, p := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if p == "" {
			continue
		}
		e := Entry{Name: filepath.Base(p), Path: p}
		if info, err := os.Lstat(p); err == nil {
			e.IsDir = info.IsDir()
			e.Size = info.Size()
			e.Mode = info.Mode()
			e.ModTime = info.ModTime()
		}
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	return entries, nil
}
