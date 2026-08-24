package screens

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"lcc2/internal/proc"
)

func forceTrueColorScreens(t *testing.T) {
	t.Helper()
	old := lipgloss.DefaultRenderer().ColorProfile()
	lipgloss.DefaultRenderer().SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.DefaultRenderer().SetColorProfile(old) })
}

// highlightSvcStatus wraps known tokens without moving any other byte:
// the stripped output must equal the input.
func TestHighlightSvcStatusPreservesText(t *testing.T) {
	forceTrueColorScreens(t)
	in := "Loaded: enabled\n     Active: active (running)\n  Main PID: 42 (sshd)\n once dead, now failed"
	out := highlightSvcStatus(in)
	if stripANSI(out) != in {
		t.Fatalf("text mutated:\n%q\n%q", in, stripANSI(out))
	}
	if n := strings.Count(out, "\x1b["); n < 4 {
		t.Errorf("expected >=4 styled spans, got %d", n)
	}
}

// The process card keeps its text content while values gain tones.
func TestProcessCardValuesToned(t *testing.T) {
	forceTrueColorScreens(t)
	d := proc.Process{PID: 1234, Name: "firefox", User: "root", State: "R",
		CPUPercent: 12.5, MemPercent: 55.0,
		Command: "/usr/bin/firefox --private-window https://example.com"}
	card := processCard(d, nil, 60)
	stripped := stripANSI(card)
	for _, want := range []string{"pid", "1234", "running", "12.5%", "55.0%", "--private-window"} {
		if !strings.Contains(stripped, want) {
			t.Errorf("card lost %q:\n%s", want, stripped)
		}
	}
	// Args after the executable render inside a separate faint span.
	if !strings.Contains(card, "/usr/bin/firefox\x1b[") {
		t.Errorf("cmdline args not dimmed: %q", card)
	}
}
