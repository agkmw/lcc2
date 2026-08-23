package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lcc2/internal/screens"
)

// Chrome regression: every section view must be exactly h lines and
// never exceed w cells, at several window sizes — this is what keeps
// bubbletea's line-diff renderer from ghosting stale rows.
func TestSectionViewsAreWellFormed(t *testing.T) {
	sizes := [][2]int{{70, 20}, {84, 22}, {100, 30}, {140, 40}}
	for _, sz := range sizes {
		w, h := sz[0], sz[1]
		for key := byte('1'); key <= '6'; key++ {
			r := New(screens.NewOverview(), screens.NewProcesses(),
				screens.NewDisks(), screens.NewFiles(),
				screens.NewServices(), screens.NewUsersGroups())
			m, _ := r.Update(tea.WindowSizeMsg{Width: w, Height: h})
			m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{rune(key)}})
			lines := strings.Split(m.View(), "\n")
			if len(lines) != h {
				t.Errorf("w=%d h=%d sec=%c: %d lines (want %d)",
					w, h, key, len(lines), h)
			}
			for i, l := range lines {
				if lw := lipgloss.Width(l); lw > w {
					t.Errorf("w=%d h=%d sec=%c line %d: %d cells > %d",
						w, h, key, i, lw, w)
					break
				}
			}
		}
	}
}

// Modal states (prompt/confirm/filter) must keep the frame intact too.
func TestModalStatesKeepFrame(t *testing.T) {
	w, h := 90, 26
	states := []struct {
		name string
		sec  byte
		keys []tea.Msg
	}{
		{"files-prompt", '4', []tea.Msg{keyRune("m")}},
		{"files-confirm", '4', []tea.Msg{keyRune("d")}},
		{"proc-filter", '2', []tea.Msg{keyRune("/"), keyRune("s")}},
		{"svc-confirm", '5', []tea.Msg{keyRune("s")}},
	}
	for _, st := range states {
		r := New(screens.NewOverview(), screens.NewProcesses(),
			screens.NewDisks(), screens.NewFiles(),
			screens.NewServices(), screens.NewUsersGroups())
		m, _ := r.Update(tea.WindowSizeMsg{Width: w, Height: h})
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{rune(st.sec)}})
		for _, k := range st.keys {
			m, _ = m.Update(k)
		}
		lines := strings.Split(m.View(), "\n")
		if len(lines) != h {
			t.Errorf("%s: %d lines (want %d)", st.name, len(lines), h)
		}
		for i, l := range lines {
			if lw := lipgloss.Width(l); lw > w {
				t.Errorf("%s: line %d %d cells > %d", st.name, i, lw, w)
				break
			}
		}
	}
}

func keyRune(r string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(r)}
}
