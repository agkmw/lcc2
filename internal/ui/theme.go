// Package ui contains the shared design system: theme, keymap and
// reusable components used by every screen of the application.
package ui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// C parses a "#RRGGBB" hex string into a color.Color.
func C(hex string) color.Color {
	var r, g, b uint8
	if _, err := fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b); err != nil {
		return nil
	}
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

// Palette is the single source of truth for colors.
var Palette = struct {
	Text    color.Color
	Muted   color.Color
	Faint   color.Color
	Blue    color.Color
	Green   color.Color
	Red     color.Color
	Yellow  color.Color
	Mauve   color.Color
	Teal    color.Color
	Peach   color.Color
	Surface color.Color
	Overlay color.Color
}{
	Text:    C("#CDD6F4"),
	Muted:   C("#9399B2"),
	Faint:   C("#585B70"),
	Blue:    C("#89B4FA"),
	Green:   C("#A6E3A1"),
	Red:     C("#F38BA8"),
	Yellow:  C("#F9E2AF"),
	Mauve:   C("#CBA6F7"),
	Teal:    C("#94E2D5"),
	Peach:   C("#FAB387"),
	Surface: C("#313244"),
	Overlay: C("#6C7086"),
}

// Section accent colors, keyed by section id.
var Accents = map[string]color.Color{
	"overview": Palette.Blue,
	"proc":     Palette.Green,
	"disk":     Palette.Peach,
	"files":    Palette.Mauve,
	"services": Palette.Teal,
	"users":    Palette.Yellow,
}

func accent(id string) color.Color {
	if c, ok := Accents[id]; ok {
		return c
	}
	return Palette.Blue
}

// Accent returns the theme color for a section id.
func Accent(id string) color.Color { return accent(id) }

var (
	appBG    = C("#1E1E2E")
	dimBG    = C("#11111B") // modal backdrop (crust)
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

// SelectedRow styles the focused list row: bold text on a surface
// background, a full-line highlight. Requires the SGR-state canvas
// painter so inner spans' resets don't punch holes in the fill.
func SelectedRow() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).
		Foreground(Palette.Text).
		Background(Palette.Surface)
}

// StateColor maps a 0-100 percentage onto threshold colors.
func StateColor(pct float64) color.Color {
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
	return ansi.Truncate(s, w, "..")
}

// ambiguousReplacer maps East-Asian-Ambiguous glyphs — which tmux and
// some locales render double-width, shifting every following column —
// to ASCII equivalents. Applied to provider text we do not control
// (e.g. systemctl output). See ADR-0010.
var ambiguousReplacer = strings.NewReplacer(
	"●", "*", "○", "o", "◐", "~", "◕", "~", "◉", "*",
	"✕", "x", "✗", "x", "✔", "+",
	"▸", ">", "◂", "<", "▴", "^", "▾", "v",
	"◈", "o", "◇", "o", "◆", "*",
	"…", "..", "›", "/", "·", "-",
)

// Narrow replaces ambiguous-width glyphs with ASCII lookalikes so
// rendered columns stay aligned in every terminal.
func Narrow(s string) string { return ambiguousReplacer.Replace(s) }
