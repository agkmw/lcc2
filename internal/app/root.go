// Package app wires the root model: layout chrome, routing between
// screens and global state such as toasts and the help overlay.
package app

import (
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"lcc2/internal/ui"
)

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
	screens  map[string]ui.Screen
	order    []string
	active   int
	width    int
	height   int
	notes    ui.NotifyStack
	helpOpen bool
	quitting bool
}

// New creates the root model with the given screens (order matters).
func New(screens ...ui.Screen) Root {
	r := Root{screens: map[string]ui.Screen{}}
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

// switchTo activates section i, lazily starting (or restarting) it.
func (r *Root) switchTo(i int) tea.Cmd {
	if i < 0 || i >= len(r.order) || i == r.active {
		return nil
	}
	r.active = i
	return tea.Batch(r.current().Init(), r.sendSize())
}

// Update handles global events and delegates to the active screen.
func (r Root) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		r.width, r.height = m.Width, m.Height
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
			case "tab":
				return r, r.switchTo((r.active + 1) % len(r.order))
			case "shift+tab":
				return r, r.switchTo((r.active - 1 + len(r.order)) % len(r.order))
			case "1", "2", "3", "4", "5", "6":
				n := int(m.String()[0] - '1')
				return r, r.switchTo(n)
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
// subtracting the tab strip, status bar and page margins.
func (r Root) contentArea() (int, int) {
	w := r.width - 4 // page margins
	h := r.height - 4
	if w < 10 {
		w = 10
	}
	if h < 4 {
		h = 4
	}
	return w, h
}

// View renders the full application frame: tab strip, body, status
// bar — then paints everything onto the app's own opaque canvas.
func (r Root) View() string {
	if r.quitting || r.width == 0 {
		return ""
	}
	cur := r.current()
	w, h := r.contentArea()

	body := ui.ClipBlock(cur.View(), w)
	lines := strings.Split("  "+strings.ReplaceAll(body, "\n", "\n  "), "\n")
	for len(lines) < h {
		lines = append(lines, "")
	}
	if len(lines) > h {
		lines = lines[:h]
	}

	frame := r.viewTabStrip() + "\n" +
		strings.Join(lines, "\n") + "\n" +
		r.viewStatusBar(cur)
	out := ui.Canvas(frame, r.width, r.height)
	if r.helpOpen {
		out = r.overlay(out, r.helpPanel())
	}
	return ui.CompositeNotes(out, r.notes)
}

// viewTabStrip renders the nvim-bufferline-style top bar: logo, one
// numbered segment per screen, dividers between. No decorative glyphs:
// East-Asian-Ambiguous shapes render double-width in tmux/locales and
// shift every following column (ADR-0010).
func (r Root) viewTabStrip() string {
	logo := lipgloss.NewStyle().Bold(true).Render("lcc2")
	segs := []string{logo}
	div := faintSty.Render("│")
	for i, id := range r.order {
		s := lookupSection(id)
		label := s.label
		if bs, ok := r.screens[id].(ui.BadgeSource); ok {
			if b := bs.Badge(); b != "" {
				label += " " + lipgloss.NewStyle().Bold(true).
					Foreground(ui.Accent(id)).Render(b)
			}
		}
		idx := faintSty.Render(strconv.Itoa(i+1))
		if i == r.active {
			segs = append(segs, idx+" "+lipgloss.NewStyle().
				Bold(true).Foreground(ui.Accent(id)).Render(label))
		} else {
			segs = append(segs, idx+" "+mutedSty.Render(label))
		}
	}
	row := " " + strings.Join(segs, " "+div+" ")
	return row + "\n" + rule(r.width)
}

func rule(w int) string {
	return lipgloss.NewStyle().Foreground(ui.Palette.Surface).
		Render(strings.Repeat("─", maxInt(w, 1)))
}

// viewStatusBar renders the bottom bar: square rule, hints left,
// context slot and clock right.
func (r Root) viewStatusBar(cur ui.Screen) string {
	hints := ""
	for _, kb := range cur.Hints() {
		if !kb.Enabled() {
			continue
		}
		hints += keycap(kb.Help().Key) + faintSty.Render(" "+kb.Help().Desc+"  ")
	}
	hints += keycap("?") + faintSty.Render(" help  ") +
		keycap("q") + faintSty.Render(" quit")

	right := ""
	if cs, ok := cur.(ui.ContextSource); ok && cs.ContextHint() != "" {
		right = cs.ContextHint() + faintSty.Render(" - ")
	}
	right += faintSty.Render(time.Now().Format("15:04"))

	budget := r.width - 2 - lipgloss.Width(right) - 1
	hints = ui.Truncate(hints, maxInt(budget, 8))
	gap := r.width - 2 - lipgloss.Width(hints) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return rule(r.width) + "\n" +
		" " + hints + strings.Repeat(" ", gap) + right + " "
}

func keycap(k string) string {
	return lipgloss.NewStyle().Bold(true).
		Foreground(ui.Palette.Text).Render("[" + k + "]")
}

// helpPanel renders the keyboard reference as a solid surface block.
func (r Root) helpPanel() string {
	chip := keycap
	head := lipgloss.NewStyle().Bold(true).
		Foreground(ui.Palette.Blue).Render("Keys") + "\n\n"
	rows := func(pairs ...[2]string) string {
		var b strings.Builder
		for _, p := range pairs {
			b.WriteString("  " + chip(p[0]) + faintSty.Render(" "+p[1]) + "\n")
		}
		return b.String()
	}
	body := head +
		rows(
			[2]string{"tab / shift+tab", "next / previous screen"},
			[2]string{"1-6", "jump to screen"},
			[2]string{"j/k", "move selection"},
			[2]string{"/", "filter list"},
			[2]string{"enter", "select / open"},
			[2]string{"esc", "back / cancel"},
			[2]string{"?", "help"},
			[2]string{"q", "quit"},
		) +
		"\n" + lipgloss.NewStyle().Bold(true).
		Render(lookupSection(r.order[r.active]).label+" keys") + "\n\n" +
		func() string {
			var b strings.Builder
			for _, kb := range r.current().Hints() {
				if kb.Enabled() {
					b.WriteString("  " + chip(kb.Help().Key) +
						faintSty.Render(" "+kb.Help().Desc) + "\n")
				}
			}
			return b.String()
		}()

	pw := 0
	for _, l := range strings.Split(body, "\n") {
		if w := lipgloss.Width(l); w > pw {
			pw = w
		}
	}
	return ui.PaintBlock(body, pw+4, ui.Palette.Surface)
}

// overlay centers a filled panel on top of the base frame without
// moving any other byte of it.
func (r Root) overlay(base, panel string) string {
	bl := strings.Split(base, "\n")
	pl := strings.Split(panel, "\n")
	y := (len(bl) - len(pl)) / 2
	x := (r.width - lipgloss.Width(panel)) / 2
	if y < 0 {
		y = 0
	}
	if x < 0 {
		x = 0
	}
	for i, src := range pl {
		row := y + i
		if row >= len(bl) {
			break
		}
		line := bl[row]
		if lw := lipgloss.Width(line); lw > x {
			line = ansi.Truncate(line, x, "")
		} else if lw < x {
			line += strings.Repeat(" ", x-lw)
		}
		bl[row] = line + src
	}
	return strings.Join(bl, "\n")
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
