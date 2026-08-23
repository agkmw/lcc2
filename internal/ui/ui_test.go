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
	if lipgloss.Width(got) > 8 || !strings.HasSuffix(got, "…") {
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
