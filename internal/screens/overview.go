// Package screens contains the top-level section models of the app.
// Screens consume data from the pure provider packages and own all
// Bubble Tea state; providers never import the UI.
package screens

import (
	"fmt"
	"os"
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
const histMax = 120   // samples kept per ring (1 s apart)
const peakWin = 60    // samples feeding the net auto-scale (rolling, L8)
const minScale = 64 << 10 // net auto-scale floor, bytes/s

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
	w, h       int
	mon        *sysinfo.NetMonitor
	snap       snapshot
	cpuHist    []float64
	coreHist   [][]float64
	rxHist     []float64
	txHist     []float64
	netWin     []float64 // recent rx/tx maxima; the auto-scale source
	netPeak    float64   // bytes/s; max of netWin (rolling window)
	graphStyle string    // "braille" or "block"; g toggles, LCC2_GRAPH seeds
	loaded     bool
	widthSet   bool
	epoch      *atomic.Uint64 // tick-chain generation; stale chains die
}

// NewOverview builds the dashboard screen.
func NewOverview() Overview {
	style := os.Getenv("LCC2_GRAPH")
	if style != "block" {
		style = "braille" // default and anything unknown
	}
	return Overview{
		mon: &sysinfo.NetMonitor{}, graphStyle: style,
		epoch: &atomic.Uint64{},
	}
}

// ID implements ui.Screen.
func (o Overview) ID() string { return "overview" }

// Title implements ui.Screen.
func (o Overview) Title() string { return "Overview" }

// Hints implements ui.Screen.
func (o Overview) Hints() []key.Binding {
	return []key.Binding{
		ui.Keys.Refresh,
		key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "graph: "+o.graphStyle)),
	}
}

// CapturingInput implements ui.Screen.
func (o Overview) CapturingInput() bool { return false }

// chart renders a time-series with the active style.
func (o Overview) chart(hist []float64, w, h int, c lipgloss.Color) string {
	if o.graphStyle == "block" {
		return ui.Graph(hist, w, h, c)
	}
	return ui.GraphBraille(hist, w, h, c)
}

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
		if m.String() == "g" {
			if o.graphStyle == "braille" {
				o.graphStyle = "block"
			} else {
				o.graphStyle = "braille"
			}
		}
	}
	return o, nil
}

// observe folds a snapshot into the history rings. The first sample
// back-fills every ring with zeros so the charts span their full
// width immediately (btop-style flat start) instead of growing from
// the right edge.
func (o *Overview) observe(s snapshot) {
	first := len(o.cpuHist) == 0
	if len(o.coreHist) != s.cpu.Cores { // first sample or hotplug
		o.coreHist = make([][]float64, s.cpu.Cores)
		for i := range o.coreHist {
			if first {
				o.coreHist[i] = zeroHist()
			}
		}
	}
	if first {
		o.cpuHist = zeroHist()
		o.rxHist = zeroHist()
		o.txHist = zeroHist()
	}
	o.cpuHist = appendHist(o.cpuHist, s.cpu.Total)
	for i, v := range s.cpu.PerCore {
		o.coreHist[i] = appendHist(o.coreHist[i], v)
	}
	o.rxHist = appendHist(o.rxHist, s.net.RecvPerSec)
	o.txHist = appendHist(o.txHist, s.net.SentPerSec)

	peak := maxFloat(s.net.RecvPerSec, s.net.SentPerSec)
	o.netWin = append(o.netWin, peak)
	if len(o.netWin) > peakWin {
		o.netWin = o.netWin[len(o.netWin)-peakWin:]
	}
	// Hysteresis: grow instantly with the rolling peak, but step down
	// only after it falls below a quarter of the current basis — the
	// graph must not shrink on every ordinary fluctuation (user report:
	// "when the speed changes the graph shrinks, hard to view").
	windowed := maxFloat(o.netWin...)
	switch {
	case windowed >= o.netPeak:
		o.netPeak = windowed
	case o.netPeak > minScale && windowed < o.netPeak/4:
		o.netPeak = maxFloat(windowed, minScale)
	}
}

