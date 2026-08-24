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

// UsersGroups is the combined users & groups section: list left, a
// detail preview always visible right. The `tab` key switches lists.
// System accounts (uid/gid < 1000, except root) render dimmed.
type UsersGroups struct {
	w, h   int
	users  []accounts.User
	groups []accounts.Group
	uTbl   ui.FilterTable
	gTbl   ui.FilterTable
	tab    string // "users" or "groups"

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
		{Title: "uid", Width: 7},
		{Title: "gid", Width: 7},
		{Title: "home", Width: 22},
		{Title: "shell", Width: 20},
	}
}

func groupCols() []table.Column {
	return []table.Column{
		{Title: "group", Width: 18},
		{Title: "gid", Width: 7},
		{Title: "members", Width: 30},
	}
}

// ID implements ui.Screen.
func (u UsersGroups) ID() string { return "users" }

// Title implements ui.Screen.
func (u UsersGroups) Title() string { return "Users & Groups" }

// Hints implements ui.Screen.
func (u UsersGroups) Hints() []key.Binding {
	return []key.Binding{
		ui.Keys.Filter,
		key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "switch to "+otherTab(u.tab))),
		ui.Keys.Refresh,
	}
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
				userCell(usr), itoa(usr.UID), itoa(usr.GID),
				ui.Truncate(usr.Home, 22), usr.Shell,
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
			grows[i] = table.Row{g.Name, itoa(g.GID), ui.Truncate(members, 30)}
			gkeys[i] = g.Name
		}
		u.gTbl.SetRowsTracked(grows, gkeys)

	case tea.KeyMsg:
		return u.handleKey(m)
	}
	return u, nil
}

// isSystemAccount reports non-human accounts; root stays highlighted.
func isSystemAccount(uid int) bool { return uid != 0 && uid < 1000 }

func userCell(usr accounts.User) string {
	if isSystemAccount(usr.UID) {
		return mutedSty.Render(usr.Name)
	}
	return lipgloss.NewStyle().Foreground(ui.Palette.Text).Render(usr.Name)
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
	case "s":
		u.tab = otherTab(u.tab)
		return u, nil
	case "r":
		return u, u.Init()
	}

	moved := false
	switch m.String() {
	case "up", "down", "j", "k", "g", "G", "home", "end", "pgup", "pgdown":
		moved = true
	}
	if u.tab == "users" {
		var cmd tea.Cmd
		u.uTbl, cmd = u.uTbl.Update(m)
		return u, cmd
	}
	var cmd tea.Cmd
	u.gTbl, cmd = u.gTbl.Update(m)
	_ = moved // preview renders synchronously from loaded slices
	return u, cmd
}

func (u *UsersGroups) layout() {
	wide, mainW, _ := splitGeom(u.w)
	tw := clampInt(u.w-2, 30, u.w)
	if wide {
		tw = mainW
	}
	th := clampInt(u.h-1, 5, u.h)
	u.uTbl.SetSize(tw, th)
	u.gTbl.SetSize(tw, th)
}

// View renders the active list with the detail preview beside it.
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
			ui.EmptyState("", "reading /etc/passwd..", "", u.w))
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
		fmt.Sprintf("s switches lists - %d entries - system accounts dimmed", count), u.w)

	wide, mainW, prevW := splitGeom(u.w)
	prev := renderPreview("users", u.previewTitle(), "", u.previewBody(), prevW, u.h-1)
	if !wide {
		keep := clampInt(u.h-8, 6, u.h-4)
		lines := strings.Split(ui.ClipBlock(tblView, mainW), "\n")
		if len(lines) > keep {
			tblView = strings.Join(lines[:keep], "\n")
		}
	}
	body := joinPanesWide(wide, tblView, prev, mainW, u.w)

	out := head + "\n" + body
	lines := strings.Split(out, "\n")
	for len(lines) < u.h {
		lines = append(lines, "")
	}
	if len(lines) > u.h {
		lines = lines[:u.h]
	}
	return strings.Join(lines, "\n")
}

func (u UsersGroups) previewTitle() string {
	if u.tab == "users" {
		if idx, ok := u.uTbl.Selected(); ok && idx < len(u.users) {
			return truncCell(u.users[idx].Name, 24)
		}
		return "user"
	}
	if idx, ok := u.gTbl.Selected(); ok && idx < len(u.groups) {
		return truncCell(u.groups[idx].Name, 24)
	}
	return "group"
}

// previewBody renders the selected user/group card from the already
// loaded slices — no blocking reads on the UI path.
func (u UsersGroups) previewBody() string {
	_, _, pw := splitGeom(u.w)
	kv := func(k, v string) string {
		return mutedSty.Render(padTo(k, 8)) + faintSty.Render(ui.Truncate(v, maxInt(pw-10, 4)))
	}
	if u.tab == "users" {
		idx, ok := u.uTbl.Selected()
		if !ok || idx >= len(u.users) {
			return ""
		}
		usr := u.users[idx]
		memberships := accounts.GroupsOf(usr.Name, usr.GID, u.groups)
		lines := []string{
			kv("uid", itoa(usr.UID)),
			kv("gid", itoa(usr.GID)),
			kv("home", usr.Home),
			kv("shell", usr.Shell),
		}
		if isSystemAccount(usr.UID) {
			lines = append(lines, kv("class", "system account"))
		} else if usr.Shell != "" && !strings.Contains(usr.Shell, "nologin") &&
			usr.Shell != "/bin/false" {
			lines = append(lines, kv("class", "human - login allowed"))
		} else {
			lines = append(lines, kv("class", "no login shell"))
		}
		if len(memberships) > 0 {
			lines = append(lines, "", mutedSty.Render("groups"),
				faintSty.Render(ui.Truncate(strings.Join(memberships, ", "), pw-2)))
		}
		return strings.Join(lines, "\n")
	}

	idx, ok := u.gTbl.Selected()
	if !ok || idx >= len(u.groups) {
		return ""
	}
	g := u.groups[idx]
	members := accounts.MembersOf(g, u.users)
	lines := []string{kv("gid", itoa(g.GID))}
	if len(members) == 0 {
		lines = append(lines, kv("members", "-"))
		return strings.Join(lines, "\n")
	}
	lines = append(lines, "", mutedSty.Render("members"))
	for _, mem := range members {
		lines = append(lines, faintSty.Render("  * "+mem))
	}
	return strings.Join(lines, "\n")
}
