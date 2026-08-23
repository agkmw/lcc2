package screens

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"lcc2/internal/ui"
)

// Shared styles for all screens, derived from the central palette.
var (
	mutedSty = lipgloss.NewStyle().Foreground(lipgloss.Color("#9399B2"))
	faintSty = lipgloss.NewStyle().Foreground(lipgloss.Color("#585B70"))
	goodSty  = lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1"))
	badSty   = lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8"))
	warnSty  = lipgloss.NewStyle().Foreground(lipgloss.Color("#F9E2AF"))
)

// joinPanes places two blocks side by side with a one-column gutter,
// keeping their tops aligned. Never pass a leading newline to
// lipgloss.JoinHorizontal: it shifts and corrupts the second block.
func joinPanes(left, right string) string {
	lines := strings.Split(right, "\n")
	for i := range lines {
		lines[i] = " " + lines[i]
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Join(lines, "\n"))
}

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
		return name + faintSty.Render(" · "+detail)
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
