package screens

import (
	"fmt"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lcc2/internal/proc"
	"lcc2/internal/sysinfo"
	"lcc2/internal/ui"
)

const procInterval = 3 * time.Second

type procTickMsg struct{ gen uint64 }

type procInspectMsg struct {
	pid int32
	d   proc.Details
	err error
}

type killDoneMsg struct {
	pid int32
	err error
}

// Processes is the process management screen: list left, a live
// preview of the selected process right.
type Processes struct {
	w, h    int
	col     *proc.Collector
	all     []proc.Process
	tbl     ui.FilterTable
	sortKey proc.SortKey

	insp    *proc.Details // preview of the selected process
	inspPID int32         // pid currently inspected (0 = none)

	scanning *atomic.Int32  // guards against overlapping full scans
	epoch    *atomic.Uint64 // tick-chain generation; stale chains die

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
		scanning: &atomic.Int32{},
		epoch:    &atomic.Uint64{},
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
		key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "terminate")),
		key.NewBinding(key.WithKeys("K"), key.WithHelp("K", "force kill")),
		ui.Keys.Refresh,
	}
}

// CapturingInput implements ui.Screen.
func (p Processes) CapturingInput() bool {
	return p.tbl.Filtering() || p.confirm != nil
}

// Init starts the refresh loop; re-entry retires the previous chain.
func (p Processes) Init() tea.Cmd { return p.tick(p.epoch.Add(1)) }

