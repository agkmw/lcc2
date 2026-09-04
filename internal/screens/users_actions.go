package screens

import (
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"lcc2/internal/accounts"
	"lcc2/internal/ui"
)

// Mutating actions for the users & groups section. Every action is
// gated on root, collected through a Form or ConfirmDialog, and run
// off the UI thread; results land as userActionMsg toasts.

// pendingOp is a confirmed, ready-to-run mutation.
type pendingOp struct {
	kind string // lock, unlock, deluser, delgroup, delmember
	name string // user or group name
	arg  string // member name; "home" for userdel -r
}

// userActionMsg reports a finished mutation.
type userActionMsg struct {
	label string
	err   error
}

// runUserAction executes a confirmed mutation off the UI thread.
func runUserAction(p pendingOp) tea.Cmd {
	return func() tea.Msg {
		switch p.kind {
		case "lock":
			return userActionMsg{label: "locked " + p.name, err: accounts.LockUser(p.name)}
		case "unlock":
			return userActionMsg{label: "unlocked " + p.name, err: accounts.UnlockUser(p.name)}
		case "deluser":
			return userActionMsg{label: "deleted user " + p.name,
				err: accounts.DeleteUser(p.name, p.arg == "home")}
		case "delgroup":
			return userActionMsg{label: "deleted group " + p.name, err: accounts.DeleteGroup(p.name)}
		case "delmember":
			return userActionMsg{label: "removed " + p.arg + " from " + p.name,
				err: accounts.RemoveMember(p.name, p.arg)}
		}
		return userActionMsg{label: "?"}
	}
}

// requireRoot blocks mutating actions when the app runs unprivileged.
func (u UsersGroups) requireRoot() (ui.Screen, tea.Cmd, bool) {
	if u.root {
		return u, nil, true
	}
	return u, ui.ErrToast("needs root - restart the app with sudo"), false
}

func (u UsersGroups) selectedUser() (accounts.User, bool) {
	idx, ok := u.uTbl.Selected()
	if !ok || idx >= len(u.users) {
		return accounts.User{}, false
	}
	return u.users[idx], true
}

func (u UsersGroups) selectedGroup() (accounts.Group, bool) {
	idx, ok := u.gTbl.Selected()
	if !ok || idx >= len(u.groups) {
		return accounts.Group{}, false
	}
	return u.groups[idx], true
}

func (u UsersGroups) formWidth() int { return clampInt(u.w-8, 48, 64) }

// --- users tab -------------------------------------------------------

func (u UsersGroups) openCreateUser() (ui.Screen, tea.Cmd) {
	if s, cmd, ok := u.requireRoot(); !ok {
		return s, cmd
	}
	f, cmd := ui.NewForm("new user", "users", []ui.Field{
		{Label: "name", Placeholder: "login name"},
		{Label: "shell", Placeholder: "/bin/bash"},
		{Label: "home", Placeholder: "/home/name"},
		{Label: "primary", Placeholder: "own group by default"},
		{Label: "groups", Placeholder: "wheel,docker"},
		{Label: "system", Placeholder: "y = system account"},
	})
	f.SetWidth(u.formWidth())
	u.form = &f
	u.formKind = "newuser"
	return u, cmd
}

func (u UsersGroups) openEditUser() (ui.Screen, tea.Cmd) {
	if s, cmd, ok := u.requireRoot(); !ok {
		return s, cmd
	}
	usr, ok := u.selectedUser()
	if !ok {
		return u, nil
	}
	primary := ""
	for _, g := range u.groups {
		if g.GID == usr.GID {
			primary = g.Name
			break
		}
	}
	f, cmd := ui.NewForm("edit user", "users", []ui.Field{
		{Label: "shell", Value: usr.Shell},
		{Label: "home", Value: usr.Home},
		{Label: "primary", Value: primary},
		{Label: "groups", Value: strings.Join(accounts.DirectMemberOf(usr.Name, u.groups), ","),
			Placeholder: "supplementary, comma list"},
	})
	f.SetWidth(u.formWidth())
	u.form = &f
	u.formKind = "edituser"
	u.formTgt = usr.Name
	return u, cmd
}

