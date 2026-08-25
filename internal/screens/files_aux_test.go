package screens

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/bubbletea/v2"

	"lcc2/internal/files"
	"lcc2/internal/ui"
)

func auxScreen(t *testing.T) Files {
	t.Helper()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "haystack.txt"), []byte("alpha\nbeta NEEDLE gamma\ndelta\n"), 0644)
	os.WriteFile(filepath.Join(dir, "needle.txt"), []byte("x\n"), 0644)
	lst, err := files.List(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	f := NewFiles()
	f.cwd = dir
	return feed(f, ui.SizeMsg{Width: 110, Height: 30}, dirListMsg{dir: dir, list: lst}).(Files)
}

// Find mode: entry, synthetic results render, esc returns to listing.
func TestAuxFindMode(t *testing.T) {
	f := auxScreen(t)
	f = feed(f, tea.KeyPressMsg{Code: 102, Text: "f"}).(Files)
	if f.mode != "find" || !f.CapturingInput() {
		t.Fatalf("find mode not entered: %q capturing=%v", f.mode, f.CapturingInput())
	}

	res := []files.Entry{
		{Name: "needle.txt", Path: filepath.Join(f.cwd, "needle.txt")},
		{Name: "sub", Path: filepath.Join(f.cwd, "sub"), IsDir: true},
	}
	f = feed(f, findResultMsg{gen: f.auxGen.Load(), entries: res}).(Files)
	if f.auxCount() != 2 {
		t.Fatalf("find results = %d, want 2", f.auxCount())
	}
	v := stripANSI(f.View())
	if !strings.Contains(v, "find") || !strings.Contains(v, "needle.txt") {
		t.Fatalf("view missing query bar or results:\n%s", v[:minInt(len(v), 400)])
	}

	f = feed(f, tea.KeyPressMsg{Code: tea.KeyEsc}).(Files)
	if f.mode != "list" {
		t.Fatalf("esc did not exit find mode: %q", f.mode)
	}
}

// Grep mode: matches render in the loc|text table; a preview keyed to
// the cursor highlights its line.
func TestAuxGrepMode(t *testing.T) {
	f := auxScreen(t)
	f = feed(f, tea.KeyPressMsg{Code: 70, Text: "F"}).(Files)
	if f.mode != "grep" {
		t.Fatalf("grep mode not entered: %q", f.mode)
	}

	hit := filepath.Join(f.cwd, "haystack.txt")
	f.grepRes = []files.Match{{Path: hit, Line: 2, Col: 6, Text: "beta NEEDLE gamma"}}
	f.syncAuxTables()
	f = feed(f, tea.KeyPressMsg{Code: tea.KeyDown}).(Files) // steer cursor, kicks preview

	pv, err := files.ReadPreviewAt(hit, 2, 10, 1<<10)
	if err != nil {
		t.Fatal(err)
	}
	f.fetching = false
	f = feed(f, filePreviewMsg{path: hit, key: f.expectKey(), p: pv, hit: 2}).(Files)
	if !strings.Contains(f.previewMeta(), "line 2") {
		t.Fatalf("preview meta missing hit line: %q", f.previewMeta())
	}
	body := stripANSI(f.previewContent())
	if !strings.Contains(body, "NEEDLE") {
		t.Fatal("preview lost match content")
	}
}

// While an aux search owns the keyboard, destructive keys must stage
// nothing and reach only the query input. Letters like j/k are legal
// query characters — only arrows steer results.
func TestAuxModeSwallowsActionKeys(t *testing.T) {
	f := auxScreen(t)
	f = feed(f, tea.KeyPressMsg{Code: 102, Text: "f"}).(Files)
	for _, k := range "djkmy" {
		f = feed(f, tea.KeyPressMsg{Code: k, Text: string(k)}).(Files)
	}
	if f.stager.Len() != 0 {
		t.Fatalf("aux mode leaked %d staged ops", f.stager.Len())
	}
	if f.auxInput.Value() != "djkmy" {
		t.Fatalf("query input got %q", f.auxInput.Value())
	}
}

// Arrow keys move the results cursor without touching the query.
func TestAuxArrowsSteerResults(t *testing.T) {
	f := auxScreen(t)
	f = feed(f, tea.KeyPressMsg{Code: 102, Text: "f"}).(Files)
	dir := f.cwd
	f.findRes = []files.Entry{
		{Name: "a.txt", Path: filepath.Join(dir, "a.txt")},
		{Name: "b.txt", Path: filepath.Join(dir, "b.txt")},
	}
	f.syncAuxTables()
	f.fetching = false
	f = feed(f, tea.KeyPressMsg{Code: tea.KeyDown}).(Files)
	if got := f.findTbl.Cursor(); got != 1 {
		t.Fatalf("cursor = %d, want 1", got)
	}
	if f.auxInput.Value() != "" {
		t.Fatalf("arrows leaked into query: %q", f.auxInput.Value())
	}
}

// ctrl+o / ctrl+i walk the directory history; entering a directory
// pushes and clears forward.
func TestDirectoryBackForwardHistory(t *testing.T) {
	root := auxScreen(t).cwd
	sub := filepath.Join(root, "sub")
	os.MkdirAll(sub, 0755)

	lst, _ := files.List(root, false)
	f := NewFiles()
	f.cwd = root
	f = feed(f, ui.SizeMsg{Width: 110, Height: 30}, dirListMsg{dir: root, list: lst}).(Files)
	// land the cursor on sub/ then enter it
	for i, e := range f.entries {
		if e.Path == sub {
			f.tbl.SetCursor(i)
			break
		}
	}
	f = feed(f, keyEnter()).(Files)
	f = feed(f, dirListMsg{dir: sub, list: mustList(t, sub)}).(Files)
	if f.cwd != sub || len(f.backStack) != 1 {
		t.Fatalf("enter did not navigate+push: cwd=%q back=%v", f.cwd, f.backStack)
	}

	f = feed(f, tea.KeyPressMsg{Code: 'o', Mod: tea.ModCtrl}).(Files)
	f = feed(f, dirListMsg{dir: root, list: lst}).(Files)
	if f.cwd != root || len(f.fwdStack) == 0 {
		t.Fatalf("ctrl+o did not go back: cwd=%q fwd=%v", f.cwd, f.fwdStack)
	}

	f = feed(f, tea.KeyPressMsg{Code: 'i', Mod: tea.ModCtrl}).(Files)
	f = feed(f, dirListMsg{dir: sub, list: mustList(t, sub)}).(Files)
	if f.cwd != sub {
		t.Fatalf("ctrl+i did not go forward: cwd=%q", f.cwd)
	}
}

func keyEnter() tea.Msg { return tea.KeyPressMsg{Code: tea.KeyEnter} }

func mustList(t *testing.T, dir string) []files.Entry {
	t.Helper()
	l, err := files.List(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	return l
}
