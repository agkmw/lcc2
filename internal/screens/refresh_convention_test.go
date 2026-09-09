package screens

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"lcc/internal/services"
	"lcc/internal/ui"
)

// r refreshes on every screen; the r-related action (services
// restart) lives on shift-r. Regression: services had r = restart
// confirm and R = refresh, the one outlier.
func TestRefreshKeyConvention(t *testing.T) {
	size := ui.SizeMsg{Width: 100, Height: 30}
	sv := NewServices()
	sv = feed(sv, size, svcListMsg{units: []services.Unit{
		{Name: "ssh.service", Active: "active", Sub: "running",
			Enabled: "enabled", Description: "ssh"},
	}}).(Services)

	sc, cmd := sv.Update(keyRunes("r"))
	ns := sc.(Services)
	if ns.confirm != nil {
		t.Fatal("r opened the restart confirm; r must refresh")
	}
	if cmd == nil {
		t.Fatal("r returned no refresh command")
	}
	sc, _ = ns.Update(keyRunes("R"))
	if sc.(Services).confirm == nil {
		t.Fatal("R did not open the restart confirm")
	}

	// Files advertises and handles r as refresh too: the returned
	// command re-lists the current directory.
	f := NewFiles()
	if _, cmd := f.Update(tea.KeyPressMsg{Code: 'r'}); cmd == nil {
		t.Fatal("files r returned no refresh command")
	}
	for _, b := range f.Hints() {
		if b.Keys()[0] == "r" && b.Help().Desc != "refresh" {
			t.Fatalf("files r hint = %q, want refresh", b.Help().Desc)
		}
	}
	// And the shared binding keeps its identity everywhere.
	if ui.Keys.Refresh.Help().Key != "r" {
		t.Fatal("shared refresh binding drifted off r")
	}
}
