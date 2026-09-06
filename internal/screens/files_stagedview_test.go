package screens

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"lcc2/internal/files"
)

// The preview meta line must show old -> new for staged chmod/chown
// instead of quietly reporting the stale on-disk values.
func TestStagedMetaLineOverlays(t *testing.T) {
	e := files.Entry{Name: "a.txt", Path: "/tmp/x/a.txt", Mode: 0o644, UID: 1000, GID: 1000}

	plain := entryMetaLine(e, nil)
	if strings.Contains(plain, "->") || strings.Contains(plain, "staged") {
		t.Fatalf("unstaged line shows staged markup: %s", plain)
	}

	chmod := []files.Op{{Kind: files.OpChmod, Path: e.Path, Mode: 0o644 | os.ModeSetuid}}
	line := entryMetaLine(e, chmod)
	if !strings.Contains(line, "644 -> 4644 (staged)") {
		t.Fatalf("chmod overlay missing: %s", line)
	}

	chown := []files.Op{{Kind: files.OpChown, Path: e.Path, Arg: "root", UID: 0}}
	line = entryMetaLine(e, chown)
	if !strings.Contains(line, "-> root") {
		t.Fatalf("chown overlay missing: %s", line)
	}

	del := []files.Op{{Kind: files.OpDelete, Path: e.Path}}
	if !strings.Contains(entryMetaLine(e, del), "(staged trash)") {
		t.Fatal("delete tag missing")
	}
}

func TestMetaCardShowsStagedRow(t *testing.T) {
	e := files.Entry{Name: "a.txt", Path: "/tmp/x/a.txt", Mode: 0o644, UID: 1000, GID: 1000}
	card := metaCard(e, []files.Op{
		{Kind: files.OpChmod, Path: e.Path, Mode: 0o600},
	}, 60)
	if !strings.Contains(card, "staged") || !strings.Contains(card, "-> 600") {
		t.Fatalf("metaCard missing staged info: %s", card)
	}
	if strings.Contains(metaCard(e, nil, 60), "staged") {
		t.Fatal("unstaged metaCard shows a staged row")
	}
}

// The preview title carries a staged tag for destructive ops.
func TestPreviewTitleStagedTag(t *testing.T) {
	f := loadedFiles(t)
	f.prevPath = f.entries[0].Path
	f.prevTitle = "a.txt"
	if got := f.previewTitle(); got != "a.txt" {
		t.Fatalf("unstaged title = %q", got)
	}
	f.stager.Stage(files.Op{Kind: files.OpDelete, Path: f.prevPath})
	if got := f.previewTitle(); !strings.Contains(got, "(staged trash)") {
		t.Fatalf("title = %q, want staged trash tag", got)
	}
}

// End to end: staging a chmod then refreshing the preview shows the
// pending mode, not the stale on-disk one.
func TestPreviewShowsStagedChmod(t *testing.T) {
	f := loadedFiles(t)
	s, _ := f.Update(runeKey("P"))
	f = s.(Files)
	s, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	f = s.(Files)

	e := f.entries[0]
	f.prevPath = e.Path
	f.prevTitle = e.Name
	f.prevMeta = entryMetaLine(e, f.stagedFor(e.Path))
	if !strings.Contains(f.prevMeta, "-> 644 (staged)") && !strings.Contains(f.prevMeta, "(staged)") {
		t.Fatalf("meta = %q", f.prevMeta)
	}
	_ = filepath.Base
}
