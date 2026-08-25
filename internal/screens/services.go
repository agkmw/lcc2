package screens

import (
	"fmt"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/bubbles/v2/key"

	"lcc2/internal/services"
	"lcc2/internal/sysinfo"
	"lcc2/internal/ui"
)

const svcRefreshInterval = 15 * time.Second

type svcListMsg struct {
	units []services.Unit
	err   error
}
type svcActionDoneMsg struct {
	unit   string
	action string
	err    error
}
type svcShowMsg struct {
	unit string
	d    services.Detail
	err  error
}
type svcJournalMsg struct {
	unit  string
	lines []string
}
type svcRawMsg struct {
	unit string
	text string
}
type editorDoneMsg struct{ err error }
type svcTickMsg struct{ gen uint64 }

// Services is the service management screen: unit list left, live
// systemctl status preview right, silent periodic refresh.
type Services struct {
	w, h    int
	spin    spinner.Model
	units   []services.Unit
	tbl     ui.FilterTable
	loaded  bool
	loadErr string

	detailUnit string // unit whose details are displayed
	det        *services.Detail
	detRaw     string // raw systemctl status fallback
	logs       []string
	fetching   bool

	confirm   *ui.ConfirmDialog
	pendingOp [2]string // unit, action

	epoch *atomic.Uint64 // auto-refresh chain generation
}

// NewServices builds the services screen.
func NewServices() Services {
	cols := []ui.Column{
		{Title: "unit", Width: 28},
		{Title: "active", Width: 10},
		{Title: "sub", Width: 12},
		{Title: "boot", Width: 10},
		{Title: "description", Width: 40},
	}
	return Services{
		tbl:   ui.NewFilterTable(cols, 80, 18),
		spin:  spinner.New(spinner.WithSpinner(spinner.Dot)),
		epoch: &atomic.Uint64{},
	}
}

// ID implements ui.Screen.
func (s Services) ID() string { return "services" }

// Title implements ui.Screen.
func (s Services) Title() string { return "Services" }

// Hints implements ui.Screen.
func (s Services) Hints() []key.Binding {
	return []key.Binding{
		ui.Keys.Filter,
		key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "start")),
		key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "stop")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "restart")),
		key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "enable")),
		key.NewBinding(key.WithKeys("E"), key.WithHelp("E", "edit unit")),
		key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "disable")),
		ui.Keys.Refresh,
	}
}

// CapturingInput implements ui.Screen.
func (s Services) CapturingInput() bool {
	return s.tbl.Filtering() || s.confirm != nil
}

// Init loads the unit list and starts the loading spinner and the
// silent refresh chain; re-entry retires stale chains via epochs.
func (s Services) Init() tea.Cmd {
	gen := s.epoch.Add(1)
	return tea.Batch(
		func() tea.Msg { u, err := services.List(); return svcListMsg{units: u, err: err} },
		func() tea.Msg { return s.spin.Tick() },
		s.tick(gen),
	)
}

func (s Services) tick(gen uint64) tea.Cmd {
	return tea.Tick(svcRefreshInterval, func(time.Time) tea.Msg {
		return svcTickMsg{gen: gen}
	})
}

func refreshSvcCmd() tea.Cmd {
	return func() tea.Msg { u, err := services.List(); return svcListMsg{units: u, err: err} }
}

// fetchDetail issues the structured show + journal pair for a unit;
// when show fails the screen follows up with the raw status dump.
func fetchDetailCmd(unit string) tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			d, err := services.Show(unit)
			return svcShowMsg{unit: unit, d: d, err: err}
		},
		func() tea.Msg { return svcJournalMsg{unit: unit, lines: services.Journal(unit, 4)} },
	)
}

func rawStatusCmd(unit string) tea.Cmd {
	return func() tea.Msg {
		// systemctl status leads lines with ambiguous-width glyphs;
		// narrow them so tmux cannot shift the pane columns.
		return svcRawMsg{unit: unit, text: ui.Narrow(services.StatusDetail(unit))}
	}
}

