package screens

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lcc2/internal/accounts"
	"lcc2/internal/files"
	"lcc2/internal/proc"
	"lcc2/internal/services"
	"lcc2/internal/ui"
)

// Preview-mode layouts must fit inside the SizeMsg box they were
// given, at every size — this is what keeps panes from bleeding
// through the app frame. Previews are always-on since the pane
// redesign, so no key feeds are needed to open them.
func TestPreviewModesFit(t *testing.T) {
	sizes := [][2]int{{64, 20}, {76, 22}, {84, 24}, {100, 30}, {140, 40}}
	for _, sz := range sizes {
		w, h := sz[0], sz[1]
		size := ui.SizeMsg{Width: w, Height: h}

		// processes: preview follows selection without any toggle
		p := NewProcesses()
		p = feed(p, size, procListMsg([]proc.Process{
			{PID: int32(os.Getpid()), Name: "self", User: "u", State: "S", Command: "/bin/self --long --args"},
			{PID: 2, Name: "other", User: "root", State: "R", CPUPercent: 50},
		})).(Processes)
		fitCheck(t, fmt.Sprintf("proc-preview@%dx%d", w, h), p.View(), w)
		p = feed(p, procInspectMsg{pid: int32(os.Getpid()), d: proc.Details{
			Process:     proc.Process{PID: int32(os.Getpid()), Name: "self", Command: "/bin/self --long --args"},
			Executable:  "/bin/self",
			CWD:         "/tmp",
			Threads:     3,
			FDs:         12,
			StartedUnix: time.Now().Unix() - 90,
		}}).(Processes)
		fitCheck(t, fmt.Sprintf("proc-inspected@%dx%d", w, h), p.View(), w)

		// services with a fetched status text
		sv := NewServices()
		sv = feed(sv, size, svcListMsg{units: []services.Unit{
			{Name: "ssh.service", Active: "active", Sub: "running",
				Description: "long description text that should wrap nicely"},
		}}).(Services)
		sv.detailText = strings.Repeat("status line\n", 12)
		sv.detailUnit = "ssh.service"
		fitCheck(t, fmt.Sprintf("svc-preview@%dx%d", w, h), sv.View(), w)

		// users + detail both tabs
		us, _ := accounts.Users()
		gs, _ := accounts.Groups()
		if len(us) == 0 {
			us = []accounts.User{{Name: "root", UID: 0, GID: 0, Home: "/root", Shell: "/bin/bash"}}
		}
		if len(gs) == 0 {
			gs = []accounts.Group{{Name: "root", GID: 0}}
		}
		ug := NewUsersGroups()
		ug = feed(ug, size, accountsMsg{users: us, groups: gs}).(UsersGroups)
		ug = feed(ug, tea.KeyMsg{Type: tea.KeyEnter}).(UsersGroups)
		fitCheck(t, fmt.Sprintf("users-detail@%dx%d", w, h), ug.View(), w)

		// files + meta pane on a real entry
		dir := t.TempDir()
		files.Mkdir(dir, "sub")
		lst, _ := files.List(dir, false)
		fi := NewFiles()
		fi.cwd = dir
		fi = feed(fi, size, dirListMsg{dir: dir, list: lst}).(Files)
		fi = feed(fi, tea.KeyMsg{Type: tea.KeyEnter}).(Files)
		fitCheck(t, fmt.Sprintf("files-meta@%dx%d", w, h), fi.View(), w)
	}
}

func fitCheck(t *testing.T, name, view string, maxW int) {
	t.Helper()
	for i, l := range strings.Split(view, "\n") {
		if lw := lipgloss.Width(l); lw > maxW {
			t.Errorf("%s: line %d is %d cells > %d: %.70q", name, i, lw, maxW, l)
			return
		}
	}
}
