package files

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mkTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write := func(p, c string) {
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(c), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write(dir+"/needle.txt", "nothing here\nhas NEEDLE inside\nnope\n")
	write(dir+"/sub/deep/needle.log", "NEEDLE again\n")
	write(dir+"/sub/skip.go", "package main\n")
	if err := os.MkdirAll(dir+"/.hiddendir", 0755); err != nil {
		t.Fatal(err)
	}
	write(dir+"/.hiddendir/secret.txt", "hidden NEEDLE\n")
	return dir
}

func TestFind(t *testing.T) {
	if _, err := findTool("fd"); err != nil {
		t.Skip("fd not installed")
	}
	dir := mkTree(t)

	got, err := Find(context.Background(), dir, "needle", false, 100)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, e := range got {
		names[e.Name] = true
	}
	if !names["needle.txt"] || !names["needle.log"] {
		t.Errorf("find missed known matches: %v", names)
	}

	hid, err := Find(context.Background(), dir, "secret", true, 100)
	if err != nil || len(hid) != 1 {
		t.Errorf("hidden find = %v, %v", hid, err)
	}
	if e := filepath.Join(dir, ".hiddendir"); len(hid) == 1 && hid[0].Path != filepath.Base(e) && !strings.HasSuffix(hid[0].Path, "secret.txt") {
		t.Errorf("unexpected hidden result path %q", hid[0].Path)
	}

	none, err := Find(context.Background(), dir, "zzz-no-match-zzz", false, 100)
	if err != nil || len(none) != 0 {
		t.Errorf("no-match find = %v, %v", none, err)
	}
}

func TestGrep(t *testing.T) {
	if _, err := findTool("rg"); err != nil {
		t.Skip("rg not installed")
	}
	dir := mkTree(t)

	got, err := Grep(context.Background(), dir, "needle", false, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) < 2 {
		t.Fatalf("grep found %d matches, want >= 2: %+v", len(got), got)
	}
	for _, m := range got {
		if m.Line < 1 || m.Text == "" || !strings.HasSuffix(m.Path, ".txt") && !strings.HasSuffix(m.Path, ".log") {
			t.Errorf("malformed match: %+v", m)
		}
	}

	capped, err := Grep(context.Background(), dir, "needle", true, 2)
	if err != nil || len(capped) > 2 {
		t.Errorf("cap respected? n=%d err=%v", len(capped), err)
	}
}

func TestMissingToolError(t *testing.T) {
	if _, err := findTool("definitely-not-a-binary-xyz"); !errors.Is(err, ErrMissingTool) {
		t.Errorf("want ErrMissingTool, got %v", err)
	}
}

func TestReadPreviewAtWindow(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "lines.txt")
	var b strings.Builder
	for i := 1; i <= 100; i++ {
		b.WriteString("line " + strings.Repeat("x", i%7) + "\n")
	}
	os.WriteFile(p, []byte(b.String()), 0644)

	prev, err := ReadPreviewAt(p, 60, 10, 16<<10)
	if err != nil {
		t.Fatal(err)
	}
	if prev.First < 51 || prev.First > 59 {
		t.Errorf("window start = %d, want ~55", prev.First)
	}
	if got := prev.Lines[60-prev.First]; got != "line xxxx" { // "i=60 -> i%%7=4 x's 
		t.Errorf("target line content = %q", got)
	}

	head, err := ReadPreview(p, 5, 16<<10)
	if err != nil || head.First != 1 || len(head.Lines) != 5 {
		t.Errorf("plain preview = first:%d n:%d err:%v", head.First, len(head.Lines), err)
	}
}
