package accounts

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// dateRe accepts YYYY-MM-DD for chage -E; empty (clear) is handled
// before the check.
var dateRe = regexp.MustCompile(`^[12]\d{3}-\d{2}-\d{2}$`)

// UserOpts configures CreateUser.
type UserOpts struct {
	Name         string
	Shell        string // empty = useradd default
	Home         string // empty = useradd default
	PrimaryGroup string // empty = useradd default (own matching group)
	Groups       []string
	System       bool
}

// CreateUser adds a local account via useradd. Human accounts get a
// home directory; system accounts do not.
func CreateUser(o UserOpts) error {
	if err := checkRoot(); err != nil {
		return err
	}
	if err := validName(o.Name); err != nil {
		return fmt.Errorf("%w: %q", ErrBadName, o.Name)
	}
	if err := validNames(o.Groups); err != nil {
		return err
	}
	if o.PrimaryGroup != "" {
		if err := validName(o.PrimaryGroup); err != nil {
			return fmt.Errorf("%w: %q", ErrBadName, o.PrimaryGroup)
		}
	}
	us, err := Users()
	if err != nil {
		return err
	}
	if _, ok := findUser(us, o.Name); ok {
		return fmt.Errorf("user %s already exists", o.Name)
	}
	args := []string{}
	if o.System {
		args = append(args, "-r")
	} else {
		args = append(args, "-m")
	}
	if o.Shell != "" {
		args = append(args, "-s", o.Shell)
	}
	if o.Home != "" {
		args = append(args, "-d", o.Home)
	}
	if o.PrimaryGroup != "" {
		args = append(args, "-g", o.PrimaryGroup)
	}
	if len(o.Groups) > 0 {
		args = append(args, "-G", strings.Join(o.Groups, ","))
	}
	return run("useradd", append(args, "--", o.Name), "")
}

// DeleteUser removes a local account via userdel; root and system
// accounts are refused.
func DeleteUser(name string, removeHome bool) error {
	if err := checkRoot(); err != nil {
		return err
	}
	if err := validName(name); err != nil {
		return err
	}
	us, err := Users()
	if err != nil {
		return err
	}
	return deleteUser(us, name, removeHome)
}

func deleteUser(us []User, name string, removeHome bool) error {
	u, ok := findUser(us, name)
	if !ok {
		return fmt.Errorf("user %s not found", name)
	}
	if u.UID < 1000 {
		return ErrSystemAccount
	}
	args := []string{}
	if removeHome {
		args = append(args, "-r")
	}
	return run("userdel", append(args, "--", name), "")
}

// ModifyUser changes shell and/or home (empty = unchanged). The home
// path is re-pointed only; files are not moved.
func ModifyUser(name, shell, home string) error {
	if err := checkRoot(); err != nil {
		return err
	}
	if err := validName(name); err != nil {
		return err
	}
	us, err := Users()
	if err != nil {
		return err
	}
	if _, ok := findUser(us, name); !ok {
		return fmt.Errorf("user %s not found", name)
	}
	args := []string{}
	if shell != "" {
		args = append(args, "-s", shell)
	}
	if home != "" {
		args = append(args, "-d", home)
	}
	if len(args) == 0 {
		return nil
	}
	return run("usermod", append(args, "--", name), "")
}

// SetGroups sets a user's primary group ("" = unchanged) and replaces
// supplementary membership with supp, applying the diff through
// gpasswd. Stripping the admin group from, or rehoming, a system
// account is refused.
func SetGroups(name, primary string, supp []string) error {
	if err := checkRoot(); err != nil {
		return err
	}
	if err := validName(name); err != nil {
		return err
	}
	if err := validNames(supp); err != nil {
		return err
	}
	if primary != "" {
		if err := validName(primary); err != nil {
			return fmt.Errorf("%w: %q", ErrBadName, primary)
		}
	}
	us, err := Users()
	if err != nil {
		return err
	}
	gs, err := Groups()
	if err != nil {
		return err
	}
	return setGroups(us, gs, name, primary, supp)
}

