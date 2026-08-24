package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestTruncate(t *testing.T) {
	if got := Truncate("hello", 10); got != "hello" {
		t.Fatalf("short string changed: %q", got)
	}
	got := Truncate("hello world", 8)
	if lipgloss.Width(got) > 8 || !strings.HasSuffix(got, "..") {
		t.Fatalf("truncate = %q", got)
	}
	if Truncate("abc", 0) != "" {
		t.Fatal("zero width must be empty")
	}
}

func TestGaugeBounds(t *testing.T) {
	for _, pct := range []float64{-5, 0, 50, 99, 150} {
		g := Gauge(pct, 20, nil)
		w := lipgloss.Width(g)
		if w < 6 || w > 24 {
			t.Fatalf("gauge(%v) width %d out of range", pct, w)
		}
	}
	red := Palette.Red
	Gauge(95, 20, &red) // custom color path
}

func TestSparkEmptyAndValues(t *testing.T) {
	if got := Spark(nil, 10, Palette.Blue); lipgloss.Width(got) != 10 {
		t.Fatalf("empty spark width = %d", lipgloss.Width(got))
	}
	got := Spark([]float64{0, 50, 100}, 3, Palette.Teal)
	if lipgloss.Width(got) != 3 {
		t.Fatalf("spark width = %d", lipgloss.Width(got))
	}
}

func TestGraphShape(t *testing.T) {
	g := Graph([]float64{0, 50, 100}, 5, 3, Palette.Blue)
	lines := strings.Split(g, "\n")
	if len(lines) != 3 {
		t.Fatalf("%d rows, want 3", len(lines))
	}
	for i, l := range lines {
		if w := lipgloss.Width(l); w != 5 {
			t.Fatalf("row %d width %d, want 5", i, w)
		}
	}
	// Newest (100%) at the right edge must fill every row of the
	// last column; the blank history columns stay empty.
	lastCol := func(s string) rune {
		runes := []rune(s)
		return runes[len(runes)-1]
	}
	for _, l := range lines {
		if lastCol(l) != '█' {
			t.Fatalf("full sample not filling column: %q", l)
		}
	}
	if first := []rune(lines[0])[0]; first != ' ' {
		t.Fatalf("unrecorded history must be blank, got %q", first)
	}
}

func TestSegGaugeSegments(t *testing.T) {
	bar := SegGauge(100, 50, 10, Palette.Green, Palette.Mauve)
	if w := lipgloss.Width(bar); w != 10 {
		t.Fatalf("width %d, want 10", w)
	}
	empty := SegGauge(0, 0, 8, Palette.Green, Palette.Mauve)
	if strings.Contains(strings.ReplaceAll(empty, "░", ""), "█") {
		t.Fatalf("zero pct must be all faint: %q", empty)
	}
}

func TestConfirmDialogAnswering(t *testing.T) {
	c := NewConfirm("Delete?", "really?", "")
	c.SetWidth(40)
	_, yes, done := c.Update(keyMsg("y"))
	if !yes || !done {
		t.Fatal("y must confirm")
	}
	c2 := NewConfirm("Delete?", "really?", "detail")
	_, yes2, done2 := c2.Update(keyMsg("n"))
	if yes2 || !done2 {
		t.Fatal("n must cancel")
	}
	dlg, _, _ := c2.Update(keyMsg("d"))
	if !dlg.ShowDet {
		t.Fatal("d toggles details")
	}
}
