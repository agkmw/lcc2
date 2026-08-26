package sysinfo

import (
	"testing"
	"time"

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

// Partition children report the same bytes as their parent device;
// summing both inflates rates roughly 2x on any partitioned drive.
func TestSumAllSkipsPartitions(t *testing.T) {
	counters := map[string]disk.IOCountersStat{
		"sda":       {Name: "sda", ReadBytes: 100, WriteBytes: 10},
		"sda1":      {Name: "sda1", ReadBytes: 90, WriteBytes: 9},
		"nvme0n1":   {Name: "nvme0n1", ReadBytes: 50, WriteBytes: 5},
		"nvme0n1p1": {Name: "nvme0n1p1", ReadBytes: 40, WriteBytes: 4},
		"mmcblk0p2": {Name: "mmcblk0p2", ReadBytes: 30, WriteBytes: 3},
		"loop0":     {Name: "loop0", ReadBytes: 999, WriteBytes: 999},
	}
	r, w := sumAll(counters)
	if r != 150 || w != 15 {
		t.Fatalf("r=%d w=%d, want 150/15 (whole devices only)", r, w)
	}
}
