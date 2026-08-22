package screens

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lcc2/internal/accounts"
	"lcc2/internal/disk"
	"lcc2/internal/files"
	"lcc2/internal/proc"
	"lcc2/internal/services"
	"lcc2/internal/sysinfo"
	"lcc2/internal/ui"
)

var sizes = [][2]int{{60, 20}, {80, 24}, {100, 30}, {140, 40}, {200, 50}}

func assertFits(t *testing.T, name string, view string, maxW int) {
	t.Helper()
	for i, l := range strings.Split(view, "\n") {
		if w := lipgloss.Width(l); w > maxW {
			t.Errorf("%s @w=%d: line %d is %d wide (> %d): %q",
				name, maxW, i, w, maxW, l[:min(60, len(l))])
			return
		}
	}
}

func feed(s ui.Screen, msgs ...tea.Msg) ui.Screen {
	for _, m := range msgs {
		sc, _ := s.Update(m)
		s = sc
	}
	return s
}

func TestScreensFitAllSizes(t *testing.T) {
	for _, sz := range sizes {
		w, h := sz[0], sz[1]
		size := ui.SizeMsg{Width: w - 20, Height: h - 4} // sidebar + chrome

		o := NewOverview()
		o.w, o.h = size.Width, size.Height // widthSet without a tick loop
		o = feed(o, size, snapshot{
			cpu:  sysinfo.CPUSample{Cores: 4, PerCore: []float64{10, 20, 30, 40}, Total: 25},
			mem:  sysinfo.Memory{Total: 8 << 30, Used: 4 << 30, UsedPercent: 50},
			load: sysinfo.Load{One: 1.5, Five: 1.2, Fifteen: 0.9},
			net:  sysinfo.NetRates{RecvPerSec: 1024, SentPerSec: 2048, RecvTotal: 1 << 30, SentTotal: 2 << 30},
			root: &disk.Filesystem{Mountpoint: "/", Total: 100 << 30, Used: 50 << 30, UsedPercent: 50},
		}).(Overview)
		assertFits(t, "overview", o.View(), w)

		pl := procListMsg([]proc.Process{
			{PID: 1, Name: "systemd", User: "root", State: "S", Command: "/sbin/init"},
			{PID: 99999, Name: "some-very-long-process-name-here", User: "aungkhant",
				State: "R", CPUPercent: 12.5, MemPercent: 3.2,
				Command: "/usr/bin/some-very-long-command --with --many --flags here"},
		})
		p := NewProcesses()
		p = feed(p, size, pl).(Processes)
		assertFits(t, "processes", p.View(), w)
		p = feed(p, tea.KeyMsg{Type: tea.KeyEnter}).(Processes)
		assertFits(t, "processes+detail", p.View(), w)

		d := NewDisks()
		d = feed(d, size, fsListMsg(disk.ListFilesystems())).(Disks)
		assertFits(t, "disks-fs", d.View(), w)

		f := NewFiles()
		lst, _ := files.List(files.Home(), false)
		f = feed(f, size, dirListMsg{dir: files.Home(), list: lst}).(Files)
		assertFits(t, "files", f.View(), w)

		sv := NewServices()
		sv = feed(sv, size, svcListMsg{units: []services.Unit{
			{Name: "ssh.service", Active: "active", Sub: "running", Enabled: "enabled",
				Description: "OpenBSD Secure Shell server"},
			{Name: "a-very-long-service-name-without-description.service", Active: "inactive",
				Sub: "dead", Enabled: "disabled"},
		}}).(Services)
		assertFits(t, "services", sv.View(), w)

		ug := NewUsersGroups()
		us, _ := accounts.Users()
		gs, _ := accounts.Groups()
		if len(us) == 0 {
			us = []accounts.User{{Name: "root", UID: 0, GID: 0, Home: "/root", Shell: "/bin/bash"}}
		}
		if len(gs) == 0 {
			gs = []accounts.Group{{Name: "wheel", GID: 10, Members: []string{"root"}}}
		}
		ug = feed(ug, size, accountsMsg{users: us, groups: gs}).(UsersGroups)
		assertFits(t, "users", ug.View(), w)
	}
}

func TestDetailPaneAlignsWithTableTop(t *testing.T) {
	p := NewProcesses()
	p = feed(p, ui.SizeMsg{Width: 100, Height: 30},
		procListMsg([]proc.Process{{PID: 42, Name: "demo", User: "u", State: "S"}})).(Processes)
	p.all[0].PID = int32(os.Getpid()) // a PID Inspect can actually open
	p = feed(p, tea.KeyMsg{Type: tea.KeyEnter}).(Processes)
	lines := strings.Split(p.View(), "\n")
	// The detail pane's top border must start on line 0 (next to the
	// table header), not shifted down onto its own row.
	found := false
	for _, l := range lines[:2] {
		if strings.Contains(l, "╭") || strings.Contains(l, "┌") {
			found = true
		}
	}
	if !found {
		t.Fatalf("detail pane not aligned with table top:\n%s", strings.Join(lines[:6], "\n"))
	}
}
