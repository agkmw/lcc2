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
	notes     ui.NotifyStack
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

	case noteExpiryMsg:
		r.notes.Dismiss(m.id)
		return r, nil

	case ui.ToastMsg:
		n := r.notes.Push(m.Kind, m.Text)
		d := 3 * time.Second // errors stay twice as long
		if m.Kind == "err" {
			d = 6 * time.Second
		}
		return r, tea.Tick(d, func(time.Time) tea.Msg {
			return noteExpiryMsg{id: n.ID}
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
// subtracting the nav/status bars, sidebar rail and page margins.
func (r Root) contentArea() (int, int) {
	w := r.width
	if r.sidebarOn {
		w -= sidebarWidth + 1 // rail + its border column
	}
	w -= 4            // page margins
	h := r.height - 5 // nav bar(2) + status bar(2) + one blank row
	if w < 10 {
		w = 10
	}
	if h < 4 {
		h = 4
	}
	return w, h
}

func (r Root) bodyH() int { return maxInt(r.height-4, 1) }

// View renders the full application frame: nav bar, body, status bar.
func (r Root) View() string {
	if r.quitting || r.width == 0 {
		return ""
	}
	cur := r.current()

	var mid string
	if r.sidebarOn {
		mid = lipgloss.JoinHorizontal(lipgloss.Top,
			r.viewSidebar(r.bodyH()), r.viewPage(cur))
	} else {
		mid = r.viewPage(cur)
	}

	frame := r.viewNavBar() + "\n" + mid + "\n" + r.viewStatusBar(cur)
	// One wide line would make Style.Render pad every line to its
	// width — clip the whole frame to the terminal first.
	out := ui.Base().Render(ui.ClipBlock(frame, r.width))
	if r.helpOpen {
		out = r.overlay(out, r.helpPanel())
	}
	return ui.CompositeNotes(out, r.notes)
}

var sectionGlyph = map[string]string{
	"overview": "◈", "proc": "⚙", "disk": "▤",
	"files": "▸", "services": "✦", "users": "◍",
}

// viewNavBar renders the top bar: plain logo, section links, clock,
// closed by a full-width square rule.
func (r Root) viewNavBar() string {
	logo := lipgloss.NewStyle().Bold(true).Render("lcc2")
	links := make([]string, 0, len(r.order))
	for i, id := range r.order {
		s := lookupSection(id)
		label := strconv.Itoa(i+1) + "  " + s.label
		if i == r.active {
			links = append(links, lipgloss.NewStyle().Bold(true).
				Foreground(ui.Accent(id)).Underline(true).Render(label))
		} else {
			links = append(links, mutedSty.Render(label))
		}
	}
	left := logo + "   " + strings.Join(links, "   ")
	clock := faintSty.Render(time.Now().Format("15:04"))
	inner := maxInt(r.width-2, 12)
	left = ui.Truncate(left, maxInt(inner-lipgloss.Width(clock)-1, 8))
	gap := inner - lipgloss.Width(left) - lipgloss.Width(clock)
	if gap < 1 {
		gap = 1
	}
	row := " " + left + strings.Repeat(" ", gap) + clock + " "
	border := lipgloss.NewStyle().Foreground(ui.Palette.Surface).
		Render(strings.Repeat("─", maxInt(r.width, 1)))
	return row + "\n" + border
}

// viewSidebar renders the nav rail with a right border column, padded
// to exactly bodyH lines.
func (r Root) viewSidebar(bodyH int) string {
	var lines []string
	lines = append(lines, "")
	for i, id := range r.order {
		s := lookupSection(id)
		glyph := faintSty.Render(sectionGlyph[s.id])
		if i == r.active {
			mark := lipgloss.NewStyle().Foreground(ui.Accent(id)).Render("▍")
			lbl := lipgloss.NewStyle().Bold(true).Foreground(ui.Accent(id)).
				Render(s.label)
			lines = append(lines, mark+" "+glyph+" "+lbl, "")
		} else {
			lines = append(lines, " "+glyph+" "+mutedSty.Render(s.label), "")
		}
	}
	borderCh := lipgloss.NewStyle().Foreground(ui.Palette.Surface).Render("│")
	out := make([]string, bodyH)
	for i := 0; i < bodyH; i++ {
		l := ""
		if i < len(lines) {
			l = lines[i]
		}
		out[i] = ui.ClipBlock(l, sidebarWidth-1) + borderCh
	}
	return strings.Join(out, "\n")
}

// viewPage renders the active screen inside the page margins.
func (r Root) viewPage(cur ui.Screen) string {
	w, _ := r.contentArea()
	body := ui.ClipBlock(cur.View(), w)
	lines := strings.Split("  "+strings.ReplaceAll(body, "\n", "\n  "), "\n")
	h := r.bodyH()
	return lipgloss.NewStyle().Height(h).MaxHeight(h).
		Render(strings.Join(lines, "\n"))
}

// viewStatusBar renders the bottom bar: square rule, hints left,
// context slot right.
func (r Root) viewStatusBar(cur ui.Screen) string {
	border := lipgloss.NewStyle().Foreground(ui.Palette.Surface).
		Render(strings.Repeat("─", maxInt(r.width, 1)))
	hints := ""
	for _, kb := range cur.Hints() {
		if !kb.Enabled() {
			continue
		}
		hints += ui.KeyBadge(cur.ID(), kb.Help().Key) +
			faintSty.Render(" "+kb.Help().Desc+"  ")
	}
	hints += ui.KeyBadge(cur.ID(), "?") + faintSty.Render(" help  ") +
		ui.KeyBadge(cur.ID(), "q") + faintSty.Render(" quit")
	right := ""
	if cs, ok := cur.(ui.ContextSource); ok {
		right = faintSty.Render(cs.ContextHint())
	}
	budget := r.width - 2
	if right != "" {
		budget -= lipgloss.Width(right) + 3
	}
	hints = ui.Truncate(hints, maxInt(budget, 8))
	gap := budget - lipgloss.Width(hints)
	if gap < 1 {
		gap = 1
	}
	row := " " + hints + strings.Repeat(" ", gap) + right + " "
	return border + "\n" + row
}

func (r Root) helpPanel() string {
	chip := func(k string) string {
		return lipgloss.NewStyle().Bold(true).Foreground(ui.Palette.Blue).
			Render("[" + k + "]")
	}
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).
		Foreground(ui.Palette.Blue).Render("Keyboard reference") + "\n\n")
	globals := []struct{ k, d string }{
		{"1-6", "jump to section"}, {"j/k", "down / up"},
		{"h/l", "back / open"}, {"enter", "select"},
		{"/", "filter list"}, {"esc", "cancel / back"},
		{"r", "refresh"}, {"?", "help"}, {"q", "quit"},
	}
	for _, g := range globals {
		b.WriteString("  " + chip(g.k) + faintSty.Render(" "+g.d) + "\n")
	}
	b.WriteString("\n" + lipgloss.NewStyle().Bold(true).
		Render("Section keys") + "\n\n")
	for _, kb := range r.current().Hints() {
		if kb.Enabled() {
			b.WriteString("  " + chip(kb.Help().Key) +
				faintSty.Render(" "+kb.Help().Desc) + "\n")
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func lookupSection(id string) section {
	for _, s := range sections {
		if s.id == id {
			return s
		}
	}
	return section{id: id, label: id}
}

type noteExpiryMsg struct{ id uint64 }