func setGroups(us []User, gs []Group, name, primary string, supp []string) error {
	u, ok := findUser(us, name)
	if !ok {
		return fmt.Errorf("user %s not found", name)
	}
	admin := AdminGroup(gs)
	if primary != "" {
		if _, ok := findGroup(gs, primary); !ok {
			return fmt.Errorf("group %s not found", primary)
		}
	}
	cur := DirectMemberOf(name, gs)
	want := dedup(supp)
	promoted := false
	if primary == "" {
		if g, ok := primaryGroupOf(u, gs); ok {
			primary = g.Name
		}
	} else if g, ok := findGroup(gs, primary); ok && g.GID != u.GID {
		promoted = true
	}
	// The (new) primary group is never a supplementary member: once
	// promoted, a stale direct entry is cleaned up via the diff below.
	want = drop(want, primary)

	if u.UID < 1000 {
		if admin != "" && contains(cur, admin) && !contains(want, admin) {
			return fmt.Errorf("refusing to strip %s from system account %s", admin, name)
		}
		if primary != "" {
			if g, ok := primaryGroupOf(u, gs); !ok || g.Name != primary {
				return fmt.Errorf("refusing to change the primary group of system account %s", name)
			}
		}
	}

	if promoted {
		if err := run("usermod", []string{"-g", primary, "--", name}, ""); err != nil {
			return err
		}
	}
	for _, g := range want {
		if !contains(cur, g) {
			if err := run("gpasswd", []string{"-a", name, g}, ""); err != nil {
				return err
			}
		}
	}
	for _, g := range cur {
		if !contains(want, g) {
			if err := run("gpasswd", []string{"-d", name, g}, ""); err != nil {
				return err
			}
		}
	}
	return nil
}

// DirectMemberOf returns the groups listing user as a direct
// (supplementary) member, excluding their primary group.
func DirectMemberOf(user string, groups []Group) []string {
	var out []string
	for _, g := range groups {
		if contains(g.Members, user) {
			out = append(out, g.Name)
		}
	}
	return out
}

// LockUser disables password login; root and system accounts are
// refused (system accounts already have no usable login).
func LockUser(name string) error { return toggleLock(name, "-L") }

// UnlockUser re-enables password login; root and system accounts are
// refused.
func UnlockUser(name string) error { return toggleLock(name, "-U") }

func toggleLock(name, flag string) error {
	if err := checkRoot(); err != nil {
		return err
	}
	if err := validName(name); err != nil {
		return err
	}
	us, err := Users()
	if err != nil {
		return err
	}
	return lockUser(us, name, flag)
}

func lockUser(us []User, name, flag string) error {
	u, ok := findUser(us, name)
	if !ok {
		return fmt.Errorf("user %s not found", name)
	}
	if u.UID < 1000 {
		return ErrSystemAccount
	}
	return run("usermod", []string{flag, "--", name}, "")
}

// SetPassword sets a local password via chpasswd; the password is
// piped over stdin, never passed as an argument.
func SetPassword(name, pw string) error {
	if err := checkRoot(); err != nil {
		return err
	}
	if err := validName(name); err != nil {
		return err
	}
	if pw == "" {
		return errors.New("empty password")
	}
	us, err := Users()
	if err != nil {
		return err
	}
	if _, ok := findUser(us, name); !ok {
		return fmt.Errorf("user %s not found", name)
	}
	return run("chpasswd", nil, name+":"+pw+"\n")
}

// SetExpiry sets the account expiry date (YYYY-MM-DD); blank clears
// it.
func SetExpiry(name, date string) error {
	if err := checkRoot(); err != nil {
		return err
	}
	if err := validName(name); err != nil {
		return err
	}
	if date != "" && !dateRe.MatchString(date) {
		return fmt.Errorf("bad date %q (want YYYY-MM-DD)", date)
	}
	if date == "" {
		date = "-1"
	}
	return run("chage", []string{"-E", date, "--", name}, "")
}

