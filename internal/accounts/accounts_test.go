package accounts

import "testing"

const passwd = `root:x:0:0:root:/root:/bin/bash
aung:x:1000:1000:Aung Khant,,,:/home/aung:/usr/bin/zsh
svc:x:900:900::/var/svc:/usr/sbin/nologin
# comment
badline
`

const group = `root:x:0:
adm:x:4:syslog,aung
docker:x:998:aung
empty:x:999:
`

func TestParsePasswd(t *testing.T) {
	us := ParsePasswd(passwd)
	if len(us) != 3 {
		t.Fatalf("got %d users", len(us))
	}
	if us[1].Name != "aung" || us[1].UID != 1000 || us[1].Shell != "/usr/bin/zsh" {
		t.Fatalf("user = %+v", us[1])
	}
}

func TestParseGroup(t *testing.T) {
	gs := ParseGroup(group)
	if len(gs) != 4 {
		t.Fatalf("got %d groups", len(gs))
	}
	if gs[1].Name != "adm" || len(gs[1].Members) != 2 || gs[1].Members[0] != "syslog" {
		t.Fatalf("group = %+v", gs[1])
	}
	if len(gs[3].Members) != 0 {
		t.Fatalf("expected empty members, got %v", gs[3].Members)
	}
}

func TestGroupsOfAndMembersOf(t *testing.T) {
	gs, _ := Groups()
	_ = gs
	all := ParseGroup(group)
	users := ParsePasswd(passwd)

	got := GroupsOf("aung", 1000, all)
	// aung is a direct member of adm and docker; primary gid 1000 has
	// no matching group entry in this fixture.
	if len(got) != 2 {
		t.Fatalf("GroupsOf = %v", got)
	}
	m := MembersOf(all[1], users) // adm: syslog + aung as direct members
	if len(m) != 2 {
		t.Fatalf("MembersOf(adm) = %v", m)
	}
}
