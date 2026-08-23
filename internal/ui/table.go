package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FilterTable is a bubbles/table wrapper with a "/" substring filter,
// vim-style g/G jumps and a consistent look across the app.
type FilterTable struct {
	cols      []table.Column
	baseCols  []table.Column // pristine widths used by fitColumns
	rows      []table.Row    // all rows
	keys      []string       // keys[i] identifies rows[i]; empty = untracked
	origIdx   []int          // origIdx[visible] = index into rows
	lastKey   string         // selection to restore on rebuild
	filtering bool
	filterStr string
	input     textinput.Model
	t         table.Model
}

// NewFilterTable creates a table with the given columns and dimensions.
func NewFilterTable(cols []table.Column, width, height int) FilterTable {
	ti := textinput.New()
	ti.Placeholder = "filter…"
	ti.Prompt = "/"
	ti.PromptStyle = lipgloss.NewStyle().Foreground(Palette.Blue)
	ti.TextStyle = lipgloss.NewStyle().Foreground(Palette.Text)
	t := table.New(
		table.WithColumns(cols),
		table.WithWidth(width),
		table.WithHeight(height),
		table.WithFocused(true),
	)
	s := table.DefaultStyles()
	// Border only under the header: bubbles renders each header cell
	// inside its own border box, so left/right borders would add
	// 2*len(columns) cells of invisible width and break alignment.
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(false).
		BorderLeft(false).
		BorderRight(false).
		BorderBottom(true).
		BorderForeground(Palette.Surface).
		Foreground(Palette.Muted).
		Bold(false)
	s.Selected = lipgloss.NewStyle().Bold(true).Foreground(Palette.Blue)
	t.SetStyles(s)
	return FilterTable{cols: cols, baseCols: cols, t: t, input: ti}
}

// SetSize updates the viewport dimensions and refits columns.
func (f *FilterTable) SetSize(w, h int) {
	f.t.SetWidth(w)
	f.t.SetHeight(h)
	f.fitColumns(w)
}

// fitColumns scales the pristine column definitions down (widest
// first, never below 3 cells) so the rendered total — column widths
// plus the header's built-in 1-cell padding on each side — never
// exceeds the table width. Without this, wide layouts wrap at the
// terminal edge and corrupt the whole screen.
func (f *FilterTable) fitColumns(w int) {
	cols := append([]table.Column(nil), f.baseCols...)
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
	f.t.SetColumns(cols)
}

// SetColumns replaces the column definitions.
func (f *FilterTable) SetColumns(cols []table.Column) {
	f.cols = cols
	f.baseCols = cols
	f.t.SetColumns(cols)
	f.fitColumns(f.Width())
}

// SetRows replaces all rows, preserving the current filter.
func (f *FilterTable) SetRows(rows []table.Row) {
	f.rows = rows
	f.keys = nil
	f.applyFilter()
}

// SetRowsTracked replaces all rows with stable identity keys; the
// cursor follows its row across rebuilds, sorts and filter changes.
func (f *FilterTable) SetRowsTracked(rows []table.Row, keys []string) {
	if len(keys) != len(rows) {
		panic("SetRowsTracked: keys/rows length mismatch")
	}
	pre := f.currentKeyOrLast()
	f.rows = rows
	f.keys = keys
	f.applyFilterWith(pre)
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
func (f *FilterTable) Rows() []table.Row { return f.rows }

// Len returns the number of visible (filtered) rows.
func (f *FilterTable) Len() int { return len(f.origIdx) }

// Selected returns the original row index under the cursor.
func (f *FilterTable) Selected() (int, bool) {
	cur := f.t.Cursor()
	if cur < 0 || cur >= len(f.origIdx) {
		return 0, false
	}
	return f.origIdx[cur], true
}

// Cursor returns the visible cursor position.
func (f *FilterTable) Cursor() int { return f.t.Cursor() }

// Filtering reports whether the filter input has focus.
func (f *FilterTable) Filtering() bool { return f.filtering }

// StartFilter activates the filter input.
func (f *FilterTable) StartFilter() tea.Cmd {
	f.filtering = true
	f.input.Focus()
	f.input.SetValue(f.filterStr)
	return textinput.Blink
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
	vis := make([]table.Row, len(f.origIdx))
	for vi, oi := range f.origIdx {
		vis[vi] = f.rows[oi]
	}
	f.t.SetRows(vis)

	// Restore the cursor onto the same logical item; without this the
	// cursor stays at an index while items shift (BACKLOG-T1).
	if want != "" && f.keys != nil {
		for vi, oi := range f.origIdx {
			if oi < len(f.keys) && f.keys[oi] == want {
				f.t.SetCursor(vi)
				f.lastKey = want
				return
			}
		}
	}
	f.lastKey = ""
	// bubbles clamps the cursor to len(rows)-1 on SetRows, which is -1
	// for an empty result. A negative cursor corrupts the viewport and
	// later panics index lookups, so normalize it whenever it is out
	// of range in either direction.
	if cur := f.t.Cursor(); cur < 0 || cur >= len(vis) {
		switch {
		case len(vis) == 0:
			// nothing to select; SetCursor would clamp to -1 again
		case cur >= len(vis):
			f.t.GotoBottom()
		default:
			f.t.SetCursor(0)
		}
	}
}

func (f *FilterTable) selKey() (string, bool) {
	cur := f.t.Cursor()
	if cur < 0 || cur >= len(f.origIdx) {
		return "", false
	}
	oi := f.origIdx[cur]
	if oi >= len(f.keys) {
		return "", false
	}
	return f.keys[oi], true
}

func containsFold(r table.Row, q string) bool {
	for _, c := range r {
		if strings.Contains(strings.ToLower(c), q) {
			return true
		}
	}
	return false
}

// Width returns the current table width.
func (f *FilterTable) Width() int { return f.t.Width() }

// VisibleOrigins returns the original row index of each visible row.
func (f *FilterTable) VisibleOrigins() []int { return f.origIdx }

// SetCursor moves the visible cursor position.
func (f *FilterTable) SetCursor(i int) { f.t.SetCursor(i) }

// Update handles input for the table or its filter box.
func (f FilterTable) Update(msg tea.Msg) (FilterTable, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "/":
			if !f.filtering {
				return f, f.StartFilter()
			}
		case "enter":
			if f.filtering {
				f.AcceptFilter()
				return f, nil
			}
		case "esc":
			if f.filtering {
				f.CancelFilter()
				return f, nil
			}
			if f.filterStr != "" {
				// A committed filter that yields nothing must be
				// clearable with esc, or the list looks permanently
				// broken ("retyping never matches again").
				f.ClearFilter()
				return f, nil
			}
		case "g":
			if !f.filtering {
				f.t.GotoTop()
				return f, nil
			}
		case "G":
			if !f.filtering {
				f.t.GotoBottom()
				return f, nil
			}
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
		return f, cmd
	}
	var cmd tea.Cmd
	f.t, cmd = f.t.Update(msg)
	return f, cmd
}

// View renders the table, optionally with an inline filter bar.
func (f FilterTable) View() string {
	out := f.t.View()
	if f.filtering || f.filterStr != "" {
		bar := faintSty.Render("filter:")
		val := f.input.View()
		if !f.filtering {
			val = mutedSty.Render("/" + f.filterStr)
		} else {
			val = "/" + val
		}
		out = bar + " " + val + "\n" + out
	}
	return out
}

// Focus is a no-op kept for interface symmetry.
func (f *FilterTable) Focus() {}

// Blur clears focus state on the inner table.
func (f *FilterTable) Blur() {}
