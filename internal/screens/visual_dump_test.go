package screens

import (
	"os"
	"strings"
	"testing"

	"lcc2/internal/disk"
	"lcc2/internal/files"
	"lcc2/internal/proc"
	"lcc2/internal/services"
	"lcc2/internal/sysinfo"
	"lcc2/internal/ui"
)

func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func TestVisualDump(t *testing.T) {
	if os.Getenv("DUMP") == "" {
		t.Skip("set DUMP=1")
	}
	w, h := 110, 32

	o := NewOverview()
	ov := feed(o, ui.SizeMsg{Width: w - 4, Height: h - 4}, snapshot{
		cpu:  sysinfo.CPUSample{Cores: 4, PerCore: []float64{5, 92, 30, 12}, Total: 33},
		mem:  sysinfo.Memory{Total: 16 << 30, Used: 9 << 30, Cached: 4 << 30, UsedPercent: 56},
		load: sysinfo.Load{One: 1.24, Five: 0.98, Fifteen: 0.77},
		net:  sysinfo.NetRates{RecvPerSec: 3 << 20, SentPerSec: 220 << 10, RecvTotal: 41 << 30, SentTotal: 9 << 30},
		fss: []disk.Filesystem{
			{Mountpoint: "/", Total: 500 << 30, Used: 310 << 30, UsedPercent: 62},
			{Mountpoint: "/home", Total: 200 << 30, Used: 40 << 30, UsedPercent: 20},
		},
	}).(Overview)
	for i := 0; i < 40; i++ { // fill history so graphs show shape
		ov.observe(snapFixture(float64(i) * 2))
	}
	dump(t, "overview", ov.View())

	p := NewProcesses()
	p = feed(p, ui.SizeMsg{Width: w - 4, Height: h - 4},
		procListMsg([]proc.Process{
			{PID: 1234, Name: "firefox", User: "aungkhant", State: "S", CPUPercent: 12.4, MemPercent: 8.1, Command: "/usr/lib/firefox/firefox"},
			{PID: 999, Name: "systemd", User: "root", State: "S", Command: "/sbin/init splash"},
		})).(Processes)
	dump(t, "processes", p.View())

	dir := t.TempDir()
	os.Mkdir(dir+"/projects", 0755)
	os.WriteFile(dir+"/README.md", []byte("# hello world\nsecond line"), 0644)
	lst, _ := files.List(dir, false)
	fi := NewFiles()
	fi = feed(fi, ui.SizeMsg{Width: w - 4, Height: h - 4}, dirListMsg{dir: dir, list: lst}).(Files)
	dump(t, "files", fi.View())
}

func snapFixture(total float64) snapshot {
	return snapshot{
		cpu: sysinfo.CPUSample{Cores: 4,
			PerCore: []float64{total, total / 2, total / 3, total},
			Total:   total},
		net: sysinfo.NetRates{RecvPerSec: total * (3 << 15), SentPerSec: total * (1 << 14)},
	}
}

func dump(t *testing.T, name, view string) {
	t.Helper()
	for _, l := range strings.Split(stripANSI(view), "\n") {
		println(l)
	}
	println("=== end " + name + " ===")
}

var _ = services.Unit{} // silence if unused in future edits
