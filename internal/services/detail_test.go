package services

import (
	"testing"
	"time"
)

func TestParseShow(t *testing.T) {
	out := "ActiveState=active\n" +
		"SubState=running\n" +
		"UnitFileState=enabled\n" +
		"MainPID=842\n" +
		"MemoryCurrent=12828672\n" +
		"CPUUsageNSec=2158349000\n" +
		"ExecMainStartTimestamp=Sun 2026-08-23 19:59:42 +0630\n" +
		"NRestarts=2\n"
	d := parseShow(out)
	if d.Active != "active" || d.Sub != "running" || d.Boot != "enabled" {
		t.Fatalf("states parsed wrong: %+v", d)
	}
	if d.PID != 842 || d.Restarts != 2 {
		t.Fatalf("pid/restarts wrong: %+v", d)
	}
	if d.MemBytes != 12828672 || d.CPUNanos != 2158349000 {
		t.Fatalf("mem/cpu wrong: %+v", d)
	}
	want, _ := time.Parse("Mon 2006-01-02 15:04:05 -0700", "Sun 2026-08-23 19:59:42 +0630")
	if !d.Since.Equal(want) {
		t.Fatalf("since = %v, want %v", d.Since, want)
	}
}

// Exited units report unset memory/cpu; the parser must tolerate it.
func TestParseShowUnsetValues(t *testing.T) {
	out := "ActiveState=inactive\nSubState=dead\nUnitFileState=disabled\n" +
		"MainPID=0\nMemoryCurrent=[not set]\nCPUUsageNSec=0\n" +
		"NRestarts=0\nExecMainStartTimestamp=\n"
	d := parseShow(out)
	if d.MemBytes != 0 || d.PID != 0 || !d.Since.IsZero() {
		t.Fatalf("unset handling broken: %+v", d)
	}
}

func TestJournalDegradesSilently(t *testing.T) {
	lines := Journal("definitely-not-a-real-unit-xyz.service", 3)
	_ = lines // may be empty (no journalctl / no logs); must not panic
	if len(lines) > 3 {
		t.Fatalf("cap ignored: %d lines", len(lines))
	}
}
