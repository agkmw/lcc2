package screens

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"lcc/internal/accounts"
	"lcc/internal/ui"
)

func runeKey(s string) tea.KeyPressMsg {
	r := []rune(s)
	return tea.KeyPressMsg{Code: r[len(r)-1], Text: s}
}

func loadedUsers(rooted bool) UsersGroups {
	u := NewUsersGroups()
	u.root = rooted
	u.w, u.h = 100, 30
	u.loaded = true
	u.users = []accounts.User{
		{Name: "root", UID: 0, GID: 0, Shell: "/bin/bash", Home: "/root"},
		{Name: "alice", UID: 1500, GID: 1500, Shell: "/bin/bash", Home: "/home/alice"},
		{Name: "svc", UID: 900, GID: 900, Shell: "/usr/sbin/nologin", Home: "/"},
	}
	u.groups = []accounts.Group{
		{Name: "wheel", GID: 10, Members: []string{"alice"}},
		{Name: "alice", GID: 1500},
		{Name: "devel", GID: 2000},
	}
	urows := make([]ui.Row, len(u.users))
	ukeys := make([]string, len(u.users))
	for i, usr := range u.users {
		urows[i] = ui.Row{usr.Name, "", "", "", ""}
		ukeys[i] = usr.Name
	}
	u.uTbl.SetRowsTracked(urows, ukeys)
	grows := make([]ui.Row, len(u.groups))
	gkeys := make([]string, len(u.groups))
	for i, g := range u.groups {
		grows[i] = ui.Row{g.Name, "", ""}
		gkeys[i] = g.Name
	}
	u.gTbl.SetRowsTracked(grows, gkeys)
	u.layout()
	return u
}

func toastOf(t *testing.T, cmd tea.Cmd) ui.ToastMsg {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected a toast cmd, got nil")
	}
	msg, ok := cmd().(ui.ToastMsg)
	if !ok {
		t.Fatalf("cmd produced %T, want ui.ToastMsg", cmd())
	}
	return msg
}

func TestUsersActionsNotRootGated(t *testing.T) {
	u := loadedUsers(false)
	for _, k := range []string{"n", "e", "p", "L", "u", "x", "E"} {
		s, cmd := u.Update(runeKey(k))
		u = s.(UsersGroups)
		if u.form != nil || u.confirm != nil {
			t.Fatalf("key %q opened a dialog without root", k)
		}
		msg := toastOf(t, cmd)
		if msg.Kind != "err" || !strings.Contains(msg.Text, "root") {
			t.Fatalf("key %q toast = %+v, want root error", k, msg)
		}
	}
	// group tab keys gate too
	s, _ := u.Update(runeKey("s"))
	u = s.(UsersGroups)
	for _, k := range []string{"n", "x", "a", "d"} {
		sc, cmd := u.Update(runeKey(k))
		u = sc.(UsersGroups)
		if u.form != nil || u.confirm != nil {
			t.Fatalf("group key %q opened a dialog without root", k)
		}
		toastOf(t, cmd)
	}
}

func TestUsersCreateUserForm(t *testing.T) {
	u := loadedUsers(true)
	s, cmd := u.Update(runeKey("n"))
	u = s.(UsersGroups)
	if u.form == nil || u.formKind != "newuser" {
		t.Fatalf("n did not open the new-user form: %+v", u.formKind)
	}
	if cmd == nil {
		t.Fatal("form focus cmd missing")
	}
	if !u.CapturingInput() {
		t.Fatal("form open but CapturingInput false")
	}
	// invalid name -> error toast, form closed
	s, _ = u.Update(runeKey("-"))
	u = s.(UsersGroups)
	sc, cmd2 := u.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	u = sc.(UsersGroups)
	if u.form != nil {
		t.Fatal("form still open after submit")
	}
	if msg := toastOf(t, cmd2); msg.Kind != "err" {
		t.Fatalf("invalid name toast = %+v", msg)
	}
}

func TestUsersFormEscCancels(t *testing.T) {
	u := loadedUsers(true)
	s, _ := u.Update(runeKey("n"))
	u = s.(UsersGroups)
	if u.form == nil {
		t.Fatal("form did not open")
	}
	s, _ = u.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	u = s.(UsersGroups)
	if u.form != nil {
		t.Fatal("esc did not close the form")
	}
	if u.CapturingInput() {
		t.Fatal("CapturingInput still true after esc")
	}
}

func TestUsersFormRendersOverList(t *testing.T) {
	u := loadedUsers(true)
	s, _ := u.Update(runeKey("n"))
	u = s.(UsersGroups)
	view := u.View()
	if !strings.Contains(view, "new user") {
		t.Fatal("form title missing from view")
	}
	lines := strings.Split(view, "\n")
	if len(lines) != u.h {
		t.Fatalf("view = %d lines, want %d", len(lines), u.h)
	}
}

