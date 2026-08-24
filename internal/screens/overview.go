// Package screens contains the top-level section models of the app.
// Screens consume data from the pure provider packages and own all
// Bubble Tea state; providers never import the UI.
package screens

import (
	"fmt"
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

type overviewTickMsg struct{ gen uint64 }

// snapshot bundles one round of measurements.
type snapshot struct {
	host sysinfo.Host
	load sysinfo.Load
	cpu  sysinfo.CPUSample
	mem  sysinfo.Memory
	net  sysinfo.NetRates
	root *disk.Filesystem
}

// collect gathers a full snapshot in one background command.
func collect(mon *sysinfo.NetMonitor) tea.Cmd {
	return func() tea.Msg {
		s := snapshot{mem: sysinfo.ReadMemory(), load: sysinfo.ReadLoad()}
		if h, err := sysinfo.ReadHost(); err == nil {
			s.host = h
		}
		if c, err := sysinfo.SampleCPU(2 * time.Second); err == nil {
			s.cpu = c
		}
		s.net = mon.Rates()
		if f, ok := disk.RootUsage(); ok {
			s.root = &f
		}
		return s
	}
}

// Overview is the monitoring dashboard.
type Overview struct {
	w, h     int
	mon      *sysinfo.NetMonitor
	snap     snapshot
	cpuHist  []float64
	rxHist   []float64
	txHist   []float64
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
		o.cpuHist = appendHist(o.cpuHist, m.cpu.Total)
		o.rxHist = appendHist(o.rxHist, pctOf(m.net.RecvPerSec, 1<<20)) // scale 1 MiB/s
		o.txHist = appendHist(o.txHist, pctOf(m.net.SentPerSec, 1<<20))
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

func appendHist(h []float64, v float64) []float64 {
	const maxLen = 60
	h = append(h, v)
	if len(h) > maxLen {
		h = h[len(h)-maxLen:]
	}
	return h
}

func pctOf(v, scale float64) float64 {
	p := v / scale * 100
	if p > 100 {
		p = 100
	}
	return p
}

// View renders the dashboard: a 2×2 card grid plus a full-width disk
// card, all equal heights per row.
func (o Overview) View() string {
	if !o.loaded || !o.widthSet {
		return o.center(ui.EmptyState("", "Gathering system information…", "", o.w))
	}
	w, h := o.w, o.h
	meta := fmt.Sprintf("load %.1f  ·  %s",
		o.snap.load.One, sysinfo.FormatUptime(o.snap.host.Uptime))
	head := pageHead("Overview", meta, w)

	gutter := 2
	cardW := (w - 4 - gutter) / 2 // two bordered cards + gutter column
	diskH := 3
	gridH := h - 1 - diskH - 2 // head + blank + gaps around grid
	rh := clampInt(gridH/2-1, 4, 20)

	topL := o.card("system", "system", kvBody(o.systemRows(), rh-6), cardW, rh)
	topR := o.card("cpu", "cpu", o.cpuBody(cardW, rh), cardW, rh)
	botL := o.card("memory", "memory", o.memBody(cardW), cardW, rh)
	botR := o.card("network", "network", o.netBody(cardW), cardW, rh)

	colW := cardW + 2 // outer card width incl. border
	blank := strings.Repeat(" ", maxInt(w, 1))
	var body string
	if w >= 72 {
		top := ui.Split(topL, topR, colW, w)
		bot := ui.Split(botL, botR, colW, w)
		body = top + "\n" + blank + "\n" + bot
	} else {
		body = lipgloss.JoinVertical(lipgloss.Left,
			topL, strings.Repeat(" ", colW), topR,
			strings.Repeat(" ", colW), botL,
			strings.Repeat(" ", colW), botR)
	}

	disk := o.diskCard(w - 2)
	out := head + "\n" + body + "\n" + disk
	lines := strings.Split(out, "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	return strings.Join(lines, "\n")
}

func (o Overview) card(id, title, body string, w, h int) string {
	return ui.Card(id, title, body, w, h)
}

func (o Overview) diskCard(w int) string {
	f := o.snap.root
	if f == nil {
		return ui.Card("disk", "disk", mutedSty.Render("no root filesystem"), w, 3)
	}
	bar := ui.Gauge(f.UsedPercent, clampInt(w/3, 16, 44), nil)
	body := bar + "  " + faintSty.Render(sysinfo.FormatBytes(float64(f.Used))+
		" / "+sysinfo.FormatBytes(float64(f.Total))+" on "+f.Mountpoint)
	return ui.Card("disk", "disk "+f.Mountpoint, body, w, 3)
}

func (o Overview) center(s string) string {
	return lipgloss.Place(o.w, o.h, lipgloss.Center, lipgloss.Center, s)
}

func (o Overview) systemRows() [][2]string {
	h := o.snap.host
	return [][2]string{
		{"host", h.Hostname},
		{"os", titleCase(h.Platform) + " " + h.PlatformVersion},
		{"kernel", h.KernelVersion + " (" + h.KernelArch + ")"},
		{"uptime", sysinfo.FormatUptime(h.Uptime)},
		{"cores", itoa(o.snap.cpu.Cores)},
		{"load", f1(o.snap.load.One) + "  " + f1(o.snap.load.Five) + "  " + f1(o.snap.load.Fifteen)},
	}
}

func kvBody(rows [][2]string, maxRows int) string {
	if maxRows < 1 {
		maxRows = 1
	}
	if len(rows) > maxRows {
		rows = rows[:maxRows]
	}
	var b strings.Builder
	for _, r := range rows {
		b.WriteString(mutedSty.Render(padTo(r[0], 7)) + faintSty.Render(ui.Truncate(r[1], 40)) + "\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func (o Overview) memBody(w int) string {
	m := o.snap.mem
	var b strings.Builder
	b.WriteString(row("ram ", ui.Gauge(m.UsedPercent, gaugeWidth(w), nil),
		sysinfo.FormatBytes(float64(m.Used))+" / "+sysinfo.FormatBytes(float64(m.Total))))
	b.WriteString("\n")
	b.WriteString(row("swap", ui.Gauge(m.SwapPercent, gaugeWidth(w), nil),
		sysinfo.FormatBytes(float64(m.SwapUsed))+" / "+sysinfo.FormatBytes(float64(m.SwapTotal))))
	return b.String()
}

func (o Overview) cpuBody(w, h int) string {
	c := o.snap.cpu
	var b strings.Builder
	b.WriteString(ui.Spark(o.cpuHist, sparkWidth(w), ui.Palette.Blue) + "\n\n")
	maxShow := clampInt(h-6, 2, 12)
	perRow := 2
	shown := min(c.Cores, maxShow)
	for i := 0; i < shown; i++ {
		b.WriteString(lipgloss.NewStyle().Width(4).Render("c"+itoa(i)) +
			ui.Gauge(c.PerCore[i], gaugeWidth(w)/2, nil))
		if (i+1)%perRow == 0 || i == shown-1 {
			b.WriteString("\n")
		} else {
			b.WriteString("  ")
		}
	}
	if c.Cores > maxShow {
		b.WriteString(faintSty.Render("… +" + itoa(c.Cores-maxShow) + " more cores"))
	}
	return strings.TrimRight(b.String(), "\n")
}

func (o Overview) netBody(w int) string {
	n := o.snap.net
	var b strings.Builder
	b.WriteString(row("rx ", ui.Spark(o.rxHist, sparkWidth(w), ui.Palette.Teal),
		sysinfo.FormatRate(n.RecvPerSec)))
	b.WriteString("\n")
	b.WriteString(row("tx ", ui.Spark(o.txHist, sparkWidth(w), ui.Palette.Peach),
		sysinfo.FormatRate(n.SentPerSec)))
	b.WriteString("\n\n")
	b.WriteString(faintSty.Render("total  ↓ " + sysinfo.FormatBytes(float64(n.RecvTotal)) +
		"   ↑ " + sysinfo.FormatBytes(float64(n.SentTotal))))
	return b.String()
}

func gaugeWidth(w int) int { return clampInt(w/2-10, 10, 40) }
func sparkWidth(w int) int { return clampInt(w-14, 10, 80) }

func row(label string, bar string, right string) string {
	return mutedSty.Render(label+" ") + bar + "  " + faintSty.Render(right)
}

func padTo(s string, w int) string {
	for lipgloss.Width(s) < w {
		s += " "
	}
	return s
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func f1(v float64) string {
	i := int(v * 10)
	return itoa(i/10) + "." + itoa(i%10)
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
