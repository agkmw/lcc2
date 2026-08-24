package screens

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lcc2/internal/files"
	"lcc2/internal/ui"
)

// Auxiliary Files modes: recursive filename search (fd) and content
// search (rg). One shared skeleton: an inline query bar above a live
// results table, fzf-style — typing narrows, j/k moves, the preview
// pane follows the cursor.
const (
	auxDebounce = 180 * time.Millisecond
	findLimit   = 1000
	grepLimit   = 500
)

type auxDebounceMsg struct {
	mode string
	gen  uint64
}

type findResultMsg struct {
	gen     uint64
	entries []files.Entry
	err     error
}

type grepResultMsg struct {
	gen     uint64
	matches []files.Match
	err     error
}

func newAuxInput() textinput.Model {
	ti := textinput.New()
	ti.Prompt = "/"
	ti.Placeholder = "type to search"
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ui.Accent("files"))
	ti.TextStyle = lipgloss.NewStyle().Foreground(ui.Palette.Text)
	return ti
}

// startAux enters find/grep mode with a focused query input.
func (f *Files) startAux(mode string) tea.Cmd {
	f.mode = mode
	f.prevHit = 0
	f.auxInput.SetValue("")
	f.auxInput.Focus()
	return textinput.Blink
}

func (f *Files) exitAux() tea.Cmd {
	f.mode = "list"
	f.auxInput.Blur()
	f.prevHit = 0
	return listDir(f.cwd, f.showHidden)
}

// bumpAux schedules the debounced spawn for the current query.
func (f *Files) bumpAux() tea.Cmd {
	gen := f.auxGen.Add(1)
	return tea.Tick(auxDebounce, func(time.Time) tea.Msg {
		return auxDebounceMsg{mode: f.mode, gen: gen}
	})
}

func auxRun(gen uint64, mode, root, query string, hidden bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if mode == "find" {
			es, err := files.Find(ctx, root, query, hidden, findLimit)
			return findResultMsg{gen: gen, entries: es, err: err}
		}
		ms, err := files.Grep(ctx, root, query, hidden, grepLimit)
		return grepResultMsg{gen: gen, matches: ms, err: err}
	}
}

// auxDebounced fires when typing pauses; spawns the real search unless
// another keystroke superseded it.
func (f Files) auxDebounced(m auxDebounceMsg) tea.Cmd {
	if m.gen != f.auxGen.Load() || m.mode != f.mode {
		return nil
	}
	f.auxSearch = true
	return auxRun(m.gen, f.mode, f.cwd, f.auxInput.Value(), f.showHidden)
}

// handleAuxKey owns the keyboard in find/grep mode. Arrow keys steer
// the results cursor while every other key feeds the query input —
// including letters like j/k, which are legitimate query characters.
func (f Files) handleAuxKey(m tea.KeyMsg) (ui.Screen, tea.Cmd) {
	steer := func(key tea.KeyType) tea.Cmd {
		tbl := f.auxTable()
		var cmd tea.Cmd
		*tbl, cmd = tbl.Update(tea.KeyMsg{Type: key})
		return tea.Batch(cmd, f.auxFollow())
	}
	switch m.String() {
	case "esc":
		return f, f.exitAux()
	case "enter":
		return f.auxOpen()
	case "up":
		return f, steer(tea.KeyUp)
	case "down":
		return f, steer(tea.KeyDown)
	case "pgup":
		return f, steer(tea.KeyPgUp)
	case "pgdown":
		return f, steer(tea.KeyPgDown)
	case "home":
		return f, steer(tea.KeyHome)
	case "end":
		return f, steer(tea.KeyEnd)
	}

	prev := f.auxInput.Value()
	var cmd tea.Cmd
	f.auxInput, cmd = f.auxInput.Update(m)
	if f.auxInput.Value() != prev {
		cmd = tea.Batch(cmd, f.bumpAux())
	}
	return f, cmd
}

// auxOpen acts on the cursor: both modes reveal the hit's directory in
// list mode (find enters directories themselves).
func (f Files) auxOpen() (ui.Screen, tea.Cmd) {
	dir := ""
	if f.mode == "find" {
		if idx, ok := f.findTbl.Selected(); ok && idx < len(f.findRes) {
			e := f.findRes[idx]
			dir = e.Path
			if !e.IsDir {
				dir = filepath.Dir(e.Path)
			}
		}
	} else if idx, ok := f.grepTbl.Selected(); ok && idx < len(f.grepRes) {
		dir = filepath.Dir(f.grepRes[idx].Path)
	}
	if dir == "" {
		return f, nil
	}
	sc := f
	sc.mode = "list"
	sc.auxInput.Blur()
	sc.prevHit = 0
	return sc, sc.navigate(dir)
}

