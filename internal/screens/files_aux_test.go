package screens

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbletea"

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
	f = feed(f, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}).(Files)
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

	f = feed(f, tea.KeyMsg{Type: tea.KeyEsc}).(Files)
	if f.mode != "list" {
		t.Fatalf("esc did not exit find mode: %q", f.mode)
	}
}

// Grep mode: matches render in the loc|text table; a preview keyed to
// the cursor highlights its line.
func TestAuxGrepMode(t *testing.T) {
	f := auxScreen(t)
	f = feed(f, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'F'}}).(Files)
	if f.mode != "grep" {
		t.Fatalf("grep mode not entered: %q", f.mode)
	}

	hit := filepath.Join(f.cwd, "haystack.txt")
	f.grepRes = []files.Match{{Path: hit, Line: 2, Col: 6, Text: "beta NEEDLE gamma"}}
	f.syncAuxTables()
	f = feed(f, tea.KeyMsg{Type: tea.KeyDown}).(Files) // steer cursor, kicks preview

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
// nothing and reach only the query input.
func TestAuxModeSwallowsActionKeys(t *testing.T) {
	f := auxScreen(t)
	f = feed(f, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}).(Files)
	for _, k := range "dmRypxw" {
		f = feed(f, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{k}}).(Files)
	}
	if f.stager.Len() != 0 {
		t.Fatalf("aux mode leaked %d staged ops", f.stager.Len())
	}
	if f.auxInput.Value() != "dmRypxw" {
		t.Fatalf("query input got %q", f.auxInput.Value())
	}
}
