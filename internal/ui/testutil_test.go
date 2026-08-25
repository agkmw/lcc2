package ui

import tea "charm.land/bubbletea/v2"

// keyMsg builds a KeyMsg from a literal rune for tests.
func keyMsg(s string) tea.Msg {
	r := []rune(s)
	return tea.KeyPressMsg{Code: r[len(r)-1], Text: s}
}
