package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"lcc2/internal/screens"
)

func newTestRoot(t *testing.T) (Root, tea.Model) {
	t.Helper()
	r := New(screens.NewOverview(), screens.NewProcesses(),
		screens.NewDisks(), screens.NewFiles(),
		screens.NewServices(), screens.NewUsersGroups())
	m, _ := r.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return r, m
}

// Tab cycles forward through the sections with wrap-around;
// shift+tab cycles backward.
func TestTabCyclesScreens(t *testing.T) {
	_, m := newTestRoot(t)
	root := m.(Root)

	for i := 1; i <= len(root.order); i++ {
		m, _ = root.Update(keyMsg("tab"))
		root = m.(Root)
		if got := root.active; got != i%len(root.order) {
			t.Fatalf("after tab %d: active=%d want %d", i, got, i%len(root.order))
		}
	}

	m, _ = root.Update(keyMsg("shift+tab"))
	root = m.(Root)
	if root.active != len(root.order)-1 {
		t.Fatalf("shift+tab from 0: active=%d want %d",
			root.active, len(root.order)-1)
	}
}

// Number keys still jump directly to a section.
func TestDigitJumpsToSection(t *testing.T) {
	_, m := newTestRoot(t)
	root := m.(Root)
	m, _ = root.Update(keyMsg("4"))
	root = m.(Root)
	if root.active != 3 {
		t.Fatalf("active=%d want 3", root.active)
	}
}

// While a screen captures input (filter, prompt), tab must not steal
// focus away — the keystroke belongs to the text field.
func TestTabSuppressedWhileCapturingInput(t *testing.T) {
	_, m := newTestRoot(t)
	root := m.(Root)
	m, _ = root.Update(keyMsg("4")) // files
	root = m.(Root)
	m, _ = root.Update(keyMsg("/")) // open filter → CapturingInput
	root = m.(Root)
	if !root.current().CapturingInput() {
		t.Fatal("files filter should capture input")
	}
	m, _ = root.Update(keyMsg("tab"))
	root = m.(Root)
	if root.active != 3 {
		t.Fatalf("tab stole focus while filtering: active=%d", root.active)
	}
}

func keyMsg(s string) tea.KeyMsg {
	if s == "shift+tab" {
		return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	}
	special := map[string]rune{
		"tab": tea.KeyTab,
		"enter": tea.KeyEnter, "esc": tea.KeyEsc,
		"up": tea.KeyUp, "down": tea.KeyDown,
		"home": tea.KeyHome, "end": tea.KeyEnd,
		"pgup": tea.KeyPgUp, "pgdown": tea.KeyPgDown,
	}
	if c, ok := special[s]; ok {
		return tea.KeyPressMsg{Code: c}
	}
	r := []rune(s)
	return tea.KeyPressMsg{Code: r[len(r)-1], Text: s}
}
