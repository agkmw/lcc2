// Package app wires the root model: layout chrome, routing between
// screens and global state such as toasts and the help overlay.
package app

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"lcc2/internal/session"
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

// Init starts the active screen plus the status-bar clock.
func (r Root) Init() tea.Cmd {
	var cmds []tea.Cmd
	if len(r.order) > 0 {
		cmds = append(cmds, r.screens[r.order[0]].Init())
	}
	cmds = append(cmds, tickClock())
	return tea.Batch(cmds...)
}

func tickClock() tea.Cmd {
	return tea.Tick(time.Minute, func(time.Time) tea.Msg { return clockTickMsg{} })
}

type clockTickMsg struct{}

func (r Root) current() ui.Screen { return r.screens[r.order[r.active]] }

// switchTo activates section i, lazily starting (or restarting) it.
func (r *Root) switchTo(i int) tea.Cmd {
	if i < 0 || i >= len(r.order) || i == r.active {
		return nil
	}
	r.saveSession() // capture the outgoing screen's prefs first
	r.active = i
	return tea.Batch(r.current().Init(), r.sendSize())
}

// Minimum terminal geometry the UI stays coherent at; below it a
// friendly notice renders instead of broken layouts.
const (
	MinW = 64
	MinH = 16
)

// NewStartingAt builds the root with the given initial section
// (clamped), for session restore.
func NewStartingAt(active int, screens ...ui.Screen) Root {
	r := New(screens...)
	if active >= 0 && active < len(r.order) {
		r.active = active
	}
	return r
}

// stateSource is implemented by screens with persisted preferences.
type stateSource interface {
	SessionState() session.State
}

// snapshot gathers the persistable state; only the Files screen
// contributes extras today.
func (r Root) snapshot() session.State {
	st := session.State{Screen: r.active, SortKey: "name"}
	if src, ok := r.current().(stateSource); ok {
		fs := src.SessionState()
		st.Cwd, st.Hidden = fs.Cwd, fs.Hidden
		st.SortKey, st.SortDesc = fs.SortKey, fs.SortDesc
	}
	return st
}

func (r Root) saveSession() { _ = session.Save(r.snapshot()) }

// Update handles global events and delegates to the active screen.
func (r Root) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		resized := r.width != m.Width || r.height != m.Height
		r.width, r.height = m.Width, m.Height
		w, h := r.contentArea()
		var cmds []tea.Cmd
		// Broadcast fresh geometry to every screen so inactive tabs
		// never carry stale layouts into their next visit.
		for _, s := range r.order {
			sc, c := r.screens[s].Update(ui.SizeMsg{Width: w, Height: h})
			r.screens[s] = sc
			cmds = append(cmds, c)
		}
		if resized {
			// tmux split/zoom leaves stale cells outside bubbletea's
			// diff expectations; force a full repaint.
			cmds = append(cmds, tea.ClearScreen)
		}
		return r, tea.Batch(cmds...)

	case ui.SizeMsg:
		cur := r.current()
		sc, cmd := cur.Update(m)
		r.screens[r.order[r.active]] = sc
		return r, cmd

	case noteExpiryMsg:
		r.notes.Dismiss(m.id)
		return r, nil

	case clockTickMsg:
		r.saveSession()
		return r, tickClock() // keep the status-bar clock honest

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
			r.saveSession()
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
	if r.width < MinW || r.height < MinH {
		return ui.CanvasWith(r.tooSmallBody(), r.width, r.height, ui.BG())
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
	if r.helpOpen {
		// Splice first, paint last: the canvas pass guarantees every
		// cell — including the strip right of the panel — carries the
		// dim backdrop. No card fill; the key list floats on it.
		frame = r.overlay(frame, r.helpPanel())
	}
	bg := ui.BG()
	if r.helpOpen {
		bg = ui.BGDim()
	}
	out := ui.CanvasWith(frame, r.width, r.height, bg)
	return ui.CompositeNotes(out, r.notes)
}

