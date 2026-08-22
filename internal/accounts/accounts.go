// Package accounts reads local users and groups from /etc/passwd and
// /etc/group with pure parsers that are easy to unit-test.
package accounts

import (
	"os"
	"strconv"
	"strings"
)

// User is one account from /etc/passwd.
type User struct {
	Name  string
	UID   int
	GID   int
	Home  string
	Shell string
}

// Group is one group from /etc/group.
type Group struct {
	Name    string
	GID     int
	Members []string
}

// ParsePasswd parses passwd file content into users.
func ParsePasswd(content string) []User {
	var out []User
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, ":")
		if len(f) < 7 {
			continue
		}
		uid, _ := strconv.Atoi(f[2])
		gid, _ := strconv.Atoi(f[3])
		out = append(out, User{Name: f[0], UID: uid, GID: gid, Home: f[5], Shell: f[6]})
	}
	return out
}

// ParseGroup parses group file content into groups.
func ParseGroup(content string) []Group {
	var out []Group
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, ":")
		if len(f) < 4 {
			continue
		}
		gid, _ := strconv.Atoi(f[2])
		members := strings.FieldsFunc(f[3], func(r rune) bool { return r == ',' || r == ' ' })
		out = append(out, Group{Name: f[0], GID: gid, Members: members})
	}
	return out
}

// Users lists all local users sorted by UID.
func Users() ([]User, error) {
	b, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return nil, err
	}
	return ParsePasswd(string(b)), nil
}

// Groups lists all local groups sorted by GID.
func Groups() ([]Group, error) {
	b, err := os.ReadFile("/etc/group")
	if err != nil {
		return nil, err
	}
	gs := ParseGroup(string(b))
	for i := range gs {
		for j := i + 1; j < len(gs); j++ {
			if gs[i].GID > gs[j].GID {
				gs[i], gs[j] = gs[j], gs[i]
			}
		}
	}
	return gs, nil
}

// GroupsOf returns names of groups listing user as a direct member,
// plus the user's primary group name when resolvable.
func GroupsOf(user string, primaryGID int, all []Group) []string {
	var out []string
	for _, g := range all {
		if g.GID == primaryGID || contains(g.Members, user) {
			out = append(out, g.Name)
		}
	}
	return out
}

// MembersOf returns a group's supplementary members plus any users
// whose primary group matches, given the full user list.
func MembersOf(group Group, users []User) []string {
	out := append([]string{}, group.Members...)
	for _, u := range users {
		if u.GID == group.GID && !contains(out, u.Name) {
			out = append(out, u.Name+" (primary)")
		}
	}
	return out
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
