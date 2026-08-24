package ui

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Gauge renders a percentage bar like "█████▌░░░░░ 52%" with a
// fractional final block for smooth motion. color picks the fill
// color; pass nil for automatic (green→yellow→red).
func Gauge(pct float64, width int, color *lipgloss.Color) string {
	if width < 8 {
		width = 8
	}
	slots := width - 5 // bar area; " NN%" label takes the other five
	exact := clampF(pct/100*float64(slots), 0, float64(slots))
	full := int(exact)
	frac := exact - float64(full)
	bar := strings.Repeat("█", full)
	if frac >= 0.125 && full < slots {
		bar += fractionBlock(frac)
		full++
	}
	var c lipgloss.Color
	switch {
	case color != nil:
		c = *color
	case pct >= 90:
		c = Palette.Red
	case pct >= 70:
		c = Palette.Yellow
	default:
		c = Palette.Green
	}
	label := padLeft(itoa(int(pct)), 3) + "%"
	rest := slots - len([]rune(bar))
	return lipgloss.NewStyle().Foreground(c).Render(bar) +
		lipgloss.NewStyle().Foreground(Palette.Faint).Render(strings.Repeat("░", rest)) +
		mutedSty.Render(" "+label)
}

func fractionBlock(f float64) string {
	ramp := []rune{'▏', '▎', '▌', '▊'}
	i := int(f * float64(len(ramp)))
	if i < 0 {
		i = 0
	}
	if i >= len(ramp) {
		return ""
	}
	return string(ramp[i])
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

// Spark renders a one-line sparkline from samples using block characters.
func Spark(samples []float64, width int, c lipgloss.Color) string {
	if len(samples) == 0 {
		return faintSty.Render(strings.Repeat("▁", width))
	}
	blocks := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	out := make([]rune, 0, width)
	start := 0
	if len(samples) > width {
		start = len(samples) - width
	}
	for _, v := range samples[start:] {
		idx := int(v / 100 * float64(len(blocks)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(blocks) {
			idx = len(blocks) - 1
		}
		out = append(out, blocks[idx])
	}
	for len(out) < width {
		out = append(out, '▁')
	}
	return lipgloss.NewStyle().Foreground(c).Render(string(out))
}

// Graph renders a multi-row area chart of 0-100 samples, newest at
// the right edge. Columns beyond the recorded history stay blank so
// the chart grows into its full width.
func Graph(samples []float64, w, h int, c lipgloss.Color) string {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	start := 0
	if len(samples) > w {
		start = len(samples) - w
	}
	vals := samples[start:]
	off := w - len(vals)
	grid := make([][]rune, h)
	for i := range grid {
		grid[i] = []rune(strings.Repeat(" ", w))
	}
	blocks := []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇'}
	for j, v := range vals {
		v = clampF(v, 0, 100)
		eighths := int(math.Round(v / 100 * float64(h*8)))
		full := eighths / 8
		part := eighths % 8
		for k := 0; k < full && k < h; k++ {
			grid[h-1-k][off+j] = '█'
		}
		if part > 0 && full < h {
			grid[h-1-full][off+j] = blocks[part]
		}
	}
	sty := lipgloss.NewStyle().Foreground(c)
	rows := make([]string, h)
	for i, r := range grid {
		rows[i] = sty.Render(string(r))
	}
	return strings.Join(rows, "\n")
}

// SegGauge renders a bar where the filled fraction uses color fill and
// the sub-fraction within it uses overlay (btop-style segmented usage,
// e.g. cache inside used memory). The empty remainder is faint.
func SegGauge(pct, innerPct float64, width int, fill, overlay lipgloss.Color) string {
	if width < 4 {
		width = 4
	}
	slots := width
	nFill := int(clampF(pct/100, 0, 1) * float64(slots))
	nInner := int(clampF(innerPct/100, 0, 1) * float64(nFill))
	fsty := lipgloss.NewStyle().Foreground(fill)
	isty := lipgloss.NewStyle().Foreground(overlay)
	esty := lipgloss.NewStyle().Foreground(Palette.Faint)
	return esty.Render(strings.Repeat("░", slots-nFill)) +
		isty.Render(strings.Repeat("█", nInner)) +
		fsty.Render(strings.Repeat("█", nFill-nInner))
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

func padLeft(s string, w int) string {
	for len(s) < w {
		s = " " + s
	}
	return s
}

// EmptyState renders a friendly placeholder for empty lists or errors.
func EmptyState(icon, title, hint string, width int) string {
	var b strings.Builder
	b.WriteString("\n\n")
	if icon != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(Palette.Faint).Render(icon))
		b.WriteString("\n")
	}
	b.WriteString(titleSty.Foreground(Palette.Muted).Render(title))
	if hint != "" {
		b.WriteString("\n")
		b.WriteString(faintSty.Render(hint))
	}
	body := b.String()
	w := width
	if w <= 0 {
		w = 60
	}
	if lipgloss.Width(body) > w {
		return body
	}
	return lipgloss.PlaceHorizontal(w, lipgloss.Center, body)
}