// Update handles messages.
func (s Services) Update(msg tea.Msg) (ui.Screen, tea.Cmd) {
	switch m := msg.(type) {
	case ui.SizeMsg:
		s.w, s.h = m.Width, m.Height
		s.layout()

	case svcListMsg:
		s.loaded = true
		if m.err != nil {
			s.loadErr = friendlySvcErr(m.err)
			s.units = nil
			return s, nil
		}
		s.loadErr = ""
		s.units = m.units
		rows := make([]ui.Row, len(m.units))
		keys := make([]string, len(m.units))
		for i, u := range m.units {
			rows[i] = ui.Row{
				unitCell(u), stateStyled(u.Active), subStyled(u.Sub),
				bootStyled(u.Enabled),
				mutedSty.Render(ui.Truncate(u.Description, 40)),
			}
			keys[i] = u.Name
		}
		s.tbl.SetRowsTracked(rows, keys)
		// Kick the preview for whatever the cursor landed on.
		var cmd tea.Cmd
		if u := s.selectedName(); u != "" && u != s.detailUnit && !s.fetching {
			s.fetching = true
			cmd = fetchDetailCmd(u)
		}
		return s, cmd

	case spinner.TickMsg:
		if s.loaded {
			return s, nil // loading done; retire the chain
		}
		var cmd tea.Cmd
		s.spin, cmd = s.spin.Update(m)
		return s, cmd

	case svcShowMsg:
		if m.unit != s.selectedName() {
			return s, nil // stale fetch
		}
		if m.err != nil { // structured show failed: fall back to raw
			return s, rawStatusCmd(m.unit)
		}
		d := m.d
		s.det, s.detRaw, s.detailUnit = &d, "", m.unit
		s.fetching = false

	case svcRawMsg:
		s.fetching = false
		if m.unit != s.selectedName() {
			return s, nil // stale fetch
		}
		s.det, s.detRaw, s.detailUnit = nil, m.text, m.unit

	case svcJournalMsg:
		s.fetching = false
		if m.unit != s.selectedName() {
			return s, nil // stale fetch
		}
		s.logs = m.lines

	case tea.MouseMsg:
		if s.confirm != nil || s.tbl.Filtering() {
			return s, nil
		}
		moved, _ := s.tbl.Mouse(m, 3)
		if moved {
			return s, fetchDetailCmd(s.selectedName())
		}
		return s, nil

	case editorDoneMsg:
		return s, s.viewerFinished(m)

	case svcTickMsg:
		if m.gen != s.epoch.Load() {
			return s, nil // stale chain from a previous Init
		}
		var cmd tea.Cmd
		cmd = tea.Batch(refreshSvcCmd(), s.tick(m.gen))
		if u := s.selectedName(); u != "" && !s.fetching {
			cmd = tea.Batch(cmd, fetchDetailCmd(u))
		}
		return s, cmd

	case svcActionDoneMsg:
		s.confirm = nil
		if m.err != nil {
			return s, tea.Batch(ui.ErrToast(m.action+" failed: "+m.err.Error()),
				fetchDetailCmd(m.unit))
		}
		return s, tea.Batch(ui.OkToast(m.action+" "+m.unit),
			refreshSvcCmd(), fetchDetailCmd(m.unit))

	case tea.KeyPressMsg:
		return s.handleKey(m)
	}
	return s, nil
}

func stateStyled(st string) string {
	switch st {
	case "active":
		return lipgloss.NewStyle().Foreground(goodSty.GetForeground()).Render("* " + st)
	case "failed":
		return lipgloss.NewStyle().Bold(true).Foreground(badSty.GetForeground()).Render("x " + st)
	case "activating", "reloading":
		return lipgloss.NewStyle().Foreground(warnSty.GetForeground()).Render("~ " + st)
	default:
		return mutedSty.Render("o " + st)
	}
}

func bootStyled(v string) string {
	switch v {
	case "enabled":
		return goodSty.Render(v)
	case "disabled":
		return faintSty.Render(v)
	case "":
		return "-"
	default:
		return v
	}
}

