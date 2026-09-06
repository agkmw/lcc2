package screens

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"lcc2/internal/files"
	"lcc2/internal/ui"
)

func loadedFiles(t *testing.T) Files {
	t.Helper()
	dir := t.TempDir()
	f := NewFiles()
	f.w, f.h = 100, 30
	f.entries = nil
	for _, n := range []string{"a.txt", "b.txt"} {
		p := filepath.Join(dir, n)
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		f.entries = append(f.entries, files.Entry{Name: n, Path: p, Mode: 0o644, UID: 1000, GID: 1000})
	}
	rows := make([]ui.Row, len(f.entries))
	keys := make([]string, len(f.entries))
	for i, e := range f.entries {
		rows[i] = ui.Row{e.Name, "", "", ""}
		keys[i] = e.Path
	}
	f.tbl.SetRowsTracked(rows, keys)
	f.layout()
	return f
}

func TestFilesChownNotRootGated(t *testing.T) {
	f := loadedFiles(t)
	orig := osGeteuid
	osGeteuid = func() int { return 1000 }
	defer func() { osGeteuid = orig }()

	s, cmd := f.Update(runeKey("O"))
	f = s.(Files)
	if f.chownForm != nil {
		t.Fatal("chown form opened without root")
	}
	msg := toastOf(t, cmd)
	if msg.Kind != "err" || msg.Text == "" {
		t.Fatalf("toast = %+v", msg)
	}
}

func TestFilesChownFormFlow(t *testing.T) {
	f := loadedFiles(t)
	orig := osGeteuid
	osGeteuid = func() int { return 0 }
	defer func() { osGeteuid = orig }()

	s, _ := f.Update(runeKey("O"))
	f = s.(Files)
	if f.chownForm == nil {
		t.Fatal("O did not open the chown form")
	}
	if !f.CapturingInput() {
		t.Fatal("form open but CapturingInput false")
	}
	if !strings.Contains(f.View(), "chown") {
		t.Fatal("chown form missing from view")
	}
	// blank submit = nothing to change
	s, cmd := f.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	f = s.(Files)
	if f.chownForm != nil {
		t.Fatal("form still open after submit")
	}
	msg := toastOf(t, cmd)
	if msg.Kind != "info" {
		t.Fatalf("blank submit toast = %+v", msg)
	}
	if f.stager.Len() != 0 {
		t.Fatal("blank submit staged an op")
	}
}

func TestFilesChownStagesOp(t *testing.T) {
	f := loadedFiles(t)
	orig := osGeteuid
	osGeteuid = func() int { return 0 }
	defer func() { osGeteuid = orig }()

	s, _ := f.Update(runeKey("O"))
	f = s.(Files)
	// type owner name; the group field stays blank (= unchanged)
	f.Update(runeKey("r"))
	f.Update(runeKey("o"))
	f.Update(runeKey("o"))
	f.Update(runeKey("t"))
	s, cmd := f.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	f = s.(Files)
	ops := f.stager.Ops()
	if len(ops) != 1 || ops[0].Kind != files.OpChown {
		t.Fatalf("staged ops = %+v", ops)
	}
	if ops[0].UID != 0 || ops[0].GID != -1 {
		t.Fatalf("resolved ids = %d,%d, want 0,-1", ops[0].UID, ops[0].GID)
	}
	if ops[0].Arg != "root" {
		t.Fatalf("Arg = %q", ops[0].Arg)
	}
	msg := toastOf(t, cmd)
	if msg.Kind != "info" || !strings.Contains(msg.Text, "staged chown") {
		t.Fatalf("toast = %+v", msg)
	}
}

func TestFilesChownUnknownUser(t *testing.T) {
	f := loadedFiles(t)
	orig := osGeteuid
	osGeteuid = func() int { return 0 }
	defer func() { osGeteuid = orig }()

	s, _ := f.Update(runeKey("O"))
	f = s.(Files)
	for _, r := range "zz-nobody" {
		f.Update(runeKey(string(r)))
	}
	s, cmd := f.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	f = s.(Files)
	if f.stager.Len() != 0 {
		t.Fatal("unknown user staged an op")
	}
	msg := toastOf(t, cmd)
	if msg.Kind != "err" || !strings.Contains(msg.Text, "unknown user") {
		t.Fatalf("toast = %+v", msg)
	}
}

func TestFilesChownEscCancels(t *testing.T) {
	f := loadedFiles(t)
	orig := osGeteuid
	osGeteuid = func() int { return 0 }
	defer func() { osGeteuid = orig }()

	s, _ := f.Update(runeKey("O"))
	f = s.(Files)
	s, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	f = s.(Files)
	if f.chownForm != nil || f.CapturingInput() {
		t.Fatal("esc did not close the form")
	}
}
