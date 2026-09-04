package accounts

import (
	"errors"
	"os"
	"regexp"
)

// ErrNotRoot means a mutation was attempted without root privileges.
// Mutating UI gates on IsRoot first; providers re-check defensively.
var ErrNotRoot = errors.New("needs root - restart the app with sudo")

// ErrSystemAccount means the target is uid/gid 0 or a system account
// (id < 1000); such accounts are never mutated through this app.
var ErrSystemAccount = errors.New("refusing to modify a system account")

// ErrBadName means an account or group name failed validation; names
// are passed as exec args, so this also blocks argument injection.
var ErrBadName = errors.New("invalid name")

// geteuid is the test seam over os.Geteuid.
var geteuid = os.Geteuid

// IsRoot reports whether the process runs with uid 0. Account and
// group mutations require it; the UI gates on this before offering
// any mutating action.
func IsRoot() bool { return geteuid() == 0 }

// checkRoot is the first line of every mutating entry point.
func checkRoot() error {
	if !IsRoot() {
		return ErrNotRoot
	}
	return nil
}

// nameRe matches the portable subset accepted by useradd/groupadd on
// mainstream distros: lowercase start, 1-32 chars of [a-z0-9_-], a
// trailing $ allowed (Samba convention). Anchoring plus the "--"
// separator on every exec call blocks argument injection.
var nameRe = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}\$?$`)

// ValidName reports whether s is an acceptable account/group name.
func ValidName(s string) bool { return nameRe.MatchString(s) }

func validName(s string) error {
	if !ValidName(s) {
		return ErrBadName
	}
	return nil
}

// AdminGroup returns the distro's sudo-capable group: wheel if
// present, else sudo, else "".
func AdminGroup(groups []Group) string {
	for _, want := range []string{"wheel", "sudo"} {
		for _, g := range groups {
			if g.Name == want {
				return g.Name
			}
		}
	}
	return ""
}

// HasAdmin reports whether user has sudo rights via adminGroup: a
// direct member, or the admin group is their primary group.
func HasAdmin(user string, primaryGID int, adminGroup string, groups []Group) bool {
	if adminGroup == "" {
		return false
	}
	for _, g := range groups {
		if g.Name != adminGroup {
			continue
		}
		return g.GID == primaryGID || contains(g.Members, user)
	}
	return false
}
