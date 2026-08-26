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
