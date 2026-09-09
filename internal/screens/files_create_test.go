package screens

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"lcc/internal/files"
)

// `n` stages an empty-file create with a phantom row (no trailing
// slash - that marks directories), and mkdir stays dirs-only.
func TestNewFileStagesCreate(t *testing.T) {
	f, dir := filesScreen(t)

	sc, _ := f.Update(keyRunes("n"))
	f = sc.(Files)
	if f.prompt == nil {
		t.Fatal("n did not open the create prompt")
	}
	in := *f.prompt
	if !strings.Contains(in.Value(), "") && in.Placeholder != "" {
		t.Fatal("unexpected prompt state")
	}
	in.SetValue("notes.txt")
	f.prompt = &in
	sc, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	f = sc.(Files)

	if f.stager.Len() != 1 {
		t.Fatalf("create not staged: %d", f.stager.Len())
	}
	op := f.stager.Ops()[0]
	if op.Kind != files.OpCreate || op.Path != filepath.Join(dir, "notes.txt") {
		t.Fatalf("staged op = %v %v", op.Kind, op.Path)
	}
	body := stripANSI(f.View())
	if !strings.Contains(body, "notes.txt") || strings.Contains(body, "notes.txt/") {
		t.Fatalf("phantom row wrong: %s", body)
	}
	if err := files.ApplyOp(op); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "notes.txt")); err != nil {
		t.Fatal("file not created on apply")
	}
}

// Pasting the clipboard into the same directory stages a deduped copy
// instead of erroring.
func TestPasteIntoSameDirDeduplicates(t *testing.T) {
	f, dir := filesScreen(t)
	f.clip = []string{filepath.Join(dir, "a.txt")}

	sc, _ := f.Update(keyRunes("p"))
	f = sc.(Files)
	if f.stager.Len() != 1 {
		t.Fatalf("paste not staged: %d", f.stager.Len())
	}
	op := f.stager.Ops()[0]
	if op.Kind != files.OpCopy {
		t.Fatalf("kind = %v", op.Kind)
	}
	if op.Arg != filepath.Join(dir, "a.2.txt") {
		t.Fatalf("dst = %q, want a.2.txt", op.Arg)
	}
	if err := files.ApplyOp(op); err != nil {
		t.Fatalf("apply: %v", err)
	}
}