// unitCell tints failed units red so they surface at a glance;
// otherwise the unit-type suffix (.service/.timer/...) dims so the
// base name carries the row.
func unitCell(u services.Unit) string {
	if u.Active == "failed" {
		return badSty.Render(u.Name)
	}
	i := strings.LastIndexByte(u.Name, '.')
	if i <= 0 {
		return u.Name
	}
	return u.Name[:i] + faintSty.Render(u.Name[i:])
}

// subStyled tones the fine-grained unit sub-state: running green,
// failed loud red, transitional states amber, quiet ones dim.
func subStyled(v string) string {
	switch v {
	case "running":
		return goodSty.Render(v)
	case "failed":
		return lipgloss.NewStyle().Bold(true).
			Foreground(badSty.GetForeground()).Render(v)
	case "activating", "reloading", "activating-done", "start", "stop",
		"reload", "mounting", "unmounting":
		return warnSty.Render(v)
	case "", "-":
		return "-"
	case "exited", "dead", "plugged", "mounted":
		return faintSty.Render(v)
	default:
		return mutedSty.Render(v)
	}
}

// editUnitFile suspends the TUI into $EDITOR on the unit's fragment
// file; a daemon-reload afterwards is left to the user (toast hints).
func (s Services) editUnitFile() tea.Cmd {
	if s.det == nil || s.det.FragmentPath == "" {
		return ui.ErrToast("no unit file to edit")
	}
	path := s.det.FragmentPath
	argv := resolveViewer()
	c := exec.Command(argv[0], append(argv[1:], path)...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorDoneMsg{err}
	})
}

// viewerFinished reacts after an external editor closes.
func (s Services) viewerFinished(m editorDoneMsg) tea.Cmd {
	cmd := fetchDetailCmd(s.selectedName())
	if m.err != nil {
		return tea.Batch(ui.ErrToast("editor: "+m.err.Error()), cmd)
	}
	return tea.Batch(ui.InfoToast("edited - run 'systemctl daemon-reload' if needed"), cmd)
}

func friendlySvcErr(err error) string {
	if err == services.ErrUnavailable {
		return "systemctl not found: service management unavailable here"
	}
	return err.Error()
}

func (s Services) selectedName() string {
	k, ok := s.tbl.SelectedKey()
	if !ok {
		return ""
	}
	return k
}

func (s Services) handleKey(m tea.KeyMsg) (ui.Screen, tea.Cmd) {
	if s.confirm != nil {
		dlg, yes, done := s.confirm.Update(m)
		*s.confirm = dlg
		if done && yes {
			s.confirm = nil
			unit, action := s.pendingOp[0], s.pendingOp[1]
			return s, func() tea.Msg {
				err := services.Action(unit, action)
				return svcActionDoneMsg{unit: unit, action: action, err: err}
			}
		}
		if done {
			s.confirm = nil
		}
		return s, nil
	}

	if s.tbl.Filtering() {
		var cmd tea.Cmd
		s.tbl, cmd = s.tbl.Update(m)
		return s, cmd
	}

	switch m.String() {
	case "r":
		return s.askAction("restart")
	case "R":
		return s, refreshSvcCmd()
	case "s":
		return s.askAction("start")
	case "t":
		return s.askAction("stop")
	case "e":
		return s.askAction("enable")
	case "E":
		return s, s.editUnitFile()
	case "D":
		return s.askAction("disable")
	}

	moved := false
	switch m.String() {
	case "up", "down", "j", "k", "g", "G", "home", "end", "pgup", "pgdown":
		moved = true
	}
	var cmd tea.Cmd
	s.tbl, cmd = s.tbl.Update(m)
	if moved {
		if u := s.selectedName(); u != "" && u != s.detailUnit {
			s.fetching = true
			cmd = tea.Batch(cmd, fetchDetailCmd(u))
		}
	}
	return s, cmd
}

func (s Services) askAction(action string) (ui.Screen, tea.Cmd) {
	u := s.selectedName()
	if u == "" {
		return s, nil
	}
	dangerous := action == "stop" || action == "disable" || action == "restart"
	verb := strings.ToUpper(action[:1]) + action[1:]
	dlg := ui.NewConfirm(verb+" service", verb+" "+u+"?", "")
	dlg.Danger = dangerous
	dlg.SetWidth(clampInt(s.w-8, 44, 70))
	s.confirm = &dlg
	s.pendingOp = [2]string{u, action}
	return s, nil
}