func (u UsersGroups) openPassword() (ui.Screen, tea.Cmd) {
	if s, cmd, ok := u.requireRoot(); !ok {
		return s, cmd
	}
	usr, ok := u.selectedUser()
	if !ok {
		return u, nil
	}
	f, cmd := ui.NewForm("set password", "users", []ui.Field{
		{Label: "password", Password: true, Placeholder: "new password"},
		{Label: "confirm", Password: true, Placeholder: "repeat it"},
	})
	f.SetWidth(u.formWidth())
	u.form = &f
	u.formKind = "password"
	u.formTgt = usr.Name
	return u, cmd
}

func (u UsersGroups) askLock(lock bool) (ui.Screen, tea.Cmd) {
	if s, cmd, ok := u.requireRoot(); !ok {
		return s, cmd
	}
	usr, ok := u.selectedUser()
	if !ok {
		return u, nil
	}
	verb, kind := "Unlock", "unlock"
	if lock {
		verb, kind = "Lock", "lock"
	}
	dlg := ui.NewConfirm(verb+" user", verb+" "+usr.Name+"?", "")
	dlg.Danger = lock
	dlg.SetWidth(clampInt(u.w-8, 44, 70))
	u.confirm = &dlg
	u.pending = pendingOp{kind: kind, name: usr.Name}
	return u, nil
}

func (u UsersGroups) askDeleteUser(removeHome bool) (ui.Screen, tea.Cmd) {
	if s, cmd, ok := u.requireRoot(); !ok {
		return s, cmd
	}
	usr, ok := u.selectedUser()
	if !ok {
		return u, nil
	}
	body, arg := "Delete user "+usr.Name+"? home directory kept", ""
	if removeHome {
		body, arg = "Delete user "+usr.Name+" AND the home directory?", "home"
	}
	dlg := ui.NewConfirm("Delete user", body, "")
	dlg.SetWidth(clampInt(u.w-8, 44, 70))
	u.confirm = &dlg
	u.pending = pendingOp{kind: "deluser", name: usr.Name, arg: arg}
	return u, nil
}

func (u UsersGroups) openExpiry() (ui.Screen, tea.Cmd) {
	if s, cmd, ok := u.requireRoot(); !ok {
		return s, cmd
	}
	usr, ok := u.selectedUser()
	if !ok {
		return u, nil
	}
	f, cmd := ui.NewForm("account expiry", "users", []ui.Field{
		{Label: "expires", Placeholder: "YYYY-MM-DD, empty = never"},
	})
	f.SetWidth(u.formWidth())
	u.form = &f
	u.formKind = "expiry"
	u.formTgt = usr.Name
	return u, cmd
}

// --- groups tab ------------------------------------------------------

func (u UsersGroups) openCreateGroup() (ui.Screen, tea.Cmd) {
	if s, cmd, ok := u.requireRoot(); !ok {
		return s, cmd
	}
	f, cmd := ui.NewForm("new group", "users", []ui.Field{
		{Label: "name", Placeholder: "group name"},
		{Label: "gid", Placeholder: "empty = auto"},
	})
	f.SetWidth(u.formWidth())
	u.form = &f
	u.formKind = "newgroup"
	return u, cmd
}

func (u UsersGroups) askDeleteGroup() (ui.Screen, tea.Cmd) {
	if s, cmd, ok := u.requireRoot(); !ok {
		return s, cmd
	}
	g, ok := u.selectedGroup()
	if !ok {
		return u, nil
	}
	dlg := ui.NewConfirm("Delete group", "Delete group "+g.Name+"?", "")
	dlg.SetWidth(clampInt(u.w-8, 44, 70))
	u.confirm = &dlg
	u.pending = pendingOp{kind: "delgroup", name: g.Name}
	return u, nil
}

func (u UsersGroups) openMember(add bool) (ui.Screen, tea.Cmd) {
	if s, cmd, ok := u.requireRoot(); !ok {
		return s, cmd
	}
	g, ok := u.selectedGroup()
	if !ok {
		return u, nil
	}
	kind, title := "delmember", "remove member"
	if add {
		kind, title = "addmember", "add member"
	}
	f, cmd := ui.NewForm(title, "users", []ui.Field{
		{Label: "user", Placeholder: "login name"},
	})
	f.SetWidth(u.formWidth())
	u.form = &f
	u.formKind = kind
	u.formTgt = g.Name
	return u, cmd
}

