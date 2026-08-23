package screens

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lcc2/internal/files"
	"lcc2/internal/sysinfo"
	"lcc2/internal/ui"
)

// Files is the file manager screen.
type Files struct {
	w, h       int
	cwd        string
	entries    []files.Entry
	tbl        ui.FilterTable
	showHidden bool

	meta      *files.Entry // metadata pane target
	prompt    *textinput.Model
	promptLbl string
	promptAct func(Files, string) tea.Cmd // executed on confirm

	permEdit   *files.PermBits
	permTarget string
	permRow    int // 0..2 selected class (u,g,o)
	permCol    int // 0..2 selected bit   (r,w,x)

	confirm   *ui.ConfirmDialog
	confirmFn func() tea.Cmd

	clipboardPath string
	clipboardMove bool
}

type dirListMsg struct {
	dir  string
	list []files.Entry
	err  error
}

// NewFiles builds the file manager rooted at $HOME.
func NewFiles() Files {
	cols := []table.Column{
		{Title: "name", Width: 34},
		{Title: "size", Width: 9},
		{Title: "mode", Width: 10},
		{Title: "owner", Width: 12},
		{Title: "modified", Width: 13},
	}
	return Files{cwd: files.Home(), tbl: ui.NewFilterTable(cols, 80, 20)}
}

// ID implements ui.Screen.
func (f Files) ID() string { return "files" }

// Title implements ui.Screen.
func (f Files) Title() string { return "Files" }

// Hints implements ui.Screen.
func (f Files) Hints() []key.Binding {
	return []key.Binding{
		ui.Keys.Filter,
		key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "hidden")),
		key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "mkdir")),
		key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "rename")),
		key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy")),
		key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "paste")),
		key.NewBinding(key.WithKeys("P"), key.WithHelp("P", "permissions")),
	}
}

// CapturingInput implements ui.Screen.
func (f Files) CapturingInput() bool {
	return f.tbl.Filtering() || f.prompt != nil || f.confirm != nil || f.permEdit != nil
}

// Init loads the starting directory.
func (f Files) Init() tea.Cmd { return listDir(f.cwd, f.showHidden) }

func listDir(dir string, hidden bool) tea.Cmd {
	return func() tea.Msg {
		l, err := files.List(dir, hidden)
		return dirListMsg{dir: dir, list: l, err: err}
	}
}

// Update handles messages.
func (f Files) Update(msg tea.Msg) (ui.Screen, tea.Cmd) {
	switch m := msg.(type) {
	case ui.SizeMsg:
		f.w, f.h = m.Width, m.Height
		f.layout()

	case dirListMsg:
		if m.err != nil {
			return f, ui.ErrToast(m.err.Error())
		}
		f.cwd = m.dir
		f.entries = m.list
		rows := make([]table.Row, len(m.list))
		for i, e := range m.list {
			name := e.Name
			if e.IsDir {
				name += "/"
			}
			rows[i] = table.Row{
				name,
				sizeOrDash(e),
				e.Mode.String(),
				files.UserName(e.UID),
				e.ModTime.Format("Jan 02 15:04"),
			}
		}
		f.tbl.SetRows(rows)

	case multiMsg:
		cmds := []tea.Cmd{}
		for _, t := range m.toasts {
			cmds = append(cmds, func(tt ui.ToastMsg) tea.Cmd { return func() tea.Msg { return tt } }(t))
		}
		if m.list != nil {
			sc, c := f.Update(*m.list)
			f = sc.(Files)
			cmds = append(cmds, c)
		}
		return f, tea.Batch(cmds...)

	case tea.KeyMsg:
		return f.handleKey(m)
	}
	return f, nil
}

func sizeOrDash(e files.Entry) string {
	if e.IsDir {
		return "-"
	}
	return sysinfo.FormatBytes(float64(e.Size))
}

