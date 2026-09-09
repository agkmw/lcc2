package files

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// infoDir creates the trash info/ sibling of the files dir that
// trashFixture returns (the fixture deliberately leaves it out).
func infoDir(t *testing.T, filesDir string) string {
	t.Helper()
	d := filepath.Join(filepath.Dir(filesDir), "info")
	if err := os.MkdirAll(d, 0o700); err != nil {
		t.Fatal(err)
	}
	return d
}

// writeTrashInfo hand-writes the .trashinfo record for a trashed entry.
func writeTrashInfo(t *testing.T, infoDir, base, origin string) {
	t.Helper()
	rec := "[Trash Info]\nPath=" + origin +
		"\nDeletionDate=" + time.Now().Format("2006-01-02T15:04:05") + "\n"
	if err := os.WriteFile(filepath.Join(infoDir, base+".trashinfo"), []byte(rec), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TrashInfo parses the record and Restore puts the entry back at its
// original path, recreating vanished parents and dropping the record.
func TestTrashInfoAndRestore(t *testing.T) {
	d := trashFixture(t)
	info := infoDir(t, d)

	// $HOME/A must not exist: Restore has to recreate it.
	origin := filepath.Join(os.Getenv("HOME"), "A", "foobar")
	src := filepath.Join(d, "foobar")
	if err := os.WriteFile(src, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeTrashInfo(t, info, "foobar", origin)

	got, deleted, ok := TrashInfo(src)
	if !ok || got != origin || deleted.IsZero() {
		t.Fatalf("TrashInfo = %q, %v, %v; want %q, non-zero, true", got, deleted, ok, origin)
	}

	if err := Restore(src); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(origin)
	if err != nil || string(data) != "payload" {
		t.Fatalf("restored file = %q, %v; want payload", data, err)
	}
	if _, err := os.Lstat(src); !os.IsNotExist(err) {
		t.Fatal("trashed entry survived the restore")
	}
	if _, err := os.Lstat(filepath.Join(info, "foobar.trashinfo")); !os.IsNotExist(err) {
		t.Fatal("trashinfo survived the restore")
	}
}

// Restore refuses to clobber whatever now sits at the origin, and
// refuses entries without a record.
func TestRestoreGuards(t *testing.T) {
	d := trashFixture(t)
	info := infoDir(t, d)

	// Occupied origin: the trashed entry must survive untouched.
	origin := filepath.Join(os.Getenv("HOME"), "A", "foobar")
	src := filepath.Join(d, "foobar")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeTrashInfo(t, info, "foobar", origin)
	if err := os.MkdirAll(filepath.Dir(origin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(origin, []byte("newer"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Restore(src); err == nil {
		t.Fatal("restore over an occupied origin must fail")
	}
	if _, err := os.Lstat(src); err != nil {
		t.Fatal("trashed entry lost after a refused restore")
	}

	// No record: nothing to restore to.
	orphan := filepath.Join(d, "norecord")
	if err := os.WriteFile(orphan, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := Restore(orphan)
	if err == nil || !strings.Contains(err.Error(), "no trash record") {
		t.Fatalf("orphan restore err = %v; want no trash record", err)
	}
}

// Stage validates restore ops up front; a valid one applies via ApplyOp.
func TestRestoreOpStages(t *testing.T) {
	d := trashFixture(t)
	info := infoDir(t, d)
	home := os.Getenv("HOME")
	s := NewStager()

	// A path outside the trash is rejected outright.
	if err := s.Stage(Op{Kind: OpRestore, Path: filepath.Join(home, "elsewhere")}); err == nil ||
		!strings.Contains(err.Error(), "not in the trash") {
		t.Fatalf("stage outside trash err = %v; want not in the trash", err)
	}

	// Occupied origin is rejected at stage time.
	taken := filepath.Join(d, "taken")
	takenOrigin := filepath.Join(home, "taken")
	if err := os.WriteFile(taken, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeTrashInfo(t, info, "taken", takenOrigin)
	if err := os.WriteFile(takenOrigin, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := s.Stage(Op{Kind: OpRestore, Path: taken, Arg: takenOrigin})
	if err == nil || !strings.Contains(err.Error(), "already exists at the original location") {
		t.Fatalf("stage over occupied origin err = %v", err)
	}

	// Free origin: stages, then ApplyOp moves it back.
	src := filepath.Join(d, "free")
	origin := filepath.Join(home, "free")
	if err := os.WriteFile(src, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeTrashInfo(t, info, "free", origin)
	if err := s.Stage(Op{Kind: OpRestore, Path: src, Arg: origin}); err != nil {
		t.Fatal(err)
	}
	if err := ApplyOp(s.Ops()[0]); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(origin); err != nil || string(data) != "keep" {
		t.Fatalf("restored file = %q, %v; want keep", data, err)
	}
	if _, err := os.Lstat(src); !os.IsNotExist(err) {
		t.Fatal("trashed entry survived the applied restore")
	}
}
