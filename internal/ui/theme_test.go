package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestGaugeConstantWidth(t *testing.T) {
	for pct := 0; pct <= 100; pct += 7 {
		for _, w := range []int{8, 12, 20, 33} {
			g := Gauge(float64(pct), w, nil)
			if got := lipgloss.Width(g); got != w {
				t.Fatalf("pct=%d w=%d: gauge width %d", pct, w, got)
			}
			if !strings.Contains(g, "%") {
				t.Fatalf("gauge missing label: %q", g)
			}
		}
	}
}

func TestGaugeFractionalFill(t *testing.T) {
	half := Gauge(50, 12, nil)
	quarter := Gauge(25, 12, nil)
	if half == quarter {
		t.Fatal("different percentages rendered identically")
	}
	if !strings.ContainsAny(half, "▏▎▌▊") && !strings.Contains(half, "███") {
		t.Fatalf("no shaded fill: %q", half)
	}
}

func TestTitledBoxEmbedsTitleInBorder(t *testing.T) {
	box := TitledBox("proc", "cpu", "body", 28) // inner width; border adds 2
	lines := strings.Split(box, "\n")
	if !strings.HasPrefix(lines[0], "┌─ cpu ") || !strings.HasSuffix(lines[0], "┐") {
		t.Fatalf("title not in border: %q", lines[0])
	}
	if lipgloss.Width(lines[0]) != 30 {
		t.Fatalf("border width %d, want 30", lipgloss.Width(lines[0]))
	}
	if !strings.Contains(box, "body") {
		t.Fatal("body lost")
	}
}

func TestKeyBadgeFormat(t *testing.T) {
	if got := KeyBadge("files", "q"); lipgloss.Width(got) != 3 || !strings.Contains(got, "[q]") {
		t.Fatalf("badge = %q", got)
	}
}

func TestStateColorThresholds(t *testing.T) {
	cases := map[float64]lipgloss.Color{
		10: Palette.Green, 69.9: Palette.Green,
		70: Palette.Yellow, 89.9: Palette.Yellow,
		90: Palette.Red, 100: Palette.Red,
	}
	for pct, want := range cases {
		if got := StateColor(pct); got != want {
			t.Fatalf("StateColor(%v) = %v, want %v", pct, got, want)
		}
	}
}
