package screens

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"lcc/internal/files"
)

// parseOctal must map unix special bits onto Go's FileMode flags; a raw
// 0o4000 in os.FileMode means nothing to os.Chmod.
func TestParseOctalMapsSpecialBits(t *testing.T) {
	m, err := parseOctal("4755")
	if err != nil {
		t.Fatal(err)
	}
	if m&os.ModeSetuid == 0 || m.Perm() != 0o755 {
		t.Fatalf("4755 -> %v", m)
	}

	m, err = parseOctal("1777")
	if err != nil || m&os.ModeSticky == 0 || m.Perm() != 0o777 {
		t.Fatalf("1777 -> %v err=%v", m, err)
	}

	if _, err := parseOctal("4855"); err == nil {
		t.Fatal("non-octal digit accepted")
	}
	if _, err := parseOctal("177777"); err == nil {
		t.Fatal("out-of-range mode accepted")
	}
}

// The perm editor's fourth row toggles setuid/setgid/sticky.
func TestPermEditorSpecialRow(t *testing.T) {
	f := loadedFiles(t)
	s, _ := f.Update(runeKey("P"))
	f = s.(Files)
	if f.permEdit == nil {
		t.Fatal("P did not open the perm editor")
	}
	// walk down to the special row, toggle setuid (col 0)
	for i := 0; i < 3; i++ {
		s, _ = f.Update(runeKey("j"))
		f = s.(Files)
	}
	if f.permRow != 3 {
		t.Fatalf("permRow = %d, want 3", f.permRow)
	}
	s, _ = f.Update(runeKey(" "))
	f = s.(Files)
	if f.permEdit.Special != 4 {
		t.Fatalf("Special = %d, want 4", f.permEdit.Special)
	}
	s, cmd := f.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	f = s.(Files)
	if f.permEdit != nil {
		t.Fatal("editor still open after enter")
	}
	ops := f.stager.Ops()
	if len(ops) != 1 || ops[0].Kind != files.OpChmod {
		t.Fatalf("ops = %+v", ops)
	}
	if ops[0].Mode&os.ModeSetuid == 0 || ops[0].Mode.Perm() != 0o644 {
		t.Fatalf("mode = %v, want setuid + 644", ops[0].Mode)
	}
	if msg := toastOf(t, cmd); !strings.Contains(msg.Text, "4644") {
		t.Fatalf("toast = %+v, want 4644", msg)
	}
}

// Typing an octal string stages that mode, overriding the grid.
func TestPermEditorOctalEntry(t *testing.T) {
	f := loadedFiles(t)
	s, _ := f.Update(runeKey("P"))
	f = s.(Files)
	for _, d := range []string{"2", "7", "5", "5"} {
		s, _ = f.Update(runeKey(d))
		f = s.(Files)
	}
	if f.permOctal != "2755" {
		t.Fatalf("buffer = %q", f.permOctal)
	}
	if !strings.Contains(f.View(), "entry:") || !strings.Contains(f.View(), "2755_") {
		t.Fatal("entry buffer missing from view")
	}
	// backspace pops, retyping restores
	s, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	f = s.(Files)
	if f.permOctal != "275" {
		t.Fatalf("buffer after backspace = %q", f.permOctal)
	}
	s, _ = f.Update(runeKey("5"))
	f = s.(Files)

	s, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	f = s.(Files)
	ops := f.stager.Ops()
	if len(ops) != 1 {
		t.Fatalf("ops = %+v", ops)
	}
	if ops[0].Mode&os.ModeSetgid == 0 || ops[0].Mode.Perm() != 0o755 {
		t.Fatalf("mode = %v, want setgid + 755", ops[0].Mode)
	}
	// the 9-grid below the buffer is ignored: "2755" wins
	if f.permEdit != nil {
		t.Fatal("editor still open")
	}
}

// A 9th-digit press is dropped; the buffer caps at four digits.
func TestPermEditorOctalCap(t *testing.T) {
	f := loadedFiles(t)
	s, _ := f.Update(runeKey("P"))
	f = s.(Files)
	for _, d := range []string{"7", "7", "7", "7", "7"} {
		s, _ = f.Update(runeKey(d))
		f = s.(Files)
	}
	if f.permOctal != "7777" {
		t.Fatalf("buffer = %q, want 7777", f.permOctal)
	}
}