func (f Files) selected() (*files.Entry, bool) {
	idx, ok := f.tbl.Selected()
	if !ok || idx >= len(f.entries) {
		return nil, false
	}
	return &f.entries[idx], true
}

func (f Files) handleKey(m tea.KeyMsg) (ui.Screen, tea.Cmd) {
	if f.confirm != nil {
		dlg, yes, done := f.confirm.Update(m)
		*f.confirm = dlg
		if done {
			fn := f.confirmFn
			f.confirm = nil
			f.confirmFn = nil
			if yes && fn != nil {
				return f, fn()
			}
		}
		return f, nil
	}

	if f.permEdit != nil {
		return f.handlePermKeys(m)
	}

	if f.prompt != nil {
		switch m.String() {
		case "enter":
			in := *f.prompt
			val := strings.TrimSpace(in.Value())
			act := f.promptAct
			f.prompt = nil
			f.promptAct = nil
			if val != "" && act != nil {
				return f, act(f, val)
			}
			return f, nil
		case "esc":
			f.prompt = nil
			f.promptAct = nil
			return f, nil
		}
		in := *f.prompt
		var cmd tea.Cmd
		in, cmd = in.Update(m)
		f.prompt = &in
		return f, cmd
	}

	if f.tbl.Filtering() {
		var cmd tea.Cmd
		f.tbl, cmd = f.tbl.Update(m)
		return f, cmd
	}

	switch m.String() {
	case "enter", "l":
		if e, ok := f.selected(); ok {
			if e.IsDir {
				return f, listDir(e.Path, f.showHidden)
			}
			f.meta = e
			f.layout()
		}
		return f, nil
	case "h":
		if !f.tbl.Filtering() {
			parent := parentDir(f.cwd)
			if parent != f.cwd {
				return f, listDir(parent, f.showHidden)
			}
		}
	case "a":
		f.showHidden = !f.showHidden
		return f, listDir(f.cwd, f.showHidden)
	case "esc":
		if f.meta != nil {
			f.meta = nil
			f.layout()
		}
	case "d":
		if e, ok := f.selected(); ok {
			dlg := ui.NewConfirm("Delete",
				fmt.Sprintf("Delete %s %q? This cannot be undone.",
					kindWord(e.IsDir), e.Name), e.Path)
			dlg.SetWidth(clampInt(f.w-8, 44, 70))
			f.confirm = &dlg
			target := e.Path
			dir := f.cwd
			f.confirmFn = func() tea.Cmd {
				return doThenRefresh(func() error { return files.Delete(target) },
					"deleted "+filepathBase(target), dir, f.showHidden)
			}
		}
		return f, nil
	case "m":
		return f.startPrompt("new directory: ", func(ff Files, name string) tea.Cmd {
			return doThenRefresh(func() error { return files.Mkdir(ff.cwd, name) },
				"created "+name, ff.cwd, ff.showHidden)
		})
	case "R":
		if e, ok := f.selected(); ok {
			cur := e.Name
			return f.startPrompt("rename: ", func(ff Files, name string) tea.Cmd {
				return doThenRefresh(func() error { return files.Rename(e.Path, name) },
					"renamed to "+name, ff.cwd, ff.showHidden)
			}, cur)
		}
	case "y":
		if e, ok := f.selected(); ok {
			f.clipboardPath = e.Path
			f.clipboardMove = false
			return f, ui.InfoToast("copied " + e.Name)
		}
	case "x":
		if e, ok := f.selected(); ok {
			f.clipboardPath = e.Path
			f.clipboardMove = true
			return f, ui.InfoToast("cut " + e.Name)
		}
	case "p":
		if f.clipboardPath != "" {
			cp := f.clipboardPath
			mv := f.clipboardMove
			dst := f.cwd
			return f, doThenRefresh(func() error {
				if mv {
					return files.Move(cp, dst)
				}
				return files.Copy(cp, dst)
			}, "pasted "+filepathBase(cp), dst, f.showHidden)
		}
		return f, ui.InfoToast("clipboard empty")
	case "P":
		if e, ok := f.selected(); ok {
			bits := files.ParsePermBits(e.Mode)
			f.permEdit = &bits
			f.permTarget = e.Path
			f.permRow = 0
		}
		return f, nil
	}

	var cmd tea.Cmd
	f.tbl, cmd = f.tbl.Update(m)
	return f, cmd
}

