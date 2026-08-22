package sysinfo

import (
	"testing"
	"time"
)

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{2048, "2 KB"},
		{1536, "1.5 KB"},
		{5 * 1024 * 1024, "5 MB"},
		{3.25 * 1024 * 1024 * 1024, "3.2 GB"},
	}
	for _, c := range cases {
		if got := FormatBytes(c.in); got != c.want {
			t.Errorf("FormatBytes(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFormatUptime(t *testing.T) {
	if got := FormatUptime(90 * time.Minute); got != "1h 30m" {
		t.Errorf("got %q", got)
	}
	if got := FormatUptime(26 * time.Hour); got != "1d 2h" {
		t.Errorf("got %q", got)
	}
	if got := FormatUptime(45 * time.Second); got != "45s" {
		t.Errorf("got %q", got)
	}
}

func TestFormatRate(t *testing.T) {
	if got := FormatRate(1024); got != "1 KB/s" {
		t.Errorf("got %q", got)
	}
}
