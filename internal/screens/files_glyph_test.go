package screens

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"

	"lcc2/internal/files"
	"lcc2/internal/ui"
)

// Regression: stagedAt[path] on an unstaged row returned the zero
// OpKind (OpMkdir), painting a phantom green "+" onto every file.
func TestUnstagedRowsCarryNoGlyph(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "plain.txt"), []byte("x"), 0644)
	files.Mkdir(dir, "subdir")
	lst, err := files.List(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	f := NewFiles()
	f = feed(f, ui.SizeMsg{Width: 100, Height: 30}, dirListMsg{dir: dir, list: lst}).(Files)
	body := stripANSI(f.View())
	for _, want := range []string{"plain.txt", "subdir/"} {
		i := strings.Index(body, want)
		if i < 2 {
			t.Fatalf("row %q missing from view", want)
		}
		if lead := body[i-2 : i]; strings.Contains(lead, "+") {
			t.Errorf("unstaged row %q carries staged glyph: %q", want, body[i-4:i+12])
		}
	}
}

// Staging one op must mark exactly its target, and the tab badge is
// ASCII ("*n") per ADR-0010.
func TestStagedGlyphAndBadgeAscii(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "gone.txt"), []byte("x"), 0644)
	lst, _ := files.List(dir, false)
	f := NewFiles()
	f = feed(f, ui.SizeMsg{Width: 100, Height: 30}, dirListMsg{dir: dir, list: lst}).(Files)
	f.stager.Stage(files.Op{Kind: files.OpDelete, Path: filepath.Join(dir, "gone.txt")})
	f.syncTable()

	if b := f.Badge(); b != "*1" {
		t.Errorf("badge = %q, want \"*1\"", b)
	}
	body := stripANSI(f.View())
	i := strings.Index(body, "gone.txt")
	if i < 2 || body[i-2:i] != "- " {
		t.Errorf("staged delete missing '- ' marker near %q", body[maxInt(0, i-3):minInt(i+8, len(body))])
	}

	lst2, _ := files.List(dir, false)
	f2 := NewFiles()
	f2 = feed(f2, ui.SizeMsg{Width: 100, Height: 30}, dirListMsg{dir: dir, list: lst2}).(Files)
	if s := stripANSI(f2.View()); strings.Contains(s, "+") {
		t.Error("fresh listing shows staged glyphs")
	}
}

// Directories keep their type bit in the mode column.
func TestDirModeShowsD(t *testing.T) {
	dir := t.TempDir()
	files.Mkdir(dir, "adir")
	lst, _ := files.List(dir, false)
	f := NewFiles()
	f = feed(f, ui.SizeMsg{Width: 100, Height: 30}, dirListMsg{dir: dir, list: lst}).(Files)
	if !strings.Contains(stripANSI(f.View()), "drwx") {
		t.Error("directory mode column lost the 'd' prefix")
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Symlinks render with an "@" tail so they are distinguishable from
// regular files at a glance.
func TestSymlinkNameGetsAtTail(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "target.txt"), []byte("x"), 0644)
	os.Symlink(filepath.Join(dir, "target.txt"), filepath.Join(dir, "link"))
	lst, err := files.List(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	f := NewFiles()
	f = feed(f, ui.SizeMsg{Width: 100, Height: 30}, dirListMsg{dir: dir, list: lst}).(Files)
	if !strings.Contains(stripANSI(f.View()), "link@") {
		t.Error("symlink missing '@' marker in listing")
	}
}

// Main-pane rows keep their semantic colors even when columns refit
// narrow — the first-party renderer no longer strips styled cells
// (H8). Asserts on raw bytes since stripANSI would hide the win.
func TestMainPaneRowsKeepColor(t *testing.T) {
	restore := ui.SetProfileOverride(colorprofile.TrueColor)
	defer restore()

	dir := t.TempDir()
	files.Mkdir(dir, "projects")
	lst, err := files.List(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	f := NewFiles()
	f = feed(f, ui.SizeMsg{Width: 80, Height: 30}, dirListMsg{dir: dir, list: lst}).(Files)
	view := f.View() // 76-col content: name column must refit below its base width
	if !strings.Contains(view, "\x1b[1;") && !strings.Contains(view, ";38;2;") {
		t.Error("main pane lost semantic styling after column refit")
	}
}
