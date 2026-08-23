// Package app wires the root model: layout chrome, routing between
// screens and global state such as toasts and the help overlay.
package app

import (
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lcc2/internal/ui"
)

const sidebarWidth = 18

type section struct {
	id    string
	label string
}

var sections = []section{
	{"overview", "Overview"},
	{"proc", "Processes"},
	{"disk", "Disks"},
	{"files", "Files"},
	{"services", "Services"},
	{"users", "Users"},
}

var (
	mutedSty = lipgloss.NewStyle().Foreground(ui.Palette.Muted)
	faintSty = lipgloss.NewStyle().Foreground(ui.Palette.Faint)
)

// Root is the top-level tea.Model.
type Root struct {
	screens   map[string]ui.Screen
	order     []string
	active    int
	width     int
	height    int
	toast     ui.ToastMsg
	toastSeq  uint64
	helpOpen  bool
	quitting  bool
	sidebarOn bool
}

// New creates the root model with the given screens (order matters).
func New(screens ...ui.Screen) Root {
	r := Root{
		screens:   map[string]ui.Screen{},
		sidebarOn: true,
	}
	for _, s := range screens {
		r.screens[s.ID()] = s
		r.order = append(r.order, s.ID())
	}
	return r
}

// Init starts the active screen.
func (r Root) Init() tea.Cmd {
	if len(r.order) == 0 {
		return nil
	}
	return r.screens[r.order[0]].Init()
}

func (r Root) current() ui.Screen { return r.screens[r.order[r.active]] }

// Update handles global events and delegates to the active screen.
func (r Root) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		r.width, r.height = m.Width, m.Height
		r.sidebarOn = m.Width >= 84
		var cmds []tea.Cmd
		for _, s := range r.order {
			sc, c := r.screens[s].Update(m)
			r.screens[s] = sc
			cmds = append(cmds, c)
		}
		cmds = append(cmds, r.sendSize())
		return r, tea.Batch(cmds...)

	case ui.SizeMsg:
		cur := r.current()
		sc, cmd := cur.Update(m)
		r.screens[r.order[r.active]] = sc
		return r, cmd

	case toastTickMsg:
		if m.seq == r.toastSeq { // only the newest toast owns its timer
			r.toast = ui.ToastMsg{}
		}
		return r, nil

	case ui.ToastMsg:
		r.toast = m
		r.toastSeq++
		seq := r.toastSeq
		d := 3 * time.Second // errors stay twice as long
		if m.Kind == "err" {
			d = 6 * time.Second
		}
		return r, tea.Tick(d, func(time.Time) tea.Msg {
			return toastTickMsg{seq: seq}
		})

	case tea.KeyMsg:
		if r.helpOpen {
			switch m.String() {
			case "?", "esc":
				r.helpOpen = false
			case "ctrl+c":
				r.quitting = true
				return r, tea.Quit
			}
			return r, nil
		}
		if m.String() == "ctrl+c" {
			r.quitting = true
			return r, tea.Quit
		}
		if !r.current().CapturingInput() {
			switch m.String() {
			case "q":
				r.quitting = true
				return r, tea.Quit
			case "?":
				r.helpOpen = true
				return r, nil
			case "1", "2", "3", "4", "5", "6":
				n := int(m.String()[0] - '1')
				if n < len(r.order) && n != r.active {
					r.active = n
					// Lazily start (or restart) the newly active section.
					return r, tea.Batch(r.current().Init(), r.sendSize())
				}
				return r, nil
			}
		}
	}

	cur := r.current()
	sc, cmd := cur.Update(msg)
	r.screens[r.order[r.active]] = sc
	return r, cmd
}

// sendSize forwards computed content dimensions to the active screen.
func (r Root) sendSize() tea.Cmd {
	w, h := r.contentArea()
	cur := r.current()
	sc, cmd := cur.Update(ui.SizeMsg{Width: w, Height: h})
	r.screens[r.order[r.active]] = sc
	return cmd
}

// contentArea returns the space available to the active screen after
// subtracting header, footer, sidebar and padding.
func (r Root) contentArea() (int, int) {
	w := r.width
	if r.sidebarOn {
		w -= sidebarWidth + 1
	}
	h := r.height - 2 // header + footer
	w -= 2            // horizontal padding
	h -= 2            // vertical breathing room
	if w < 10 {
		w = 10
	}
	if h < 4 {
		h = 4
	}
	return w, h
}

// View renders the full application frame.
func (r Root) View() string {
	if r.quitting || r.width == 0 {
		return ""
	}
	cur := r.current()

	var body string
	if r.sidebarOn {
		body = lipgloss.JoinHorizontal(lipgloss.Top,
			r.viewSidebar(), r.viewContent(cur))
	} else {
		body = r.viewTabStrip() + "\n" + r.viewContent(cur)
	}

	// The body must occupy exactly the space between header and
	// footer: short content is padded, tall content is clipped.
	// Otherwise the footer floats up on sparse screens and slips off
	// the bottom on crowded ones.
	bodyH := r.height - 2 // header + footer
	if bodyH < 1 {
		bodyH = 1
	}
	body = lipgloss.NewStyle().Height(bodyH).MaxHeight(bodyH).Render(body)

	frame := r.viewHeader(cur) + "\n" + body + "\n" + r.viewFooter(cur)
	out := ui.Base().Render(frame)
	if r.helpOpen {
		out = r.overlay(out, r.helpPanel())
	}
	if r.toast.Text != "" {
		out = r.overlay(out, r.toastPanel())
	}
	return out
}

