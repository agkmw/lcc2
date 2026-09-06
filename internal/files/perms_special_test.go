package files

import (
	"os"
	"path/filepath"
	"testing"
)

// Special bits must survive the editor's octal readout instead of being
// silently dropped from staged chmods.
func TestPermBitsSpecialOctal(t *testing.T) {
	cases := []struct {
		mode os.FileMode
		want string
	}{
		{0o755 | os.ModeSetuid, "4755"},
		{0o644 | os.ModeSetgid, "2644"},
		{0o777 | os.ModeSticky, "1777"},
		{0o644, "644"}, // no leading digit noise without special bits
	}
	for _, tc := range cases {
		if got := ParsePermBits(tc.mode).Octal(); got != tc.want {
			t.Fatalf("ParsePermBits(%v).Octal() = %q, want %q", tc.mode, got, tc.want)
		}
	}
}

// ToggleSpecial flips exactly one special bit per call.
func TestToggleSpecial(t *testing.T) {
	p := ParsePermBits(0o755)
	if p.Special != 0 {
		t.Fatalf("start Special = %d", p.Special)
	}
	p.ToggleSpecial(0) // setuid
	if p.Special != 4 {
		t.Fatalf("after setuid toggle Special = %d", p.Special)
	}
	p.ToggleSpecial(1) // setgid
	if p.Special != 6 {
		t.Fatalf("after setgid toggle Special = %d", p.Special)
	}
	p.ToggleSpecial(0) // setuid off again
	if p.Special != 2 {
		t.Fatalf("after second toggle Special = %d", p.Special)
	}
	if got := p.Octal(); got != "2755" {
		t.Fatalf("Octal = %q, want 2755", got)
	}
}

// The apply path honors a mode carrying special bits.
func TestChmodAppliesSpecialBits(t *testing.T) {
	f := filepath.Join(t.TempDir(), "prog")
	os.WriteFile(f, nil, 0o755)

	err := ApplyOp(Op{Kind: OpChmod, Path: f, Mode: 0o755 | os.ModeSetuid})
	if err != nil {
		t.Fatal(err)
	}
	st, serr := os.Stat(f)
	if serr != nil {
		t.Fatal(serr)
	}
	if st.Mode()&os.ModeSetuid == 0 || st.Mode().Perm() != 0o755 {
		t.Fatalf("setuid lost: %v", st.Mode())
	}
}
