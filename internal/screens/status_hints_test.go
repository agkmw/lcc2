package screens

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestFilesStatusHintsState(t *testing.T) {
	f := loadedFiles(t)
	h := f.StatusHints()
	if len(h) != 1 || !strings.Contains(h[0].Help().Desc, "sort") {
		t.Fatalf("idle StatusHints = %+v, want just sort", h)
	}

	// stage two ops -> review + undo hints appear
	s, _ := f.Update(runeKey("P"))
	f = s.(Files)
	s, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	f = s.(Files)
	f.Update(runeKey("j"))
	sc, _ := f.Update(runeKey("P"))
	f = sc.(Files)
	sc, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	f = sc.(Files)

	var descs []string
	for _, kb := range f.StatusHints() {
		descs = append(descs, kb.Help().Key+" "+kb.Help().Desc)
	}
	joined := strings.Join(descs, "|")
	if !strings.Contains(joined, "review 2") || !strings.Contains(joined, "undo op") {
		t.Fatalf("staged StatusHints = %s", joined)
	}
}

func TestUsersStatusHintsRoot(t *testing.T) {
	rooted := loadedUsers(true)
	if h := rooted.StatusHints(); len(h) != 0 {
		t.Fatalf("root StatusHints = %+v, want empty", h)
	}
	plain := loadedUsers(false)
	h := plain.StatusHints()
	if len(h) != 1 || !strings.Contains(h[0].Help().Desc, "sudo") {
		t.Fatalf("not-root StatusHints = %+v, want sudo pointer", h)
	}
}
