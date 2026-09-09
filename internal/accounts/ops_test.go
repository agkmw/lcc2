package accounts

import (
	"errors"
	"strings"
	"testing"
)

// recCall records one execer invocation.
type recCall struct {
	name  string
	args  []string
	stdin string
}

type recorder struct {
	calls []recCall
}

func (r *recorder) stub() func(string, []string, string) (string, error) {
	return func(name string, args []string, stdin string) (string, error) {
		r.calls = append(r.calls, recCall{name: name, args: args, stdin: stdin})
		return "", nil
	}
}

func (r *recorder) one(t *testing.T) recCall {
	t.Helper()
	if len(r.calls) != 1 {
		t.Fatalf("got %d exec calls, want 1: %+v", len(r.calls), r.calls)
	}
	return r.calls[0]
}

func asRoot(t *testing.T) {
	t.Helper()
	orig := geteuid
	geteuid = fakeEuid(0)
	t.Cleanup(func() { geteuid = orig })
}

var fixtures = struct {
	users  []User
	groups []Group
}{
	users: []User{
		{Name: "root", UID: 0, GID: 0},
		{Name: "daemon", UID: 1, GID: 1},
		{Name: "alice", UID: 1500, GID: 1500},
		{Name: "svc", UID: 900, GID: 900},
	},
	groups: []Group{
		{Name: "root", GID: 0},
		{Name: "wheel", GID: 10, Members: []string{"alice", "svc"}},
		{Name: "docker", GID: 900, Members: []string{"alice"}},
		{Name: "alice", GID: 1500},
		{Name: "devel", GID: 2000},
	},
}

