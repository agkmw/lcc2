package screens

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lcc2/internal/services"
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
type svcDetailMsg struct {
	unit string
	text string
}
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

	detailUnit string // unit whose status is displayed
	detailText string
	fetching   bool

	confirm   *ui.ConfirmDialog
	pendingOp [2]string // unit, action

	epoch *atomic.Uint64 // auto-refresh chain generation
}

// NewServices builds the services screen.
func NewServices() Services {
	cols := []table.Column{
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
		s.spin.Tick,
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

func statusCmd(unit string) tea.Cmd {
	return func() tea.Msg {
		// systemctl leads lines with ambiguous-width glyphs; narrow
		// them so tmux cannot shift the pane columns.
		return svcDetailMsg{unit: unit, text: ui.Narrow(services.StatusDetail(unit))}
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
		rows := make([]table.Row, len(m.units))
		keys := make([]string, len(m.units))
		for i, u := range m.units {
			rows[i] = table.Row{
				u.Name, stateStyled(u.Active), u.Sub,
				bootStyled(u.Enabled),
				ui.Truncate(u.Description, 40),
			}
			keys[i] = u.Name
		}
		s.tbl.SetRowsTracked(rows, keys)
		// Kick the preview for whatever the cursor landed on.
		var cmd tea.Cmd
		if u := s.selectedName(); u != "" && u != s.detailUnit && !s.fetching {
			s.fetching = true
			cmd = statusCmd(u)
		}
		return s, cmd

	case spinner.TickMsg:
		if s.loaded {
			return s, nil // loading done; retire the chain
		}
		var cmd tea.Cmd
		s.spin, cmd = s.spin.Update(m)
		return s, cmd

	case svcDetailMsg:
		s.fetching = false
		if m.unit == s.selectedName() { // ignore stale fetches
			s.detailUnit, s.detailText = m.unit, m.text
		}

	case svcTickMsg:
		if m.gen != s.epoch.Load() {
			return s, nil // stale chain from a previous Init
		}
		var cmd tea.Cmd
		cmd = tea.Batch(refreshSvcCmd(), s.tick(m.gen))
		if u := s.selectedName(); u != "" && !s.fetching {
			cmd = tea.Batch(cmd, statusCmd(u))
		}
		return s, cmd

	case svcActionDoneMsg:
		s.confirm = nil
		if m.err != nil {
			return s, tea.Batch(ui.ErrToast(m.action+" failed: "+m.err.Error()),
				statusCmd(m.unit))
		}
		return s, tea.Batch(ui.OkToast(m.action+" "+m.unit),
			refreshSvcCmd(), statusCmd(m.unit))

	case tea.KeyMsg:
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
			cmd = tea.Batch(cmd, statusCmd(u))
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
	body := strings.ToUpper(action[:1]) + action[1:] + " " + u + "?"
	dlg := ui.NewConfirm(strings.ToUpper(action[:1])+action[1:]+" service", body, "")
	dlg.SetWidth(clampInt(s.w-8, 44, 70))
	if !dangerous {
		dlg.Title = action
	}
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
	if s.confirm != nil {
		return lipgloss.Place(s.w, s.h, lipgloss.Center, lipgloss.Center, s.confirm.View())
	}
	return strings.Join(lines, "\n")
}

// previewBody renders the right-hand pane for the selected unit.
func (s Services) previewBody() string {
	_, _, prevW := splitGeom(s.w)
	name := s.selectedName()
	title, meta := "unit", ""
	if name != "" {
		for _, u := range s.units {
			if u.Name == name {
				meta = u.Active + " - " + u.Enabled
				break
			}
		}
		title = name
	} else {
		title = "unit"
	}
	var body string
	switch {
	case name == "":
		body = faintSty.Render("select a unit..")
	case s.detailText == "" || (s.fetching && name != s.detailUnit):
		body = faintSty.Render("loading status..")
	default:
		body = lipgloss.NewStyle().Width(maxInt(prevW-2, 10)).
			Render(strings.TrimSpace(s.detailText))
	}
	return renderPreview("services", truncCell(title, maxInt(prevW-4, 6)), meta, body, prevW, s.h-1)
}
