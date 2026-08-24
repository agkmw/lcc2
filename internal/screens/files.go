package screens

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
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

// Files is the file manager screen: listing left, preview right, and
// the oil.nvim model — every mutation is staged and only applied on
// save. Nothing touches the disk until `w`.
type Files struct {
	w, h       int
	cwd        string
	entries    []files.Entry
	tbl        ui.FilterTable
	showHidden bool

	mode      string // "list", "find" (fd) or "grep" (rg)
	auxInput  textinput.Model
	auxGen    *atomic.Uint64 // debounce/spawn generation
	auxSearch bool           // a search spawn is in flight
	findRes   []files.Entry  // latest fd results
	grepRes   []files.Match  // latest rg results
	findTbl   ui.FilterTable
	grepTbl   ui.FilterTable

	backStack []string // visited dirs; ctrl+o pops
	fwdStack  []string // forward history; ctrl+i pops

	stager *files.Stager // shared across value copies: closures stage into it
	marked map[string]bool // marked paths (multi-select)

	clip     []string // clipboard paths
	clipMove bool

	prompt    *textinput.Model
	promptLbl string
	promptAct func(Files, string) tea.Cmd

	permEdit   *files.PermBits
	permTarget string
	permRow    int
	permCol    int

	prevPath  string // path whose preview is displayed
	prevBody  string // rendered body lines
	prevTitle string
	prevMeta  string
	prevHit   int // highlighted line number in the preview (0 = none)
	fetching  bool

	saving   bool
	saveOps  []files.Op
	saveDone int

	opCount *atomic.Int32 // in-flight fs operations; >0 blocks input
}

type dirListMsg struct {
	dir  string
	list []files.Entry
	err  error
}

type filePreviewMsg struct {
	path string // file the preview belongs to
	key  string // fetch identity used for staleness checks
	p    files.Preview
	err  error
	hit  int // highlight this line when the preview came from a grep hit
}

type dirPreviewMsg struct {
	path string
	key  string
	list []files.Entry
}

type stageStepMsg struct {
	done  int
	total int
	label string
	err   error
}

// NewFiles builds the file manager rooted at $HOME.
func NewFiles() Files {
	cols := []table.Column{
		{Title: "name", Width: 34},
		{Title: "size", Width: 8},
		{Title: "mode", Width: 10},
		{Title: "owner", Width: 10},
	}
	return Files{
		cwd:      files.Home(),
		marked:   map[string]bool{},
		stager:   files.NewStager(),
		tbl:      ui.NewFilterTable(cols, 80, 20),
		opCount:  &atomic.Int32{},
		mode:     "list",
		auxGen:   &atomic.Uint64{},
		auxInput: newAuxInput(),
		findTbl: ui.NewFilterTable([]table.Column{
			{Title: "name", Width: 30}, {Title: "dir", Width: 40}, {Title: "size", Width: 8},
		}, 80, 18),
		grepTbl: ui.NewFilterTable([]table.Column{
			{Title: "loc", Width: 30}, {Title: "text", Width: 60},
		}, 80, 18),
	}
}

// ID implements ui.Screen.
func (f Files) ID() string { return "files" }

// Title implements ui.Screen.
func (f Files) Title() string { return "Files" }

// Hints implements ui.Screen.
func (f Files) Hints() []key.Binding {
	if f.mode != "list" { // aux search: the query bar owns everything
		return []key.Binding{
			key.NewBinding(key.WithKeys("up/down"), key.WithHelp("up/down", "results")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "reveal dir")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "exit search")),
		}
	}
	h := []key.Binding{
		ui.Keys.Filter,
		key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "find")),
		key.NewBinding(key.WithKeys("F"), key.WithHelp("F", "grep")),
		key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "mark")),
		key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "hidden")),
		key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "mkdir")),
		key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "rename")),
		key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy")),
		key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "cut")),
		key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "paste")),
	}
	if n := f.stager.Len(); n > 0 {
		h = append(h,
			key.NewBinding(key.WithKeys("w"), key.WithHelp("w", fmt.Sprintf("save %d", n))),
			key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "undo op")))
	}
	if len(f.backStack) > 0 {
		h = append(h, key.NewBinding(key.WithKeys("ctrl+o"),
			key.WithHelp("ctrl+o", "back")))
	}
	if len(f.fwdStack) > 0 {
		h = append(h, key.NewBinding(key.WithKeys("ctrl+i"),
			key.WithHelp("ctrl+i", "forward")))
	}
	return h
}