func TestUsersPasswordMismatch(t *testing.T) {
	u := loadedUsers(true)
	// cursor on alice (row 1)
	u.uTbl.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	s, _ := u.Update(runeKey("p"))
	u = s.(UsersGroups)
	if u.form == nil || u.formKind != "password" || u.formTgt != "alice" {
		t.Fatalf("p did not open the password form for alice: %q %q", u.formKind, u.formTgt)
	}
	u.Update(runeKey("a")) // field 1: "a"
	u.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	u.Update(runeKey("b")) // field 2: "b"
	s, cmd := u.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	u = s.(UsersGroups)
	if u.form != nil {
		t.Fatal("form still open after mismatch")
	}
	msg := toastOf(t, cmd)
	if msg.Kind != "err" || !strings.Contains(msg.Text, "match") {
		t.Fatalf("mismatch toast = %+v", msg)
	}
}

func TestUsersLockConfirmFlow(t *testing.T) {
	u := loadedUsers(true)
	u.uTbl.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // alice
	s, _ := u.Update(runeKey("L"))
	u = s.(UsersGroups)
	if u.confirm == nil || u.pending.kind != "lock" || u.pending.name != "alice" {
		t.Fatalf("L did not stage a lock confirm: %+v", u.pending)
	}
	if !u.CapturingInput() {
		t.Fatal("confirm open but CapturingInput false")
	}
	s, cmd := u.Update(runeKey("y"))
	u = s.(UsersGroups)
	if u.confirm != nil {
		t.Fatal("confirm still open after y")
	}
	if cmd == nil {
		t.Fatal("y did not produce an action cmd")
	}
	// The cmd would exec usermod; correctness of argv is covered by
	// the accounts package tests, so it is not run here.
}

func TestUsersConfirmDecline(t *testing.T) {
	u := loadedUsers(true)
	u.uTbl.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // alice
	sc, _ := u.Update(runeKey("x"))
	u = sc.(UsersGroups)
	if u.confirm == nil {
		t.Fatal("x did not open the delete confirm")
	}
	s, cmd := u.Update(runeKey("n"))
	u = s.(UsersGroups)
	if u.confirm != nil {
		t.Fatal("confirm still open after n")
	}
	if cmd != nil {
		t.Fatal("decline produced a cmd")
	}
}

func TestUsersEditPrefillsCurrentValues(t *testing.T) {
	u := loadedUsers(true)
	u.uTbl.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // alice
	s, _ := u.Update(runeKey("e"))
	u = s.(UsersGroups)
	if u.form == nil || u.formKind != "edituser" || u.formTgt != "alice" {
		t.Fatalf("e did not open the edit form: %q", u.formKind)
	}
	vals := u.form.Values()
	if vals[0] != "/bin/bash" || vals[1] != "/home/alice" || vals[2] != "alice" {
		t.Fatalf("prefills = %v", vals)
	}
}

func TestGroupsTabActions(t *testing.T) {
	u := loadedUsers(true)
	s, _ := u.Update(runeKey("s")) // groups tab
	u = s.(UsersGroups)
	if u.tab != "groups" {
		t.Fatalf("tab = %q", u.tab)
	}
	s, _ = u.Update(runeKey("n"))
	u = s.(UsersGroups)
	if u.form == nil || u.formKind != "newgroup" {
		t.Fatalf("n did not open the group form: %q", u.formKind)
	}
	s, _ = u.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	u = s.(UsersGroups)

	s, _ = u.Update(runeKey("a"))
	u = s.(UsersGroups)
	if u.formKind != "addmember" || u.formTgt != "wheel" {
		t.Fatalf("a did not target wheel: %q %q", u.formKind, u.formTgt)
	}
	s, _ = u.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	u = s.(UsersGroups)

	s, _ = u.Update(runeKey("d"))
	u = s.(UsersGroups)
	if u.formKind != "delmember" {
		t.Fatalf("d did not open remove-member: %q", u.formKind)
	}
	u.Update(runeKey("a")) // member name "a" (form is pointer-shared)
	s2, _ := u.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	u = s2.(UsersGroups)
	// removal routes through a confirm, not straight to exec
	if u.confirm == nil || u.pending.arg != "a" {
		t.Fatalf("remove member did not confirm: %+v", u.pending)
	}
}

func TestUsersPreviewShowsSudo(t *testing.T) {
	u := loadedUsers(true)
	body := u.previewBody()
	if !strings.Contains(body, "sudo") {
		t.Fatal("user preview lacks the sudo line")
	}
	// alice is in wheel -> the yes form
	u.uTbl.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	body = u.previewBody()
	if !strings.Contains(body, "yes") {
		t.Fatalf("alice's sudo flag missing: %s", body)
	}
}

func TestUsersHeadNotRootBadge(t *testing.T) {
	u := loadedUsers(false)
	if !strings.Contains(u.View(), "read-only") {
		t.Fatal("unprivileged head lacks the read-only badge")
	}
	u2 := loadedUsers(true)
	if strings.Contains(u2.View(), "read-only") {
		t.Fatal("root view carries the read-only badge")
	}
}
