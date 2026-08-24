package screens

import (
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
