package screens

import (
	"time"

	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"lcc2/internal/proc"
	"lcc2/internal/services"
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

// The structured services card renders every known fact, the failed
// banner when failed, and nothing else.
func TestServicesDetailCard(t *testing.T) {
	forceTrueColorScreens(t)
	d := services.Detail{Active: "active", Sub: "running", Boot: "enabled",
		PID: 842, MemBytes: 12828672, CPUNanos: 2158349000,
		Since:     time.Now().Add(-90 * time.Minute),
		Timestamp: "Sun 2026-08-23 19:59:42 +0630", Restarts: 2}
	card := detailCard(d, 50)
	s := stripANSI(card)
	for _, want := range []string{"active (running)", "enabled",
		"1h 30m ago", "Sun 2026-08-23", "842", "12.2 MB", "2.2s", "restarts"} {
		if !strings.Contains(s, want) {
			t.Errorf("card missing %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "FAILED") {
		t.Error("running unit must not show the failed banner")
	}

	d.Active = "failed"
	d.Sub = "exit-code"
	failed := stripANSI(detailCard(d, 50))
	if !strings.Contains(failed, "FAILED") {
		t.Error("failed unit missing banner")
	}
}

// relSince buckets like systemd does.
func TestRelSince(t *testing.T) {
	cases := map[time.Duration]string{
		45 * time.Second: "45s ago",
		5 * time.Minute:  "5m ago",
		90 * time.Minute: "1h 30m ago",
		26 * time.Hour:   "1d 2h ago",
		48 * time.Hour:   "2d ago",
	}
	for d, want := range cases {
		if got := relSince(d); got != want {
			t.Errorf("relSince(%v) = %q, want %q", d, got, want)
		}
	}
}
