package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Column is one table column definition.
type Column struct {
	Title string
	Width int
}

// Row is one table row of cell strings.
type Row = []string

// FilterTable is a first-party filtered table: "/" substring filter,
// vim-style jumps, cursor tracking and a consistent look across the
// app. Rows are rendered here rather than by bubbles/table so styled
// cells keep their ANSI at any width — bubbles truncates by rune
// count, which slices escape sequences and once forced us to strip
// every color from the main panes.
type FilterTable struct {
	cols      []Column
	baseCols  []Column // pristine widths used by fitColumns
	rows      []Row    // all rows
	keys      []string // keys[i] identifies rows[i]; empty = untracked
	origIdx   []int    // origIdx[visible] = index into rows
	lastKey   string   // selection to restore on rebuild
	filtering bool
	filterStr string
	input     textinput.Model

	vis    []Row // visible (filtered) rows
	cursor int   // index into vis
	yOff   int   // first visible body row

	lastClickRow int
	lastClickAt  time.Time

	headerSty lipgloss.Style

	vw, vh int // total block size incl. position/filter line
}

var headerStyle = func() lipgloss.Style {
	// Border only under the header text: per-column bordered blocks
	// concatenate into one continuous rule, matching the old look.
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(false).
		BorderLeft(false).
		BorderRight(false).
		BorderBottom(true).
		BorderForeground(Palette.Surface).
		Foreground(Palette.Muted).
		Bold(false).
		Padding(0, 1)
}()

// NewFilterTable creates a table with the given columns and dimensions.
func NewFilterTable(cols []Column, width, height int) FilterTable {
	ti := textinput.New()
	ti.Placeholder = "type to filter"
	ti.Prompt = "/"
	styles := textinput.DefaultStyles(true) // dark background app
	styles.Focused.Prompt = lipgloss.NewStyle().Foreground(Palette.Blue)
	styles.Blurred.Prompt = styles.Focused.Prompt
	styles.Focused.Text = lipgloss.NewStyle().Foreground(Palette.Text)
	styles.Blurred.Text = styles.Focused.Text
	ti.SetStyles(styles)
	ft := FilterTable{cols: cols, baseCols: cols, input: ti, headerSty: headerStyle}
	ft.SetSize(width, height)
	return ft
}

// bodyH is the number of body rows below the two header lines.
func (f *FilterTable) bodyH() int {
	h := f.vh - 3 // stats/filter line + header text + header rule
	if h < 0 {
		h = 0
	}
	return h
}

// SetSize updates the viewport dimensions and refits columns.
// h is the total block height: the position/filter line reserves the
// first row, the table gets the rest.
func (f *FilterTable) SetSize(w, h int) {
	f.vw, f.vh = w, h
	f.fitColumns(w)
	f.syncScroll()
}

// fitColumns scales the pristine column definitions down (widest
// first, never below 3 cells) so the rendered total — column widths
// plus the header's built-in 1-cell padding on each side — never
// exceeds the table width. Without this, wide layouts wrap at the
// terminal edge and corrupt the whole screen.
func (f *FilterTable) fitColumns(w int) {
	cols := append([]Column(nil), f.baseCols...)
	budget := w - 2*len(cols)
	if budget < len(cols)*3 {
		budget = len(cols) * 3 // degenerate terminal; stay usable
	}
	total := 0
	for _, c := range cols {
		total += c.Width
	}
	overflow := total - budget
	for overflow > 0 {
		maxIdx, maxW := -1, 0
		for i, c := range cols {
			if c.Width > maxW {
				maxW, maxIdx = c.Width, i
			}
		}
		if maxIdx < 0 || maxW <= 3 {
			break
		}
		shrink := overflow
		if avail := maxW - 3; shrink > avail {
			shrink = avail
		}
		cols[maxIdx].Width -= shrink
		overflow -= shrink
	}
	f.cols = cols
}

// SetColumns replaces the column definitions.
func (f *FilterTable) SetColumns(cols []Column) {
	f.cols = cols
	f.baseCols = cols
	f.fitColumns(f.Width())
}

// SetRows replaces all rows, preserving the current filter.
func (f *FilterTable) SetRows(rows []Row) {
	f.rows = clipCells(rows, f.cols)
	f.keys = nil
	f.applyFilter()
}

