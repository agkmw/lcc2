// Package app wires the root model: layout chrome, routing between
// screens and global state such as toasts and the help overlay.
package app

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"lcc/internal/session"
	"lcc/internal/ui"
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
	screens    map[string]ui.Screen
	order      []string
	active     int
	width      int
	height     int
	notes      ui.NotifyStack
	helpOpen   bool
	helpScroll int
	quitting   bool
	// tabSpans lives behind a pointer because View renders through
	// value receivers: a plain slice write would be discarded with
	// the receiver copy and stripHit would never see it.
	tabSpans *[]tabSpan
}

// tabSpan is the horizontal hit range of one tab segment on row 0.
type tabSpan struct {
	start, end, idx int
}

// stripHit maps an x column on the tab strip to a section index.
func (r Root) stripHit(x int) (int, bool) {
	if r.tabSpans == nil {
		return 0, false
	}
	for _, s := range *r.tabSpans {
		if x >= s.start && x <= s.end {
			return s.idx, true
		}
	}
	return 0, false
}

// New creates the root model with the given screens (order matters).
func New(screens ...ui.Screen) Root {
	r := Root{screens: map[string]ui.Screen{}, tabSpans: &[]tabSpan{}}
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
		cmds = append(cmds, r.screens[r.order[r.active]].Init())
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
	r.active = i
	// Snapshot after the flip: Screen must record the destination, and
	// Files prefs come from the screen map regardless of which section
	// is active.
	r.saveSession()
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

// snapshot gathers the persistable state. The Files screen contributes
// its extras regardless of which section is active — otherwise any
// save made elsewhere (clock tick, quit) resets cwd/sort to defaults.
func (r Root) snapshot() session.State {
	st := session.State{Screen: r.active, SortKey: "name"}
	if src, ok := r.screens["files"].(stateSource); ok {
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

	case tea.MouseWheelMsg:
		if r.helpOpen {
			// The help overlay owns the wheel: scroll its key list
			// instead of the content underneath.
			switch m.Button {
			case tea.MouseWheelUp:
				r.clampHelpScroll(-3)
			case tea.MouseWheelDown:
				r.clampHelpScroll(3)
			}
			return r, nil
		}
		return r.delegate(msg)

	case tea.MouseClickMsg:
		if r.helpOpen {
			return r, nil // nothing behind the overlay is clickable
		}
		ev := m.Mouse()
		if ev.Y == 0 { // tab strip
			if idx, ok := r.stripHit(ev.X); ok {
				return r, r.switchTo(idx)
			}
		}
		// Everything else belongs to the active screen.
		return r.delegate(m)

	case tea.KeyPressMsg:
		if r.helpOpen {
			switch m.String() {
			case "?", "esc", "q":
				r.helpOpen = false
			case "ctrl+c":
				r.quitting = true
				r.saveSession()
				return r, tea.Quit
			case "ctrl+d", "pgdown":
				r.clampHelpScroll(r.helpViewport())
			case "ctrl+u", "pgup":
				r.clampHelpScroll(-r.helpViewport())
			case "j", "down":
				r.clampHelpScroll(1)
			case "k", "up":
				r.clampHelpScroll(-1)
			case "tab", "shift+tab", "1", "2", "3", "4", "5", "6":
				// The overlay follows the active screen: helpContent
				// rebuilds from r.current() on every render.
				r.helpScroll = 0
				return r, r.switchTo(helpTarget(m.String(), r.active, len(r.order)))
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
				r.saveSession()
				return r, tea.Quit
			case "?":
				r.helpOpen = true
				r.helpScroll = 0
				return r, nil
			case "tab":
				return r, r.switchTo(helpTarget("tab", r.active, len(r.order)))
			case "shift+tab":
				return r, r.switchTo(helpTarget("shift+tab", r.active, len(r.order)))
			case "1", "2", "3", "4", "5", "6":
				return r, r.switchTo(helpTarget(m.String(), r.active, len(r.order)))
			}
		}
	}

	return r.delegate(msg)
}

// delegate forwards a message to the active screen.
func (r Root) delegate(msg tea.Msg) (tea.Model, tea.Cmd) {
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
// viewString renders the full frame as a string (pre-v2 style).
func (r Root) viewString() string {
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
	logo := lipgloss.NewStyle().Bold(true).Render("lcc")

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
		spans := make([]tabSpan, 0, len(tabs))
		x := 1 + lipgloss.Width(logo) // leading global space + logo
		for i := range tabs {
			ti := tabs[i]
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
			var seg string
			if i == r.active {
				// Inverted chip marks the current section at a glance;
				// the SGR-state canvas keeps the fill intact.
				seg = lipgloss.NewStyle().
					Bold(true).Foreground(ui.Accent(ti.id)).
					Background(ui.Palette.Surface).
					Render(" " + label + " ")
			} else {
				seg = mutedSty.Render(label)
			}
			if i > 0 {
				x += 3 // separator " │ "
			}
			w := lipgloss.Width(seg)
			spans = append(spans, tabSpan{start: x, end: x + w - 1, idx: i})
			segs = append(segs, seg)
			x += w
		}
		*r.tabSpans = spans
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
// context slot and clock right. StatusSource screens show only their
// state hints plus a "? keys (n)" pointer into the help overlay —
// the full key list lives there, so the bar never overflows.
func (r Root) viewStatusBar(cur ui.Screen) string {
	hints := ""
	if ss, ok := cur.(ui.StatusSource); ok {
		for _, kb := range ss.StatusHints() {
			if kb.Enabled() {
				hints += keycap(kb.Help().Key) + faintSty.Render(" "+kb.Help().Desc+"  ")
			}
		}
		n := 0
		for _, kb := range cur.Hints() {
			if kb.Enabled() {
				n++
			}
		}
		hints += keycap("?") + faintSty.Render(fmt.Sprintf(" keys (%d)  ", n))
	} else {
		for _, kb := range cur.Hints() {
			if !kb.Enabled() {
				continue
			}
			hints += keycap(kb.Help().Key) + faintSty.Render(" "+kb.Help().Desc+"  ")
		}
	}
	hints += keycap("q") + faintSty.Render(" quit")

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

// View implements tea.Model: wraps the string frame in a tea.View
// with the alternate screen enabled.
func (r Root) View() tea.View {
	v := tea.NewView(r.viewString())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// helpContent builds the scrollable key list. Every row is derived
// from what actually handles the key: root's own bindings first, the
// shared list-navigation row when the current screen drives a
// FilterTable, then the screen's actions, then the mouse reference.
// Nothing here may advertise a key the view under the overlay ignores.
func (r Root) helpContent() []string {
	rows := func(pairs ...[2]string) []string {
		out := make([]string, 0, len(pairs))
		for _, p := range pairs {
			out = append(out, "  "+keycap(p[0])+faintSty.Render(" "+p[1]))
		}
		return out
	}
	content := rows(
		[2]string{"tab / shift+tab", "next / previous screen"},
		[2]string{fmt.Sprintf("1-%d", len(r.order)), "jump to screen"},
		[2]string{"?", "help"},
		[2]string{"q", "quit"},
	)
	if screenHasList(r.current()) {
		content = append(content, rows([2]string{"j/k", "move selection"})...)
	}
	for _, kb := range r.current().Hints() {
		if kb.Enabled() {
			content = append(content, "  "+keycap(kb.Help().Key)+
				faintSty.Render(" "+kb.Help().Desc))
		}
	}
	content = append(content, "", faintSty.Render(
		"mouse (after esc): click rows - wheel scrolls - click tabs to switch"))
	return content
}

// screenHasList reports whether the screen drives a FilterTable: it
// advertises the shared "/" filter binding, whose list answers to j/k.
// Screens without a list (Overview) skip the shared rows.
func screenHasList(s ui.Screen) bool {
	for _, kb := range s.Hints() {
		if kb.Enabled() && slices.Contains(kb.Keys(), "/") {
			return true
		}
	}
	return false
}

// helpTarget maps a section-switch key to the target index; -1 (a
// no-op for switchTo) for anything else.
func helpTarget(k string, active, n int) int {
	switch k {
	case "tab":
		return (active + 1) % n
	case "shift+tab":
		return (active - 1 + n) % n
	case "1", "2", "3", "4", "5", "6":
		return int(k[0] - '1')
	}
	return -1
}

// helpViewport is the visible row count of the key list: the content
// area minus the panel's title/footer and borders. Fixed per terminal
// size - the panel is the same height on every screen.
func (r Root) helpViewport() int {
	_, ch := r.contentArea()
	vh := ch - 8
	if vh < 4 {
		vh = 4
	}
	return vh
}

// clampHelpScroll applies delta to the scroll offset, clamped to the
// list length.
func (r *Root) clampHelpScroll(delta int) {
	max := len(r.helpContent()) - r.helpViewport()
	if max < 0 {
		max = 0
	}
	r.helpScroll += delta
	if r.helpScroll > max {
		r.helpScroll = max
	}
	if r.helpScroll < 0 {
		r.helpScroll = 0
	}
}

// helpPanel renders the keyboard reference as a bordered card floating
// on the dimmed canvas (ADR-0011 supersedes the no-card clause for
// help): a visible boundary is the cue that a modal layer is open.
// The card is a fixed-height viewport - identical across screens -
// scrolled with the wheel or ctrl+d/ctrl+u.
func (r Root) helpPanel() string {
	section := lookupSection(r.order[r.active]).label
	content := r.helpContent()

	vh := r.helpViewport()
	maxTop := len(content) - vh
	if maxTop < 0 {
		maxTop = 0
	}
	top := r.helpScroll
	if top > maxTop {
		top = maxTop
	}
	if top < 0 {
		top = 0
	}
	end := top + vh
	if end > len(content) {
		end = len(content)
	}
	view := content[top:end]
	for len(view) < vh {
		view = append(view, "")
	}

	body := lipgloss.NewStyle().Bold(true).Foreground(ui.Palette.Blue).
		Render("Keys") + faintSty.Render("  "+section) + "\n\n" +
		strings.Join(view, "\n") + "\n\n" +
		faintSty.Render(fmt.Sprintf("%d-%d of %d", top+1, end, len(content))) +
		faintSty.Render("   ctrl+d/u scroll   esc/q close")

	return ui.Panel().
		BorderForeground(ui.Palette.Surface).
		Width(clampHelpWidth(r.width)).
		Height(vh+4).
		Padding(0, 1).
		Render(body)
}

func clampHelpWidth(w int) int {
	v := w - 12
	if v < 44 {
		v = 44
	}
	if v > 76 {
		v = 76
	}
	return v
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
		// Terminate the base row's SGR state at the cut point: a style
		// still open in the truncated run would bleed bold/bright into
		// the panel text appended after it.
		bl[row] = line + "\x1b[0m" + src
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
