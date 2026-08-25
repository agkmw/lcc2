package screens

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func forceTrueColorHl(t *testing.T) {
	t.Helper()
	old := lipgloss.DefaultRenderer().ColorProfile()
	lipgloss.DefaultRenderer().SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.DefaultRenderer().SetColorProfile(old) })
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

	old := lipgloss.DefaultRenderer().ColorProfile()
	lipgloss.DefaultRenderer().SetColorProfile(termenv.Ascii)
	if out := highlightCode("main.go", lines); out != nil {
		lipgloss.DefaultRenderer().SetColorProfile(old)
		t.Error("ascii profile should not highlight")
	}
	lipgloss.DefaultRenderer().SetColorProfile(old)
}
