// Package ui contains the shared design system: theme, keymap and
// reusable components used by every screen of the application.
package ui

import "github.com/charmbracelet/lipgloss"

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

// Box returns a rounded border style tinted with the section accent.
func Box(sectionID string) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent(sectionID))
}

// Panel returns a neutral rounded panel.
func Panel() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Palette.Surface)
}

// SelectedRow is the highlight style for the focused list/table row.
func SelectedRow() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#11111B"))
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
