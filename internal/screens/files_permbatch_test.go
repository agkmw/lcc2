package screens

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"lcc/internal/files"
	"lcc/internal/ui"
)

// markedFiles marks both fixture entries.
func markedFiles(t *testing.T) Files {
	t.Helper()
	f := loadedFiles(t)
	s, _ := f.Update(runeKey(" ")) // mark a.txt
	f = s.(Files)
	s, _ = f.Update(runeKey("j"))
	f = s.(Files)
	s, _ = f.Update(runeKey(" ")) // mark b.txt
	f = s.(Files)
	return f
}

func TestFilesChownMultiTarget(t *testing.T) {
	f := markedFiles(t)
	orig := osGeteuid
	osGeteuid = func() int { return 0 }
	defer func() { osGeteuid = orig }()

	s, _ := f.Update(runeKey("O"))
	f = s.(Files)
	f.Update(runeKey("r"))
	f.Update(runeKey("o"))
	f.Update(runeKey("o"))
	f.Update(runeKey("t"))
	s, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	f = s.(Files)
	ops := f.stager.Ops()
	if len(ops) != 2 {
		t.Fatalf("staged %d ops, want 2 (one per mark)", len(ops))
	}
	for _, op := range ops {
		if op.Kind != files.OpChown || op.UID != 0 || op.GID != -1 {
			t.Fatalf("op = %+v", op)
		}
	}
}

func TestFilesPermsMultiTarget(t *testing.T) {
	f := markedFiles(t)
	s, _ := f.Update(runeKey("P"))
	f = s.(Files)
	s, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	f = s.(Files)
	ops := f.stager.Ops()
	if len(ops) != 2 || ops[0].Kind != files.OpChmod {
		t.Fatalf("staged = %+v, want 2 chmods", ops)
	}
}

func TestFilesPermsRecursive(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	paths := []string{
		filepath.Join(dir, "a.txt"),
		filepath.Join(sub, "b.txt"),
		filepath.Join(sub, "c.txt"),
	}
	for _, p := range paths {
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	f := NewFiles()
	f.w, f.h = 100, 30
	f.entries = []files.Entry{
		{Name: filepath.Base(dir), Path: dir, IsDir: true, Mode: 0o755 | os.ModeDir},
	}
	rows := []ui.Row{{"dir", "", "", "", ""}}
	f.tbl.SetRowsTracked(rows, []string{dir})
	f.layout()

	s, _ := f.Update(runeKey("P"))
	f = s.(Files)
	s, _ = f.Update(runeKey("r")) // recursive on
	f = s.(Files)
	if !strings.Contains(f.View(), "recursive") {
		t.Fatal("recursive indicator missing from editor view")
	}
	s, cmd := f.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	f = s.(Files)
	ops := f.stager.Ops()
	if len(ops) != 5 { // dir + a.txt + sub + b.txt + c.txt
		t.Fatalf("staged %d ops, want 5", len(ops))
	}
	msg := toastOf(t, cmd)
	if !strings.Contains(msg.Text, "5 paths") {
		t.Fatalf("toast = %+v", msg)
	}
}

func TestFilesPermsRecursiveReviewGate(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 101; i++ { // big batch: staging is dialog-free now
		p := filepath.Join(dir, fmt.Sprintf("f%03d", i))
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	f := NewFiles()
	f.w, f.h = 100, 30
	f.entries = []files.Entry{
		{Name: filepath.Base(dir), Path: dir, IsDir: true, Mode: 0o755 | os.ModeDir},
	}
	f.tbl.SetRowsTracked([]ui.Row{{"dir", "", "", "", ""}}, []string{dir})
	f.layout()

	s, _ := f.Update(runeKey("P"))
	f = s.(Files)
	s, _ = f.Update(runeKey("r"))
	f = s.(Files)
	s, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	f = s.(Files)
	// no stage-time dialog: all 102 ops queued directly
	if len(f.stager.Ops()) != 102 {
		t.Fatalf("staged %d ops, want 102", len(f.stager.Ops()))
	}
	if f.review != nil {
		t.Fatal("review opened at stage time")
	}

	// w opens the review listing the whole batch
	s, _ = f.Update(runeKey("w"))
	f = s.(Files)
	if f.review == nil {
		t.Fatal("w did not open the review")
	}
	if !f.CapturingInput() {
		t.Fatal("review open but CapturingInput false")
	}
	body := stripANSI(f.View())
	if !strings.Contains(body, "102 staged changes") || !strings.Contains(body, "chmod") {
		t.Fatal("review view missing batch summary")
	}
	// esc backs out without applying
	s, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	f = s.(Files)
	if f.review != nil || len(f.stager.Ops()) != 102 {
		t.Fatal("esc must close review and keep the queue")
	}
}

func TestFilesGroupColumn(t *testing.T) {
	f := loadedFiles(t)
	f.syncTable()
	if !strings.Contains(f.tbl.View(), "group") {
		t.Fatal("group column header missing")
	}
	e := f.entries[0]
	if !strings.Contains(entryMetaLine(e, nil), ":") {
		t.Fatal("meta line lacks owner:group")
	}
}
