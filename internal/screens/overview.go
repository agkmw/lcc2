// Package screens contains the top-level section models of the app.
// Screens consume data from the pure provider packages and own all
// Bubble Tea state; providers never import the UI.
package screens

import (
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

// View renders the dashboard panels.
func (o Overview) View() string {
	if !o.loaded || !o.widthSet {
		return o.center(ui.EmptyState("", "Gathering system information…", "", o.w))
	}
	w := o.w
	leftW := w * 2 / 5
	rightW := w - leftW - 2

	left := lipgloss.JoinVertical(lipgloss.Left,
		o.systemPanel(leftW),
		o.memoryPanel(leftW),
	)
	right := lipgloss.JoinVertical(lipgloss.Left,
		o.cpuPanel(rightW),
		o.networkPanel(rightW),
	)

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	if w < 72 {
		body = lipgloss.JoinVertical(lipgloss.Left,
			o.systemPanel(w), o.cpuPanel(w), o.memoryPanel(w), o.networkPanel(w))
	}

	diskLine := ""
	if o.snap.root != nil {
		f := o.snap.root
		diskLine = "\n" + mutedSty.Render("disk  ") +
			ui.Gauge(f.UsedPercent, min(40, w-16), nil) +
			mutedSty.Render("  "+sysinfo.FormatBytes(float64(f.Used))+" / "+
				sysinfo.FormatBytes(float64(f.Total))+" on "+f.Mountpoint)
	}
	return body + diskLine
}

func (o Overview) center(s string) string {
	return lipgloss.Place(o.w, o.h, lipgloss.Center, lipgloss.Center, s)
}

func (o Overview) systemPanel(w int) string {
	h := o.snap.host
	rows := [][2]string{
		{"host", h.Hostname},
		{"os", titleCase(h.Platform) + " " + h.PlatformVersion},
		{"kernel", h.KernelVersion + " (" + h.KernelArch + ")"},
		{"uptime", sysinfo.FormatUptime(h.Uptime)},
		{"cores", itoa(o.snap.cpu.Cores)},
		{"load", f1(o.snap.load.One) + "  " + f1(o.snap.load.Five) + "  " + f1(o.snap.load.Fifteen)},
	}
	return panel("system", w-2, kvRows(rows))
}

func (o Overview) memoryPanel(w int) string {
	m := o.snap.mem
	var b strings.Builder
	b.WriteString(row("ram ", ui.Gauge(m.UsedPercent, gaugeWidth(w), nil),
		sysinfo.FormatBytes(float64(m.Used))+" / "+sysinfo.FormatBytes(float64(m.Total))))
	b.WriteString("\n")
	b.WriteString(row("swap", ui.Gauge(m.SwapPercent, gaugeWidth(w), nil),
		sysinfo.FormatBytes(float64(m.SwapUsed))+" / "+sysinfo.FormatBytes(float64(m.SwapTotal))))
	return panel("memory", w-2, b.String())
}

func (o Overview) cpuPanel(w int) string {
	c := o.snap.cpu
	var b strings.Builder
	b.WriteString(ui.Spark(o.cpuHist, sparkWidth(w), ui.Palette.Blue) + "\n\n")
	const maxShow = 12
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
		b.WriteString(faintSty.Render("… +"+itoa(c.Cores-maxShow)+" more cores") + "\n")
	}
	return panel("cpu", w-2, strings.TrimRight(b.String(), "\n"))
}

func (o Overview) networkPanel(w int) string {
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
	return panel("network", w-2, b.String())
}

func gaugeWidth(w int) int { return clampInt(w/2-10, 10, 40) }
func sparkWidth(w int) int { return clampInt(w-14, 10, 80) }

func row(label string, bar string, right string) string {
	return mutedSty.Render(label+" ") + bar + "  " + faintSty.Render(right)
}

func panel(title string, width int, body string) string {
	head := lipgloss.NewStyle().Bold(true).
		Foreground(ui.Accent("overview")).Render(title)
	return ui.Panel().
		BorderForeground(ui.Accent("overview")).
		Width(width).
		Padding(0, 1).
		Render(head + "\n" + body)
}

func kvRows(rows [][2]string) string {
	var b strings.Builder
	for _, r := range rows {
		b.WriteString(mutedSty.Render(padTo(r[0], 7)) + faintSty.Render(r[1]) + "\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
