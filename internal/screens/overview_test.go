package screens

import (
	"strings"
	"testing"

	"lcc2/internal/disk"
	"lcc2/internal/sysinfo"
	"lcc2/internal/ui"
)

func overviewFixture() Overview {
	o := NewOverview()
	o = feed(o, ui.SizeMsg{Width: 96, Height: 26}, snapshot{
		cpu: sysinfo.CPUSample{Cores: 2, PerCore: []float64{10, 90}, Total: 50},
		mem: sysinfo.Memory{Total: 8 << 30, Used: 4 << 30,
			Cached: 1 << 30, UsedPercent: 50},
		load: sysinfo.Load{One: 1.5},
		net: sysinfo.NetRates{RecvPerSec: 1024, SentPerSec: 2048,
			RecvTotal: 1 << 30, SentTotal: 2 << 30},
		fss: []disk.Filesystem{
			{Mountpoint: "/", Total: 100 << 30, Used: 50 << 30, UsedPercent: 50},
		},
	}).(Overview)
	return o
}

// The dashboard must show every btop-style section header.
func TestOverviewSectionsPresent(t *testing.T) {
	v := overviewFixture().View()
	for _, want := range []string{"cpu", "mem", "net", "disk", "load"} {
		if !strings.Contains(v, want) {
			t.Errorf("overview missing %q section", want)
		}
	}
}

// Net graph auto-scale: after a burst, the scale label reflects the
// peak instead of a fixed 1 MiB/s ceiling (backlog L5).
func TestOverviewNetPeakScale(t *testing.T) {
	o := NewOverview()
	o.observe(snapshot{net: sysinfo.NetRates{RecvPerSec: 8 << 20}})
	o.observe(snapshot{net: sysinfo.NetRates{SentPerSec: 3 << 20}})
	if o.netPeak != 8<<20 {
		t.Fatalf("netPeak = %v, want 8 MiB", o.netPeak)
	}
}

// Per-core history rings track core count changes.
func TestOverviewCoreRings(t *testing.T) {
	o := NewOverview()
	o.observe(snapshot{cpu: sysinfo.CPUSample{Cores: 2, PerCore: []float64{1, 2}}})
	o.observe(snapshot{cpu: sysinfo.CPUSample{Cores: 2, PerCore: []float64{3, 4}}})
	if len(o.coreHist) != 2 || len(o.coreHist[0]) != 2 {
		t.Fatalf("rings = %d x %d", len(o.coreHist), len(o.coreHist[0]))
	}
}
