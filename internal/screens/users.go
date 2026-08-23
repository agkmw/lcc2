package screens

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lcc2/internal/accounts"
	"lcc2/internal/ui"
)

type accountsMsg struct {
	users  []accounts.User
	groups []accounts.Group
	err    error
}

// UsersGroups is the combined users & groups section. The `tab` key
// switches between the two lists; enter opens a detail pane.
type UsersGroups struct {
	w, h   int
	users  []accounts.User
	groups []accounts.Group
	uTbl   ui.FilterTable
	gTbl   ui.FilterTable
	tab    string // "users" or "groups"

	detailOpen bool
	detail     string // rendered detail body
	detailName string

	loaded bool
	err    string
}

// NewUsersGroups builds the section.
func NewUsersGroups() UsersGroups {
	return UsersGroups{
		tab:  "users",
		uTbl: ui.NewFilterTable(userCols(), 80, 18),
		gTbl: ui.NewFilterTable(groupCols(), 80, 18),
	}
}

func userCols() []table.Column {
	return []table.Column{
		{Title: "user", Width: 18},
		{Title: "uid", Width: 8},
		{Title: "gid", Width: 8},
		{Title: "home", Width: 24},
		{Title: "shell", Width: 22},
	}
}

func groupCols() []table.Column {
	return []table.Column{
		{Title: "group", Width: 20},
		{Title: "gid", Width: 8},
		{Title: "members", Width: 52},
	}
}

// ID implements ui.Screen.
func (u UsersGroups) ID() string { return "users" }

// Title implements ui.Screen.
func (u UsersGroups) Title() string { return "Users & Groups" }

// Hints implements ui.Screen.
func (u UsersGroups) Hints() []key.Binding {
	hints := []key.Binding{
		ui.Keys.Filter,
		key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch to "+otherTab(u.tab))),
		ui.Keys.Select,
	}
	return hints
}

func otherTab(cur string) string {
	if cur == "users" {
		return "groups"
	}
	return "users"
}

// CapturingInput implements ui.Screen.
func (u UsersGroups) CapturingInput() bool {
	return u.uTbl.Filtering() || u.gTbl.Filtering()
}

// Init loads passwd/group.
func (u UsersGroups) Init() tea.Cmd {
	return func() tea.Msg {
		us, uerr := accounts.Users()
		gs, gerr := accounts.Groups()
		if uerr != nil {
			return accountsMsg{err: uerr}
		}
		if gerr != nil {
			return accountsMsg{err: gerr}
		}
		return accountsMsg{users: us, groups: gs}
	}
}

// Update handles messages.
func (u UsersGroups) Update(msg tea.Msg) (ui.Screen, tea.Cmd) {
	switch m := msg.(type) {
	case ui.SizeMsg:
		u.w, u.h = m.Width, m.Height
		u.layout()

	case accountsMsg:
		if m.err != nil {
			u.err = m.err.Error()
			break
		}
		u.loaded = true
		u.users = m.users
		u.groups = m.groups
		urows := make([]table.Row, len(m.users))
		ukeys := make([]string, len(m.users))
		for i, usr := range m.users {
			urows[i] = table.Row{
				usr.Name, itoa(usr.UID), itoa(usr.GID),
				ui.Truncate(usr.Home, 24), usr.Shell,
			}
			ukeys[i] = usr.Name
		}
		u.uTbl.SetRowsTracked(urows, ukeys)
		grows := make([]table.Row, len(m.groups))
		gkeys := make([]string, len(m.groups))
		for i, g := range m.groups {
			members := strings.Join(g.Members, ", ")
			if members == "" {
				members = "-"
			}
			grows[i] = table.Row{g.Name, itoa(g.GID), ui.Truncate(members, 52)}
			gkeys[i] = g.Name
		}
		u.gTbl.SetRowsTracked(grows, gkeys)

	case tea.KeyMsg:
		return u.handleKey(m)
	}
	return u, nil
}

