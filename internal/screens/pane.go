package screens

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"lcc2/internal/ui"
)

// The main|preview pattern shared by every data screen: the list lives
// left, a borderless preview pane right, stacked below the fold width.
// Nothing here draws boxes — separation comes from whitespace and a
// thin rule under the preview title.

const stackFold = 96 // below this width panes stack vertically

// splitGeom resolves the pane geometry for a content width w: main,
// one divider column, then preview.
func splitGeom(w int) (wide bool, mainW, prevW int) {
	if w < stackFold {
		return false, w, clampInt(w-2, 24, w)
	}
	pw := clampInt(w*2/5, 34, 56)
	return true, w - pw - 3, pw
}

// renderPreview renders the right-hand pane: accent title, faint meta
// right-aligned, thin rule, body clipped to exactly pw × ph cells.
func renderPreview(id, title, meta, body string, pw, ph int) string {
	t := lipgloss.NewStyle().Bold(true).Foreground(ui.Accent(id)).Render(title)
	gap := pw - lipgloss.Width(t) - lipgloss.Width(meta) - 1
	if gap < 1 {
		gap = 1
	}
	head := t
	if meta != "" {
		head += " " + faintSty.Render(meta)
	}
	rule := lipgloss.NewStyle().Foreground(ui.Palette.Surface).
		Render(strings.Repeat("─", maxInt(pw, 1)))
	lines := []string{ui.ClipBlock(head, pw), rule}
	lines = append(lines, strings.Split(body, "\n")...)

	out := make([]string, 0, ph)
	for i := 0; i < ph; i++ {
		l := ""
		if i < len(lines) {
			l = lines[i]
		}
		out = append(out, ui.ClipBlock(l, pw))
	}
	return strings.Join(out, "\n")
}

// joinPanesWide places main and preview side by side with a vertical
// divider column between them; stacks with a horizontal rule otherwise.
func joinPanesWide(wide bool, main, preview string, mainW, total int) string {
	if wide {
		div := lipgloss.NewStyle().Foreground(ui.Palette.Surface).Render("│")
		lines := strings.Split(main, "\n")
		pl := strings.Split(preview, "\n")
		n := len(lines)
		if len(pl) > n {
			n = len(pl)
		}
		out := make([]string, n)
		for i := 0; i < n; i++ {
			l := ""
			if i < len(lines) {
				l = lines[i]
			}
			r := ""
			if i < len(pl) {
				r = pl[i]
			}
			out[i] = ui.ClipBlock(l, mainW) + " " + div + " " + ui.ClipBlock(r, total-mainW-3)
		}
		return strings.Join(out, "\n")
	}
	rule := lipgloss.NewStyle().Foreground(ui.Palette.Surface).
		Render(strings.Repeat("-", maxInt(total, 1)))
	return main + "\n" + rule + "\n" + preview
}
