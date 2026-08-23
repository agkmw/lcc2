package screens

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lcc2/internal/services"
	"lcc2/internal/ui"
)

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

// Services is the service management screen.
type Services struct {
	w, h    int
	spin    spinner.Model
	units   []services.Unit
	tbl     ui.FilterTable
	loaded  bool
	loadErr string

	detailUnit string
	detailText string
	detailOpen bool

	confirm   *ui.ConfirmDialog
	pendingOp [2]string // unit, action
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
		tbl:  ui.NewFilterTable(cols, 80, 18),
		spin: spinner.New(spinner.WithSpinner(spinner.Dot)),
	}
}

// ID implements ui.Screen.
func (s Services) ID() string { return "services" }

// Title implements ui.Screen.
func (s Services) Title() string { return "Services" }

// Hints implements ui.Screen.
func (s Services) Hints() []key.Binding {
	hints := []key.Binding{ui.Keys.Filter, ui.Keys.Select}
	if !s.detailOpen {
		hints = append(hints,
			key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "start")),
			key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "stop")),
			key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "restart")),
			key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "enable")),
			key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "disable")),
		)
	}
	return hints
}

// CapturingInput implements ui.Screen.
func (s Services) CapturingInput() bool {
	return s.tbl.Filtering() || s.confirm != nil || s.detailOpen
}

// Init loads the unit list and runs the loading spinner.
func (s Services) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg { u, err := services.List(); return svcListMsg{units: u, err: err} },
		s.spin.Tick,
	)
}

func refreshSvcCmd() tea.Cmd {
	return func() tea.Msg { u, err := services.List(); return svcListMsg{units: u, err: err} }
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
			break
		}
		s.loadErr = ""
		s.units = m.units
		rows := make([]table.Row, len(m.units))
		keys := make([]string, len(m.units))
		for i, u := range m.units {
			rows[i] = table.Row{
				u.Name, stateStyled(u.Active), u.Sub,
				orDash(u.Enabled),
				ui.Truncate(u.Description, 40),
			}
			keys[i] = u.Name
		}
		s.tbl.SetRowsTracked(rows, keys)

	case spinner.TickMsg:
		if s.loaded {
			return s, nil // loading done; retire the chain
		}
		var cmd tea.Cmd
		s.spin, cmd = s.spin.Update(m)
		return s, cmd

	case svcDetailMsg:
		s.detailText = m.text
		return s, nil

	case svcActionDoneMsg:
		s.confirm = nil
		if m.err != nil {
			return s, ui.ErrToast(m.action + " failed: " + m.err.Error())
		}
		return s, tea.Batch(ui.OkToast(m.action+" "+m.unit), refreshSvcCmd())

	case tea.KeyMsg:
		return s.handleKey(m)
	}
	return s, nil
}

func detailWidthIf(open bool) int {
	if open {
		return clampInt(48, 34, 52)
	}
	return 0
}

func orDash(v string) string {
	if v == "" {
		return "-"
	}
	return v
}

func stateStyled(st string) string { return st }

func friendlySvcErr(err error) string {
	if err == services.ErrUnavailable {
		return "systemctl not found — service management unavailable here"
	}
	return err.Error()
}

func (s Services) selectedUnit() (*services.Unit, bool) {
	idx, ok := s.tbl.Selected()
	if !ok || idx >= len(s.units) {
		return nil, false
	}
	return &s.units[idx], true
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

	if s.detailOpen {
		switch m.String() {
		case "esc", "enter":
			s.detailOpen = false
			s.detailUnit = ""
			s.layout()
			return s, nil
		}
		return s, nil
	}

	if s.tbl.Filtering() {
		var cmd tea.Cmd
		s.tbl, cmd = s.tbl.Update(m)
		return s, cmd
	}

	switch m.String() {
	case "enter":
		if u, ok := s.selectedUnit(); ok {
			s.detailOpen = true
			s.detailUnit = u.Name
			s.detailText = "loading status…"
			s.layout()
			name := u.Name
			return s, func() tea.Msg {
				return svcDetailMsg{unit: name, text: services.StatusDetail(name)}
			}
		}
		return s, nil
	case "s":
		return s.askAction("start")
	case "t":
		return s.askAction("stop")
	case "r":
		return s.askAction("restart")
	case "e":
		return s.askAction("enable")
	case "D":
		return s.askAction("disable")
	}

	var cmd tea.Cmd
	s.tbl, cmd = s.tbl.Update(m)
	return s, cmd
}

func (s Services) askAction(action string) (ui.Screen, tea.Cmd) {
	u, ok := s.selectedUnit()
	if !ok {
		return s, nil
	}
	dangerous := action == "stop" || action == "disable" || action == "restart"
	body := strings.ToUpper(action[:1]) + action[1:] + " " + u.Name + "?"
	if dangerous && u.Active == "active" && action != "restart" {
		body += " The service is currently running."
	}
	dlg := ui.NewConfirm(strings.ToUpper(action[:1])+action[1:]+" service", body, "")
	dlg.SetWidth(clampInt(s.w-8, 44, 70))
	if !dangerous {
		dlg.Title = action
	}
	s.confirm = &dlg
	s.pendingOp = [2]string{u.Name, action}
	return s, nil
}

func (s *Services) layout() {
	dh := clampInt(s.h-4, 5, s.h)
	s.tbl.SetSize(clampInt(s.w-detailWidthIf(s.detailOpen)-3, 30, s.w), dh)
}

// View renders the list, empty state or details pane.
func (s Services) View() string {
	if s.w == 0 {
		return ""
	}
	head := faintSty.Render(itoa(len(s.units)) + " units")
	body := head + "\n" + s.tbl.View()

	if !s.loaded {
		return lipgloss.Place(s.w, s.h, lipgloss.Center, lipgloss.Center,
			lipgloss.JoinVertical(lipgloss.Center,
				lipgloss.NewStyle().Foreground(ui.Accent("services")).Render(s.spin.View()),
				faintSty.Render("querying systemd…")))
	}
	if s.loadErr != "" {
		return lipgloss.Place(s.w, s.h, lipgloss.Center, lipgloss.Center,
			ui.EmptyState("", s.loadErr,
				"this environment may not use systemd", s.w))
	}

	if s.detailOpen {
		paneW := detailWidthIf(true)
		content := lipgloss.NewStyle().Bold(true).Foreground(ui.Accent("services")).
			Render(s.detailUnit) + "\n\n" +
			lipgloss.NewStyle().Width(paneW-4).Render(strings.TrimSpace(s.detailText))
		pane := ui.Panel().BorderForeground(ui.Accent("services")).
			Width(paneW).Height(paneHeight(lipgloss.Height(content), s.h)).
			Padding(0, 1).Render(content)
		body = joinPanes(body, pane)
	}

	if s.confirm != nil {
		return lipgloss.Place(s.w, s.h, lipgloss.Center, lipgloss.Center, s.confirm.View())
	}
	return body
}
