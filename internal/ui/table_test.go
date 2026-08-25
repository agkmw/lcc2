package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func newTestTable() FilterTable {
	cols := []Column{{Title: "name", Width: 20}}
	ft := NewFilterTable(cols, 30, 5)
	ft.SetRows([]Row{
		{"alpha"}, {"beta"}, {"alphabet"}, {"gamma"},
	})
	return ft
}

// The table's own chrome must never emit East-Asian-Ambiguous glyphs
// (ADR-0010): the app-level denylist test only sees empty tables, so
// the stats line and filter placeholder escape it unless scanned here.
func TestTableChromeGlyphClean(t *testing.T) {
	deny := func(r rune) bool {
		return strings.ContainsRune("●○◌◐◑◕◉✕✗✔✖▸◂▴▾◄►◊◈◇◆•·‣›‹…—", r)
	}
	scan := func(name, s string) {
		t.Helper()
		for _, r := range stripSeq(s) {
			if deny(r) {
				t.Errorf("%s emits ambiguous glyph %q", name, string(r))
			}
		}
	}
	ft := newTestTable()
	scan("stats", ft.View())
	ft, _ = ft.Update(keyMsg("/"))
	scan("filter", ft.View())
}

func stripSeq(s string) string {
	var b strings.Builder
	esc := false
	for _, r := range s {
		switch {
		case r == '\x1b':
			esc = true
		case esc:
			if r == 'm' {
				esc = false
			}
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func typeString(ft *FilterTable, s string) {
	for _, r := range s {
		*ft, _ = ft.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

func backspaces(ft *FilterTable, n int) {
	for i := 0; i < n; i++ {
		*ft, _ = ft.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	}
}

// Regression: mistyping until zero matches left the inner table
// cursor clamped at -1 by bubbles. Every later index lookup then read
// origIdx[-1] (panic on enter) and the corrupted viewport made
// recovered results invisible — "retyping never matches again".
func TestFilterEmptyResultThenRecover(t *testing.T) {
	ft := newTestTable()
	ft, _ = ft.Update(keyMsg("/")) // enter filter mode

	typeString(&ft, "zzz") // mistype into a dead end
	if ft.Len() != 0 {
		t.Fatalf("expected zero matches, got %d", ft.Len())
	}
	if _, ok := ft.Selected(); ok {
		t.Fatal("empty result must not report a selection")
	}

	// delete the mistake one char at a time; matches must reappear
	// as soon as the query is valid again ("a" matches 2 rows)
	backspaces(&ft, 3)
	if ft.Len() != 4 { // empty query shows everything again
		t.Fatalf("empty query should show all rows, got %d", ft.Len())
	}
	idx, ok := ft.Selected()
	if !ok || idx < 0 || idx >= len(ft.Rows()) {
		t.Fatalf("selection broken after recovery: idx=%d ok=%v", idx, ok)
	}

	// enter (accept) and enter (open) must be safe in every state
	ft, _ = ft.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	ft, _ = ft.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	// narrow to a real match and select it end-to-end
	ft, _ = ft.Update(keyMsg("/"))
	typeString(&ft, "alp")
	if ft.Len() != 2 {
		t.Fatalf("query 'alp' should match 2 rows, got %d", ft.Len())
	}
	ft, _ = ft.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	idx, ok = ft.Selected()
	if !ok || (ft.Rows()[idx][0] != "alpha" && ft.Rows()[idx][0] != "alphabet") {
		t.Fatalf("selection after narrowed filter wrong: idx=%d ok=%v", idx, ok)
	}
}

// A committed-but-empty filter state must also be recoverable via esc.
func TestFilterAcceptEmptyThenEscape(t *testing.T) {
	ft := newTestTable()
	ft, _ = ft.Update(keyMsg("/"))
	typeString(&ft, "qq")
	ft, _ = ft.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // commit empty result
	if ft.Len() != 0 {
		t.Fatalf("committed query should stay empty, got %d", ft.Len())
	}
	ft, _ = ft.Update(keyMsg("esc")) // cancel clears the filter
	if ft.Len() != 4 {
		t.Fatalf("esc after empty committed filter should restore all rows, got %d", ft.Len())
	}
	if _, ok := ft.Selected(); !ok {
		t.Fatal("selection must be usable right after recovery")
	}
}

func TestSelectedRejectsNegativeCursor(t *testing.T) {
	ft := newTestTable()
	ft.SetRows([]Row{}) // forces bubbles' cursor clamp to -1
	if _, ok := ft.Selected(); ok {
		t.Fatal("negative cursor must not yield a selection")
	}
	if !strings.Contains(ft.View(), "name") {
		t.Fatal("view should still render header")
	}
}

// BACKLOG-T1: the cursor must follow the same logical item across
// row rebuilds (refresh churn, deletions, sort changes) and filters.
func TestTrackingFollowsItemThroughDelete(t *testing.T) {
	ft := NewFilterTable([]Column{{Title: "name", Width: 20}}, 30, 5)
	ft.SetRowsTracked(rows5(), []string{"a", "b", "c", "d", "e"})
	ft.SetCursor(2)
	if k, _ := ft.SelectedKey(); k != "c" {
		t.Fatalf("setup wrong: key=%q", k)
	}

	// delete an item BEFORE the selection: index shifts, identity holds
	ft.SetRowsTracked([]Row{{"alpha"}, {"gamma"}, {"delta"}, {"echo"}},
		[]string{"a", "c", "d", "e"})
	if k, ok := ft.SelectedKey(); !ok || k != "c" {
		t.Fatalf("selection lost after delete-shift: %q %v", k, ok)
	}

	// delete the tracked item itself: cursor lands on nearest survivor
	ft.SetRowsTracked([]Row{{"alpha"}, {"delta"}, {"echo"}},
		[]string{"a", "d", "e"})
	if _, ok := ft.SelectedKey(); !ok {
		t.Fatal("no selection after tracking target vanished")
	}
}

func TestTrackingThroughFilterNarrowAndWiden(t *testing.T) {
	ft := NewFilterTable([]Column{{Title: "name", Width: 20}}, 30, 8)
	ft.SetRowsTracked(rows5(), []string{"a", "b", "c", "d", "e"})
	ft.SetCursor(3) // delta

	ft, _ = ft.Update(keyMsg("/"))
	typeString(&ft, "del")
	ft, _ = ft.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // commit filter
	if k, _ := ft.SelectedKey(); k != "d" {
		t.Fatalf("filter lost selection: %q", k)
	}

	ft.ClearFilter()
	if ft.Len() != 5 {
		t.Fatalf("widen failed: %d", ft.Len())
	}
	if k, _ := ft.SelectedKey(); k != "d" {
		t.Fatalf("clear-filter lost selection: %q", k)
	}
}

func rows5() []Row {
	return []Row{
		{"alpha"}, {"beta"}, {"gamma"}, {"delta"}, {"echo"},
	}
}

// Regression (H8): bubbles/table truncated cells by rune count, so
// styled cells had to be stripped to plain text before rendering —
// which drained ALL color from the main panes. The first-party
// renderer must keep styled cells intact at every width.
func TestStyledCellsKeepColorAtAnyWidth(t *testing.T) {
	forceTrueColor(t)
	cols := []Column{{Title: "name", Width: 30}, {Title: "pct", Width: 6}}
	styled := lipgloss.NewStyle().Bold(true).Foreground(Palette.Red)
	rows := []Row{
		{styled.Render("a-very-long-directory-name-indeed/"), styled.Render("99%")},
		{"plain-but-long-plaintext-name-here.txt", "0.0"},
	}
	ft := NewFilterTable(cols, 80, 4)
	ft.SetRowsTracked(rows, []string{"a", "b"})
	for _, w := range []int{60, 40, 26, 18} {
		ft.SetSize(w, 4)
		view := ft.View()
		if !strings.Contains(view, ";38;2;") {
			t.Fatalf("w=%d: styled cell lost its color", w)
		}
		for i, l := range strings.Split(view, "\n") {
			if lw := lipgloss.Width(l); lw > w {
				t.Fatalf("w=%d: line %d is %d cells (cap %d)", w, i, lw, w)
			}
			if !balancedEscapes(l) {
				t.Fatalf("w=%d: line %d has a sliced escape sequence: %q", w, i, l)
			}
		}
	}
}

// The built-in scroller keeps the cursor row visible through moves,
// page jumps and list shrinks.
func TestTableViewportKeepsCursorVisible(t *testing.T) {
	cols := []Column{{Title: "name", Width: 20}}
	rows := make([]Row, 30)
	keys := make([]string, 30)
	for i := range rows {
		rows[i] = Row{fmt.Sprintf("row%02d", i)}
		keys[i] = fmt.Sprintf("k%02d", i)
	}
	ft := NewFilterTable(cols, 24, 9) // bodyH = 9-3 = 6
	ft.SetRowsTracked(rows, keys)

	down := tea.KeyPressMsg{Code: tea.KeyDown}
	for i := 0; i < 10; i++ { // walk past the window edge
		ft, _ = ft.Update(down)
	}
	if ft.cursor != 10 {
		t.Fatalf("cursor = %d, want 10", ft.cursor)
	}
	if !strings.Contains(ft.View(), "row10") {
		t.Error("cursor row scrolled out of view")
	}

	end := tea.KeyPressMsg{Code: tea.KeyEnd}
	ft, _ = ft.Update(end)
	v := ft.View()
	if ft.cursor != 29 || !strings.Contains(v, "row29") {
		t.Errorf("goto-bottom failed: cursor=%d view=%q", ft.cursor, v)
	}

	home := tea.KeyPressMsg{Code: tea.KeyHome}
	ft, _ = ft.Update(home)
	if ft.cursor != 0 || !strings.Contains(ft.View(), "row00") {
		t.Error("goto-top failed")
	}

	// Shrinking the list under the cursor clamps instead of panicking.
	ft.SetRowsTracked(rows[:4], keys[:4])
	if c := ft.Cursor(); c > 3 {
		t.Errorf("cursor %d out of range after shrink", c)
	}
}

// balancedEscapes reports whether every ESC[ sequence in s reaches a
// proper final byte instead of being cut at a cell boundary.
func balancedEscapes(s string) bool {
	for i := 0; i < len(s); {
		if s[i] != '\x1b' {
			i++
			continue
		}
		j := i + 1
		if j < len(s) && s[j] == '[' {
			j++
		}
		for j < len(s) && (s[j] == ';' || (s[j] >= '0' && s[j] <= '9')) {
			j++
		}
		if j >= len(s) || s[j] < 0x40 || s[j] > 0x7e {
			return false
		}
		i = j + 1
	}
	return true
}

// Mouse: wheel moves the cursor, clicks select, RowAt accounts for
// the scroll offset.
func TestTableMouse(t *testing.T) {
	cols := []Column{{Title: "name", Width: 12}}
	rows := make([]Row, 20)
	keys := make([]string, 20)
	for i := range rows {
		rows[i] = Row{fmt.Sprintf("row%02d", i)}
		keys[i] = fmt.Sprintf("k%02d", i)
	}
	ft := NewFilterTable(cols, 16, 9) // bodyH = 6
	ft.SetRowsTracked(rows, keys)

	wheel := tea.MouseWheelMsg{Button: tea.MouseWheelDown}
	for i := 0; i < 8; i++ {
		ft.Mouse(wheel, 0)
	}
	if ft.Cursor() != 8 {
		t.Fatalf("wheel cursor = %d, want 8", ft.Cursor())
	}
	if ft.yOff != 3 { // 8 - bodyH(6) + 1
		t.Fatalf("yOff = %d, want 3", ft.yOff)
	}

	// Click on the third visible row: abs y = header(2) + (row - yOff).
	click := tea.MouseClickMsg{Button: tea.MouseLeft, X: 5, Y: 2 + 2} // vis row 5
	moved, dbl := ft.Mouse(click, 0)
	if !moved || ft.Cursor() != 5 {
		t.Fatalf("click moved=%v cursor=%d", moved, ft.Cursor())
	}

	// Double-click the same row within the window.
	_, dbl = ft.Mouse(click, 0)
	if !dbl {
		t.Error("second click not detected as double-click")
	}

	// Header and rule rows are never hits.
	if _, ok := ft.RowAt(0); ok {
		t.Error("header text row should not map to a data row")
	}
}