// Badge implements ui.BadgeSource: pending-change count for the tab strip.
func (f Files) Badge() string {
	if n := f.stager.Len(); n > 0 {
		return "*" + itoa(n)
	}
	return ""
}

// CapturingInput implements ui.Screen.
func (f Files) CapturingInput() bool {
	return f.tbl.Filtering() || f.prompt != nil || f.permEdit != nil ||
		f.mode != "list"
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
		f.pruneMarks()
		f.syncTable()
		if e, ok := f.selected(); ok && e.Path != f.prevPath && !f.fetching {
			f.fetching = true
			return f, fetchPreview(e)
		}
		return f, nil

	case filePreviewMsg:
		f.fetching = false
		if m.key != f.expectKey() {
			break // stale: the cursor moved on to something else
		}
		e, _ := f.entryByPath(m.path)
		f.prevPath = m.path
		f.prevHit = m.hit
		switch {
		case m.err == nil && !m.p.Binary:
			if e != nil {
				f.prevTitle = e.Name
			} else {
				f.prevTitle = filepathBase(m.path)
			}
			f.prevMeta = sysinfo.FormatBytes(float64(m.p.Size))
			if e != nil {
				f.prevMeta = entryMetaLine(*e)
			}
			if m.hit > 0 {
				f.prevMeta += " - line " + itoa(m.hit)
			}
			body := numberLines(m.p.Lines, m.p.First, m.hit)
			if m.p.Truncated {
				body += "\n" + faintSty.Render(".. truncated")
			}
			f.prevBody = body
		case m.err != nil:
			f.prevTitle = filepathBase(m.path)
			f.prevMeta = ""
			f.prevBody = badSty.Render(ui.Truncate(m.err.Error(), f.paneW()-2))
		default: // binary: metadata fallback
			f.prevTitle = filepathBase(m.path)
			f.prevMeta = sysinfo.FormatBytes(float64(m.p.Size))
			f.prevBody = faintSty.Render("binary file - no text preview")
			if e != nil {
				f.prevBody = metaCard(*e, f.paneW()-2) + "\n\n" + f.prevBody
			}
		}

	case dirPreviewMsg:
		f.fetching = false
		if m.key != f.expectKey() {
			break
		}
		e, _ := f.entryByPath(m.path)
		f.prevPath = m.path
		f.prevBody = dirListingCard(m.list)
		f.prevMeta = itoa(len(m.list)) + " entries"
		if e != nil {
			f.prevTitle, f.prevMeta = e.Name, entryMetaLine(*e)+" - "+itoa(len(m.list))+" entries"
		} else {
			f.prevTitle = filepathBase(m.path)
		}

	case stageStepMsg:
		return f.stageStep(m)

	case auxDebounceMsg:
		if f.mode != "list" {
			return f, f.auxDebounced(m)
		}

	case findResultMsg:
		f.findRes = nil
		f.auxSearch = false
		if m.gen != f.auxGen.Load() {
			return f, nil // stale search
		}
		if m.err != nil {
			return f, ui.ErrToast(m.err.Error())
		}
		f.findRes = m.entries
		f.syncAuxTables()
		return f, f.auxFollow()

	case grepResultMsg:
		f.grepRes = nil
		f.auxSearch = false
		if m.gen != f.auxGen.Load() {
			return f, nil // stale search
		}
		if m.err != nil {
			return f, ui.ErrToast(m.err.Error())
		}
		f.grepRes = m.matches
		f.syncAuxTables()
		return f, f.auxFollow()

	case tea.KeyMsg:
		return f.handleKey(m)
	}
	return f, nil
}

