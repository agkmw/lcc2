package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

var sectionEdge = lipgloss.NewStyle().Foreground(Palette.Surface)

// Section renders a btop-style box: square border with the title
// embedded in the top edge (accent-colored) and an optional
// right-aligned detail string. The body is force-fit into the inner
// w-2 × h-2 area. Only narrow-safe box glyphs are drawn (ADR-0010).
func Section(id, title, right string, w, h int, body string) string {
	if w < 10 {
		w = 10
	}
	if h < 2 {
		h = 2
	}
	iw := w - 2

	head := lipgloss.NewStyle().Bold(true).Foreground(Accent(id)).Render(title)
	if right != "" {
		head += " " + faintSty.Render(right)
	}
	// Top row budget: "┌─ "(3) + head(H) + sep(S) + dashes(F) + "┐"(1)
	// must equal iw+2 cells, so F = iw - 2 - H - S. The corner then
	// sits in the same column as every body row's right edge.
	sep := " "
	fill := iw - 2 - lipgloss.Width(head) - 1
	if fill < 0 {
		head = Truncate(head, maxInt(iw-2, 1))
		fill = maxInt(iw-2-lipgloss.Width(head), 0)
		sep = ""
	}
	top := sectionEdge.Render("┌─ ") + head + sep +
		sectionEdge.Render(strings.Repeat("─", fill)+"┐")
	bottom := sectionEdge.Render("└" + strings.Repeat("─", iw) + "┘")

	lines := strings.Split(body, "\n")
	out := make([]string, 0, h)
	out = append(out, top)
	for i := 0; i < h-2; i++ {
		l := ""
		if i < len(lines) {
			l = lines[i]
		}
		out = append(out, sectionEdge.Render("│")+" "+
			ClipBlock(l, iw-2)+" "+sectionEdge.Render("│"))
	}
	out = append(out, bottom)
	return strings.Join(out, "\n")
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
