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

// Per-core history rings track core count changes; rings prefill with
// zeros so charts span full width immediately.
func TestOverviewCoreRings(t *testing.T) {
	o := NewOverview()
	o.observe(snapshot{cpu: sysinfo.CPUSample{Cores: 2, PerCore: []float64{1, 2}}})
	o.observe(snapshot{cpu: sysinfo.CPUSample{Cores: 2, PerCore: []float64{3, 4}}})
	if len(o.coreHist) != 2 {
		t.Fatalf("rings = %d x %d", len(o.coreHist), len(o.coreHist[0]))
	}
	r0 := o.coreHist[0]
	if r0[len(r0)-2] != 1 || r0[len(r0)-1] != 3 {
		t.Fatalf("core0 ring tail = %v, want [1 3]", r0[len(r0)-2:])
	}
	r1 := o.coreHist[1]
	if r1[len(r1)-2] != 2 || r1[len(r1)-1] != 4 {
		t.Fatalf("core1 ring tail = %v, want [2 4]", r1[len(r1)-2:])
	}
}

// The net auto-scale uses a rolling window with hysteresis: one burst
// must not pin the scale forever after (backlog L8); the basis decays
// only after a long quiet stretch and never below the 64 KiB floor.
func TestOverviewNetPeakDecays(t *testing.T) {
	o := NewOverview()
	o.observe(snapshot{net: sysinfo.NetRates{RecvPerSec: 8 << 20}})
	for i := 0; i < peakWin+10; i++ {
		o.observe(snapshot{net: sysinfo.NetRates{RecvPerSec: 1 << 10}})
	}
	if o.netPeak != minScale {
		t.Fatalf("netPeak = %v after quiet window, want the %v floor", o.netPeak, minScale)
	}
}

// Hysteresis: ordinary fluctuations must not shrink the scale basis.
func TestOverviewNetPeakHysteresis(t *testing.T) {
	o := NewOverview()
	o.observe(snapshot{net: sysinfo.NetRates{RecvPerSec: 4 << 20}})
	for i := 0; i < 30; i++ { // mild noise around half the peak
		o.observe(snapshot{net: sysinfo.NetRates{RecvPerSec: 3 << 20}})
	}
	if o.netPeak < 3<<20 {
		t.Fatalf("basis shrank under mild load: %v", o.netPeak)
	}
}

// Net graphs plot recorded history, not just the latest sample.
func TestOverviewNetHistoryPlotted(t *testing.T) {
	o := NewOverview()
	for i := 0; i < histMax; i++ {
		v := float64(i) / histMax * 100
		o.observe(snapshot{cpu: sysinfo.CPUSample{Cores: 1},
			net: sysinfo.NetRates{RecvPerSec: v * minScale}})
	}
	if len(o.rxHist) != histMax {
		t.Fatalf("rxHist = %d samples", len(o.rxHist))
	}
	plot := scaleHist(o.rxHist, snapScale(o.netPeak))
	nonZero := 0
	for _, v := range plot {
		if v > 1 {
			nonZero++
		}
	}
	if nonZero < histMax/2 {
		t.Fatalf("history plot mostly flat: %d/%d non-zero", nonZero, histMax)
	}
}
