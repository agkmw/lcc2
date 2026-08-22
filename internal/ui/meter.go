package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Gauge renders a percentage bar like "█████░░░░░ 52%".
// color picks the fill color; pass nil for automatic (green→yellow→red).
func Gauge(pct float64, width int, color *lipgloss.Color) string {
	if width < 8 {
		width = 8
	}
	filled := int(pct/100*float64(width-6) + 0.5)
	if filled > width-6 {
		filled = width - 6
	}
	if filled < 0 {
		filled = 0
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
	bar := strings.Repeat("█", filled)
	rest := strings.Repeat("░", width-6-filled)
	label := padLeft(itoa(int(pct)), 3) + "%"
	return lipgloss.NewStyle().Foreground(c).Render(bar) +
		lipgloss.NewStyle().Foreground(Palette.Faint).Render(rest) +
		mutedSty.Render(" "+label)
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
	if w <= 0 || w > 80 {
		w = 60
	}
	return lipgloss.PlaceHorizontal(w, lipgloss.Center, body)
}
