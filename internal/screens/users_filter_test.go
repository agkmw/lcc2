package screens

import (
	"strings"
	"testing"
)

// Regression: the groups-tab filter guard routed keys to the users
// table, so typing in the groups filter fed the wrong input and the
// filter never received a character.
func TestGroupsFilterReceivesKeys(t *testing.T) {
	u := loadedUsers(true)
	s, _ := u.Update(runeKey("s")) // groups tab
	u = s.(UsersGroups)

	s, _ = u.Update(runeKey("/"))
	u = s.(UsersGroups)
	if !u.gTbl.Filtering() {
		t.Fatal("/ did not start the groups filter")
	}

	s, _ = u.Update(runeKey("w"))
	u = s.(UsersGroups)
	if !strings.Contains(u.gTbl.View(), "w") {
		t.Fatal("typed key did not reach the groups filter input")
	}
}
