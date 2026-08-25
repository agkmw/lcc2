package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"lcc2/internal/screens"
)

// denyGlyphs are East-Asian-Ambiguous codepoints: tmux and CJK-locale
// terminals render them double-width while our cell math counts one,
// shifting every following column (the "services bleed"). Chrome and
// provider-derived text must never emit them. See ADR-0010.
var denyGlyphs = func() map[rune]bool {
	m := map[rune]bool{}
	for _, r := range "●○◌◐◑◕◉✕✗✔✖▸◂▴▾◄►◊◈◇◆•·‣›‹…—" {
		m[r] = true
	}
	return m
}()

func offending(frame string) string {
	for i, r := range frame {
		if denyGlyphs[r] {
			return string(r) + " (offset " + itoaOff(i) + ")"
		}
	}
	return ""
}

func itoaOff(i int) string {
	s := ""
	if i == 0 {
		return "0"
	}
	for i > 0 {
		s = string(rune('0'+i%10)) + s
		i /= 10
	}
	return s
}

// Every rendered section, at several sizes, must be glyph-clean.
func TestFramesContainNoAmbiguousGlyphs(t *testing.T) {
	for _, sz := range [][2]int{{80, 24}, {110, 32}, {150, 44}} {
		w, h := sz[0], sz[1]
		r := New(screens.NewOverview(), screens.NewProcesses(),
			screens.NewDisks(), screens.NewFiles(),
			screens.NewServices(), screens.NewUsersGroups())
		m, _ := r.Update(tea.WindowSizeMsg{Width: w, Height: h})
		for key := byte('1'); key <= '6'; key++ {
			m, _ = m.Update(keyMsg(string(key)))
			if off := offending(viewString(m)); off != "" {
				t.Errorf("w=%d sec=%c: ambiguous glyph %s", w, key, off)
			}
		}
		// help overlay path too
		m, _ = m.Update(keyMsg("?"))
		if off := offending(viewString(m)); off != "" {
			t.Errorf("w=%d help: ambiguous glyph %s", w, off)
		}
	}
}