func kindWord(isDir bool) string {
	if isDir {
		return "directory"
	}
	return "file"
}

func parentDir(p string) string {
	i := strings.LastIndexByte(p, '/')
	if i <= 0 {
		return "/"
	}
	return p[:i]
}

func filepathBase(p string) string {
	i := strings.LastIndexByte(p, '/')
	if i < 0 {
		return p
	}
	return p[i+1:]
}

func doThenRefresh(fn func() error, okMsg, dir string, hidden bool) tea.Cmd {
	return func() tea.Msg {
		if err := fn(); err != nil {
			return ui.ToastMsg{Kind: "err", Text: err.Error()}
		}
		// refresh listing as part of the same message flow
		l, lerr := files.List(dir, hidden)
		if lerr == nil {
			return multiMsg{toasts: []ui.ToastMsg{{Kind: "ok", Text: okMsg}}, list: &dirListMsg{dir: dir, list: l}}
		}
		return ui.ToastMsg{Kind: "ok", Text: okMsg}
	}
}

// multiMsg carries both a toast and a fresh directory listing.
type multiMsg struct {
	toasts []ui.ToastMsg
	list   *dirListMsg
}

func (f Files) startPrompt(label string, act func(Files, string) tea.Cmd, prefill ...string) (ui.Screen, tea.Cmd) {
	ti := textinput.New()
	ti.Focus()
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ui.Accent("files"))
	if len(prefill) > 0 {
		ti.SetValue(prefill[0])
	}
	f.prompt = &ti
	f.promptLbl = label
	f.promptAct = act
	return f, textinput.Blink
}

func (f Files) handlePermKeys(m tea.KeyMsg) (ui.Screen, tea.Cmd) {
	switch m.String() {
	case "esc":
		f.permEdit = nil
		return f, nil
	case "enter":
		bits := *f.permEdit
		target := f.permTarget
		f.permEdit = nil
		mode, _ := parseOctal(bits.Octal())
		return f, doThenRefresh(func() error { return files.Chmod(target, mode) },
			"chmod "+bits.Octal()+" "+filepathBase(target), f.cwd, f.showHidden)
	case "h", "left":
		f.permCol = (f.permCol + 2) % 3
	case "l", "right", "tab":
		f.permCol = (f.permCol + 1) % 3
	case "j", "down":
		f.permRow = (f.permRow + 1) % 3
	case "k", "up":
		f.permRow = (f.permRow + 2) % 3
	case " ", "s":
		f.permEdit.Toggle(f.permRow, f.permCol)
	}
	return f, nil
}

func parseOctal(s string) (os.FileMode, error) {
	var v uint32
	for _, c := range s {
		if c < '0' || c > '7' {
			return 0, fmt.Errorf("bad octal %q", s)
		}
		v = v*8 + uint32(c-'0')
	}
	return os.FileMode(v), nil
}

func (f *Files) layout() {
	th := clampInt(f.h-3, 5, f.h)
	tw := clampInt(f.w-2, 40, f.w)
	if f.meta != nil {
		tw = clampInt(tw*2/3, 40, tw)
	}
	f.tbl.SetSize(tw, th)
}

