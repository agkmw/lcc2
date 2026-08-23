package screens

import (
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
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