// CreateGroup adds a local group; gid 0 lets groupadd pick one.
func CreateGroup(name string, gid int) error {
	if err := checkRoot(); err != nil {
		return err
	}
	if err := validName(name); err != nil {
		return err
	}
	gs, err := Groups()
	if err != nil {
		return err
	}
	if _, ok := findGroup(gs, name); ok {
		return fmt.Errorf("group %s already exists", name)
	}
	args := []string{}
	if gid > 0 {
		args = append(args, "-g", strconv.Itoa(gid))
	}
	return run("groupadd", append(args, "--", name), "")
}

// DeleteGroup removes a local group. System groups (gid < 1000) and
// any user's primary group are refused.
func DeleteGroup(name string) error {
	if err := checkRoot(); err != nil {
		return err
	}
	if err := validName(name); err != nil {
		return err
	}
	gs, err := Groups()
	if err != nil {
		return err
	}
	us, err := Users()
	if err != nil {
		return err
	}
	return deleteGroup(us, gs, name)
}

func deleteGroup(us []User, gs []Group, name string) error {
	g, ok := findGroup(gs, name)
	if !ok {
		return fmt.Errorf("group %s not found", name)
	}
	if g.GID < 1000 {
		return ErrSystemAccount
	}
	for _, u := range us {
		if u.GID == g.GID {
			return fmt.Errorf("group %s is the primary group of %s", name, u.Name)
		}
	}
	return run("groupdel", []string{"--", name}, "")
}

// AddMember adds user to group (supplementary) via gpasswd.
func AddMember(group, user string) error {
	return memberOp(group, user, "-a")
}

// RemoveMember drops user from group via gpasswd; stripping the admin
// group from a system account is refused, as is trying to remove a
// primary membership.
func RemoveMember(group, user string) error {
	return memberOp(group, user, "-d")
}

func memberOp(group, user, flag string) error {
	if err := checkRoot(); err != nil {
		return err
	}
	if err := validName(group); err != nil {
		return err
	}
	if err := validName(user); err != nil {
		return err
	}
	us, err := Users()
	if err != nil {
		return err
	}
	gs, err := Groups()
	if err != nil {
		return err
	}
	return memberOpChecked(us, gs, group, user, flag)
}

func memberOpChecked(us []User, gs []Group, group, user, flag string) error {
	g, ok := findGroup(gs, group)
	if !ok {
		return fmt.Errorf("group %s not found", group)
	}
	u, ok := findUser(us, user)
	if !ok {
		return fmt.Errorf("user %s not found", user)
	}
	if flag == "-d" {
		if u.GID == g.GID {
			return fmt.Errorf("%s's primary group is %s - change the primary group instead", user, group)
		}
		if admin := AdminGroup(gs); admin == group && u.UID < 1000 {
			return fmt.Errorf("refusing to strip %s from system account %s", group, user)
		}
	}
	return run("gpasswd", []string{flag, user, group}, "")
}

func findUser(us []User, name string) (User, bool) {
	for _, u := range us {
		if u.Name == name {
			return u, true
		}
	}
	return User{}, false
}

func findGroup(gs []Group, name string) (Group, bool) {
	for _, g := range gs {
		if g.Name == name {
			return g, true
		}
	}
	return Group{}, false
}

// primaryGroupOf resolves the user's primary group entry.
func primaryGroupOf(u User, gs []Group) (Group, bool) {
	for _, g := range gs {
		if g.GID == u.GID {
			return g, true
		}
	}
	return Group{}, false
}

func validNames(ss []string) error {
	for _, s := range ss {
		if err := validName(s); err != nil {
			return fmt.Errorf("%w: %q", ErrBadName, s)
		}
	}
	return nil
}

func dedup(ss []string) []string {
	var out []string
	for _, s := range ss {
		if !contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}

// drop removes name from ss without mutating ss.
func drop(ss []string, name string) []string {
	var out []string
	for _, s := range ss {
		if s != name {
			out = append(out, s)
		}
	}
	return out
}
