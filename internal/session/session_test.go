package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	want := State{Screen: 3, Cwd: "/tmp/x", Hidden: true, SortKey: "size", SortDesc: true}
	if err := Save(want); err != nil {
		t.Fatal(err)
	}
	got := Load()
	if got != want {
		t.Fatalf("round trip: %+v != %+v", got, want)
	}
	p := filepath.Join(dir, "lcc2", "state.json")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("state file missing: %v", err)
	}
}

func TestLoadDefaultsOnCorruption(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	p := filepath.Join(dir, "lcc2", "state.json")
	os.MkdirAll(filepath.Dir(p), 0o755)
	os.WriteFile(p, []byte("{not json"), 0o600)

	got := Load()
	if got.Screen != 0 || got.SortKey != "name" || got.Cwd != "" {
		t.Fatalf("corrupt file yielded %+v", got)
	}
}

func TestLoadClampsScreen(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	Save(State{Screen: -2})
	if got := Load(); got.Screen != 0 {
		t.Fatalf("negative screen not clamped: %d", got.Screen)
	}
}
