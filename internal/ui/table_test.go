package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

func newTestTable() FilterTable {
	cols := []table.Column{{Title: "name", Width: 20}}
	ft := NewFilterTable(cols, 30, 5)
	ft.SetRows([]table.Row{
		{"alpha"}, {"beta"}, {"alphabet"}, {"gamma"},
	})
	return ft
}

func typeString(ft *FilterTable, s string) {
	for _, r := range s {
		*ft, _ = ft.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

func backspaces(ft *FilterTable, n int) {
	for i := 0; i < n; i++ {
		*ft, _ = ft.Update(tea.KeyMsg{Type: tea.KeyBackspace})
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
	ft, _ = ft.Update(tea.KeyMsg{Type: tea.KeyEnter})
	ft, _ = ft.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// narrow to a real match and select it end-to-end
	ft, _ = ft.Update(keyMsg("/"))
	typeString(&ft, "alp")
	if ft.Len() != 2 {
		t.Fatalf("query 'alp' should match 2 rows, got %d", ft.Len())
	}
	ft, _ = ft.Update(tea.KeyMsg{Type: tea.KeyEnter})
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
	ft, _ = ft.Update(tea.KeyMsg{Type: tea.KeyEnter}) // commit empty result
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
	ft.SetRows([]table.Row{}) // forces bubbles' cursor clamp to -1
	if _, ok := ft.Selected(); ok {
		t.Fatal("negative cursor must not yield a selection")
	}
	if !strings.Contains(ft.View(), "name") {
		t.Fatal("view should still render header")
	}
}