func TestCreateUserArgv(t *testing.T) {
	asRoot(t)
	var rec recorder
	execer = rec.stub()
	t.Cleanup(func() { execer = realExec })

	err := CreateUser(UserOpts{
		Name: "zz-lcc-t", Shell: "/bin/zsh", Home: "/srv/zz",
		PrimaryGroup: "ops", Groups: []string{"docker", "wheel"},
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	c := rec.one(t)
	if c.name != "useradd" {
		t.Fatalf("tool = %s", c.name)
	}
	want := "-m -s /bin/zsh -d /srv/zz -g ops -G docker,wheel -- zz-lcc-t"
	if got := strings.Join(c.args, " "); got != want {
		t.Fatalf("argv = %q, want %q", got, want)
	}
}

func TestCreateUserSystemArgv(t *testing.T) {
	asRoot(t)
	var rec recorder
	execer = rec.stub()
	t.Cleanup(func() { execer = realExec })

	if err := CreateUser(UserOpts{Name: "zz-lcc-s", System: true}); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	c := rec.one(t)
	want := "-r -- zz-lcc-s"
	if got := strings.Join(c.args, " "); got != want {
		t.Fatalf("argv = %q, want %q", got, want)
	}
}

func TestCreateUserCollision(t *testing.T) {
	asRoot(t)
	var rec recorder
	execer = rec.stub()
	t.Cleanup(func() { execer = realExec })
	// root always exists in /etc/passwd.
	if err := CreateUser(UserOpts{Name: "root"}); err == nil ||
		!strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CreateUser(root) = %v, want already-exists error", err)
	}
	if len(rec.calls) != 0 {
		t.Fatalf("collision path exec'd: %+v", rec.calls)
	}
}

func TestCreateUserNotRoot(t *testing.T) {
	orig := geteuid
	geteuid = fakeEuid(1000)
	defer func() { geteuid = orig }()
	if err := CreateUser(UserOpts{Name: "zz"}); !errors.Is(err, ErrNotRoot) {
		t.Fatalf("CreateUser as user = %v, want ErrNotRoot", err)
	}
}

func TestCreateUserBadName(t *testing.T) {
	asRoot(t)
	for _, n := range []string{"--help", "", "a b"} {
		if err := CreateUser(UserOpts{Name: n}); !errors.Is(err, ErrBadName) {
			t.Fatalf("CreateUser(%q) = %v, want ErrBadName", n, err)
		}
	}
}

func TestDeleteUserRefusesSystem(t *testing.T) {
	asRoot(t)
	if err := DeleteUser("root", false); !errors.Is(err, ErrSystemAccount) {
		t.Fatalf("DeleteUser(root) = %v, want ErrSystemAccount", err)
	}
}

func TestDeleteUserNotFound(t *testing.T) {
	asRoot(t)
	var rec recorder
	execer = rec.stub()
	t.Cleanup(func() { execer = realExec })
	if err := DeleteUser("zz-lcc-none", false); err == nil ||
		!strings.Contains(err.Error(), "not found") {
		t.Fatalf("DeleteUser = %v, want not-found", err)
	}
	if len(rec.calls) != 0 {
		t.Fatalf("not-found path exec'd: %+v", rec.calls)
	}
}

func TestDeleteUserArgv(t *testing.T) {
	asRoot(t)
	var rec recorder
	execer = rec.stub()
	t.Cleanup(func() { execer = realExec })

	if err := deleteUser(fixtures.users, "alice", true); err != nil {
		t.Fatalf("deleteUser: %v", err)
	}
	c := rec.one(t)
	if got := strings.Join(c.args, " "); got != "-r -- alice" {
		t.Fatalf("argv = %q", got)
	}

	var rec2 recorder
	execer = rec2.stub()
	if err := deleteUser(fixtures.users, "alice", false); err != nil {
		t.Fatalf("deleteUser: %v", err)
	}
	if got := strings.Join(rec2.one(t).args, " "); got != "-- alice" {
		t.Fatalf("argv without -r = %q", got)
	}
}

func TestDeleteUserGuard(t *testing.T) {
	var rec recorder
	execer = rec.stub()
	t.Cleanup(func() { execer = realExec })

	if err := deleteUser(fixtures.users, "root", false); !errors.Is(err, ErrSystemAccount) {
		t.Fatalf("deleteUser(root) = %v", err)
	}
	if err := deleteUser(fixtures.users, "svc", false); !errors.Is(err, ErrSystemAccount) {
		t.Fatalf("deleteUser(uid 900) = %v", err)
	}
	if len(rec.calls) != 0 {
		t.Fatalf("refused paths still exec'd: %+v", rec.calls)
	}
}

func TestSetPasswordStdin(t *testing.T) {
	asRoot(t)
	var rec recorder
	execer = rec.stub()
	t.Cleanup(func() { execer = realExec })

	if err := SetPassword("root", "s3cret"); err != nil {
		t.Fatalf("SetPassword: %v", err)
	}
	c := rec.one(t)
	if c.name != "chpasswd" || len(c.args) != 0 {
		t.Fatalf("chpasswd argv = %+v", c)
	}
	if c.stdin != "root:s3cret\n" {
		t.Fatalf("stdin = %q", c.stdin)
	}
	if err := SetPassword("root", ""); err == nil {
		t.Fatal("empty password accepted")
	}
}

func TestSetExpiry(t *testing.T) {
	asRoot(t)
	var rec recorder
	execer = rec.stub()
	t.Cleanup(func() { execer = realExec })

	if err := SetExpiry("root", "2027-01-31"); err != nil {
		t.Fatalf("SetExpiry: %v", err)
	}
	if got := strings.Join(rec.one(t).args, " "); got != "-E 2027-01-31 -- root" {
		t.Fatalf("argv = %q", got)
	}
	rec.calls = nil
	if err := SetExpiry("root", ""); err != nil {
		t.Fatalf("SetExpiry clear: %v", err)
	}
	if got := strings.Join(rec.one(t).args, " "); got != "-E -1 -- root" {
		t.Fatalf("clear argv = %q", got)
	}
	if err := SetExpiry("root", "tomorrow"); err == nil {
		t.Fatal("bad date accepted")
	}
}

func TestToggleLockGuards(t *testing.T) {
	asRoot(t)
	var rec recorder
	execer = rec.stub()
	t.Cleanup(func() { execer = realExec })

	if err := LockUser("root"); !errors.Is(err, ErrSystemAccount) {
		t.Fatalf("LockUser(root) = %v", err)
	}
	if err := UnlockUser("daemon"); !errors.Is(err, ErrSystemAccount) {
		t.Fatalf("UnlockUser(daemon) = %v", err)
	}
	if len(rec.calls) != 0 {
		t.Fatalf("refused paths still exec'd: %+v", rec.calls)
	}
}

func TestToggleLockArgv(t *testing.T) {
	us := []User{{Name: "alice", UID: 1500, GID: 1500}}
	var rec recorder
	execer = rec.stub()
	t.Cleanup(func() { execer = realExec })

	if err := lockUser(us, "alice", "-L"); err != nil {
		t.Fatalf("lockUser: %v", err)
	}
	if got := strings.Join(rec.one(t).args, " "); got != "-L -- alice" {
		t.Fatalf("argv = %q", got)
	}
}

func TestSetGroupsDiff(t *testing.T) {
	var rec recorder
	execer = rec.stub()
	t.Cleanup(func() { execer = realExec })

	// alice: primary alice(1500), direct member of wheel+docker.
	// Want: primary docker, drop wheel, add audio.
	err := setGroups(fixtures.users, fixtures.groups, "alice", "docker",
		[]string{"docker", "audio"})
	if err != nil {
		t.Fatalf("setGroups: %v", err)
	}
	var got []string
	for _, c := range rec.calls {
		got = append(got, c.name+" "+strings.Join(c.args, " "))
	}
	want := []string{
		"usermod -g docker -- alice",
		"gpasswd -a alice audio",
		"gpasswd -d alice wheel",
		"gpasswd -d alice docker",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("exec sequence:\n%s\nwant:\n%s",
			strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestSetGroupsNoop(t *testing.T) {
	var rec recorder
	execer = rec.stub()
	t.Cleanup(func() { execer = realExec })

	// Same supplementary set, primary unchanged -> no exec at all.
	if err := setGroups(fixtures.users, fixtures.groups, "alice", "",
		[]string{"wheel", "docker"}); err != nil {
		t.Fatalf("setGroups: %v", err)
	}
	if len(rec.calls) != 0 {
		t.Fatalf("noop setGroups exec'd: %+v", rec.calls)
	}
}

func TestSetGroupsRefusesSystemStrip(t *testing.T) {
	var rec recorder
	execer = rec.stub()
	t.Cleanup(func() { execer = realExec })

	// svc (uid 900) is a direct member of wheel; removing it refused.
	err := setGroups(fixtures.users, fixtures.groups, "svc", "", []string{})
	if err == nil || !strings.Contains(err.Error(), "refusing to strip wheel") {
		t.Fatalf("setGroups(svc) = %v, want admin-strip refusal", err)
	}
	// Rehoming a system account refused too (wheel membership kept so
	// the strip guard does not fire first).
	err = setGroups(fixtures.users, fixtures.groups, "svc", "devel", []string{"wheel"})
	if err == nil || !strings.Contains(err.Error(), "primary group") {
		t.Fatalf("setGroups(svc, primary) = %v, want primary refusal", err)
	}
	if len(rec.calls) != 0 {
		t.Fatalf("refused paths still exec'd: %+v", rec.calls)
	}
}

func TestSetGroupsPrimaryOnly(t *testing.T) {
	var rec recorder
	execer = rec.stub()
	t.Cleanup(func() { execer = realExec })

	// alice -> primary wheel with no supplementary set: the promotion
	// cleans up the stale direct wheel entry and drops docker.
	if err := setGroups(fixtures.users, fixtures.groups, "alice", "wheel", nil); err != nil {
		t.Fatalf("setGroups: %v", err)
	}
	var got []string
	for _, c := range rec.calls {
		got = append(got, c.name+" "+strings.Join(c.args, " "))
	}
	want := []string{
		"usermod -g wheel -- alice",
		"gpasswd -d alice wheel",
		"gpasswd -d alice docker",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("exec sequence = %v, want %v", got, want)
	}
}

func TestDeleteGroupGuards(t *testing.T) {
	var rec recorder
	execer = rec.stub()
	t.Cleanup(func() { execer = realExec })

	// gid < 1000 refused.
	if err := deleteGroup(fixtures.users, fixtures.groups, "wheel"); !errors.Is(err, ErrSystemAccount) {
		t.Fatalf("deleteGroup(wheel) = %v", err)
	}
	// primary group of a user refused.
	if err := deleteGroup(fixtures.users, fixtures.groups, "alice"); err == nil ||
		!strings.Contains(err.Error(), "primary group") {
		t.Fatalf("deleteGroup(alice) = %v, want primary refusal", err)
	}
	if len(rec.calls) != 0 {
		t.Fatalf("refused paths still exec'd: %+v", rec.calls)
	}

	// deletable group (gid >= 1000, not a primary) goes through.
	if err := deleteGroup(fixtures.users, fixtures.groups, "devel"); err != nil {
		t.Fatalf("deleteGroup(devel): %v", err)
	}
	if got := strings.Join(rec.one(t).args, " "); got != "-- devel" {
		t.Fatalf("argv = %q", got)
	}
}

func TestMemberOps(t *testing.T) {
	var rec recorder
	execer = rec.stub()
	t.Cleanup(func() { execer = realExec })

	// Removing a primary membership is refused with guidance.
	err := memberOpChecked(fixtures.users, fixtures.groups, "docker", "svc", "-d")
	if err == nil || !strings.Contains(err.Error(), "primary group") {
		t.Fatalf("remove primary member = %v", err)
	}
	// Stripping admin from a system account refused.
	err = memberOpChecked(fixtures.users, fixtures.groups, "wheel", "svc", "-d")
	if err == nil || !strings.Contains(err.Error(), "refusing to strip wheel") {
		t.Fatalf("strip admin from system = %v", err)
	}
	// Humans may lose admin rights; system accounts may gain groups.
	if err := memberOpChecked(fixtures.users, fixtures.groups, "wheel", "alice", "-d"); err != nil {
		t.Fatalf("remove alice from wheel: %v", err)
	}
	if err := memberOpChecked(fixtures.users, fixtures.groups, "docker", "svc", "-a"); err != nil {
		t.Fatalf("add svc to docker: %v", err)
	}
	want := []string{"gpasswd -d alice wheel", "gpasswd -a svc docker"}
	var got []string
	for _, c := range rec.calls {
		got = append(got, c.name+" "+strings.Join(c.args, " "))
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("exec sequence = %v, want %v", got, want)
	}
}
