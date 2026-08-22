package screens

import (
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lcc2/internal/proc"
	"lcc2/internal/sysinfo"
	"lcc2/internal/ui"
)

const procInterval = 3 * time.Second

type procTickMsg struct{}

type killDoneMsg struct {
	pid int32
	err error
}

// Processes is the process management screen.
type Processes struct {
	w, h    int
	col     *proc.Collector
	all     []proc.Process
	tbl     ui.FilterTable
	sortKey proc.SortKey

	detailOpen bool
	detail     proc.Details
	dvp        viewport.Model

	selectedPID int32
	scanning    *atomic.Int32 // guards against overlapping full scans

	confirm     *ui.ConfirmDialog
	pendingSig  syscall.Signal
	pendingPID  int32
	pendingName string

	loaded bool
}

// NewProcesses builds the process screen.
func NewProcesses() Processes {
	cols := []table.Column{
		{Title: "pid", Width: 8},
		{Title: "user", Width: 11},
		{Title: "cpu%", Width: 6},
		{Title: "mem%", Width: 6},
		{Title: "s", Width: 1},
		{Title: "command", Width: 40},
	}
	return Processes{
		col:      proc.NewCollector(),
		sortKey:  proc.SortCPU,
		tbl:      ui.NewFilterTable(cols, 80, 20),
		dvp:      viewport.New(40, 10),
		scanning: &atomic.Int32{},
	}
}

// ID implements ui.Screen.
func (p Processes) ID() string { return "proc" }

// Title implements ui.Screen.
func (p Processes) Title() string { return "Processes" }

// Hints implements ui.Screen.
func (p Processes) Hints() []key.Binding {
	return []key.Binding{
		ui.Keys.Filter, key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort: "+string(p.sortKey))),
		ui.Keys.Select,
		key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "terminate")),
		key.NewBinding(key.WithKeys("K"), key.WithHelp("K", "force kill")),
		ui.Keys.Refresh,
	}
}

// CapturingInput implements ui.Screen.
func (p Processes) CapturingInput() bool {
	return p.tbl.Filtering() || p.confirm != nil || p.detailOpen
}

// Init starts the refresh loop.
func (p Processes) Init() tea.Cmd { return p.tick() }

func (p Processes) tick() tea.Cmd {
	return tea.Batch(p.refresh(), tea.Tick(procInterval, func(time.Time) tea.Msg {
		return procTickMsg{}
	}))
}

func (p Processes) refresh() tea.Cmd {
	if !p.scanning.CompareAndSwap(0, 1) {
		return nil // a scan is still in flight; the next tick will retry
	}
	return func() tea.Msg {
		defer p.scanning.Store(0)
		ps, err := p.col.Snapshot()
		if err != nil {
			return ui.ToastMsg{Kind: "err", Text: "process scan failed"}
		}
		return procListMsg(ps)
	}
}

type procListMsg []proc.Process

// Update handles input and data messages.
func (p Processes) Update(msg tea.Msg) (ui.Screen, tea.Cmd) {
	switch m := msg.(type) {
	case ui.SizeMsg:
		p.w, p.h = m.Width, m.Height
		p.layout()

	case procListMsg:
		// Store only; the single tick chain owns all scheduling.
		p.all = m
		proc.Sort(p.all, p.sortKey)
		p.syncTable()
		p.loaded = true
		if p.detailOpen {
			if d, err := proc.Inspect(p.detail.PID); err == nil {
				p.detail = d
				p.setDetailContent()
			}
		}

	case procTickMsg:
		return p, p.tick()

	case killDoneMsg:
		p.confirm = nil
		if m.err != nil {
			return p, ui.ErrToast("cannot signal " + itoa(int(m.pid)))
		}
		sig := "terminated"
		if p.pendingSig == syscall.SIGKILL {
			sig = "killed"
		}
		return p, ui.OkToast(sig + " " + itoa(int(m.pid)))

	case tea.KeyMsg:
		return p.handleKey(m)
	}
	return p, nil
}

func (p Processes) handleKey(m tea.KeyMsg) (ui.Screen, tea.Cmd) {
	if p.confirm != nil {
		dlg, yes, done := p.confirm.Update(m)
		*p.confirm = dlg
		if done && yes {
			return p, signalCmd(p.pendingPID, p.pendingSig)
		}
		if done {
			p.confirm = nil
		}
		return p, nil
	}

	if p.detailOpen {
		switch m.String() {
		case "esc", "enter", "q":
			p.detailOpen = false
			p.layout()
			return p, nil
		}
		var cmd tea.Cmd
		p.dvp, cmd = p.dvp.Update(m)
		return p, cmd
	}

	switch m.String() {
	case "s":
		p.sortKey = proc.NextSortKey(p.sortKey)
		proc.Sort(p.all, p.sortKey)
		p.syncTable()
		return p, nil
	case "r":
		return p, p.refresh()
	case "enter":
		if idx, ok := p.tbl.Selected(); ok && idx < len(p.all) {
			sel := p.all[idx]
			if d, err := proc.Inspect(sel.PID); err == nil {
				p.detail = d
				p.detailOpen = true
				p.layout()
				p.setDetailContent()
			} else {
				return p, ui.ErrToast("cannot inspect " + sel.Name)
			}
		}
		return p, nil
	case "x":
		_, cmd := p.askSignal(syscall.SIGTERM)
		return p, cmd
	case "K":
		_, cmd := p.askSignal(syscall.SIGKILL)
		return p, cmd
	}

	var cmd tea.Cmd
	p.tbl, cmd = p.tbl.Update(m)
	return p, cmd
}