func (r Root) viewHeader(cur ui.Screen) string {
	a := ui.Accent(cur.ID())
	left := lipgloss.NewStyle().Bold(true).Foreground(a).Render("lcc2")
	title := lipgloss.NewStyle().Bold(true).
		Render(" / " + cur.Title())
	right := faintSty.Render(time.Now().Format("15:04"))
	gap := r.width - lipgloss.Width(left+title+right)
	if gap < 1 {
		gap = 1
	}
	return left + title + strings.Repeat(" ", gap) + right
}

func (r Root) viewSidebar() string {
	var b strings.Builder
	b.WriteString("\n")
	for i, id := range r.order {
		s := lookupSection(id)
		sty := mutedSty
		marker := "  "
		if i == r.active {
			a := ui.Accent(s.id)
			marker = lipgloss.NewStyle().Foreground(a).Bold(true).Render("> ")
			sty = lipgloss.NewStyle().Bold(true).Foreground(a)
		}
		b.WriteString(marker + sty.Render(s.label) + "\n\n")
	}
	return lipgloss.NewStyle().Width(sidebarWidth).Render(b.String())
}

func (r Root) viewTabStrip() string {
	parts := make([]string, 0, len(r.order))
	for i, id := range r.order {
		s := lookupSection(id)
		label := strconv.Itoa(i+1) + " " + s.label
		if i == r.active {
			a := ui.Accent(s.id)
			parts = append(parts, lipgloss.NewStyle().Bold(true).
				Foreground(a).Underline(true).
				Render(" "+label+" "))
		} else {
			parts = append(parts, faintSty.Render(" "+label+" "))
		}
	}
	return strings.Join(parts, "")
}

func (r Root) viewContent(cur ui.Screen) string {
	w, _ := r.contentArea()
	content := cur.View()
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		lines[i] = " " + l
	}
	padded := strings.Join(lines, "\n")
	return lipgloss.NewStyle().Width(w + 1).MaxHeight(r.height - 2).
		Render(padded)
}

func (r Root) viewFooter(cur ui.Screen) string {
	hints := " "
	for _, kb := range cur.Hints() {
		if !kb.Enabled() {
			continue
		}
		hints += lipgloss.NewStyle().Bold(true).Foreground(ui.Accent(cur.ID())).
			Render(kb.Help().Key) +
			faintSty.Render(" "+kb.Help().Desc+"   ")
	}
	hints += faintSty.Render("? help   q quit")
	toastW := 0
	if r.toast.Text != "" {
		t := r.toastPanelInline()
		toastW = lipgloss.Width(t)
		hints = ui.Truncate(hints, r.width-toastW-2)
		gap := r.width - lipgloss.Width(hints) - toastW
		if gap < 1 {
			gap = 1
		}
		return ui.Truncate(hints, r.width) + strings.Repeat(" ", gap) + t
	}
	return ui.Truncate(hints, r.width)
}

func (r Root) toastPanelInline() string {
	var sty lipgloss.Style
	icon := "*"
	switch r.toast.Kind {
	case "ok":
		sty = lipgloss.NewStyle().Foreground(ui.Palette.Green)
		icon = "+"
	case "err":
		sty = lipgloss.NewStyle().Foreground(ui.Palette.Red)
		icon = "x"
	default:
		sty = lipgloss.NewStyle().Foreground(ui.Palette.Blue)
	}
	return sty.Bold(true).Render(icon + " " + r.toast.Text + " ")
}

func (r Root) toastPanel() string {
	t := ui.Panel().Padding(0, 1).Render(r.toastPanelInline())
	return lipgloss.Place(r.width, lipgloss.Height(t), lipgloss.Right, lipgloss.Bottom, t,
		lipgloss.WithWhitespaceForeground(ui.Palette.Faint))
}

func (r Root) helpPanel() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ui.Palette.Blue).
		Render("Keyboard reference") + "\n\n")
	globals := []struct{ k, d string }{
		{"1-6", "jump to section"}, {"j / k", "down / up"},
		{"h / l", "back / open"}, {"enter", "select"},
		{"/", "filter list"}, {"esc", "cancel / back"},
		{"r", "refresh"}, {"?", "toggle this help"}, {"q", "quit"},
	}
	for _, g := range globals {
		b.WriteString("  " + lipgloss.NewStyle().Bold(true).
			Foreground(ui.Palette.Blue).Render(pad(g.k, 7)) +
			faintSty.Render(g.d) + "\n")
	}
	b.WriteString("\n" + lipgloss.NewStyle().Bold(true).
		Render("Section keys") + "\n\n")
	for _, kb := range r.current().Hints() {
		if kb.Enabled() {
			b.WriteString("  " + lipgloss.NewStyle().Bold(true).
				Foreground(ui.Accent(r.current().ID())).Render(pad(kb.Help().Key, 7)) +
				faintSty.Render(kb.Help().Desc) + "\n")
		}
	}
	return ui.Panel().Padding(1, 3).Render(b.String())
}

// overlay centers a panel on top of the base frame.
func (r Root) overlay(base, panel string) string {
	return lipgloss.Place(r.width, lipgloss.Height(base),
		lipgloss.Center, lipgloss.Center, panel,
		lipgloss.WithWhitespaceForeground(ui.Palette.Faint))
}

func pad(s string, w int) string {
	for lipgloss.Width(s) < w {
		s += " "
	}
	return s
}

func lookupSection(id string) section {
	for _, s := range sections {
		if s.id == id {
			return s
		}
	}
	return section{id: id, label: id}
}

type toastTickMsg struct{ seq uint64 }