func (p Processes) tick(gen uint64) tea.Cmd {
	return tea.Batch(p.refresh(), p.inspectSelected(),
		tea.Tick(procInterval, func(time.Time) tea.Msg {
			return procTickMsg{gen: gen}
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

// inspectSelected refreshes the preview for the cursor's process; nil
// when nothing is selected.
func (p Processes) inspectSelected() tea.Cmd {
	idx, ok := p.tbl.Selected()
	if !ok || idx >= len(p.all) {
		return nil
	}
	pid := p.all[idx].PID
	return func() tea.Msg {
		d, err := proc.Inspect(pid)
		return procInspectMsg{pid: pid, d: d, err: err}
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

	case procInspectMsg:
		if m.pid != p.selectedPID() {
			return p, nil // stale inspection of a since-moved cursor
		}
		if m.err != nil {
			p.insp, p.inspPID = nil, 0 // list row keeps the pane filled
			break
		}
		d := m.d
		p.insp, p.inspPID = &d, m.pid

	case procTickMsg:
		if m.gen != p.epoch.Load() {
			return p, nil // stale chain from a previous Init
		}
		return p, p.tick(m.gen)

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

	if p.tbl.Filtering() {
		var cmd tea.Cmd
		p.tbl, cmd = p.tbl.Update(m)
		return p, cmd
	}

	switch m.String() {
	case "s":
		p.sortKey = proc.NextSortKey(p.sortKey)
		proc.Sort(p.all, p.sortKey)
		p.syncTable()
		return p, nil
	case "r":
		return p, tea.Batch(p.refresh(), p.inspectSelected())
	case "x":
		_, cmd := p.askSignal(syscall.SIGTERM)
		return p, cmd
	case "K":
		_, cmd := p.askSignal(syscall.SIGKILL)
		return p, cmd
	}

	moved := false
	switch m.String() {
	case "up", "down", "j", "k", "g", "G", "home", "end", "pgup", "pgdown":
		moved = true
	}
	var cmd tea.Cmd
	p.tbl, cmd = p.tbl.Update(m)
	if moved {
		cmd = tea.Batch(cmd, p.inspectSelected())
	}
	return p, cmd
}

// selectedPID returns the PID under the cursor (0 = none).
func (p Processes) selectedPID() int32 {
	pr, ok := p.selectedProc()
	if !ok {
		return 0
	}
	return pr.PID
}

func (p Processes) selectedProc() (proc.Process, bool) {
	idx, ok := p.tbl.Selected()
	if !ok || idx >= len(p.all) {
		return proc.Process{}, false
	}
	return p.all[idx], true
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
	wide, _, _ := splitGeom(p.w)
	th := clampInt(p.h-1, 5, p.h)
	tw := p.w - 2
	if wide {
		_, mainW, _ := splitGeom(p.w)
		tw = mainW
	}
	p.tbl.SetSize(tw, th)
}

// syncTable rebuilds visible rows; the cursor follows its PID.
func (p *Processes) syncTable() {
	rows := make([]table.Row, len(p.all))
	keys := make([]string, len(p.all))
	for i, pr := range p.all {
		rows[i] = table.Row{
			faintSty.Render(itoa(int(pr.PID))),
			userCell(pr.User),
			pctCell(pr.CPUPercent),
			memPctCell(pr.MemPercent),
			stateCell(pr.State),
			cmdCell(pr),
		}
		keys[i] = itoa(int(pr.PID))
	}
	p.tbl.SetRowsTracked(rows, keys)
}

// pctCell colors cpu percentages on the shared 70/90 scale.
func pctCell(v float64) string {
	return lipgloss.NewStyle().Foreground(ui.StateColor(v)).Render(f1(v))
}

// memPctCell uses memory-appropriate thresholds — a process holding
// half of RAM is remarkable in a way 50% cpu is not.
func memPctCell(v float64) string {
	c := ui.Palette.Green
	switch {
	case v >= 70:
		c = ui.Palette.Red
	case v >= 40:
		c = ui.Palette.Yellow
	}
	return lipgloss.NewStyle().Foreground(c).Render(f1(v))
}

// userCell tones the row owner: root loud, everyone else muted.
func userCell(u string) string {
	if u == "root" {
		return warnSty.Render(truncCell(u, 11))
	}
	return mutedSty.Render(truncCell(u, 11))
}

// cmdCell dims kernel threads ([kthreadd] and children) entirely; for
// real processes the arguments after the executable go faint so the
// binary name carries the row.
func cmdCell(pr proc.Process) string {
	if strings.HasPrefix(pr.Name, "[") {
		return mutedSty.Render(truncCell(pr.Command, 60))
	}
	if i := strings.IndexByte(pr.Command, ' '); i > 0 {
		rest := pr.Command[i+1:]
		exe := truncCell(pr.Command[:i], maxInt(60-len(rest)-1, 8))
		args := faintSty.Render(truncCell(rest, maxInt(58-lipgloss.Width(exe), 6)))
		return exe + " " + args
	}
	return truncCell(pr.Command, 60)
}

func stateCell(s string) string {
	var c lipgloss.TerminalColor
	switch s {
	case "R":
		c = goodSty.GetForeground()
	case "D":
		c = warnSty.GetForeground()
	case "Z", "X":
		c = badSty.GetForeground()
	default:
		return mutedSty.Render(s)
	}
	return lipgloss.NewStyle().Bold(true).Foreground(c).Render(s)
}

func truncCell(s string, w int) string {
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	return string(r[:w-2]) + ".."
}

// View renders the list with the live process preview beside it.
func (p Processes) View() string {
	if !p.loaded || p.w == 0 {
		return lipgloss.Place(p.w, p.h, lipgloss.Center, lipgloss.Center,
			ui.EmptyState("", "Scanning processes..", "", p.w))
	}
	head := pageHead("Processes",
		fmt.Sprintf("sorted by %s - %d running", p.sortKey, len(p.all)), p.w)

	main := p.tbl.View()
	prev := p.previewBody()
	wide, mainW, _ := splitGeom(p.w)
	if !wide { // stacked: give the list most of the height
		keep := clampInt(p.h-8, 6, p.h-4)
		main = ui.ClipBlock(main, mainW)
		mainLines := strings.Split(main, "\n")
		if len(mainLines) > keep {
			main = strings.Join(mainLines[:keep], "\n")
		}
	}
	body := joinPanesWide(wide, main, prev, mainW, p.w)

	out := head + "\n" + body
	lines := strings.Split(out, "\n")
	if len(lines) > p.h {
		lines = lines[:p.h]
	}
	for len(lines) < p.h {
		lines = append(lines, "")
	}
	view := strings.Join(lines, "\n")
	if p.confirm != nil {
		return overlayCenter(view, p.confirm.View(), p.w)
	}
	return view
}

// previewBody renders the right-hand pane for the selected process.
// It draws straight from the list row so the pane is never empty
// while the async inspection is in flight.
func (p Processes) previewBody() string {
	_, _, prevW := splitGeom(p.w)
	pr, ok := p.selectedProc()
	var title, meta, body string
	switch {
	case !ok:
		// nothing selected: blank pane
	case p.insp != nil && p.inspPID == pr.PID:
		title = truncCell(p.insp.Name, maxInt(prevW-4, 6))
		meta = "pid " + itoa(int(p.insp.PID))
		body = processCard(p.insp.Process, p.insp, prevW)
	default:
		title = truncCell(pr.Name, maxInt(prevW-4, 6))
		meta = "pid " + itoa(int(pr.PID))
		body = processCard(pr, nil, prevW)
	}
	return renderPreview("proc", title, meta, body, prevW, p.h-1)
}

// processCard renders key/value lines from the list row, enriched
// when full details are available.
func processCard(d proc.Process, insp *proc.Details, w int) string {
	kv := func(k, v string) string {
		return mutedSty.Render(padTo(k, 9)) + faintSty.Render(ui.Truncate(v, maxInt(w-11, 4)))
	}
	head := lipgloss.NewStyle().Bold(true).Foreground(ui.Accent("proc")).
		Render(truncCell(d.Name, maxInt(w-2, 6)))
	lines := []string{head, "",
		kv("pid", itoa(int(d.PID))),
		kv("user", d.User),
		kv("state", stateName(d.State)),
		kv("cpu", f1(d.CPUPercent)+"%"),
		kv("memory", f1(d.MemPercent)+"% ("+sysinfo.FormatBytes(float64(d.RSS))+")"),
	}
	if insp != nil {
		set := func(k, v string) {
			for i, l := range lines {
				if strings.HasPrefix(l, padTo(k, 9)) {
					lines[i] = kv(k, v)
					return
				}
			}
			lines = append(lines, kv(k, v))
		}
		set("parent", itoa(int(insp.PPID)))
		set("threads", itoa(int(insp.Threads)))
		set("fds", itoa(int(insp.FDs)))
		set("nice", itoa(int(insp.Nice)))
		set("started", proc.FormatAge(insp.StartedUnix))
		if insp.Executable != "" {
			lines = append(lines, kv("exe", insp.Executable))
		}
		if insp.CWD != "" {
			lines = append(lines, kv("cwd", insp.CWD))
		}
	}
	lines = append(lines, "", faintSty.Render("cmdline"))
	lines = append(lines, strings.Split(wordWrap(d.Command, maxInt(w-2, 10)), "\n")...)
	return strings.Join(lines, "\n")
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
				r := []rune(line)
				out = append(out, string(r[:w]))
				line = string(r[w:])
			}
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}