func (p Processes) askSignal(sig syscall.Signal) (ui.Screen, tea.Cmd) {
	idx, ok := p.tbl.Selected()
	if !ok || idx >= len(p.all) {
		return p, nil
	}
	target := p.all[idx]
	if target.PID == int32(1) {
		return p, ui.ErrToast("refusing to signal init")
	}
	p.pendingPID = target.PID
	p.pendingName = target.Name
	p.pendingSig = sig
	action := "Terminate"
	body := "Send SIGTERM to " + target.Name + " (" + itoa(int(target.PID)) + ")?"
	if sig == syscall.SIGKILL {
		action = "Force kill"
		body = "Send SIGKILL to " + target.Name + " (" + itoa(int(target.PID)) + ")? The process cannot clean up."
	}
	dlg := ui.NewConfirm(action, body, "")
	dlg.SetWidth(clampInt(p.w-8, 44, 70))
	p.confirm = &dlg
	return p, nil
}

func signalCmd(pid int32, sig syscall.Signal) tea.Cmd {
	return func() tea.Msg {
		err := proc.Signal(pid, sig)
		return killDoneMsg{pid: pid, err: err}
	}
}

func (p *Processes) layout() {
	detailW := 0
	if p.detailOpen {
		detailW = clampInt(p.w/3, 34, 52)
	}
	tw := clampInt(p.w-detailW-3, 30, p.w)
	th := clampInt(p.h-(p.filterBarHeight()), 5, p.h)
	p.tbl.SetSize(tw, th)
	p.dvp.Width = detailW - 4
	p.dvp.Height = th
}

func (p Processes) filterBarHeight() int {
	if p.tbl.Filtering() || p.tbl.FilterString() != "" {
		return 2
	}
	return 1
}

// syncTable rebuilds visible rows, preserving cursor by PID.
func (p *Processes) syncTable() {
	if oldIdx, hasOld := p.tbl.Selected(); hasOld && oldIdx < len(p.all) {
		p.selectedPID = p.all[oldIdx].PID
	}
	rows := make([]table.Row, len(p.all))
	for i, pr := range p.all {
		rows[i] = table.Row{
			itoa(int(pr.PID)),
			truncCell(pr.User, 11),
			f1(pr.CPUPercent),
			f1(pr.MemPercent),
			pr.State,
			truncCell(pr.Command, 60),
		}
	}
	p.tbl.SetRows(rows)
	if p.selectedPID > 0 {
		for vi, oi := range p.tbl.VisibleOrigins() {
			if oi < len(p.all) && p.all[oi].PID == p.selectedPID {
				p.tbl.SetCursor(vi)
				break
			}
		}
	}
}

func truncCell(s string, w int) string {
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	return string(r[:w-1]) + "…"
}

func (p *Processes) setDetailContent() {
	d := p.detail
	var b strings.Builder
	kv := func(k, v string) {
		b.WriteString(mutedSty.Render(padTo(k, 9)) + faintSty.Render(v) + "\n")
	}
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ui.Accent("proc")).
		Render(d.Name) + "\n\n")
	kv("pid", itoa(int(d.PID)))
	kv("parent", itoa(int(d.PPID)))
	kv("user", d.User)
	kv("state", stateName(d.State))
	kv("cpu", f1(d.CPUPercent)+"%")
	kv("memory", f1(d.MemPercent)+"% ("+sysinfo.FormatBytes(float64(d.RSS))+")")
	kv("threads", itoa(int(d.Threads)))
	kv("fds", itoa(int(d.FDs)))
	kv("nice", itoa(int(d.Nice)))
	kv("started", proc.FormatAge(d.StartedUnix))
	if d.Executable != "" {
		kv("exe", d.Executable)
	}
	if d.CWD != "" {
		kv("cwd", d.CWD)
	}
	b.WriteString("\n" + faintSty.Render("cmdline") + "\n")
	b.WriteString(wordWrap(d.Command, p.dvp.Width))
	p.dvp.SetContent(b.String())
	p.dvp.GotoTop()
}

// View renders table plus optional detail pane and confirm dialog.
func (p Processes) View() string {
	if !p.loaded || p.w == 0 {
		return lipgloss.Place(p.w, p.h, lipgloss.Center, lipgloss.Center,
			ui.EmptyState("", "Scanning processes…", "", p.w))
	}
	head := faintSty.Render("sorted by "+string(p.sortKey)+"  ") +
		mutedSty.Render(itoa(len(p.all))+" processes")
	body := head + "\n" + p.tbl.View()

	if p.detailOpen {
		detailW := clampInt(p.w/3, 34, 52)
		pane := ui.Panel().BorderForeground(ui.Accent("proc")).
			Width(detailW).
			Padding(0, 1).
			Render(p.dvp.View())
		body = joinPanes(body, pane)
	}
	if p.confirm != nil {
		return lipgloss.Place(p.w, p.h, lipgloss.Center, lipgloss.Center, p.confirm.View())
	}
	return body
}

func stateName(s string) string {
	switch s {
	case "R":
		return "running"
	case "S":
		return "sleeping"
	case "D":
		return "disk wait"
	case "Z":
		return "zombie"
	case "T", "t":
		return "stopped"
	case "I":
		return "idle"
	default:
		return s
	}
}

func wordWrap(s string, w int) string {
	if w < 10 {
		w = 10
	}
	var out []string
	for _, para := range strings.Split(s, "\n") {
		line := ""
		for _, word := range strings.Split(para, " ") {
			if line != "" && lipgloss.Width(line)+1+lipgloss.Width(word) > w {
				out = append(out, line)
				line = word
				continue
			}
			if line == "" {
				line = word
			} else {
				line += " " + word
			}
			for lipgloss.Width(line) > w { // break very long words
				out = append(out, line[:w])
				line = line[w:]
			}
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}
