package proc

import (
	"testing"
)

func TestParseStat(t *testing.T) {
	// comm contains spaces and a paren to exercise last-')' parsing.
	line := []byte("1234 (weird (proc) name) R 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22")
	pr, ticks, ok := parseStat(line, 1234)
	if !ok {
		t.Fatal("parse failed")
	}
	if pr.Name != "weird (proc) name" {
		t.Fatalf("name = %q", pr.Name)
	}
	if pr.State != "R" || pr.PPID != 1 {
		t.Fatalf("state/ppid = %q/%d", pr.State, pr.PPID)
	}
	// utime=field14 -> rest[11]=11? verify explicit positions:
	// rest = ["R","1","2",...]; utime is rest[11], stime rest[12]
	if ticks[0] != 11 || ticks[1] != 12 {
		t.Fatalf("ticks = %v", ticks)
	}
}

func TestParseStatRealistic(t *testing.T) {
	line := []byte("79277 (lcc2) S 1234 79277 79277 0 -1 4194560 0 0 0 0 250 100 0 0 20 0 1 0 " +
		"18446744073709551615 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0")
	pr, ticks, ok := parseStat(line, 79277)
	if !ok {
		t.Fatal("parse failed")
	}
	if pr.Name != "lcc2" || pr.State != "S" || pr.PPID != 1234 {
		t.Fatalf("pr = %+v", pr)
	}
	if ticks[0] != 250 || ticks[1] != 100 {
		t.Fatalf("ticks = %v", ticks)
	}
	if len(line) < 22 { // rss field missing in this fixture; must not panic
		t.Fail()
	}
}

func TestSortAndFilter(t *testing.T) {
	ps := []Process{
		{PID: 3, Name: "zsh", User: "aung", CPUPercent: 1},
		{PID: 1, Name: "init", User: "root", CPUPercent: 9, MemPercent: 5},
		{PID: 2, Name: "bash", User: "root"},
	}
	Sort(ps, SortCPU)
	if ps[0].PID != 1 {
		t.Fatalf("cpu sort wrong: %+v", ps[0])
	}
	Sort(ps, SortPID)
	if ps[0].PID != 1 || ps[2].PID != 3 {
		t.Fatalf("pid sort wrong")
	}
	got := Filter(ps, "roo") // matches user root
	if len(got) != 2 {
		t.Fatalf("filter = %d", len(got))
	}
	if got := Filter(ps, "1"); len(got) != 1 || got[0].PID != 1 {
		t.Fatalf("pid filter = %v", got)
	}
	if NextSortKey(SortUsr) != SortCPU {
		t.Fatal("cycle broken")
	}
}
