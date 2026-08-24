package screens

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Regression: "tab" is pinned globally to screen cycling (root
// intercepts it), so the users/groups list switch moved to "s" — the
// old binding could never fire.
func TestUsersSwitchKeySwapsLists(t *testing.T) {
	u := NewUsersGroups()
	if u.tab != "users" {
		t.Fatalf("initial tab = %q", u.tab)
	}
	sKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	u = feed(u, sKey).(UsersGroups)
	if u.tab != "groups" {
		t.Fatalf("s did not switch lists, tab = %q", u.tab)
	}
	u = feed(u, sKey).(UsersGroups)
	if u.tab != "users" {
		t.Fatalf("s did not switch back, tab = %q", u.tab)
	}
}
