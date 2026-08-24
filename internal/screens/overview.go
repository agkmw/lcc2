// Package screens contains the top-level section models of the app.
// Screens consume data from the pure provider packages and own all
// Bubble Tea state; providers never import the UI.
package screens

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lcc2/internal/disk"
	"lcc2/internal/sysinfo"
	"lcc2/internal/ui"
)

const overviewInterval = time.Second
const histMax = 120 // samples kept per ring (1 s apart)

type overviewTickMsg struct{ gen uint64 }

// snapshot bundles one round of measurements.
type snapshot struct {
	host sysinfo.Host
	load sysinfo.Load
	cpu  sysinfo.CPUSample
	mem  sysinfo.Memory
	net  sysinfo.NetRates
	fss  []disk.Filesystem
}

// collect gathers a full snapshot in one background command. CPU uses
// a zero interval — diff since the previous call — so the tick never
// blocks and 1 s really means 1 s.
func collect(mon *sysinfo.NetMonitor) tea.Cmd {
	return func() tea.Msg {
		s := snapshot{mem: sysinfo.ReadMemory(), load: sysinfo.ReadLoad()}
		if h, err := sysinfo.ReadHost(); err == nil {
			s.host = h
		}
		if c, err := sysinfo.SampleCPU(0); err == nil {
			s.cpu = c
		}
		s.net = mon.Rates()
		s.fss = disk.ListFilesystems()
		return s
	}
}

// Overview is the monitoring dashboard: btop-style stacked sections
// for cpu, memory, network and disks on one borderless canvas.
type Overview struct {
	w, h     int
	mon      *sysinfo.NetMonitor
	snap     snapshot
	cpuHist  []float64
	coreHist [][]float64
	rxHist   []float64
	txHist   []float64
	netPeak  float64 // bytes/s; monotonic auto-scale for the net graphs
	loaded   bool
	widthSet bool
	epoch    *atomic.Uint64 // tick-chain generation; stale chains die
}

// NewOverview builds the dashboard screen.
func NewOverview() Overview {
	return Overview{mon: &sysinfo.NetMonitor{}, epoch: &atomic.Uint64{}}
}

// ID implements ui.Screen.
func (o Overview) ID() string { return "overview" }

// Title implements ui.Screen.
func (o Overview) Title() string { return "Overview" }

// Hints implements ui.Screen.
func (o Overview) Hints() []key.Binding {
	return []key.Binding{ui.Keys.Refresh}
}

// CapturingInput implements ui.Screen.
func (o Overview) CapturingInput() bool { return false }

// Init starts the periodic refresh loop; re-entry retires the previous chain.
func (o Overview) Init() tea.Cmd {
	return o.tick(o.epoch.Add(1))
}

func (o Overview) tick(gen uint64) tea.Cmd {
	return tea.Batch(collect(o.mon), tea.Tick(overviewInterval, func(time.Time) tea.Msg {
		return overviewTickMsg{gen: gen}
	}))
}

// Update handles ticks and resize events.
func (o Overview) Update(msg tea.Msg) (ui.Screen, tea.Cmd) {
	switch m := msg.(type) {
	case ui.SizeMsg:
		o.w, o.h = m.Width, m.Height
		o.widthSet = true
	case snapshot:
		// Data messages only store state; rescheduling happens solely
		// in the tick handler so exactly one refresh loop stays alive.
		o.snap = m
		o.loaded = true
		o.observe(m)
	case overviewTickMsg:
		if m.gen != o.epoch.Load() {
			return o, nil // stale chain from a previous Init
		}
		return o, o.tick(m.gen)
	case tea.KeyMsg:
		if m.String() == "r" {
			o.loaded = false
			return o, o.tick(o.epoch.Load())
		}
	}
	return o, nil
}

// observe folds a snapshot into the history rings.
func (o *Overview) observe(s snapshot) {
	o.cpuHist = appendHist(o.cpuHist, s.cpu.Total)
	if len(o.coreHist) != s.cpu.Cores { // first sample or hotplug
		o.coreHist = make([][]float64, s.cpu.Cores)
	}
	for i, v := range s.cpu.PerCore {
		o.coreHist[i] = appendHist(o.coreHist[i], v)
	}
	o.rxHist = appendHist(o.rxHist, s.net.RecvPerSec)
	o.txHist = appendHist(o.txHist, s.net.SentPerSec)
	o.netPeak = maxFloat(o.netPeak, s.net.RecvPerSec, s.net.SentPerSec)
}

