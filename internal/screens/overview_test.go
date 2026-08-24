package screens

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"lcc2/internal/disk"
	"lcc2/internal/proc"
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

// Net charts must share the cpu chart's exact geometry: full interior
// width, no label gutter — direction labels live on their own lines.
func TestNetBoxAlignedLikeCpu(t *testing.T) {
	o := overviewFixture()
	for i := 0; i < 40; i++ {
		o.observe(snapshot{net: sysinfo.NetRates{
			RecvPerSec: float64(i) * (1 << 15), SentPerSec: float64(i) * (1 << 14),
		}})
	}
	box := o.netBox(80, 3)
	lines := strings.Split(box, "\n")
	if len(lines) != 3*2+2+1+2 { // netH*2 charts + 2 labels + totals + borders
		t.Fatalf("net box has %d lines", len(lines))
	}
	labels, charts := 0, 0
	for _, l := range lines {
		s := stripANSI(l)
		switch {
		case strings.HasPrefix(s, "┌") || strings.HasPrefix(s, "└"):
			continue
		case strings.Contains(s, "total"):
			continue
		case strings.HasPrefix(s, "│ down") || strings.HasPrefix(s, "│ up"):
			labels++
			continue
		}
		charts++
		// Between the side borders: pad + full-width chart (76) + pad.
		body := strings.TrimSuffix(strings.TrimPrefix(s, "│"), "│")
		if lipgloss.Width(body) != 78 {
			t.Errorf("chart row interior %d cells, want 78", lipgloss.Width(body))
		}
	}
	if labels != 2 || charts != 6 {
		t.Errorf("labels=%d charts=%d, want 2 and 6", labels, charts)
	}
	// The rx chart reaches the right edge: rising history ends non-zero
	// in the final plot column.
	rxLast := strings.TrimSuffix(strings.TrimPrefix(stripANSI(lines[4]), "│"), "│")
	if rxLast[76] == ' ' {
		t.Errorf("rx chart does not span the full interior: %q", rxLast)
	}
}

// Wide terminals earn the density extras: available-memory stat in the
// mem rows and the procs/threads/running strip in the cpu box. Narrow
// layouts keep the compact form.
func TestOverviewWideExtrasGate(t *testing.T) {
	o := overviewFixture()
	o.snap.host.Platform = "ubuntu"
	o.snap.host.PlatformVersion = "24.04"
	o.snap.host.KernelVersion = "6.8.0-49-generic"
	o.snap.counts = proc.Counts{Processes: 412, Threads: 1834, Running: 3}

	wide := feed(o, ui.SizeMsg{Width: 150, Height: 40}).(Overview).View()
	if !strings.Contains(wide, "avail ") || !strings.Contains(wide, "procs") ||
		!strings.Contains(wide, "kernel 6.8.0-49-generic") {
		t.Error("wide layout missing extras")
	}

	narrow := feed(overviewFixture(), ui.SizeMsg{Width: 70, Height: 26}).(Overview)
	narrow.snap = o.snap // same data, small terminal
	nv := narrow.View()
	if strings.Contains(nv, "avail ") || strings.Contains(nv, "procs 412") ||
		strings.Contains(nv, "kernel") {
		t.Error("narrow layout must stay compact")
	}
}