// View renders the manager, optional metadata pane, modals and editor.
func (f Files) View() string {
	if f.w == 0 {
		return ""
	}
	head := renderCrumbs(f.cwd, f.w) +
		faintSty.Render(fmt.Sprintf("%d items · hidden %s",
			len(f.entries), onOff(f.showHidden)))
	body := head + "\n" + f.tbl.View()

	if f.meta != nil {
		paneW := clampInt(f.w/3, 30, 48)
		body = joinPanes(body, metaPane(*f.meta, paneW))
	}
	if f.permEdit != nil {
		return lipgloss.Place(f.w, f.h, lipgloss.Center, lipgloss.Center,
			f.permEditorView(), lipgloss.WithWhitespaceForeground(ui.Palette.Faint))
	}
	if f.prompt != nil {
		box := ui.Panel().BorderForeground(ui.Accent("files")).Width(clampInt(f.w/2, 40, 60)).
			Padding(0, 1).Render(f.promptLbl + f.prompt.View())
		return lipgloss.Place(f.w, f.h, lipgloss.Center, lipgloss.Center, box)
	}
	if f.confirm != nil {
		return lipgloss.Place(f.w, f.h, lipgloss.Center, lipgloss.Center, f.confirm.View())
	}
	return body
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

func metaPane(e files.Entry, w int) string {
	var b strings.Builder
	title := lipgloss.NewStyle().Bold(true).Foreground(ui.Accent("files")).
		Render(ui.Truncate(e.Name, w-4))
	b.WriteString(title + "\n\n")
	kv := func(k, v string) {
		b.WriteString(mutedSty.Render(padTo(k, 9)) + faintSty.Render(ui.Truncate(v, w-12)) + "\n")
	}
	kv("kind", kindWord(e.IsDir))
	kv("size", sysinfo.FormatBytes(float64(e.Size)))
	kv("owner", files.UserName(e.UID))
	kv("group", files.GroupName(e.GID))
	bits := files.ParsePermBits(e.Mode)
	kv("perms", bits.Symbolic()+" ("+bits.Octal()+")")
	kv("path", e.Path)
	kv("modified", e.ModTime.Format(time.RFC3339))
	b.WriteString("\n" + faintSty.Render("esc close · P edit permissions"))
	return ui.Panel().BorderForeground(ui.Accent("files")).Width(w).
		Height(clampInt(14, 6, 24)).Padding(0, 1).Render(b.String())
}

// permEditorView draws the interactive rwx matrix with a live octal readout.
func (f Files) permEditorView() string {
	bits := *f.permEdit
	target := filepathBase(f.permTarget)

	header := lipgloss.NewStyle().Bold(true).Foreground(ui.Palette.Text).
		Render("        r      w      x")
	grid := header + "\n"
	labels := [3]string{"user", "group", "other"}
	cells := [3][3]bool{bits.U, bits.G, bits.O}
	for who := 0; who < 3; who++ {
		rowLabel := mutedSty.Render(padTo(labels[who], 7))
		if who == f.permRow {
			rowLabel = lipgloss.NewStyle().Bold(true).
				Foreground(ui.Accent("files")).Render("> " + padTo(labels[who], 5))
		}
		grid += rowLabel
		for which := 0; which < 3; which++ {
			cell := "[ ]"
			if cells[who][which] {
				cell = "[x]"
			}
			sty := mutedSty
			if who == f.permRow && which == f.permCol {
				sty = warnSty
			} else if cells[who][which] {
				sty = goodSty.Bold(true)
			}
			grid += "  " + sty.Render(cell)
		}
		grid += "\n"
	}
	help := faintSty.Render("h/j/k/l move · space toggle · enter apply · esc cancel")

	body := lipgloss.NewStyle().Bold(true).Foreground(ui.Accent("files")).
		Render("permissions — "+target) + "\n\n" +
		grid + "\n" +
		mutedSty.Render(bits.Symbolic()) + "  " +
		lipgloss.NewStyle().Bold(true).Foreground(ui.Palette.Yellow).
			Render(bits.Octal()) + "\n\n" + help

	return ui.Panel().BorderForeground(ui.Accent("files")).
		Width(clampInt(f.w/2, 40, 56)).Padding(1, 2).Render(body)
}