func (s *Services) layout() {
	wide, mainW, _ := splitGeom(s.w)
	tw := clampInt(s.w-2, 30, s.w)
	if wide {
		tw = mainW
	}
	s.tbl.SetSize(tw, clampInt(s.h-1, 5, s.h))
}

// View renders the list with the status preview beside it.
func (s Services) View() string {
	if s.w == 0 {
		return ""
	}
	head := pageHead("Services", fmt.Sprintf("%d units - auto-refresh %s",
		len(s.units), svcRefreshInterval), s.w)

	if !s.loaded {
		return lipgloss.Place(s.w, s.h, lipgloss.Center, lipgloss.Center,
			lipgloss.JoinVertical(lipgloss.Center,
				lipgloss.NewStyle().Foreground(ui.Accent("services")).Render(s.spin.View()),
				faintSty.Render("querying systemd..")))
	}
	if s.loadErr != "" {
		return lipgloss.Place(s.w, s.h, lipgloss.Center, lipgloss.Center,
			ui.EmptyState("", s.loadErr,
				"this environment may not use systemd", s.w))
	}

	main := s.tbl.View()
	prev := s.previewBody()
	wide, mainW, _ := splitGeom(s.w)
	if !wide {
		keep := clampInt(s.h-8, 6, s.h-4)
		mainLines := strings.Split(ui.ClipBlock(main, mainW), "\n")
		if len(mainLines) > keep {
			main = strings.Join(mainLines[:keep], "\n")
		}
	}
	body := joinPanesWide(wide, main, prev, mainW, s.w)

	out := head + "\n" + body
	lines := strings.Split(out, "\n")
	for len(lines) < s.h {
		lines = append(lines, "")
	}
	if len(lines) > s.h {
		lines = lines[:s.h]
	}
	view := strings.Join(lines, "\n")
	if s.confirm != nil {
		return overlayCenter(view, s.confirm.View(), s.w)
	}
	return view
}

// previewBody renders the right-hand pane for the selected unit.
// previewBody renders the right-hand pane: a structured facts card
// when `systemctl show` answered, the raw highlighted dump otherwise.
func (s Services) previewBody() string {
	_, _, prevW := splitGeom(s.w)
	name := s.selectedName()
	title, meta := "unit", ""
	if name != "" {
		for _, u := range s.units {
			if u.Name == name {
				meta = stateStyled(u.Active) + faintSty.Render(" - ") +
					bootStyled(u.Enabled) + restartsBadge(u, s.det)
				break
			}
		}
		title = unitTitle(name)
	}
	var body string
	switch {
	case name == "":
		// blank pane until a unit is selected
	case s.det == nil && s.detRaw == "":
		body = faintSty.Render("loading status..")
	case s.det != nil:
		body = detailCard(*s.det, prevW)
	default:
		body = lipgloss.NewStyle().Width(maxInt(prevW-2, 10)).
			Render(highlightSvcStatus(strings.TrimSpace(s.detRaw)))
	}
	if len(s.logs) > 0 && name != "" {
		var b strings.Builder
		b.WriteString(body)
		b.WriteString("\n\n" + mutedSty.Render("last logs") + "\n")
		for _, l := range s.logs {
			b.WriteString(faintSty.Render(ui.ClipBlock(l, maxInt(prevW-4, 8))) + "\n")
		}
		body = strings.TrimRight(b.String(), "\n")
	}
	return renderPreview("services", truncCell(title, maxInt(prevW-4, 6)), meta, body, prevW, s.h-1)
}

// unitTitle dims the unit-type suffix like the table rows do.
func unitTitle(name string) string {
	i := strings.LastIndexByte(name, '.')
	if i <= 0 {
		return truncCell(name, 24)
	}
	return name[:i] + faintSty.Render(name[i:])
}

