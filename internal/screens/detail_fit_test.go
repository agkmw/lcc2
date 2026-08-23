package screens

import (
	"fmt"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lcc2/internal/accounts"
	"lcc2/internal/files"
	"lcc2/internal/proc"
	"lcc2/internal/services"
	"lcc2/internal/ui"
)

// Detail-mode layouts must fit inside the SizeMsg box they were given,
// at every size — this is what keeps panes from bleeding through the
// app frame.
func TestDetailModesFit(t *testing.T) {
	sizes := [][2]int{{64, 20}, {76, 22}, {84, 24}, {100, 30}, {140, 40}}
	for _, sz := range sizes {
		w, h := sz[0], sz[1]
		size := ui.SizeMsg{Width: w, Height: h}

		// processes + detail open
		p := NewProcesses()
		p = feed(p, size, procListMsg([]proc.Process{
			{PID: int32(os.Getpid()), Name: "self", User: "u", State: "S", Command: "/bin/self --long --args"},
			{PID: 2, Name: "other", User: "root", State: "R", CPUPercent: 50},
		})).(Processes)
		p = feed(p, tea.KeyMsg{Type: tea.KeyEnter}).(Processes)
		fitCheck(t, fmt.Sprintf("proc-detail@%dx%d", w, h), p.View(), w)

		// services + detail
		sv := NewServices()
		sv = feed(sv, size, svcListMsg{units: []services.Unit{
			{Name: "ssh.service", Active: "active", Sub: "running",
				Description: "long description text that should wrap nicely"},
		}}).(Services)
		sv.detailText = strings.Repeat("status line\n", 12)
		sv = feed(sv, tea.KeyMsg{Type: tea.KeyEnter}).(Services)
		fitCheck(t, fmt.Sprintf("svc-detail@%dx%d", w, h), sv.View(), w)

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
