package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// ErrorDialog shows an error with an expandable technical detail section.
type ErrorDialog struct {
	Title   string
	Message string
	Details string
	ShowDet bool
	vp      viewport.Model
	width   int
	height  int
}

// NewError builds an error dialog.
func NewError(title, message, details string) ErrorDialog {
	return ErrorDialog{
		Title: title, Message: message, Details: details,
		vp: viewport.New(56, 6), width: 60, height: 14,
	}
}

// Update handles scrolling when details are visible.
func (e *ErrorDialog) Update(msg tea.Msg) (ErrorDialog, bool) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc", "enter", "q":
			return *e, true
		case "d", "D":
			e.ShowDet = !e.ShowDet
		case "up", "k":
			if e.ShowDet {
				e.vp.LineUp(1)
			}
		case "down", "j":
			if e.ShowDet {
				e.vp.LineDown(1)
			}
		}
	}
	return *e, false
}

// View renders the error dialog.
func (e ErrorDialog) View() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(Palette.Red).Render(e.Title))
	b.WriteString("\n\n")
	b.WriteString(Truncate(e.Message, e.width-4))
	b.WriteString("\n\n")
	b.WriteString(mutedSty.Render("d") + faintSty.Render(" details   ") + mutedSty.Render("esc") + faintSty.Render(" dismiss"))
	if e.ShowDet && e.Details != "" {
		e.vp.Width = e.width - 6
		e.vp.SetContent(faintSty.Render(e.Details))
		b.WriteString("\n\n")
		b.WriteString(e.vp.View())
	}
	return Panel().
		BorderForeground(Palette.Red).
		Width(e.width).
		Padding(1, 2).
		Render(b.String())
}
