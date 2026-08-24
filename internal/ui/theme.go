// Package ui contains the shared design system: theme, keymap and
// reusable components used by every screen of the application.
package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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

// Panel returns a neutral square-bordered panel for modals (prompt,
// permission editor). Main content is borderless per ADR-0009.
func Panel() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(Palette.Surface)
}

// TitledBox, Card, card and KeyBadge were removed with the borderless
// canvas redesign (ADR-0009); panes are now drawn by screens via the
// shared preview scaffold and whitespace, not bordered boxes.

// SelectedRow styles the focused list row: bold foreground only.
// Background fills are reserved for intentional surfaces (dialogs,
// toasts, help) — never scattered per-widget backgrounds.
func SelectedRow() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(Palette.Text)
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

// Truncate clips s to w cells appending an ellipsis when needed,
// preserving ANSI escape sequences intact.
func Truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= w {
		return s
	}
	return ansi.Truncate(s, w, "…")
}
