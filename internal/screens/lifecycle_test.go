package screens

import (
	"os"
	"strings"
	"testing"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"lcc2/internal/files"
	"lcc2/internal/ui"
)

func spinTick() spinner.TickMsg { return spinner.TickMsg{} }

// BACKLOG-H2: re-entering a section used to stack refresh chains, and
// ticks landing on inactive screens killed them. Epochs retire stale
// chains deterministically instead.

func TestStaleTickChainsRetire(t *testing.T) {
	p := NewProcesses()
	gen := p.epoch.Add(1)
	sc, cmd := p.Update(procTickMsg{gen: gen})
	if cmd == nil {
		t.Fatal("live process chain must reschedule")
	}
	p = sc.(Processes)
	if _, cmd := p.Update(procTickMsg{gen: gen + 7}); cmd != nil {
		t.Fatal("stale process chain must die")
	}

	o := NewOverview()
	ogen := o.epoch.Add(1)
	sc, cmd = o.Update(overviewTickMsg{gen: ogen})
	if cmd == nil {
		t.Fatal("live overview chain must reschedule")
	}
	o = sc.(Overview)
	if _, cmd := o.Update(overviewTickMsg{gen: ogen + 7}); cmd != nil {
		t.Fatal("stale overview chain must die")
	}
}

func TestDisksSpinnerChainRetiresWhenIdle(t *testing.T) {
	d := NewDisks()
	d.busy = false
	if _, cmd := d.Update(spinTick()); cmd != nil {
		t.Fatal("idle disks must not keep spinning")
	}
	d.busy = true
	if _, cmd := d.Update(spinTick()); cmd == nil {
		t.Fatal("busy disks must keep the spinner alive")
	}
}

// BACKLOG-H4: while a fs operation is in flight, Files input is locked
// and the head shows a working indicator; the counter drains after the op.
func TestFilesBusyGuard(t *testing.T) {
	f := seededFiles(t)

	if f.opCount.Load() != 0 {
		t.Fatal("fresh screen must be idle")
	}
	f.opCount.Store(1)
	sc, _ := f.Update(keyRunes("a"))
	f = sc.(Files)
	if f.showHidden {
		t.Fatal("hidden toggle fired while busy")
	}
	sc, _ = f.Update(keyRunes("d"))
	f = sc.(Files)
	if f.stager.Len() != 0 {
		t.Fatal("delete staged while busy")
	}
	f.opCount.Store(0)
	sc, _ = f.Update(keyRunes("a"))
	f = sc.(Files)
	if !f.showHidden {
		t.Fatal("guard stuck: toggle refused after drain")
	}
}

// T1 end-to-end: after a listing refresh that removes an earlier row,
// the cursor stays on the same file.
func TestFilesCursorTracksAcrossRefresh(t *testing.T) {
	dir := t.TempDir()
	mk := func(names ...string) tea.Msg {
		for _, n := range names {
			os.WriteFile(dir+"/"+n, []byte("x"), 0644)
		}
		lst, err := files.List(dir, false)
		if err != nil {
			t.Fatal(err)
		}
		return dirListMsg{dir: dir, list: lst}
	}
	f := NewFiles()
	f = feed(f, ui.SizeMsg{Width: 100, Height: 30},
		mk("a.txt", "b.txt", "c.txt")).(Files)
	f.tbl.SetCursor(1) // b.txt

	os.Remove(dir + "/a.txt")
	f = feed(f, mk("b.txt", "c.txt")).(Files)

	k, ok := f.tbl.SelectedKey()
	if !ok || !strings.HasSuffix(k, "/b.txt") {
		t.Fatalf("cursor did not track b.txt: %q %v", k, ok)
	}
}

// S1: pane height hugs content and never exceeds the available area.
func TestPaneHeight(t *testing.T) {
	if got := paneHeight(10, 20); got != 10 {
		t.Fatalf("content-sized pane = %d, want 10", got)
	}
	if got := paneHeight(50, 20); got != 18 {
		t.Fatalf("oversized pane = %d, want avail-2 = 18", got)
	}
	if got := paneHeight(1, 20); got != 4 {
		t.Fatalf("tiny pane = %d, want floor 4", got)
	}
}