// --- form submission -------------------------------------------------

// submitForm turns a submitted form into an action command (or, for
// membership removal, a confirm dialog).
func (u UsersGroups) submitForm() (ui.Screen, tea.Cmd) {
	vals := u.form.Values()
	u.form = nil
	switch u.formKind {
	case "newuser":
		name := vals[0]
		if !accounts.ValidName(name) {
			return u, ui.ErrToast("new user: invalid name " + strconv.Quote(name))
		}
		o := accounts.UserOpts{
			Name:         name,
			Shell:        vals[1],
			Home:         vals[2],
			PrimaryGroup: vals[3],
			Groups:       splitList(vals[4]),
			System:       vals[5] == "y" || vals[5] == "yes",
		}
		return u, func() tea.Msg {
			return userActionMsg{label: "created user " + name, err: accounts.CreateUser(o)}
		}

	case "edituser":
		name := u.formTgt
		shell, home, primary := vals[0], vals[1], vals[2]
		groups := splitList(vals[3])
		// Skip no-change execs against the loaded snapshot.
		if cur, ok := u.lookupUser(name); ok {
			if shell == cur.Shell {
				shell = ""
			}
			if home == cur.Home {
				home = ""
			}
			for _, g := range u.groups {
				if g.GID == cur.GID && g.Name == primary {
					primary = ""
				}
			}
		}
		return u, func() tea.Msg {
			if err := accounts.ModifyUser(name, shell, home); err != nil {
				return userActionMsg{label: "edit " + name, err: err}
			}
			if err := accounts.SetGroups(name, primary, groups); err != nil {
				return userActionMsg{label: "groups " + name, err: err}
			}
			return userActionMsg{label: "updated " + name}
		}

	case "password":
		name := u.formTgt
		if vals[0] == "" {
			return u, ui.ErrToast("password: empty")
		}
		if vals[0] != vals[1] {
			return u, ui.ErrToast("passwords do not match")
		}
		return u, func() tea.Msg {
			return userActionMsg{label: "password set for " + name,
				err: accounts.SetPassword(name, vals[0])}
		}

	case "expiry":
		name, date := u.formTgt, vals[0]
		return u, func() tea.Msg {
			err := accounts.SetExpiry(name, date)
			if err == nil && date == "" {
				return userActionMsg{label: "cleared expiry for " + name}
			}
			return userActionMsg{label: "expiry " + date + " for " + name, err: err}
		}

	case "newgroup":
		name := vals[0]
		if !accounts.ValidName(name) {
			return u, ui.ErrToast("new group: invalid name " + strconv.Quote(name))
		}
		gid := 0
		if vals[1] != "" {
			n, err := strconv.Atoi(vals[1])
			if err != nil || n < 1 {
				return u, ui.ErrToast("bad gid " + strconv.Quote(vals[1]))
			}
			gid = n
		}
		return u, func() tea.Msg {
			return userActionMsg{label: "created group " + name,
				err: accounts.CreateGroup(name, gid)}
		}

	case "addmember", "delmember":
		group, member := u.formTgt, vals[0]
		if !accounts.ValidName(member) {
			return u, ui.ErrToast("invalid user name " + strconv.Quote(member))
		}
		if u.formKind == "addmember" {
			return u, func() tea.Msg {
				return userActionMsg{label: member + " -> " + group,
					err: accounts.AddMember(group, member)}
			}
		}
		dlg := ui.NewConfirm("Remove member", "Remove "+member+" from "+group+"?", "")
		dlg.SetWidth(clampInt(u.w-8, 44, 70))
		u.confirm = &dlg
		u.pending = pendingOp{kind: "delmember", name: group, arg: member}
		return u, nil
	}
	return u, nil
}

func (u UsersGroups) lookupUser(name string) (accounts.User, bool) {
	for _, usr := range u.users {
		if usr.Name == name {
			return usr, true
		}
	}
	return accounts.User{}, false
}

// splitList parses a comma-separated group list, dropping empties.
func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
