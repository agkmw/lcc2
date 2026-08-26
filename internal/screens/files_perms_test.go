package screens

import (
	"os"
	"testing"
)

// parseOctal must map unix special bits onto Go's FileMode flags; a raw
// 0o4000 in os.FileMode means nothing to os.Chmod.
func TestParseOctalMapsSpecialBits(t *testing.T) {
	m, err := parseOctal("4755")
	if err != nil {
		t.Fatal(err)
	}
	if m&os.ModeSetuid == 0 || m.Perm() != 0o755 {
		t.Fatalf("4755 -> %v", m)
	}

	m, err = parseOctal("1777")
	if err != nil || m&os.ModeSticky == 0 || m.Perm() != 0o777 {
		t.Fatalf("1777 -> %v err=%v", m, err)
	}

	if _, err := parseOctal("4855"); err == nil {
		t.Fatal("non-octal digit accepted")
	}
	if _, err := parseOctal("177777"); err == nil {
		t.Fatal("out-of-range mode accepted")
	}
}