// restartsBadge surfaces non-zero restart counts in the meta line.
func restartsBadge(u services.Unit, d *services.Detail) string {
	if u.Active == "failed" || d == nil || d.Restarts <= 0 {
		return ""
	}
	return faintSty.Render(" - ") + warnSty.Render("restarted "+itoa(d.Restarts))
}

// detailCard renders the structured facts of one unit.
func detailCard(d services.Detail, pw int) string {
	kv := func(k, v string) string {
		return mutedSty.Render(padTo(k, 9)) + ui.Truncate(v, maxInt(pw-11, 6))
	}
	stateVal := stateStyled(d.Active)
	if d.Sub != "" && d.Sub != "-" {
		stateVal += faintSty.Render(" (" + d.Sub + ")")
	}
	lines := []string{kv("state", stateVal), kv("boot", bootStyled(d.Boot))}
	if !d.Since.IsZero() {
		lines = append(lines,
			kv("since", relSince(time.Since(d.Since))),
			kv("at", faintSty.Render(d.Timestamp)))
	} else if d.Timestamp != "" {
		lines = append(lines, kv("at", faintSty.Render(d.Timestamp)))
	}
	pid := "-"
	if d.PID > 0 {
		pid = itoa(int(d.PID))
	}
	mem := "-"
	if d.MemBytes > 0 {
		mem = dimUnit(sysinfo.FormatBytes(float64(d.MemBytes)))
	}
	cpu := "-"
	if d.CPUNanos > 0 {
		cpu = fmt.Sprintf("%.1fs", float64(d.CPUNanos)/1e9)
	}
	lines = append(lines, kv("pid", pid), kv("memory", mem), kv("cpu", cpu))
	if d.Restarts > 0 {
		lines = append(lines, kv("restarts", warnSty.Render(itoa(d.Restarts))))
	}
	out := strings.Join(lines, "\n")
	if d.Active == "failed" { // banner first: the fact that matters
		banner := lipgloss.NewStyle().Bold(true).
			Background(badSty.GetForeground()).
			Foreground(ui.C("#11111B")).
			Render(" FAILED ")
		out = banner + "\n" + out
	}
	return out
}

// relSince humanizes an age the way systemd does ("3 days ago").
func relSince(d time.Duration) string {
	switch {
	case d < time.Minute:
		return itoa(int(d.Seconds())) + "s ago"
	case d < time.Hour:
		return itoa(int(d.Minutes())) + "m ago"
	case d < 24*time.Hour:
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		if m > 0 {
			return itoa(h) + "h " + itoa(m) + "m ago"
		}
		return itoa(h) + "h ago"
	default:
		days := int(d.Hours()) / 24
		h := int(d.Hours()) % 24
		if h > 0 {
			return itoa(days) + "d " + itoa(h) + "h ago"
		}
		return itoa(days) + "d ago"
	}
}

// svcTokens are the systemctl status phrases worth surfacing, wrapped
// with their semantic styles wherever they appear.
var svcTokens = []struct {
	phrase string
	sty    lipgloss.Style
}{
	{"active (running)", goodSty},
	{"activating", warnSty},
	{"reloading", warnSty},
	{"failed", lipgloss.NewStyle().Bold(true).Foreground(badSty.GetForeground())},
	{"enabled", goodSty},
	{"disabled", faintSty},
	{"inactive", faintSty},
	{"dead", faintSty},
}

// highlightSvcStatus wraps known status tokens with semantic styles;
// every other byte passes through unchanged, so line widths never move.
func highlightSvcStatus(text string) string {
	var b strings.Builder
	for text != "" {
		bestPos, bestLen, bestSty := -1, 0, lipgloss.Style{}
		for _, t := range svcTokens {
			if p := strings.Index(text, t.phrase); p >= 0 &&
				(bestPos == -1 || p < bestPos ||
					(p == bestPos && len(t.phrase) > bestLen)) {
				bestPos, bestLen, bestSty = p, len(t.phrase), t.sty
			}
		}
		if bestPos < 0 {
			b.WriteString(text)
			break
		}
		b.WriteString(text[:bestPos])
		b.WriteString(bestSty.Render(text[bestPos : bestPos+bestLen]))
		text = text[bestPos+bestLen:]
	}
	return b.String()
}
