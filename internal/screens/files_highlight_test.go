package screens

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"

	"lcc2/internal/ui"
)

func forceTrueColorHl(t *testing.T) {
	t.Helper()
	restore := ui.SetProfileOverride(colorprofile.TrueColor)
	t.Cleanup(restore)
}

// Go sources highlight under truecolor; output keeps the line count.
func TestHighlightCodeGo(t *testing.T) {
	forceTrueColorHl(t)
	lines := []string{"package main", "", "func main() {}"}
	out := highlightCode("main.go", lines)
	if out == nil {
		t.Fatal("go file not highlighted")
	}
	if len(out) != len(lines) {
		t.Fatalf("line count %d != %d", len(out), len(lines))
	}
	if !strings.Contains(strings.Join(out, "\n"), "\x1b[") {
		t.Error("no escape sequences in highlighted output")
	}
}

// Gate matrix: unknown language, NO_COLOR and Ascii profile all fall
// back to nil (plain preview).
func TestHighlightCodeGates(t *testing.T) {
	forceTrueColorHl(t)
	lines := []string{"hello: world"}

	if out := highlightCode("data.xyz", lines); out != nil {
		t.Error("unknown extension should not highlight")
	}
	t.Setenv("NO_COLOR", "1")
	if out := highlightCode("main.go", lines); out != nil {
		t.Error("NO_COLOR should disable highlighting")
	}
	os.Unsetenv("NO_COLOR")

	restore := ui.SetProfileOverride(colorprofile.ASCII)
	defer restore()
	if out := highlightCode("main.go", lines); out != nil {
		restore()
		t.Error("ascii profile should not highlight")
	}
	restore()
}
