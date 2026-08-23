// Package ui contains the shared design system: theme, keymap and
// reusable components used by every screen of the application.
package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Palette is the single source of truth for colors.
var Palette = struct {
	Text    lipgloss.Color
	Muted   lipgloss.Color
	Faint   lipgloss.Color
	Blue    lipgloss.Color
	Green   lipgloss.Color
	Red     lipgloss.Color
	Yellow  lipgloss.Color
	Mauve   lipgloss.Color
	Teal    lipgloss.Color
	Peach   lipgloss.Color
	Surface lipgloss.Color
	Overlay lipgloss.Color
}{
	Text:    lipgloss.Color("#CDD6F4"),
	Muted:   lipgloss.Color("#9399B2"),
	Faint:   lipgloss.Color("#585B70"),
	Blue:    lipgloss.Color("#89B4FA"),
	Green:   lipgloss.Color("#A6E3A1"),
	Red:     lipgloss.Color("#F38BA8"),
	Yellow:  lipgloss.Color("#F9E2AF"),
	Mauve:   lipgloss.Color("#CBA6F7"),
	Teal:    lipgloss.Color("#94E2D5"),
	Peach:   lipgloss.Color("#FAB387"),
	Surface: lipgloss.Color("#313244"),
	Overlay: lipgloss.Color("#6C7086"),
}

// Section accent colors, keyed by section id.
var Accents = map[string]lipgloss.Color{
	"overview": Palette.Blue,
	"proc":     Palette.Green,
	"disk":     Palette.Peach,
	"files":    Palette.Mauve,
	"services": Palette.Teal,
	"users":    Palette.Yellow,
}

func accent(id string) lipgloss.Color {
	if c, ok := Accents[id]; ok {
		return c
	}
	return Palette.Blue
}

// Accent returns the theme color for a section id.
func Accent(id string) lipgloss.Color { return accent(id) }

var (
	appBG    = lipgloss.Color("#1E1E2E")
	base     = lipgloss.NewStyle().Foreground(Palette.Text)
	titleSty = lipgloss.NewStyle().Bold(true)
	mutedSty = lipgloss.NewStyle().Foreground(Palette.Muted)
	faintSty = lipgloss.NewStyle().Foreground(Palette.Faint)
)

// Base returns the base text style applied to the whole app. No
// background is painted so the terminal's own palette stays intact
// and cell alignment never depends on background fills.
func Base() lipgloss.Style { return base }

// Title styles bold primary text.
func Title() lipgloss.Style { return titleSty }

// Muted styles secondary text.
func Muted() lipgloss.Style { return mutedSty }

// Faint styles hints and decorations.
func Faint() lipgloss.Style { return faintSty }

// Box returns a square border style tinted with the section accent.
func Box(sectionID string) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(accent(sectionID))
}

// Panel returns a neutral square-bordered panel.
func Panel() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(Palette.Surface)
}

// TitledBox renders an accent-bordered square panel with the title
// spliced into its top border: ┌─ title ──────┐. height > 0 pins the
// inner block height (border adds two rows on top).
func TitledBox(sectionID, title, body string, width int) string {
	return card(sectionID, title, body, width, 0)
}

// Card is TitledBox with an optional fixed inner height.
func Card(sectionID, title, body string, width, height int) string {
	return card(sectionID, title, body, width, height)
}

func card(sectionID, title, body string, width, height int) string {
	sty := Panel().BorderForeground(Accent(sectionID)).Width(width)
	if height > 0 {
		sty = sty.Height(height)
	}
	p := sty.Render(body)
	lines := strings.Split(p, "\n")
	if len(lines) == 0 || !strings.HasPrefix(lines[0], "┌") {
		return p
	}
	w := lipgloss.Width(lines[0])
	label := lipgloss.NewStyle().Bold(true).Foreground(Accent(sectionID)).
		Render(" " + title + " ")
	lw := 1 + lipgloss.Width(label)
	fill := w - 2 - lw
	if fill < 0 {
		fill = 0
	}
	lines[0] = "┌─" + label + strings.Repeat("─", fill) + "┐"
	return strings.Join(lines, "\n")
}

// KeyBadge renders a footer keycap like "[q]" in the section accent.
func KeyBadge(sectionID, key string) string {
	return lipgloss.NewStyle().Bold(true).Foreground(Accent(sectionID)).
		Render("[" + key + "]")
}

// SelectedRow is the full-line highlight for the focused list row.
func SelectedRow() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).
		Background(Palette.Surface).Foreground(Palette.Text)
}

// StateColor maps a 0-100 percentage onto threshold colors.
func StateColor(pct float64) lipgloss.Color {
	switch {
	case pct >= 90:
		return Palette.Red
	case pct >= 70:
		return Palette.Yellow
	default:
		return Palette.Green
	}
}

// Danger styles destructive text such as delete prompts.
func Danger() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(Palette.Red)
}

// KeyHint renders a key cap like "j".
func KeyHint(k string) string {
	return lipgloss.NewStyle().Bold(true).Foreground(accent("overview")).Render(k)
}

// Truncate clips s to w cells appending an ellipsis when needed.
func Truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= w {
		return s
	}
	runes := []rune(s)
	out := make([]rune, 0, w)
	width := 0
	for _, r := range runes {
		rw := lipgloss.Width(string(r))
		if width+rw > w-1 {
			break
		}
		out = append(out, r)
		width += rw
	}
	return string(out) + "…"
}