// SetRowsTracked replaces all rows with stable identity keys; the
// cursor follows its row across rebuilds, sorts and filter changes.
func (f *FilterTable) SetRowsTracked(rows []Row, keys []string) {
	if len(keys) != len(rows) {
		panic("SetRowsTracked: keys/rows length mismatch")
	}
	pre := f.currentKeyOrLast()
	f.rows = clipCells(rows, f.cols)
	f.keys = keys
	f.applyFilterWith(pre)
}

// clipCells force-fits every cell into its column's fitted width.
// Truncation is display-cell aware and ANSI-preserving; nothing in
// this renderer ever truncates by rune count, so styled cells keep
// their escapes intact at any width.
func clipCells(rows []Row, cols []Column) []Row {
	if len(cols) == 0 {
		return rows
	}
	out := make([]Row, len(rows))
	for i, r := range rows {
		c := make(Row, len(r))
		for j, cell := range r {
			if j < len(cols) {
				cell = Truncate(cell, cols[j].Width)
			}
			c[j] = cell
		}
		out[i] = c
	}
	return out
}

// SelectedKey returns the stable key of the cursor's row.
func (f *FilterTable) SelectedKey() (string, bool) {
	idx, ok := f.Selected()
	if !ok || idx >= len(f.keys) {
		return "", false
	}
	return f.keys[idx], true
}

// Rows returns all rows currently set on the table.
func (f *FilterTable) Rows() []Row { return f.rows }

// Len returns the number of visible (filtered) rows.
func (f *FilterTable) Len() int { return len(f.origIdx) }

// Selected returns the original row index under the cursor.
func (f *FilterTable) Selected() (int, bool) {
	cur := f.cursor
	if cur < 0 || cur >= len(f.origIdx) {
		return 0, false
	}
	return f.origIdx[cur], true
}

// Cursor returns the visible cursor position.
func (f *FilterTable) Cursor() int { return f.cursor }

// Filtering reports whether the filter input has focus.
func (f *FilterTable) Filtering() bool { return f.filtering }

// StartFilter activates the filter input.
func (f *FilterTable) StartFilter() tea.Cmd {
	f.filtering = true
	cmd := f.input.Focus()
	f.input.SetValue(f.filterStr)
	return cmd
}

// CancelFilter exits filter mode without applying changes.
func (f *FilterTable) CancelFilter() {
	f.filtering = false
	f.input.Blur()
	if f.filterStr != "" {
		f.filterStr = ""
		f.applyFilter()
	}
}

// ClearFilter drops the active filter and restores all rows.
func (f *FilterTable) ClearFilter() {
	f.filtering = false
	f.input.Blur()
	f.input.SetValue("")
	f.filterStr = ""
	f.applyFilter()
}

// AcceptFilter commits the current input as the active filter.
func (f *FilterTable) AcceptFilter() {
	f.filtering = false
	f.input.Blur()
	f.filterStr = strings.TrimSpace(f.input.Value())
	f.applyFilter()
}

// FilterString returns the active filter text.
func (f *FilterTable) FilterString() string { return f.filterStr }

func (f *FilterTable) applyFilter() { f.applyFilterWith(f.currentKeyOrLast()) }

// currentKeyOrLast resolves the selection to restore: the live cursor's
// key when state is consistent, else the last tracked key.
func (f *FilterTable) currentKeyOrLast() string {
	if k, ok := f.selKey(); ok {
		return k
	}
	return f.lastKey
}

func (f *FilterTable) applyFilterWith(want string) {
	q := strings.ToLower(f.filterStr)
	f.origIdx = f.origIdx[:0]
	for i, r := range f.rows {
		if q == "" || containsFold(r, q) {
			f.origIdx = append(f.origIdx, i)
		}
	}
	f.vis = make([]Row, len(f.origIdx))
	for vi, oi := range f.origIdx {
		f.vis[vi] = f.rows[oi]
	}

	// Restore the cursor onto the same logical item; without this the
	// cursor stays at an index while items shift (BACKLOG-T1).
	if want != "" && f.keys != nil {
		for vi, oi := range f.origIdx {
			if oi < len(f.keys) && f.keys[oi] == want {
				f.cursor = vi
				f.lastKey = want
				f.syncScroll()
				return
			}
		}
	}
	f.lastKey = ""
	switch {
	case len(f.vis) == 0:
		f.cursor = 0 // nothing selectable
	case f.cursor >= len(f.vis):
		f.cursor = len(f.vis) - 1
	case f.cursor < 0:
		f.cursor = 0
	}
	f.syncScroll()
}

