package screens

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"lcc2/internal/accounts"
	"lcc2/internal/files"
	"lcc2/internal/proc"
	"lcc2/internal/services"
	"lcc2/internal/ui"
)

// Regression for BACKLOG-C2 / ADR-0004: while a FilterTable owns focus,
// screen-level action keys must never fire; every key goes to the input.

func keyRunes(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func typeQuery(t *testing.T, s ui.Screen, q string) ui.Screen {
	t.Helper()
	s, _ = s.Update(keyRunes("/"))
	for _, r := range q {
		s, _ = s.Update(keyRunes(string(r)))
	}
	return s
}

func seededFiles(t *testing.T) Files {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(dir, 0755)
	f := NewFiles()
	lst, err := files.List(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	return feed(f, ui.SizeMsg{Width: 100, Height: 30}, dirListMsg{dir: dir, list: lst}).(Files)
}

func TestFilterModeSwallowsActionKeysOnFiles(t *testing.T) {
	f := seededFiles(t)
	f = typeQuery(t, f, "dxmpRah").(Files)

	if !f.tbl.Filtering() {
		t.Fatal("filter mode lost")
	}
	if f.prompt != nil || f.permEdit != nil || f.stager.Len() > 0 {
		t.Fatal("action fired during filtering")
	}
	if f.showHidden {
		t.Fatal("hidden toggle fired during filtering")
	}
	if !strings.Contains(f.tbl.FilterString(), "dxmpRah") {
		t.Fatalf("query corrupted: %q", f.tbl.FilterString())
	}
}

func TestFilterEnterCommitsNotNavigatesOnFiles(t *testing.T) {
	f := seededFiles(t)
	f = typeQuery(t, f, "zz").(Files)
	f = feed(f, tea.KeyMsg{Type: tea.KeyEnter}).(Files)

	if f.tbl.Filtering() || f.stager.Len() > 0 {
		t.Fatalf("enter did not commit filter: filtering=%v staged=%d",
			f.tbl.Filtering(), f.stager.Len())
	}
	if f.tbl.FilterString() != "zz" {
		t.Fatalf("committed filter wrong: %q", f.tbl.FilterString())
	}
}

func TestFilterModeSwallowsActionKeysOnProcesses(t *testing.T) {
	p := NewProcesses()
	p = feed(p, ui.SizeMsg{Width: 100, Height: 30},
		procListMsg([]proc.Process{{PID: 1, Name: "init", User: "root", State: "S"}})).(Processes)

	p = typeQuery(t, p, "sxKr").(Processes)

	if p.confirm != nil {
		t.Fatal("signal fired during filtering")
	}
	if !strings.Contains(p.tbl.FilterString(), "sxKr") {
		t.Fatalf("query corrupted: %q", p.tbl.FilterString())
	}
}

func TestFilterModeSwallowsActionKeysOnServices(t *testing.T) {
	sv := NewServices()
	sv = feed(sv, ui.SizeMsg{Width: 100, Height: 30}, svcListMsg{units: []services.Unit{
		{Name: "ssh.service", Active: "active", Sub: "running"},
	}}).(Services)

	sv = typeQuery(t, sv, "strDe").(Services)

	if sv.confirm != nil {
		t.Fatal("service action fired during filtering")
	}
	if !strings.Contains(sv.tbl.FilterString(), "strDe") {
		t.Fatalf("query corrupted: %q", sv.tbl.FilterString())
	}
}

func TestServicesConfirmDismissedOnActionStart(t *testing.T) {
	sv := NewServices()
	sv = feed(sv, ui.SizeMsg{Width: 100, Height: 30}, svcListMsg{units: []services.Unit{
		{Name: "ssh.service", Active: "active", Sub: "running"},
	}}).(Services)

	sv = feed(sv, keyRunes("s")).(Services)
	if sv.confirm == nil {
		t.Fatal("confirm did not open")
	}
	sv = feed(sv, keyRunes("y")).(Services)
	if sv.confirm != nil {
		t.Fatal("confirm still open after yes")
	}

	for _, res := range []error{nil, os.ErrPermission} {
		sv2, _ := sv.Update(svcActionDoneMsg{unit: "ssh.service", action: "start", err: res})
		sv = sv2.(Services)
		if sv.confirm != nil {
			t.Fatalf("confirm resurrected by done msg (err=%v)", res)
		}
	}
}

func TestFilterModeGuardsOnDisksAndUsers(t *testing.T) {
	d := NewDisks()
	d = feed(d, ui.SizeMsg{Width: 100, Height: 30}).(Disks)
	d = typeQuery(t, d, "re").(Disks)
	d = feed(d, tea.KeyMsg{Type: tea.KeyEnter}).(Disks)
	if d.mode != "fs" || d.busy || d.fsTbl.Filtering() {
		t.Fatalf("disks: enter mishandled mode=%s busy=%v filtering=%v",
			d.mode, d.busy, d.fsTbl.Filtering())
	}
	if d.fsTbl.FilterString() != "re" {
		t.Fatalf("disks committed filter wrong: %q", d.fsTbl.FilterString())
	}

	ug := NewUsersGroups()
	us, _ := accounts.Users()
	gs, _ := accounts.Groups()
	if len(us) == 0 {
		us = []accounts.User{{Name: "root", UID: 0, GID: 0}}
	}
	if len(gs) == 0 {
		gs = []accounts.Group{{Name: "root", GID: 0}}
	}
	ug = feed(ug, ui.SizeMsg{Width: 100, Height: 30}, accountsMsg{users: us, groups: gs}).(UsersGroups)
	ug = typeQuery(t, ug, "te").(UsersGroups)
	ug = feed(ug, tea.KeyMsg{Type: tea.KeyTab}).(UsersGroups)
	if ug.tab != "users" {
		t.Fatalf("users: tab leaked through filter tab=%s", ug.tab)
	}
	if !ug.uTbl.Filtering() || !strings.Contains(ug.uTbl.FilterString(), "te") {
		t.Fatalf("users filter state wrong: filtering=%v q=%q",
			ug.uTbl.Filtering(), ug.uTbl.FilterString())
	}
}
