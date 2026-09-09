package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveOwner(t *testing.T) {
	uid, gid, err := ResolveOwner("", "")
	if uid != -1 || gid != -1 || err != nil {
		t.Fatalf("empty resolve = %d,%d,%v", uid, gid, err)
	}
	uid, gid, err = ResolveOwner("root", "root")
	if err != nil {
		t.Fatalf("ResolveOwner(root,root): %v", err)
	}
	if uid != 0 || gid != 0 {
		t.Fatalf("root resolved to %d:%d, want 0:0", uid, gid)
	}
	if _, _, err := ResolveOwner("zz-lcc-nouser", ""); err == nil {
		t.Fatal("unknown user accepted")
	}
	if _, _, err := ResolveOwner("", "zz-lcc-nogroup"); err == nil {
		t.Fatal("unknown group accepted")
	}
}

func TestStageAndApplyOpChown(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	st := NewStager()
	// -1/-1 = unchanged: the only chown unprivileged tests may apply.
	op := Op{Kind: OpChown, Path: p, UID: -1, GID: -1, Arg: "root:root"}
	if err := st.Stage(op); err != nil {
		t.Fatalf("Stage: %v", err)
	}
	if err := ApplyOp(st.Ops()[0]); err != nil {
		t.Fatalf("ApplyOp: %v", err)
	}
}

func TestStageOpChownVanished(t *testing.T) {
	st := NewStager()
	err := st.Stage(Op{Kind: OpChown, Path: "/nonexistent/lcc", UID: 0, GID: 0})
	if err == nil {
		t.Fatal("staged a chown on a missing path")
	}
}

func TestOpChownLabel(t *testing.T) {
	got := Op{Kind: OpChown, Path: "/tmp/x/file.txt", UID: 0, GID: 0, Arg: "root:root"}.Label()
	if got != "chown root:root file.txt" {
		t.Fatalf("Label = %q", got)
	}
	if OpChown.String() != "chown" {
		t.Fatalf("String = %q", OpChown.String())
	}
}