func (f *FilterTable) selKey() (string, bool) {
	cur := f.cursor
	if cur < 0 || cur >= len(f.origIdx) {
		return "", false
	}
	oi := f.origIdx[cur]
	if oi >= len(f.keys) {
		return "", false
	}
	return f.keys[oi], true
}

func containsFold(r Row, q string) bool {
	for _, c := range r {
		if strings.Contains(strings.ToLower(c), q) {
			return true
		}
	}
	return false
}

// Width returns the current table width.
func (f *FilterTable) Width() int { return f.vw }

// VisibleOrigins returns the original row index of each visible row.
func (f *FilterTable) VisibleOrigins() []int { return f.origIdx }

// SetCursor moves the visible cursor position.
func (f *FilterTable) SetCursor(i int) {
	if i < 0 {
		i = 0
	}
	if i > len(f.vis)-1 {
		i = len(f.vis) - 1
	}
	if i < 0 {
		i = 0
	}
	f.cursor = i
	f.syncScroll()
}

// Focus is a no-op kept for interface symmetry.
func (f *FilterTable) Focus() {}

// Blur clears focus state on the inner table.
func (f *FilterTable) Blur() {}

// --- movement ---------------------------------------------------------

func clampI(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// move shifts the cursor by d rows and keeps it in view.
func (f *FilterTable) move(d int) {
	if len(f.vis) == 0 {
		f.cursor, f.yOff = 0, 0
		return
	}
	f.cursor = clampI(f.cursor+d, 0, len(f.vis)-1)
	f.syncScroll()
}

func (f *FilterTable) gotoTop() {
	if len(f.vis) == 0 {
		return
	}
	f.cursor = 0
	f.yOff = 0
}

func (f *FilterTable) gotoBottom() {
	if len(f.vis) == 0 {
		return
	}
	f.cursor = len(f.vis) - 1
	f.syncScroll()
}

// syncScroll keeps the cursor inside the [yOff, yOff+bodyH) window,
// clamping when the list shrinks.
func (f *FilterTable) syncScroll() {
	h := f.bodyH()
	n := len(f.vis)
	if h <= 0 || n <= 0 {
		f.yOff = 0
		return
	}
	if n <= h {
		f.yOff = 0
		return
	}
	if f.cursor < f.yOff {
		f.yOff = f.cursor
	}
	if f.cursor >= f.yOff+h {
		f.yOff = f.cursor - h + 1
	}
	if f.yOff > n-h {
		f.yOff = n - h
	}
	if f.yOff < 0 {
		f.yOff = 0
	}
}

// Update handles input for the table or its filter box. Movement keys
// mirror the bubbles/table defaults exactly (up/k, down/j, b/pgup,
// f/pgdown/space, u/ctrl+u, d/ctrl+d, home/g, end/G).
func (f *FilterTable) Update(msg tea.Msg) (FilterTable, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "/":
			if !f.filtering {
				return *f, f.StartFilter()
			}
		case "enter":
			if f.filtering {
				f.AcceptFilter()
				return *f, nil
			}
		case "esc":
			if f.filtering {
				f.CancelFilter()
				return *f, nil
			}
			if f.filterStr != "" {
				// A committed filter that yields nothing must be
				// clearable with esc, or the list looks permanently
				// broken ("retyping never matches again").
				f.ClearFilter()
				return *f, nil
			}
		}
		if !f.filtering {
			half := f.bodyH()/2 + 1
			switch key.String() {
			case "up", "k":
				f.move(-1)
			case "down", "j":
				f.move(1)
			case "b", "pgup":
				f.move(-maxI(f.bodyH(), 1))
			case "f", "pgdown", " ", "space":
				f.move(maxI(f.bodyH(), 1))
			case "u", "ctrl+u":
				f.move(-half)
			case "d", "ctrl+d":
				f.move(half)
			case "home", "g":
				f.gotoTop()
			case "end", "G":
				f.gotoBottom()
			}
			return *f, nil
		}
	}
	if f.filtering {
		var cmd tea.Cmd
		prev := f.input.Value()
		f.input, cmd = f.input.Update(msg)
		if f.input.Value() != prev {
			f.filterStr = f.input.Value()
			f.applyFilter()
		}
		return *f, cmd
	}
	return *f, nil
}

