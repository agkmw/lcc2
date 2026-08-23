package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ConfirmDialog asks for yes/no before a potentially destructive action.
type ConfirmDialog struct {
	Title   string
	Body    string
	Details string // optional technical detail shown with "d"
	ShowDet bool
	width   int
}

// NewConfirm builds a confirmation dialog.
func NewConfirm(title, body, details string) ConfirmDialog {
	return ConfirmDialog{Title: title, Body: body, Details: details, width: 60}
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

// View renders the dialog.
func (c ConfirmDialog) View() string {
	var b strings.Builder
	b.WriteString(Danger().Render(c.Title))
	b.WriteString("\n\n")
	b.WriteString(Truncate(c.Body, c.width-4))
	b.WriteString("\n\n")
	hints := mutedSty.Render("y") + faintSty.Render(" confirm   ") +
		mutedSty.Render("n") + faintSty.Render("/") + mutedSty.Render("esc") + faintSty.Render(" cancel")
	if c.Details != "" {
		hints += faintSty.Render("   ") + mutedSty.Render("d") + faintSty.Render(" details")
	}
	b.WriteString(hints)
	if c.ShowDet && c.Details != "" {
		b.WriteString("\n\n")
		b.WriteString(faintSty.Render(Truncate(c.Details, c.width-4)))
	}
	return Panel().
		BorderForeground(Palette.Red).
		Width(c.width).
		Padding(1, 2).
		Render(b.String())
}
