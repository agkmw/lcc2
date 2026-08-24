package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Notification is one transient floating window, nvim-notify style.
type Notification struct {
	ID   uint64
	Kind string // "info", "ok", "err"
	Text string
}

const maxNotes = 4

// NotifyStack holds active notifications, newest first.
type NotifyStack struct {
	next  uint64
	items []Notification
}

// Push adds a notification and returns it (for expiry scheduling).
func (n *NotifyStack) Push(kind, text string) Notification {
	n.next++
	notif := Notification{ID: n.next, Kind: kind, Text: text}
	n.items = append([]Notification{notif}, n.items...)
	if len(n.items) > maxNotes {
		n.items = n.items[:maxNotes]
	}
	return notif
}

// Dismiss removes the notification with the given id; no-op if gone.
func (n *NotifyStack) Dismiss(id uint64) {
	for i, it := range n.items {
		if it.ID == id {
			n.items = append(n.items[:i], n.items[i+1:]...)
			return
		}
	}
}

// Items returns a copy of the active notifications.
func (n NotifyStack) Items() []Notification {
	out := make([]Notification, len(n.items))
	copy(out, n.items)
	return out
}

// Len reports how many notifications are active.
func (n NotifyStack) Len() int { return len(n.items) }

// CompositeNotes splices notification windows into base, right-aligned,
// starting one line below the top. Content outside each window rectangle
// is preserved byte-for-byte; nothing else on screen moves.
func CompositeNotes(base string, s NotifyStack) string {
	if s.Len() == 0 {
		return base
	}
	var win []string
	for _, it := range s.Items() {
		win = append(win, noteWindow(it)...)
	}
	if len(win) == 0 {
		return base
	}

	bl := strings.Split(base, "\n")
	target := 0
	for _, l := range bl {
		if w := lipgloss.Width(l); w > target {
			target = w
		}
	}
	const margin = 1
	cut := target - margin
	avail := len(bl) - 2 // rows 1..len-2 stay clear of header/footer
	if avail < len(win) {
		if avail < 0 {
			avail = 0
		}
		win = win[:avail] // never let a window straddle the footer
	}
	for i, wl := range win {
		r := 1 + i // below the header line
		line := bl[r]
		start := cut - lipgloss.Width(wl)
		if start < 0 {
			continue
		}
		if lipgloss.Width(line) > start {
			line = ansi.Truncate(line, start, "")
		} else {
			line += strings.Repeat(" ", start-lipgloss.Width(line))
		}
		bl[r] = line + wl
	}
	return strings.Join(bl, "\n")
}

func noteWindow(it Notification) []string {
	var color lipgloss.Color
	icon, kind := "*", "info"
	switch it.Kind {
	case "ok":
		color, icon, kind = Palette.Green, "+", "ok"
	case "err":
		color, icon, kind = Palette.Red, "x", "error"
	}
	caption := faintSty.Render(kind)
	body := icon + " " + it.Text
	inner := clampW(max(lipgloss.Width(body), 6), 16, 44)
	if lipgloss.Width(body) > inner {
		body = Truncate(body, inner)
	}
	pad := inner - lipgloss.Width(body)
	content := strings.Repeat(" ", max(0, inner-6)) + caption + "\n" +
		body + strings.Repeat(" ", max(0, pad))
	panel := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(color).
		Padding(0, 1).
		Render(content)
	// Solid fill: the window must read as a layer above the canvas,
	// not punch through to the terminal background.
	return strings.Split(PaintBlock(panel, lipgloss.Width(panel), Palette.Surface), "\n")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clampW(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