func maxI(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// --- mouse ------------------------------------------------------------

// dblClickWindow is the max gap between clicks that counts as a
// double-click.
const dblClickWindow = 450 * time.Millisecond

// RowAt maps a terminal-row hit relative to the table's own View
// (0 = header text line, 1 = header rule, 2+ = body rows) onto a
// visible row index.
func (f *FilterTable) RowAt(rel int) (int, bool) {
	if rel < 2 {
		return 0, false
	}
	i := f.yOff + rel - 2
	if i >= 0 && i < len(f.vis) {
		return i, true
	}
	return 0, false
}

// Mouse handles wheel and left-click over the table. It returns
// moved=true when the cursor changed and dbl=true for a double-click
// on the same row (callers turn that into their "enter" action).
func (f *FilterTable) Mouse(m tea.MouseMsg, topAbs int) (moved, dbl bool) {
	switch ev := m.(type) {
	case tea.MouseWheelMsg:
		switch ev.Button {
		case tea.MouseWheelUp:
			f.move(-1)
			return true, false
		case tea.MouseWheelDown:
			f.move(1)
			return true, false
		}
		return false, false
	case tea.MouseClickMsg:
		if ev.Button != tea.MouseLeft {
			return false, false
		}
		row, ok := f.RowAt(ev.Y - topAbs)
		if !ok {
			return false, false
		}
		now := time.Now()
		dbl = now.Sub(f.lastClickAt) <= dblClickWindow && f.lastClickRow == row &&
			row == f.cursor // already selected before this press
		f.lastClickAt, f.lastClickRow = now, row
		if row != f.cursor {
			f.SetCursor(row)
			return true, false
		}
		return false, dbl
	}
	return false, false
}

// View renders the table with one chrome row: the filter prompt above
// while filtering, otherwise a position strip below the table.
func (f *FilterTable) View() string {
	out := f.headersView() + "\n" + f.bodyView()
	if f.filtering || f.filterStr != "" {
		bar := faintSty.Render("filter")
		val := f.input.View()
		if !f.filtering {
			val = mutedSty.Render("/" + f.filterStr)
		} else {
			val = "/" + val
		}
		return uiPanellessRow(f.vw, bar+" "+val) + "\n" + out
	}
	if total := len(f.rows); total > 0 {
		stats := faintSty.Render(fmt.Sprintf("%d/%d | %d%%",
			len(f.origIdx), total, f.scrollPct()))
		out += "\n" + uiPanellessRow(f.vw, stats)
	}
	return out
}

// headersView renders the title row plus its continuous rule.
func (f *FilterTable) headersView() string {
	parts := make([]string, 0, len(f.cols))
	for _, col := range f.cols {
		if col.Width <= 0 {
			continue
		}
		// Each block is two lines (text + bottom border); joining
		// horizontally weaves them into one title line + one rule.
		// Content is pre-fitted so blocks span exactly colW+2.
		inner := clipLine(Truncate(col.Title, col.Width), col.Width)
		parts = append(parts, f.headerSty.Render(inner))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// bodyView renders the visible window of rows, blank-filled to the
// viewport height.
func (f *FilterTable) bodyView() string {
	h := f.bodyH()
	lines := make([]string, 0, h)
	for i := f.yOff; i < f.yOff+h; i++ {
		if i >= len(f.vis) {
			lines = append(lines, "")
			continue
		}
		row := f.renderRow(i)
		if i == f.cursor {
			// Full-line highlight: pad before wrapping so the fill
			// covers trailing space too (the canvas painter keeps it).
			row = SelectedRow().Render(clipLine(row, maxI(f.vw, 1)))
		}
		lines = append(lines, row)
	}
	for len(lines) < h {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// renderRow joins one row's cells: each padded to its fitted width,
// separated by the header's 1-cell gutters.
func (f *FilterTable) renderRow(i int) string {
	var b strings.Builder
	for j, col := range f.cols {
		if col.Width <= 0 {
			continue
		}
		cell := ""
		if j < len(f.vis[i]) {
			cell = f.vis[i][j]
		}
		b.WriteString(" " + clipLine(cell, col.Width) + " ")
	}
	return b.String()
}

func uiPanellessRow(w int, content string) string {
	gap := w - lipgloss.Width(content)
	if gap < 0 {
		gap = 0
	}
	return content + strings.Repeat(" ", gap)
}

func (f *FilterTable) scrollPct() int {
	n := len(f.origIdx)
	if n <= 1 {
		return 100
	}
	cur := f.cursor
	return int(float64(cur) / float64(n-1) * 100)
}
