package screens

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

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

var _ = tea.KeyRunes // keep tea import if feeds change

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
