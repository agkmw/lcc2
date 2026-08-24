package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfirmDialog asks for yes/no before a potentially destructive action.
type ConfirmDialog struct {
	Title   string
	Body    string
	Details string // optional technical detail shown with "d"
	ShowDet bool
	// Danger styles the dialog with the red destructive band; neutral
	// surface otherwise (start/enable are not destructive).
	Danger bool
	width  int
}

// NewConfirm builds a confirmation dialog; dangerous actions keep the
// default red band, benign ones should clear Danger.
func NewConfirm(title, body, details string) ConfirmDialog {
	return ConfirmDialog{Title: title, Body: body, Details: details, Danger: true, width: 60}
}

// SetWidth sets dialog width.
func (c *ConfirmDialog) SetWidth(w int) { c.width = w }

// Update handles dialog keys; returns true when answered.
func (c *ConfirmDialog) Update(msg tea.Msg) (ConfirmDialog, bool, bool) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return *c, false, false
	}
	switch key.String() {
	case "y", "Y":
		return *c, true, true
	case "n", "N", "esc":
		return *c, false, true
	case "d", "D":
		c.ShowDet = !c.ShowDet
		return *c, false, false
	}
	return *c, false, false
}

// View renders the dialog with a band and a button row.
func (c ConfirmDialog) View() string {
	inner := c.width - 4
	bandC, borderC := Palette.Red, Palette.Red
	if !c.Danger {
		bandC, borderC = Palette.Surface, Palette.Surface
	}
	band := lipgloss.NewStyle().Bold(true).
		Background(bandC).
		Foreground(lipgloss.Color("#11111B")).
		Width(inner).
		Render(" " + c.Title)
	var b strings.Builder
	b.WriteString(band)
	b.WriteString("\n\n")
	b.WriteString(Truncate(c.Body, inner))
	b.WriteString("\n\n")
	hints := KeyBadgeStyle(Palette.Text).Render("[y]") + faintSty.Render(" confirm   ") +
		KeyBadgeStyle(Palette.Text).Render("[n]") + faintSty.Render("/") +
		KeyBadgeStyle(Palette.Text).Render("[esc]") + faintSty.Render(" cancel")
	if c.Details != "" {
		hints += faintSty.Render("   ") + KeyBadgeStyle(Palette.Text).Render("[d]") +
			faintSty.Render(" details")
	}
	b.WriteString(hints)
	if c.ShowDet && c.Details != "" {
		b.WriteString("\n\n")
		b.WriteString(faintSty.Render(Truncate(c.Details, inner)))
	}
	return Panel().
		BorderForeground(borderC).
		Width(c.width).
		Padding(1, 2).
		Render(b.String())
}

// KeyBadgeStyle renders "[k]" chips where no section accent applies.
func KeyBadgeStyle(c lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(c)
}