func appendHist(h []float64, v float64) []float64 {
	h = append(h, v)
	if len(h) > histMax {
		h = h[len(h)-histMax:]
	}
	return h
}

// View renders stacked full-width sections: cpu, mem, net, disk.
func (o Overview) View() string {
	if !o.loaded || !o.widthSet {
		return lipgloss.Place(o.w, o.h, lipgloss.Center, lipgloss.Center,
			ui.EmptyState("", "Gathering system information..", "", o.w))
	}
	w, h := o.w, o.h

	head := pageHead("Overview",
		o.snap.host.Hostname+" - up "+sysinfo.FormatUptime(o.snap.host.Uptime), w)

	// Budget: 13 fixed rows + graphH + cRows + 2*netH + fsRows == h.
	// Fixed: head, blank, cpu title, blank, mem title, ram, swap,
	// blank, net title, net totals, blank, disk title (+1 spare).
	cRows := o.coreRows(w)
	left := clampInt(h-14-cRows, 3, 1000)
	graphH := clampInt(left*2/5, 3, 10)
	rem := maxInt(left-graphH, 0)
	netH := clampInt(rem/4, 1, 6)
	fsRows := clampInt(rem-2*netH, 1, len(o.snap.fss))

	var b strings.Builder
	b.WriteString(head + "\n\n")
	b.WriteString(o.cpuSection(w, graphH, cRows))
	b.WriteString("\n\n")
	b.WriteString(o.memSection(w))
	b.WriteString("\n\n")
	b.WriteString(o.netSection(w, netH))
	b.WriteString("\n\n")
	b.WriteString(o.diskSection(w, fsRows))

	lines := strings.Split(b.String(), "\n")
	for len(lines) < h {
		lines = append(lines, "")
	}
	if len(lines) > h {
		lines = lines[:h]
	}
	return strings.Join(lines, "\n")
}

func pctOfScaled(v, scale float64) []float64 {
	p := clampF(v/scale*100, 0, 100)
	return []float64{p}
}

// secTitle renders an accent-colored section label with a faint
// detail string right-aligned within w.
func secTitle(id, label, right string, w int) string {
	l := lipgloss.NewStyle().Bold(true).Foreground(ui.Accent(id)).
		Render(label)
	gap := w - lipgloss.Width(l) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return ui.ClipBlock(l+" "+faintSty.Render(right), w)
}

func (o Overview) cpuSection(w, graphH, cRows int) string {
	c := o.snap.cpu
	right := fmt.Sprintf("total %s%% - load %s %s %s - %d cores",
		f1(c.Total), f1(o.snap.load.One), f1(o.snap.load.Five),
		f1(o.snap.load.Fifteen), c.Cores)
	var b strings.Builder
	b.WriteString(secTitle("overview", "cpu", right, w) + "\n")
	b.WriteString(ui.Graph(o.cpuHist, w, graphH, ui.Palette.Blue))
	b.WriteString("\n")
	b.WriteString(strings.TrimRight(o.coreGrid(w), "\n"))
	return b.String()
}

// coreGrid renders per-core sparkline cells, as many columns as fit.
func (o Overview) coreGrid(w int) string {
	shown := min(len(o.snap.cpu.PerCore), 32)
	if shown == 0 {
		return ""
	}
	const cellW = 17
	perRow := maxInt(w/cellW, 1)
	var out []string
	for start := 0; start < shown; start += perRow {
		end := min(start+perRow, shown)
		cells := make([]string, 0, end-start)
		for i := start; i < end; i++ {
			v := o.snap.cpu.PerCore[i]
			lbl := "c" + itoa(i)
			for lipgloss.Width(lbl) < 4 {
				lbl += " "
			}
			pct := padLeft(itoa(int(v+0.5)), 3) + "%"
			cells = append(cells, lbl+
				ui.Spark(o.coreHist[i], 8, ui.StateColor(v))+
				mutedSty.Render(pct))
		}
		out = append(out, strings.Join(cells, " "))
	}
	return strings.Join(out, "\n") + "\n"
}

// coreRows reports how many lines coreGrid emits at width w.
func (o Overview) coreRows(w int) int {
	shown := min(len(o.snap.cpu.PerCore), 32)
	if shown == 0 {
		return 0
	}
	perRow := maxInt(w/17, 1)
	return (shown + perRow - 1) / perRow
}

