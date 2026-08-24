package screens

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"lcc2/internal/ui"
)

// Shared styles for all screens, derived from the central palette.
var (
	mutedSty = lipgloss.NewStyle().Foreground(ui.Palette.Muted)
	faintSty = lipgloss.NewStyle().Foreground(ui.Palette.Faint)
	goodSty  = lipgloss.NewStyle().Foreground(ui.Palette.Green)
	badSty   = lipgloss.NewStyle().Foreground(ui.Palette.Red)
	warnSty  = lipgloss.NewStyle().Foreground(ui.Palette.Yellow)
)

// paneHeight sizes a bordered pane to hug its content: lipgloss
// Height() bounds the inner block and the border adds two rows, so the
// returned value keeps the rendered pane within avail.
func paneHeight(contentLines, avail int) int {
	inner := min(contentLines, avail-2)
	if inner < 4 {
		inner = 4
	}
	return inner
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// contextHint is the shared shape of the status-bar right slot.
func contextSel(name, detail string) string {
	if name == "" {
		return ""
	}
	if detail != "" {
		return name + faintSty.Render(" - "+detail)
	}
	return name
}

// pageHead renders the standard two-part page title row: bold title
// left, faint meta right, padded to w.
func pageHead(title, metaRight string, w int) string {
	title = lipgloss.NewStyle().Bold(true).Render(title)
	meta := faintSty.Render(metaRight)
	gap := w - lipgloss.Width(title) - lipgloss.Width(metaRight)
	if gap < 1 {
		gap = 1
	}
	return ui.ClipBlock(title+strings.Repeat(" ", gap)+meta, w)
}

// modeCell colors a permission string by meaning: the type bit stands
// out (d/l accent-bold), executable bits tint green, everything else
// stays faint.
func modeCell(mode os.FileMode) string {
	s := ui.Narrow(mode.String())
	var b strings.Builder
	for i, r := range s {
		switch {
		case i == 0 && (r == 'd' || r == 'l'):
			b.WriteString(lipgloss.NewStyle().Bold(true).
				Foreground(ui.Accent("files")).Render(string(r)))
		case r == 'x':
			b.WriteString(goodSty.Render(string(r)))
		default:
			b.WriteString(faintSty.Render(string(r)))
		}
	}
	return b.String()
}

// idCell tones numeric ids by account class: root loud, system
// accounts dim, humans normal.
func idCell(v int) string {
	switch {
	case v == 0:
		return warnSty.Render(itoa(v))
	case v < 1000:
		return faintSty.Render(itoa(v))
	default:
		return itoa(v)
	}
}

// shellCell dims shells that cannot log in.
func shellCell(sh string) string {
	if sh == "" || sh == "/bin/false" || strings.Contains(sh, "nologin") {
		return faintSty.Render(ui.Truncate(sh, 20))
	}
	return ui.Truncate(sh, 20)
}
