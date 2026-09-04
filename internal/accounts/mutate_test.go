package accounts

import (
	"errors"
	"testing"
)

func fakeEuid(uid int) func() int { return func() int { return uid } }

func TestIsRoot(t *testing.T) {
	orig := geteuid
	defer func() { geteuid = orig }()

	geteuid = fakeEuid(0)
	if !IsRoot() {
		t.Fatal("IsRoot() = false for euid 0")
	}
	geteuid = fakeEuid(1000)
	if IsRoot() {
		t.Fatal("IsRoot() = true for euid 1000")
	}
	if err := checkRoot(); !errors.Is(err, ErrNotRoot) {
		t.Fatalf("checkRoot() = %v, want ErrNotRoot", err)
	}
}

func TestValidName(t *testing.T) {
	good := []string{"a", "root", "aung-khant", "svc_2", "user$", "x1234567890123456789012345678901"}
	for _, s := range good {
		if !ValidName(s) {
			t.Errorf("ValidName(%q) = false, want true", s)
		}
	}
	bad := []string{"", "Root", "1abc", "-abc", "a b", "a;b", "a\nb", "--help",
		"a/b", "x12345678901234567890123456789012", "ä"}
	for _, s := range bad {
		if ValidName(s) {
			t.Errorf("ValidName(%q) = true, want false", s)
		}
	}
}

func TestAdminGroup(t *testing.T) {
	groups := []Group{{Name: "sudo", GID: 27}, {Name: "wheel", GID: 10}}
	if got := AdminGroup(groups); got != "wheel" {
		t.Fatalf("AdminGroup = %q, want wheel (wheel preferred)", got)
	}
	if got := AdminGroup(groups[:1]); got != "sudo" {
		t.Fatalf("AdminGroup = %q, want sudo", got)
	}
	if got := AdminGroup(nil); got != "" {
		t.Fatalf("AdminGroup(nil) = %q, want empty", got)
	}
}

func TestHasAdmin(t *testing.T) {
	groups := []Group{
		{Name: "wheel", GID: 10, Members: []string{"alice"}},
		{Name: "ops", GID: 20},
	}
	cases := []struct {
		user  string
		pgid  int
		admin string
		want  bool
	}{
		{"alice", 100, "wheel", true}, // direct member
		{"bob", 10, "wheel", true},    // primary group is wheel
		{"bob", 100, "wheel", false},  // no relation
		{"alice", 100, "", false},     // no admin group at all
		{"alice", 100, "sudo", false}, // admin group not present
	}
	for _, c := range cases {
		got := HasAdmin(c.user, c.pgid, c.admin, groups)
		if got != c.want {
			t.Errorf("HasAdmin(%q, %d, %q) = %v, want %v", c.user, c.pgid, c.admin, got, c.want)
		}
	}
}
