package ui

import tea "github.com/charmbracelet/bubbletea"

// keyMsg builds a KeyMsg from a literal rune for tests.
func keyMsg(s string) tea.Msg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}