func zeroHist() []float64 { return make([]float64, histMax) }

func appendHist(h []float64, v float64) []float64 {
	h = append(h, v)
	if len(h) > histMax {
		h = h[len(h)-histMax:]
	}
	return h
}

// View renders stacked full-width btop-style boxes: cpu, mem, net,
// disk. Heights are budgeted so the stack lands exactly on h.
func (o Overview) View() string {
	if !o.loaded || !o.widthSet {
		return lipgloss.Place(o.w, o.h, lipgloss.Center, lipgloss.Center,
			ui.EmptyState("", "Gathering system information..", "", o.w))
	}
	w, h := o.w, o.h

	head := pageHead("Overview",
		o.snap.host.Hostname+" - up "+sysinfo.FormatUptime(o.snap.host.Uptime), w)

	// Budget: head(1) + borders(8) + ram+swap(2) + net totals(1) plus
	// graphs and rows must land exactly on h.
	cRows := o.coreRows(w - 4)
	rem := h - 12 - cRows
	graphH := clampInt(rem*2/5, 3, 12)
	rest := maxInt(rem-graphH, 0)
	netH := clampInt(rest/3, 1, 6)
	fsRows := clampInt(rest-2*netH, 1, len(o.snap.fss))

	var b strings.Builder
	b.WriteString(head + "\n")
	b.WriteString(o.cpuBox(w, graphH, cRows) + "\n")
	b.WriteString(o.memBox(w) + "\n")
	b.WriteString(o.netBox(w, netH) + "\n")
	b.WriteString(o.diskBox(w, fsRows))

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

// scaleHist normalizes a byte-rate history against the current scale.
func scaleHist(hist []float64, scale float64) []float64 {
	out := make([]float64, len(hist))
	for i, v := range hist {
		out[i] = clampF(v/scale*100, 0, 100)
	}
	return out
}

func (o Overview) cpuBox(w, graphH, cRows int) string {
	c := o.snap.cpu
	right := fmt.Sprintf("%s%% - load %s %s %s - %d cores",
		f1(c.Total), f1(o.snap.load.One), f1(o.snap.load.Five),
		f1(o.snap.load.Fifteen), c.Cores)
	iw := w - 4
	body := o.chart(o.cpuHist, iw, graphH, ui.Palette.Blue)
	if grid := strings.TrimRight(o.coreGrid(iw), "\n"); grid != "" {
		body += "\n" + grid
	}
	return ui.Section("overview", "cpu", right, w, graphH+cRows+2, body)
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
			if i >= len(o.coreHist) {
				break // ring not seeded yet (hotplug race); render the rest
			}
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
	return strings.Join(out, "\n")
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

func (o Overview) memBox(w int) string {
	m := o.snap.mem
	cachePct := 0.0
	if m.Total > 0 {
		cachePct = float64(m.Cached) / float64(m.Total) * 100
	}
	barW := clampInt((w-4)/2-8, 20, 60)

	ramBar := ui.SegGauge(m.UsedPercent, cachePct, barW,
		ui.StateColor(m.UsedPercent), ui.Palette.Mauve) +
		mutedSty.Render(" "+padLeft(itoa(int(m.UsedPercent+0.5)), 3)+"%")
	ramStats := "used " + sysinfo.FormatBytes(float64(m.Used)) +
		" - cache " + sysinfo.FormatBytes(float64(m.Cached)) +
		" - free " + sysinfo.FormatBytes(float64(m.Total)-float64(m.Used))

	line := memLine("ram ", ramBar,
		sysinfo.FormatBytes(float64(m.Used))+" / "+sysinfo.FormatBytes(float64(m.Total)),
		ramStats, w-4)
	if m.SwapTotal > 0 {
		swapBar := ui.Gauge(m.SwapPercent, barW, nil)
		line += "\n" + memLine("swap", swapBar,
			sysinfo.FormatBytes(float64(m.SwapUsed))+" / "+
				sysinfo.FormatBytes(float64(m.SwapTotal)), "", w-4)
	} else {
		line += "\n" + mutedSty.Render("swap none")
	}
	return ui.Section("disk", "mem", "total "+
		sysinfo.FormatBytes(float64(m.Total)), w, 4, line)
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

func (o Overview) netBox(w, netH int) string {
	n := o.snap.net
	scale := snapScale(o.netPeak)

	iw := w - 4
	const labelW = 6 // "down " / "up   "
	rxRate := faintSty.Render(sysinfo.FormatRate(n.RecvPerSec))
	txRate := faintSty.Render(sysinfo.FormatRate(n.SentPerSec))
	suffix := lipgloss.Width(rxRate)
	if s := lipgloss.Width(txRate); s > suffix {
		suffix = s
	}
	gw := clampInt(iw-labelW-2-suffix, 10, iw)

	// Plot the recorded history (absolute bytes/s), normalized against
	// the current scale — the whole line breathes together instead of
	// a lone point per tick.
	rxRows := strings.Split(o.chart(scaleHist(o.rxHist, scale), gw, netH, ui.Palette.Teal), "\n")
	txRows := strings.Split(o.chart(scaleHist(o.txHist, scale), gw, netH, ui.Palette.Peach), "\n")
	downLbl := mutedSty.Render(padTo("down", labelW-1))
	upLbl := mutedSty.Render(padTo("up", labelW-1))

	var body []string
	for i := 0; i < netH; i++ {
		if i == 0 {
			body = append(body, downLbl+rxRows[i]+"  "+rxRate)
		} else {
			body = append(body, strings.Repeat(" ", labelW)+rxRows[i])
		}
	}
	for i := 0; i < netH; i++ {
		if i == 0 {
			body = append(body, upLbl+txRows[i]+"  "+txRate)
		} else {
			body = append(body, strings.Repeat(" ", labelW)+txRows[i])
		}
	}
	body = append(body, faintSty.Render(fmt.Sprintf("down total %s   up total %s",
		sysinfo.FormatBytes(float64(n.RecvTotal)),
		sysinfo.FormatBytes(float64(n.SentTotal)))))
	return ui.Section("services", "net", "scale "+sysinfo.FormatRate(scale),
		w, len(body)+2, strings.Join(body, "\n"))
}

// snapScale snaps the auto-scale basis upward to the next power of
// two so the axis label moves in recognizable steps; minScale floors
// it at 64 KiB/s.
func snapScale(peak float64) float64 {
	if peak < minScale {
		return minScale
	}
	p := float64(minScale)
	for p < peak {
		p *= 2
	}
	return p
}

func (o Overview) diskBox(w, rows int) string {
	fss := append([]disk.Filesystem(nil), o.snap.fss...)
	sort.Slice(fss, func(i, j int) bool { return fss[i].Total > fss[j].Total })
	iw := w - 4
	shown := min(rows, len(fss))
	barW := clampInt(iw/2-16, 16, 44)
	mountW := 8
	for i := 0; i < shown; i++ {
		if n := lipgloss.Width(fss[i].Mountpoint); n+1 > mountW && n < iw/3 {
			mountW = n + 1
		}
	}
	var lines []string
	for i := 0; i < shown; i++ {
		f := fss[i]
		bar := ui.Gauge(f.UsedPercent, barW, nil)
		usage := sysinfo.FormatBytes(float64(f.Used)) + " / " +
			sysinfo.FormatBytes(float64(f.Total))
		left := mutedSty.Render(padTo(f.Mountpoint, mountW)) + bar
		gap := iw - lipgloss.Width(left) - len(usage) - 1
		if gap >= 1 {
			lines = append(lines, left+strings.Repeat(" ", gap)+
				faintSty.Render(usage))
		} else {
			lines = append(lines, left+"  "+faintSty.Render(usage))
		}
	}
	if len(fss) > shown {
		lines = append(lines, faintSty.Render(
			".. +" + itoa(len(fss)-shown) + " more - see Disks"))
	}
	if len(lines) == 0 {
		lines = append(lines, mutedSty.Render("no mounts"))
	}
	return ui.Section("proc", "disk", itoa(len(fss))+" mounts",
		w, len(lines)+2, strings.Join(lines, "\n"))
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
