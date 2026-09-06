package screens

import (
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// A staged create must appear in the listing even though the path is
// not on disk yet - it used to be invisible (glyph could never render
// on a row that did not exist).
func TestStagedMkdirGetsPhantomRow(t *testing.T) {
	f, dir := filesScreen(t)

	sc, _ := f.Update(keyRunes("m"))
	f = sc.(Files)
	in := *f.prompt
	in.SetValue("brandnew")
	f.prompt = &in
	sc, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	f = sc.(Files)

	if f.stager.Len() != 1 {
		t.Fatalf("mkdir not staged: %d", f.stager.Len())
	}
	body := stripANSI(f.View())
	if !strings.Contains(body, "brandnew/") {
		t.Fatalf("phantom row missing from listing: %s", body)
	}
	if !strings.Contains(body, "+") {
		t.Fatal("phantom row missing the staged-create glyph")
	}
	_ = dir
}

// Cursor onto the phantom row shows a staged-create preview instead
// of silently keeping the previous file's preview.
func TestPhantomPreview(t *testing.T) {
	f, dir := filesScreen(t)

	sc, _ := f.Update(keyRunes("m"))
	f = sc.(Files)
	in := *f.prompt
	in.SetValue("brandnew")
	f.prompt = &in
	sc, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	f = sc.(Files)

	// land the cursor just above the phantom row and move down onto it
	// so the preview hook runs
	f.tbl.SetCursor(len(f.entries) - 1)
	sc, _ = f.Update(keyRunes("j"))
	f = sc.(Files)
	if f.prevPath != filepath.Join(dir, "brandnew") {
		t.Fatalf("prevPath = %q, want the phantom path", f.prevPath)
	}
	body := stripANSI(f.View())
	if !strings.Contains(body, "staged create") || !strings.Contains(body, "created on save") {
		t.Fatalf("phantom preview missing: %s", body)
	}
}
