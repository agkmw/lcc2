package ui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/key"
)

// SizeMsg carries the computed content dimensions to the active screen,
// after the root subtracts the chrome (header, sidebar, footer).
type SizeMsg struct{ Width, Height int }

// ToastMsg asks the root to display a transient status message.
type ToastMsg struct {
	Kind string // "info", "ok", "err"
	Text string
}

// InfoToast returns an informational toast command.
func InfoToast(text string) tea.Cmd { return toast("info", text) }

// OkToast returns a success toast command.
func OkToast(text string) tea.Cmd { return toast("ok", text) }

// ErrToast returns an error toast command.
func ErrToast(text string) tea.Cmd { return toast("err", text) }

func toast(kind, text string) tea.Cmd {
	return func() tea.Msg { return ToastMsg{Kind: kind, Text: text} }
}

// Screen is implemented by every top-level section of the application.
type Screen interface {
	Init() tea.Cmd
	Update(tea.Msg) (Screen, tea.Cmd)
	View() string
	ID() string    // section id, used for accents and routing
	Title() string // human readable section title
	Hints() []key.Binding
	// CapturingInput is true while a text field owns the keyboard,
	// which suppresses global shortcuts like q and the number keys.
	CapturingInput() bool
}

// ContextSource is an optional Screen extension: whatever it returns
// shows up in the status bar's right-hand slot.
type ContextSource interface {
	ContextHint() string
}

// BadgeSource is an optional Screen extension: a short status string
// rendered next to the section's tab title, e.g. a pending-change
// count. Return "" when there is nothing to report.
type BadgeSource interface {
	Badge() string
}
