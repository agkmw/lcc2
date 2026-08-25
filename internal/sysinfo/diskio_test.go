package sysinfo

import (
	"time"
	"testing"

	"github.com/shirou/gopsutil/v3/disk"
)

// sumAll aggregates while skipping loop/ram noise devices.
func TestSumAllSkipsLoopDevices(t *testing.T) {
	counters := map[string]disk.IOCountersStat{
		"sda":   {Name: "sda", ReadBytes: 100, WriteBytes: 50},
		"loop0": {Name: "loop0", ReadBytes: 999, WriteBytes: 999},
		"ram3":  {Name: "ram3", ReadBytes: 500, WriteBytes: 500},
	}
	r, w := sumAll(counters)
	if r != 100 || w != 50 {
		t.Fatalf("r=%d w=%d, want 100/50", r, w)
	}
}

// A counter reset between samples yields totals only (no negative
// rates), mirroring NetMonitor's contract.
func TestRatesCounterReset(t *testing.T) {
	m := &IOMonitor{}
	m.Rates() // seed
	m.t = m.t.Add(-time.Second)
	m.last = map[string]disk.IOCountersStat{
		"sda": {Name: "sda", ReadBytes: 10_000_000, WriteBytes: 5_000_000},
	}
	out := m.Rates()
	if out.ReadPerSec < 0 || out.WritePerSec < 0 {
		t.Fatalf("negative rates on reset: %+v", out)
	}
}
