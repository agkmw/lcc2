package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestNotifyPushStacksNewestFirst(t *testing.T) {
	var s NotifyStack
	a := s.Push("info", "first")
	b := s.Push("ok", "second")
	if a.ID == b.ID {
		t.Fatal("ids must be unique")
	}
	items := s.Items()
	if items[0].Text != "second" || items[1].Text != "first" {
		t.Fatalf("wrong order: %+v", items)
	}
	s.Dismiss(a.ID)
	if s.Len() != 1 || s.Items()[0].Text != "second" {
		t.Fatal("dismiss removed wrong item")
	}
	s.Dismiss(999) // unknown id: no-op, no panic
}

func TestCompositeNotesPreservesContentOutsideWindow(t *testing.T) {
	rows := []string{}
	for _, ch := range []string{"A", "B", "C", "D", "E", "F", "G"} {
		rows = append(rows, strings.Repeat(ch, 60))
	}
	base := strings.Join(rows, "\n")

	var s NotifyStack
	s.Push("ok", "saved")
	out := CompositeNotes(base, s)
	lines := strings.Split(out, "\n")

	if lines[0] != strings.Split(base, "\n")[0] {
		t.Fatal("header line must be untouched")
	}
	if !strings.HasPrefix(lines[5], "FFFF") || !strings.HasPrefix(lines[6], "GGGG") {
		t.Fatalf("content destroyed below window: %q / %q", lines[5][:8], lines[6][:8])
	}
	found := false
	for _, l := range lines[1:] {
		if strings.Contains(l, "saved") {
			found = true
		}
	}
	if !found {
		t.Fatal("window text missing below header")
	}
}

func TestCompositeNotesEmptyIsIdentity(t *testing.T) {
	base := "hello\nworld"
	if got := CompositeNotes(base, NotifyStack{}); got != base {
		t.Fatal("empty stack must return base unchanged")
	}
}

func TestCompositeNotesTruncatesUnderWindowOnly(t *testing.T) {
	sty := lipgloss.NewStyle().Foreground(Palette.Red)
	base := strings.Repeat("H", 80) + "\n" +
		sty.Render(strings.Repeat("Z", 80)) + "\n" +
		strings.Repeat("Y", 80) + "\n" +
		strings.Repeat("W", 80) + "\n" +
		strings.Repeat("V", 80) + "\n" +
		strings.Repeat("U", 80) + "\n" +
		strings.Repeat("F", 80)
	var s NotifyStack
	s.Push("err", "boom")
	out := CompositeNotes(base, s)
	lines := strings.Split(out, "\n")
	if w := lipgloss.Width(lines[1]); w > 80 {
		t.Fatalf("line grew beyond terminal width: %d", w)
	}
	if !strings.Contains(lines[3], "boom") {
		t.Fatalf("window body lost over styled line: %q", lines[3][:40])
	}
}

func TestCompositeNotesNeverCoversFooter(t *testing.T) {
	var s NotifyStack
	for i := 0; i < maxNotes+2; i++ { // more notes than fit above the footer
		s.Push("info", strings.Repeat("n", 30))
	}
	base := "H\n" + strings.Repeat("x", 70) + "\nfooter"
	out := CompositeNotes(base, s)
	lines := strings.Split(out, "\n")
	if last := lines[len(lines)-1]; !strings.Contains(last, "footer") {
		t.Fatalf("footer covered: %q", last)
	}
}