// isTextual was folded into the filePreviewMsg handler.

// stageStep consumes one save step; failures stop the run while
// already-applied ops are dropped from the queue.
func (f Files) stageStep(m stageStepMsg) (ui.Screen, tea.Cmd) {
	f.saveDone = m.done
	if m.err != nil {
		f.saving = false
		f.opCount.Add(-1)
		f.stager.DropFirst(m.done - 1)
		return f, tea.Batch(
			ui.ErrToast(m.label+": "+m.err.Error()),
			listDir(f.cwd, f.showHidden))
	}
	if m.done < m.total {
		return f, runStageStep(f.saveOps, m.done)
	}
	n := m.total
	f.saving = false
	f.opCount.Add(-1)
	f.stager.Clear()
	f.saveOps = nil
	f.syncTable()
	return f, tea.Batch(ui.OkToast(fmt.Sprintf("saved %d change%s", n, plural(n))),
		listDir(f.cwd, f.showHidden))
}

func runStageStep(ops []files.Op, i int) tea.Cmd {
	op := ops[i]
	return func() tea.Msg {
		err := files.ApplyOp(op)
		return stageStepMsg{done: i + 1, total: len(ops), label: op.Label(), err: err}
	}
}

// startSave begins applying the staged queue one op per message so the
// header can show progress.
func (f *Files) startSave() tea.Cmd {
	if f.stager.Len() == 0 || f.opCount.Load() > 0 || f.saving {
		return nil
	}
	f.saving = true
	f.saveOps = f.stager.Ops()
	f.saveDone = 0
	f.opCount.Add(1)
	return runStageStep(f.saveOps, 0)
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// targets returns the entries an operation applies to: the marked set
// in listing order, or the cursor entry.
func (f Files) targets() []files.Entry {
	if len(f.marked) == 0 {
		if e, ok := f.selected(); ok {
			return []files.Entry{*e}
		}
		return nil
	}
	var out []files.Entry
	for _, e := range f.entries {
		if f.marked[e.Path] {
			out = append(out, e)
		}
	}
	return out
}

func (f Files) selectedPath() string {
	if e, ok := f.selected(); ok {
		return e.Path
	}
	return ""
}

func (f Files) entryByPath(p string) (*files.Entry, bool) {
	for i := range f.entries {
		if f.entries[i].Path == p {
			return &f.entries[i], true
		}
	}
	return nil, false
}

// pruneMarks drops marks whose paths left the listing (deleted or
// navigated away); marks only make sense within the visible dir.
func (f *Files) pruneMarks() {
	if len(f.marked) == 0 {
		return
	}
	live := map[string]bool{}
	for _, e := range f.entries {
		live[e.Path] = true
	}
	for p := range f.marked {
		if !live[p] {
			delete(f.marked, p)
		}
	}
}

// syncTable rebuilds rows; marks and staged glyphs live in the name
// cell so the name column keeps its full width. Cursor follows.
func (f *Files) syncTable() {
	stagedAt := map[string]files.OpKind{}
	for _, op := range f.stager.Ops() {
		switch op.Kind {
		case files.OpMkdir:
			stagedAt[op.Path] = op.Kind
		case files.OpDelete, files.OpRename, files.OpChmod:
			stagedAt[op.Path] = op.Kind
		}
	}
	rows := make([]table.Row, len(f.entries))
	keys := make([]string, len(f.entries))
	for i, e := range f.entries {
		mark := "  "
		if f.marked[e.Path] {
			mark = lipgloss.NewStyle().Foreground(ui.Palette.Mauve).Render("* ")
		}
		glyph := ""
		if k, ok := stagedAt[e.Path]; ok {
			glyph = stagedGlyph(k) + " "
		}
		name := e.Name
		if e.IsDir {
			name = lipgloss.NewStyle().Bold(true).
				Foreground(ui.Accent("files")).Render(name + "/")
		}
		rows[i] = table.Row{
			mark + glyph + name,
			sizeOrDash(e),
			modeCell(e.Mode),
			mutedSty.Render(files.UserName(e.UID)),
		}
		keys[i] = e.Path
	}
	f.tbl.SetRowsTracked(rows, keys)
}

func stagedGlyph(k files.OpKind) string {
	var s string
	switch k {
	case files.OpMkdir:
		s = "+"
	case files.OpDelete:
		s = "-"
	case files.OpRename:
		s = ">"
	case files.OpChmod:
		s = "~"
	default:
		return ""
	}
	c := ui.Palette.Green
	switch k {
	case files.OpDelete:
		c = ui.Palette.Red
	case files.OpRename:
		c = ui.Palette.Yellow
	case files.OpChmod:
		c = ui.Palette.Blue
	}
	return lipgloss.NewStyle().Bold(true).Foreground(c).Render(s)
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

// navigate records a directory change into the back/forward history.
func (f *Files) navigate(to string) tea.Cmd {
	if to == f.cwd {
		return listDir(to, f.showHidden)
	}
	f.backStack = append(f.backStack, f.cwd)
	f.fwdStack = nil
	return listDir(to, f.showHidden)
}

// goBack / goForward walk the traversal history (ctrl+o / ctrl+i).
func (f *Files) goBack() tea.Cmd {
	if len(f.backStack) == 0 {
		return nil
	}
	prev := f.backStack[len(f.backStack)-1]
	f.backStack = f.backStack[:len(f.backStack)-1]
	f.fwdStack = append(f.fwdStack, f.cwd)
	return listDir(prev, f.showHidden)
}

func (f *Files) goForward() tea.Cmd {
	if len(f.fwdStack) == 0 {
		return nil
	}
	next := f.fwdStack[len(f.fwdStack)-1]
	f.fwdStack = f.fwdStack[:len(f.fwdStack)-1]
	f.backStack = append(f.backStack, f.cwd)
	return listDir(next, f.showHidden)
}

func (f Files) handleKey(m tea.KeyMsg) (ui.Screen, tea.Cmd) {
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

	if f.opCount.Load() > 0 { // operation in flight: input stays locked
		return f, nil
	}
	if f.mode != "list" { // find/grep: the query bar owns the keyboard
		return f.handleAuxKey(m)
	}
	if f.tbl.Filtering() {
		var cmd tea.Cmd
		f.tbl, cmd = f.tbl.Update(m)
		return f, cmd
	}

	switch m.String() {
	case "f":
		return f, f.startAux("find")
	case "F":
		return f, f.startAux("grep")
	case "ctrl+o":
		return f, f.goBack()
	case "ctrl+i":
		return f, f.goForward()
	case "enter", "l":
		if e, ok := f.selected(); ok && e.IsDir {
			f.clearMarks()
			return f, f.navigate(e.Path)
		}
		return f, nil
	case "h":
		parent := parentDir(f.cwd)
		if parent != f.cwd {
			f.clearMarks()
			return f, f.navigate(parent)
		}
	case "a":
		f.showHidden = !f.showHidden
		return f, listDir(f.cwd, f.showHidden)
	case " ":
		if p := f.selectedPath(); p != "" {
			if f.marked[p] {
				delete(f.marked, p)
			} else {
				f.marked[p] = true
			}
			f.syncTable()
		}
		return f, nil
	case "esc":
		if len(f.marked) > 0 {
			f.clearMarks()
		}
		return f, nil
	case "d":
		ts := f.targets()
		var errs []string
		for _, e := range ts {
			if err := f.stager.Stage(files.Op{Kind: files.OpDelete, Path: e.Path}); err != nil {
				errs = append(errs, err.Error())
			}
		}
		cmd := f.afterStage(errs, fmt.Sprintf("staged delete %d", len(ts)))
		return f, cmd
	case "m":
		return f.startPrompt("new directory: ", func(ff Files, name string) tea.Cmd {
			p := filepath.Join(ff.cwd, name)
			if err := ff.stager.Stage(files.Op{Kind: files.OpMkdir, Path: p}); err != nil {
				return ui.ErrToast(err.Error())
			}
			return ui.InfoToast("staged create " + name)
		})
	case "R":
		if e, ok := f.selected(); ok {
			cur := e.Name
			target := e.Path
			return f.startPrompt("rename: ", func(ff Files, name string) tea.Cmd {
				if err := ff.stager.Stage(files.Op{Kind: files.OpRename, Path: target, Arg: name}); err != nil {
					return ui.ErrToast(err.Error())
				}
				return ui.InfoToast("staged rename -> " + name)
			}, cur)
		}
	case "P":
		if e, ok := f.selected(); ok {
			bits := files.ParsePermBits(e.Mode)
			f.permEdit = &bits
			f.permTarget = e.Path
			f.permRow = 0
		}
		return f, nil
	case "y":
		return f, f.copyTargets(false)
	case "x":
		return f, f.copyTargets(true)
	case "p":
		if len(f.clip) == 0 {
			return f, ui.InfoToast("clipboard empty")
		}
		kind := files.OpCopy
		verb := "copy"
		if f.clipMove {
			kind = files.OpMove
			verb = "move"
		}
		var errs []string
		n := 0
		for _, src := range f.clip {
			err := f.stager.Stage(files.Op{Kind: kind, Path: src, Arg: f.cwd})
			if err != nil {
				errs = append(errs, err.Error())
			} else {
				n++
			}
		}
		return f, f.afterStage(errs, fmt.Sprintf("staged %s %d", verb, n))
	case "u":
		if op, ok := f.stager.Undo(); ok {
			f.syncTable()
			return f, ui.InfoToast("unstaged " + op.Label())
		}
		return f, nil
	case "U":
		n := f.stager.Len()
		f.stager.Clear()
		f.syncTable()
		return f, ui.InfoToast(fmt.Sprintf("discarded %d change%s", n, plural(n)))
	case "w":
		return f, f.startSave()
	}

	moved := false
	switch m.String() {
	case "up", "down", "j", "k", "g", "G", "home", "end", "pgup", "pgdown":
		moved = true
	}
	var cmd tea.Cmd
	f.tbl, cmd = f.tbl.Update(m)
	if moved {
		if p := f.selectedPath(); p != "" && p != f.prevPath && !f.fetching {
			f.fetching = true
			e, _ := f.selected()
			cmd = tea.Batch(cmd, fetchPreviewCmd(*e))
		}
	}
	return f, cmd
}

func (f *Files) clearMarks() {
	if len(f.marked) == 0 {
		return
	}
	f.marked = map[string]bool{}
	f.syncTable()
}

func (f *Files) copyTargets(move bool) tea.Cmd {
	ts := f.targets()
	if len(ts) == 0 {
		return nil
	}
	f.clip = make([]string, len(ts))
	for i, e := range ts {
		f.clip[i] = e.Path
	}
	f.clipMove = move
	verb := "copied"
	if move {
		verb = "cut"
	}
	what := ts[0].Name
	if len(ts) > 1 {
		what = fmt.Sprintf("%d paths", len(ts))
	}
	return ui.InfoToast(verb + " " + what)
}

func (f Files) afterStage(errs []string, okMsg string) tea.Cmd {
	f.syncTable()
	if len(errs) > 0 {
		return ui.ErrToast(strings.Join(errs, "; "))
	}
	return ui.InfoToast(okMsg)
}

func dirFileWord(isDir bool) string {
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
		err := f.stager.Stage(files.Op{Kind: files.OpChmod, Path: target, Mode: mode})
		if err != nil {
			return f, ui.ErrToast(err.Error())
		}
		f.syncTable()
		return f, ui.InfoToast("staged chmod " + bits.Octal() + " " + filepathBase(target))
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
	wide, mainW, _ := splitGeom(f.w)
	tw := clampInt(f.w-2, 30, f.w)
	if wide {
		tw = mainW
	}
	// Two chrome rows live above the table: the page head and the
	// main pane's own breadcrumb header.
	f.tbl.SetSize(tw, clampInt(f.h-2, 5, f.h))
	// Aux search adds one more: the query bar.
	f.findTbl.SetSize(tw, clampInt(f.h-3, 5, f.h))
	f.grepTbl.SetSize(tw, clampInt(f.h-3, 5, f.h))
}

func (f Files) paneW() int {
	_, _, pw := splitGeom(f.w)
	return pw
}

// View renders listing | preview plus modals; staged state shows in
// rows, badge and head.
func (f Files) View() string {
	if f.w == 0 {
		return ""
	}
	var meta string
	if f.mode != "list" {
		meta = fmt.Sprintf("%s %q - %d results - enter reveals dir",
			f.mode, f.auxInput.Value(), f.auxCount())
	}
	if meta == "" && f.showHidden {
		meta = "hidden on"
	}
	if f.mode == "list" {
		if n := f.stager.Len(); n > 0 {
			if meta != "" {
				meta += " - "
			}
			meta += fmt.Sprintf("%d pending", n)
		}
		if f.saving {
			meta += fmt.Sprintf(" - saving %d/%d", f.saveDone, len(f.saveOps))
		} else if f.opCount.Load() > 0 {
			meta += " - working.."
		}
	}
	head := pageHead("Files", meta, f.w)

	var main string
	if f.mode != "list" {
		main = f.auxBar() + "\n" + f.auxTable().View()
	} else {
		main = f.mainHeader() + "\n" + f.tbl.View()
	}
	wide, mainW, _ := splitGeom(f.w)
	var prev string
	if wide {
		prev = renderPreview("preview", f.previewTitle(), f.previewMeta(),
			f.previewContent(), f.paneW(), f.h-1)
	} else { // stacked: split the body between list, rule and preview
		avail := f.h - 2 // head + rule
		keep := clampInt(avail*3/5, 4, avail-1)
		lines := strings.Split(ui.ClipBlock(main, mainW), "\n")
		if len(lines) > keep {
			main = strings.Join(lines[:keep], "\n")
		}
		prev = renderPreview("preview", f.previewTitle(), f.previewMeta(),
			f.previewContent(), f.paneW(), avail-keep-1)
	}
	body := joinPanesWide(wide, main, prev, mainW, f.w)

	out := head + "\n" + body
	lines := strings.Split(out, "\n")
	for len(lines) < f.h {
		lines = append(lines, "")
	}
	if len(lines) > f.h {
		lines = lines[:f.h]
	}

	view := strings.Join(lines, "\n")
	if f.permEdit != nil {
		return overlayCenter(view, f.permEditorView(), f.w)
	}
	if f.prompt != nil {
		box := ui.Panel().BorderForeground(ui.Accent("files")).Width(clampInt(f.w/2, 40, 60)).
			Padding(0, 1).Render(f.promptLbl + f.prompt.View())
		return overlayCenter(view, box, f.w)
	}
	return view
}

// paneMainW returns the main column width for the current geometry.
func (f Files) paneMainW() int {
	_, mainW, _ := splitGeom(f.w)
	return mainW
}

// mainHeader renders the main pane's own title row: accent breadcrumb,
// entry count right.
func (f Files) mainHeader() string {
	crumb := crumbMeta(f.cwd)
	count := mutedSty.Render(itoa(len(f.entries)) + " items")
	gap := f.paneMainW() - lipgloss.Width(crumb) - lipgloss.Width(count) - 1
	if gap < 1 {
		gap = 1
	}
	return ui.ClipBlock(crumb+" "+strings.Repeat(" ", gap)+count, f.paneMainW())
}

// --- preview plumbing -------------------------------------------------

func entryMetaLine(e files.Entry) string {
	bits := files.ParsePermBits(e.Mode)
	return bits.Octal() + " - " + files.UserName(e.UID) +
		" - " + sysinfo.FormatBytes(float64(e.Size))
}

// fetchPreview issues the right async read for the given entry.
func fetchPreview(e *files.Entry) tea.Cmd { return fetchPreviewCmd(*e) }

func fetchPreviewCmd(e files.Entry) tea.Cmd {
	key := e.Path
	if e.IsDir {
		return func() tea.Msg {
			l, err := files.List(e.Path, false)
			if err != nil {
				return filePreviewMsg{path: e.Path, key: key, err: err}
			}
			const capEntries = 40
			if len(l) > capEntries {
				l = l[:capEntries]
			}
			return dirPreviewMsg{path: e.Path, key: key, list: l}
		}
	}
	return func() tea.Msg {
		p, err := files.ReadPreview(e.Path, 60, 16<<10)
		return filePreviewMsg{path: e.Path, key: key, p: p, err: err}
	}
}

// numberLines prefixes faint line numbers gutter-aligned to the block;
// hit (when non-zero) is highlighted as an accent line.
func numberLines(lines []string, first, hit int) string {
	gutter := len([]rune(itoa(first + len(lines))))
	if gutter < 2 {
		gutter = 2
	}
	hitSty := lipgloss.NewStyle().Bold(true).Foreground(ui.Accent("files"))
	var b strings.Builder
	for i, l := range lines {
		n := first + i
		prefix := faintSty.Render(padLeft(itoa(n), gutter)+"  ")
		line := prefix + l
		if n == hit {
			b.WriteString(hitSty.Render(padLeft(itoa(n), gutter)) + "  " +
				hitSty.Render(l))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func dirListingCard(list []files.Entry) string {
	var b strings.Builder
	for _, e := range list {
		name := e.Name
		if e.IsDir {
			name = lipgloss.NewStyle().Bold(true).
				Foreground(ui.Accent("files")).Render(name + "/")
		}
		b.WriteString("  " + name + strings.Repeat(" ",
			maxInt(30-lipgloss.Width(name), 1)) +
			faintSty.Render(sizeOrDash(e)) + "\n")
	}
	if len(list) == 0 {
		b.WriteString(faintSty.Render("empty directory"))
	}
	return strings.TrimRight(b.String(), "\n")
}

func metaCard(e files.Entry, w int) string {
	kv := func(k, v string) string {
		return mutedSty.Render(padTo(k, 9)) + faintSty.Render(ui.Truncate(v, maxInt(w-11, 4)))
	}
	bits := files.ParsePermBits(e.Mode)
	lines := []string{
		kv("kind", dirFileWord(e.IsDir)),
		kv("size", sysinfo.FormatBytes(float64(e.Size))),
		kv("owner", files.UserName(e.UID)),
		kv("group", files.GroupName(e.GID)),
		kv("perms", bits.Symbolic()+" ("+bits.Octal()+")"),
		kv("modified", e.ModTime.Format(time.RFC3339)),
	}
	if e.Link != "" {
		lines = append(lines, kv("link →", e.Link))
	}
	return strings.Join(lines, "\n")
}

func (f Files) previewTitle() string {
	if f.prevTitle != "" {
		return truncCell(f.prevTitle, maxInt(f.paneW()-4, 6))
	}
	return "preview"
}

func (f Files) previewMeta() string { return f.prevMeta }

func (f Files) previewContent() string {
	if f.prevBody != "" {
		return f.prevBody
	}
	if f.fetching {
		return faintSty.Render("loading..")
	}
	return "" // nothing to show yet: leave the pane blank
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
	help := faintSty.Render("h/j/k/l move - space toggle - enter stage - esc cancel")

	body := lipgloss.NewStyle().Bold(true).Foreground(ui.Accent("files")).
		Render("permissions - "+target) + "\n\n" +
		grid + "\n" +
		mutedSty.Render(bits.Symbolic()) + "  " +
		lipgloss.NewStyle().Bold(true).Foreground(ui.Palette.Yellow).
			Render(bits.Octal()) + "\n\n" + help

	return ui.Panel().BorderForeground(ui.Accent("files")).
		Width(clampInt(f.w/2, 40, 56)).Padding(1, 2).Render(body)
}
