package services

import "testing"

func TestParseListUnits(t *testing.T) {
	out := `  ssh.service     loaded active running OpenBSD Secure Shell server
nginx.service loaded failed failed  A high performance web server
cups.socket loaded active running CUPS Scheduler
notaservice loaded active running should be skipped
`
	units := parseListUnits(out)
	if len(units) != 2 {
		t.Fatalf("got %d units, want 2", len(units))
	}
	if units[0].Name != "ssh.service" || units[0].Active != "active" ||
		units[0].Sub != "running" {
		t.Fatalf("unit0 = %+v", units[0])
	}
	if units[0].Description != "OpenBSD Secure Shell server" {
		t.Fatalf("desc = %q", units[0].Description)
	}
	if units[1].Active != "failed" || units[1].Sub != "failed" {
		t.Fatalf("unit1 = %+v", units[1])
	}
}

func TestParseUnitFiles(t *testing.T) {
	out := `ssh.service enabled enabled
nginx.service disabled disabled
udisks2.service static -
`
	m := parseUnitFiles(out)
	if m["ssh.service"] != "enabled" || m["nginx.service"] != "disabled" ||
		m["udisks2.service"] != "static" {
		t.Fatalf("map = %v", m)
	}
}