func (u UsersGroups) handleKey(m tea.KeyMsg) (ui.Screen, tea.Cmd) {
	if u.tab == "users" && u.uTbl.Filtering() {
		var cmd tea.Cmd
		u.uTbl, cmd = u.uTbl.Update(m)
		return u, cmd
	}
	if u.tab == "groups" && u.gTbl.Filtering() {
		var cmd tea.Cmd
		u.gTbl, cmd = u.gTbl.Update(m)
		return u, cmd
	}

	switch m.String() {
	case "tab":
		u.tab = otherTab(u.tab)
		u.detailOpen = false
		u.layout()
		return u, nil
	case "enter":
		if u.tab == "users" {
			if idx, ok := u.uTbl.Selected(); ok && idx < len(u.users) {
				sel := u.users[idx]
				u.detailName = sel.Name
				u.detail = u.renderUser(sel)
				u.detailOpen = true
				u.layout()
			}
		} else {
			if idx, ok := u.gTbl.Selected(); ok && idx < len(u.groups) {
				sel := u.groups[idx]
				u.detailName = sel.Name
				u.detail = u.renderGroup(sel)
				u.detailOpen = true
				u.layout()
			}
		}
		return u, nil
	case "esc":
		if u.detailOpen {
			u.detailOpen = false
			u.layout()
			return u, nil
		}
	}

	if u.tab == "users" {
		var cmd tea.Cmd
		u.uTbl, cmd = u.uTbl.Update(m)
		return u, cmd
	}
	var cmd tea.Cmd
	u.gTbl, cmd = u.gTbl.Update(m)
	return u, cmd
}

func (u *UsersGroups) layout() {
	th := clampInt(u.h-5, 4, u.h)
	w := u.w - 2
	if u.detailOpen && u.w >= 100 {
		w = u.w - clampInt(u.w*2/5, 32, 52) - 3
	}
	u.uTbl.SetSize(clampInt(w, 30, u.w), th)
	u.gTbl.SetSize(clampInt(w, 30, u.w), th)
}

func (u UsersGroups) renderUser(usr accounts.User) string {
	allGroups, _ := accounts.Groups()
	memberships := accounts.GroupsOf(usr.Name, usr.GID, allGroups)
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).
		Foreground(ui.Accent("users")).Render(usr.Name) + "\n\n")
	kv := func(k, v string) {
		b.WriteString(mutedSty.Render(padTo(k, 9)) + faintSty.Render(v) + "\n")
	}
	kv("uid", itoa(usr.UID))
	kv("gid", itoa(usr.GID))
	kv("home", usr.Home)
	kv("shell", usr.Shell)
	if usr.Shell != "/usr/sbin/nologin" && usr.Shell != "/bin/false" &&
		!strings.Contains(usr.Shell, "nologin") {
		kv("login", "allowed")
	} else {
		kv("login", "no shell")
	}
	if len(memberships) > 0 {
		kv("groups", strings.Join(memberships, ", "))
	}
	return b.String()
}

func (u UsersGroups) renderGroup(g accounts.Group) string {
	members := accounts.MembersOf(g, u.users)
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).
		Foreground(ui.Accent("users")).Render(g.Name) + "\n\n")
	kv := func(k, v string) {
		b.WriteString(mutedSty.Render(padTo(k, 9)) + faintSty.Render(v) + "\n")
	}
	kv("gid", itoa(g.GID))
	if len(members) == 0 {
		kv("members", "-")
	} else {
		b.WriteString("\n" + mutedSty.Render("members") + "\n")
		for _, mem := range members {
			b.WriteString(faintSty.Render("  • "+mem) + "\n")
		}
	}
	return b.String()
}

// View renders the active list plus optional detail pane.
func (u UsersGroups) View() string {
	if u.w == 0 {
		return ""
	}
	if !u.loaded {
		if u.err != "" {
			return lipgloss.Place(u.w, u.h, lipgloss.Center, lipgloss.Center,
				ui.EmptyState("", "cannot read account database", u.err, u.w))
		}
		return lipgloss.Place(u.w, u.h, lipgloss.Center, lipgloss.Center,
			ui.EmptyState("", "reading /etc/passwd…", "", u.w))
	}

	tblView := u.uTbl.View()
	count := len(u.users)
	if u.tab == "groups" {
		tblView = u.gTbl.View()
		count = len(u.groups)
	}
	tabLabel := "Users"
	if u.tab == "groups" {
		tabLabel = "Groups"
	}
	head := pageHead(tabLabel,
		fmt.Sprintf("%s · tab switches · %d entries", otherTab(u.tab), count), u.w)
	body := head + "\n\n" + tblView

	if u.detailOpen {
		wide := u.w >= 100
		inner := clampInt(u.w*2/5, 32, 52)
		if !wide {
			inner = u.w - 2
		}
		tbl := u.uTbl.Width()
		if u.tab == "groups" {
			tbl = u.gTbl.Width()
		}
		title := u.detailName
		if title == "" {
			title = u.tab
		}
		pane := ui.TitledBox("users", truncCell(title, maxInt(inner-6, 8)),
			ui.ClipBlock(u.detail, maxInt(inner-4, 10)), inner)
		if wide {
			body = ui.Split(body, pane, tbl, u.w)
		} else {
			body += "\n" + pane
		}
	}
	return body
}