func (o Overview) memSection(w int) string {
	m := o.snap.mem
	cachePct := 0.0
	if m.Total > 0 {
		cachePct = float64(m.Cached) / float64(m.Total) * 100
	}
	barW := clampInt(w/2-8, 20, 60)

	ramBar := ui.SegGauge(m.UsedPercent, cachePct, barW,
		ui.StateColor(m.UsedPercent), ui.Palette.Mauve) +
		mutedSty.Render(" "+padLeft(itoa(int(m.UsedPercent+0.5)), 3)+"%")
	ramStats := "used " + sysinfo.FormatBytes(float64(m.Used)) +
		" - cache " + sysinfo.FormatBytes(float64(m.Cached)) +
		" - free " + sysinfo.FormatBytes(float64(m.Total)-float64(m.Used))

	line := memLine("ram ", ramBar,
		sysinfo.FormatBytes(float64(m.Used))+" / "+sysinfo.FormatBytes(float64(m.Total)),
		ramStats, w)
	var b strings.Builder
	b.WriteString(secTitle("disk", "mem", "total "+
		sysinfo.FormatBytes(float64(m.Total)), w) + "\n")
	b.WriteString(line)
	if m.SwapTotal > 0 {
		swapBar := ui.Gauge(m.SwapPercent, barW, nil)
		b.WriteString("\n" + memLine("swap", swapBar,
			sysinfo.FormatBytes(float64(m.SwapUsed))+" / "+
				sysinfo.FormatBytes(float64(m.SwapTotal)), "", w))
	} else {
		b.WriteString("\n" + mutedSty.Render("swap none"))
	}
	return b.String()
}

// memLine places label + bar + usage left and stats right when both fit.
func memLine(label, bar, usage, stats string, w int) string {
	line := mutedSty.Render(label+" ") + bar + "  " + faintSty.Render(usage)
	if stats == "" || lipgloss.Width(line)+2+lipgloss.Width(stats) > w {
		return line
	}
	gap := w - lipgloss.Width(line) - lipgloss.Width(stats)
	return line + strings.Repeat(" ", maxInt(gap, 2)) + faintSty.Render(stats)
}

func (o Overview) netSection(w, graphH int) string {
	n := o.snap.net
	scale := maxFloat(o.netPeak, 64<<10) // floor: 64 KiB/s
	rx := pctOfScaled(n.RecvPerSec, scale)
	tx := pctOfScaled(n.SentPerSec, scale)
	right := fmt.Sprintf("↓ %s - ↑ %s - scale %s",
		sysinfo.FormatRate(n.RecvPerSec), sysinfo.FormatRate(n.SentPerSec),
		sysinfo.FormatRate(scale))
	var b strings.Builder
	b.WriteString(secTitle("services", "net", right, w) + "\n")
	b.WriteString(ui.Graph(rx, w, graphH, ui.Palette.Teal) + "\n")
	b.WriteString(ui.Graph(tx, w, graphH, ui.Palette.Peach) + "\n")
	totals := fmt.Sprintf("↓ total %s   ↑ total %s",
		sysinfo.FormatBytes(float64(n.RecvTotal)),
		sysinfo.FormatBytes(float64(n.SentTotal)))
	b.WriteString(faintSty.Render(totals))
	return b.String()
}

func (o Overview) diskSection(w, rows int) string {
	fss := append([]disk.Filesystem(nil), o.snap.fss...)
	sort.Slice(fss, func(i, j int) bool { return fss[i].Total > fss[j].Total })
	var b strings.Builder
	b.WriteString(secTitle("proc", "disk", itoa(len(fss))+" mounts", w))
	barW := clampInt(w/2-12, 16, 48)
	shown := min(rows, len(fss))
	for i := 0; i < shown; i++ {
		f := fss[i]
		bar := ui.Gauge(f.UsedPercent, barW, nil)
		usage := sysinfo.FormatBytes(float64(f.Used)) + " / " +
			sysinfo.FormatBytes(float64(f.Total))
		line := mutedSty.Render(padTo(f.Mountpoint, 14)) + bar + "  " +
			faintSty.Render(usage)
		b.WriteString("\n" + ui.Truncate(line, w))
	}
	if len(fss) > shown {
		b.WriteString("\n" + faintSty.Render(
			"  .. +" + itoa(len(fss)-shown) + " more - see Disks"))
	}
	return b.String()
}

func padTo(s string, w int) string {
	for lipgloss.Width(s) < w {
		s += " "
	}
	return s
}

func padLeft(s string, w int) string {
	for len(s) < w {
		s = " " + s
	}
	return s
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func f1(v float64) string {
	return strconv.FormatFloat(v, 'f', 1, 64)
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func maxFloat(vals ...float64) float64 {
	m := vals[0]
	for _, v := range vals {
		if v > m {
			m = v
		}
	}
	return m
}
