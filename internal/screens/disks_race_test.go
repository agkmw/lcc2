package screens

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"lcc2/internal/disk"
)

// A completion from a superseded scan must not clobber the newer
// scan's view; the fresh one still applies.
func TestDisksIgnoresStaleScanCompletion(t *testing.T) {
	d := NewDisks()
	d.w, d.h = 100, 30

	sc, _ := d.startScan("/slow-a")
	dA := sc.(Disks)
	genA := dA.scanGen.Load()

	sc, _ = dA.startScan("/fast-b")
	dB := sc.(Disks)

	sc, _ = dB.Update(scanDoneMsg{
		root: "/slow-a", res: &disk.ScanResult{Root: "/slow-a"}, gen: genA})
	d = sc.(Disks)
	if d.path == "/slow-a" || d.items != nil {
		t.Fatalf("stale completion applied: path=%q", d.path)
	}
	if !d.busy {
		t.Fatal("stale completion cleared the live scan's busy state")
	}

	resB := &disk.ScanResult{Root: "/fast-b"}
	sc, _ = d.Update(scanDoneMsg{root: "/fast-b", res: resB, gen: d.scanGen.Load()})
	d = sc.(Disks)
	if d.items != resB || d.busy {
		t.Fatal("fresh completion dropped or busy stuck")
	}
}

// Enter during an in-flight scan must not start an overlapping drill-down.
func TestDisksEnterIgnoredWhileBusy(t *testing.T) {
	d := NewDisks()
	sc, _ := d.startScan("/busy")
	d = sc.(Disks)

	sc, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	d2 := sc.(Disks)
	if len(d2.stack) != 0 || d2.scanGen.Load() != d.scanGen.Load() {
		t.Fatal("enter drilled down mid-scan")
	}
}