// expectKey is the fetch identity of the current cursor: previews for
// anything else are stale on arrival.
func (f Files) expectKey() string {
	switch f.mode {
	case "find":
		if idx, ok := f.findTbl.Selected(); ok && idx < len(f.findRes) {
			return f.findRes[idx].Path
		}
	case "grep":
		if idx, ok := f.grepTbl.Selected(); ok && idx < len(f.grepRes) {
			m := f.grepRes[idx]
			return fmt.Sprintf("%s:%d", m.Path, m.Line)
		}
	default:
		return f.selectedPath()
	}
	return ""
}

// auxFollow kicks the preview for the cursor's result.
func (f *Files) auxFollow() tea.Cmd {
	switch f.mode {
	case "find":
		idx, ok := f.findTbl.Selected()
		if !ok || idx >= len(f.findRes) || f.fetching {
			return nil
		}
		f.fetching = true
		return fetchPreviewCmd(f.findRes[idx])
	case "grep":
		idx, ok := f.grepTbl.Selected()
		if !ok || idx >= len(f.grepRes) || f.fetching {
			return nil
		}
		mt := f.grepRes[idx]
		f.fetching = true
		key := fmt.Sprintf("%s:%d", mt.Path, mt.Line)
		return func() tea.Msg {
			p, err := files.ReadPreviewAt(mt.Path, mt.Line, 60, 16<<10)
			return filePreviewMsg{path: mt.Path, key: key, p: p, err: err, hit: mt.Line}
		}
	}
	return nil
}

// auxTable returns whichever results table is active. Pointer
// receiver: callers mutate through the returned field, so it must
// alias the caller's struct, not a receiver copy.
func (f *Files) auxTable() *ui.FilterTable {
	if f.mode == "find" {
		return &f.findTbl
	}
	return &f.grepTbl
}

// syncAuxTables rebuilds result rows from the latest response.
func (f *Files) syncAuxTables() {
	rows := make([]table.Row, len(f.findRes))
	keys := make([]string, len(f.findRes))
	for i, e := range f.findRes {
		name := e.Name
		if e.IsDir {
			name = lipgloss.NewStyle().Bold(true).
				Foreground(ui.Accent("files")).Render(name + "/")
		}
		rows[i] = table.Row{name, ui.Truncate(filepath.Dir(e.Path), 46), sizeOrDash(e)}
		keys[i] = e.Path
	}
	f.findTbl.SetRowsTracked(rows, keys)

	rows2 := make([]table.Row, len(f.grepRes))
	keys2 := make([]string, len(f.grepRes))
	for i, m := range f.grepRes {
		loc := filepath.Base(m.Path) + ":" + strconv.Itoa(m.Line)
		rows2[i] = table.Row{
			lipgloss.NewStyle().Foreground(ui.Accent("files")).Render(loc),
			ui.Truncate(strings.TrimSpace(m.Text), 60),
		}
		keys2[i] = fmt.Sprintf("%s:%d:%d", m.Path, m.Line, m.Col)
	}
	f.grepTbl.SetRowsTracked(rows2, keys2)
}

// auxCount reports result rows for the header meta.
func (f Files) auxCount() int {
	if f.mode == "find" {
		return len(f.findRes)
	}
	return len(f.grepRes)
}

// auxStatus is the right-hand query-bar slot: live feedback while a
// search runs, the tally otherwise.
func (f Files) auxStatus() string {
	if f.auxSearch {
		return warnSty.Render("searching..")
	}
	return faintSty.Render(strconv.Itoa(f.auxCount()) + " results")
}

// auxBar renders the query row above the results.
func (f Files) auxBar() string {
	label := mutedSty.Render(f.mode+" ") + f.auxInput.View()
	status := f.auxStatus()
	budget := f.paneMainW() - lipgloss.Width(status) - 2
	bar := ui.ClipBlock(label, maxInt(budget, 12))
	gap := f.paneMainW() - lipgloss.Width(bar) - lipgloss.Width(status)
	if gap < 1 {
		gap = 1
	}
	return bar + strings.Repeat(" ", gap) + status
}
