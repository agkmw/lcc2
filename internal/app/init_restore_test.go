package app

import (
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"lcc2/internal/ui"
)

// recScreen records how often Init fired.
type recScreen struct{ inits int }

func (s *recScreen) Init() tea.Cmd                         { s.inits++; return nil }
func (s *recScreen) Update(m tea.Msg) (ui.Screen, tea.Cmd) { return s, nil }
func (s *recScreen) View() string                          { return "" }
func (s *recScreen) ID() string                            { return "rec" }
func (s *recScreen) Title() string                         { return "Rec" }
func (s *recScreen) Hints() []key.Binding                  { return nil }
func (s *recScreen) CapturingInput() bool                  { return false }

// Session restore lands on a non-overview section: exactly that
// screen's Init must run at startup, or its data loop never starts
// and the section hangs on "Scanning.." forever.
func TestInitStartsRestoredScreen(t *testing.T) {
	a, b := &recScreen{}, &recScreen{}
	r := NewStartingAt(1, a, b)
	if r.Init() == nil {
		t.Fatal("Init returned no cmd")
	}
	if a.inits != 0 || b.inits != 1 {
		t.Fatalf("inits: active=%d inactive=%d, want 0/1", a.inits, b.inits)
	}
}