// viewTabStrip renders the nvim-bufferline-style top bar: logo, one
// numbered segment per screen, dividers between. No decorative glyphs:
// East-Asian-Ambiguous shapes render double-width in tmux/locales and
// shift every following column (ADR-0010).
//
// Narrow terminals degrade by priority instead of clipping mid-tab
// (backlog L9): drop badges, then numbers, then shrink all labels.
func (r Root) viewTabStrip() string {
	logo := lipgloss.NewStyle().Bold(true).Render("lcc2")

	type tabInfo struct {
		id    string
		label string
		badge string
	}
	tabs := make([]tabInfo, 0, len(r.order))
	for _, id := range r.order {
		ti := tabInfo{id: id, label: lookupSection(id).label}
		if bs, ok := r.screens[id].(ui.BadgeSource); ok {
			ti.badge = bs.Badge()
		}
		tabs = append(tabs, ti)
	}

	render := func(numbers, badges bool, maxLabel int) string {
		segs := []string{logo}
		for i, ti := range tabs {
			label := ti.label
			if badges && ti.badge != "" {
				label += " " + lipgloss.NewStyle().Bold(true).
					Foreground(ui.Accent(ti.id)).Render(ti.badge)
			}
			if i != r.active { // the active label degrades last
				label = clipPlain(label, maxLabel)
			}
			if numbers {
				label = faintSty.Render(strconv.Itoa(i+1)) + " " + label
			}
			if i == r.active {
				// Inverted chip marks the current section at a glance;
				// the SGR-state canvas keeps the fill intact.
				segs = append(segs, lipgloss.NewStyle().
					Bold(true).Foreground(ui.Accent(ti.id)).
					Background(ui.Palette.Surface).
					Render(" "+label+" "))
			} else {
				segs = append(segs, mutedSty.Render(label))
			}
		}
		return " " + strings.Join(segs, " "+faintSty.Render("│")+" ")
	}

	degradations := []struct {
		nums, badges bool
		maxLabel     int
	}{
		{true, true, 64}, {true, false, 64}, {false, false, 64},
		{false, false, 14}, {false, false, 8}, {false, false, 3},
	}
	row := ""
	for _, d := range degradations {
		row = render(d.nums, d.badges, d.maxLabel)
		if lipgloss.Width(row) <= r.width {
			return row + "\n" + rule(r.width)
		}
	}
	return ui.ClipBlock(row, r.width) + "\n" + rule(r.width)
}

// clipPlain shortens an unstyled label to n cells with an ellipsis.
func clipPlain(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n < 3 {
		return string(r[:n])
	}
	return string(r[:n-2]) + ".."
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

// helpPanel renders the keyboard reference as a bordered card floating
// on the dimmed canvas (ADR-0011 supersedes the no-card clause for
// help): a visible boundary is the cue that a modal layer is open.
func (r Root) helpPanel() string {
	chip := keycap
	rows := func(pairs ...[2]string) []string {
		out := make([]string, 0, len(pairs))
		for _, p := range pairs {
			out = append(out, "  "+chip(p[0])+faintSty.Render(" "+p[1]))
		}
		return out
	}
	section := lookupSection(r.order[r.active]).label
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(ui.Palette.Blue).
			Render("Keys") + faintSty.Render("  "+section),
		"",
	}
	lines = append(lines, rows(
		[2]string{"tab / shift+tab", "next / previous screen"},
		[2]string{"1-6", "jump to screen"},
		[2]string{"j/k", "move selection"},
		[2]string{"/", "filter list"},
		[2]string{"enter", "select / open"},
		[2]string{"esc", "back / cancel"},
		[2]string{"?", "help"},
		[2]string{"q", "quit"},
	)...)
	lines = append(lines, "")
	for _, kb := range r.current().Hints() {
		if kb.Enabled() {
			lines = append(lines, "  "+chip(kb.Help().Key)+
				faintSty.Render(" "+kb.Help().Desc))
		}
	}
	return ui.Panel().
		BorderForeground(ui.Palette.Surface).
		Padding(0, 1).
		Render(strings.Join(lines, "\n"))
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

// tooSmallBody builds the below-floor notice: centered dims message
// on an otherwise empty frame. Kept to short lines so even absurdly
// narrow terminals show every fact.
func (r Root) tooSmallBody() string {
	bold := lipgloss.NewStyle().Bold(true).Foreground(ui.Palette.Yellow)
	body := lipgloss.JoinVertical(lipgloss.Center,
		bold.Render("terminal too small"),
		faintSty.Render(fmt.Sprintf("need >= %dx%d", MinW, MinH)),
		faintSty.Render(fmt.Sprintf("have %dx%d", r.width, r.height)),
		"",
		faintSty.Render("resize the terminal"),
		faintSty.Render("to continue"),
	)
	lines := strings.Split(lipgloss.PlaceHorizontal(
		maxInt(r.width, 1), lipgloss.Center, body), "\n")
	for len(lines) < maxInt(r.height, 1) {
		lines = append(lines, "")
	}
	if len(lines) > r.height {
		lines = lines[:r.height]
	}
	return strings.Join(lines, "\n")
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
