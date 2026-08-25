package screens

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/bubbles/v2/key"

	"lcc2/internal/disk"
	"lcc2/internal/sysinfo"
	"lcc2/internal/ui"
)

type fsListMsg []disk.Filesystem
type scanDoneMsg struct {
	root string
	res  *disk.ScanResult
	err  error
}

// Disks is the storage screen: filesystem overview plus interactive
// directory usage analysis, both with a live preview pane.
type Disks struct {
	w, h   int
	spin   spinner.Model
	fss    []disk.Filesystem
	fsTbl  ui.FilterTable
	mode   string // "fs" or "scan"
	path   string // current analyzed directory
	stack  []string
	items  *disk.ScanResult
	dirTbl ui.FilterTable
	busy   bool
	cancel context.CancelFunc
	err    string
}

// NewDisks builds the disks screen.
func NewDisks() Disks {
	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	return Disks{
		spin:   sp,
		mode:   "fs",
		fsTbl:  ui.NewFilterTable(fsCols(), 80, 12),
		dirTbl: ui.NewFilterTable(itemCols(), 80, 16),
	}
}

func fsCols() []ui.Column {
	return []ui.Column{
		{Title: "mount", Width: 16},
		{Title: "device", Width: 18},
		{Title: "type", Width: 7},
		{Title: "size", Width: 9},
		{Title: "use%", Width: 6},
	}
}

func itemCols() []ui.Column {
	return []ui.Column{
		{Title: "name", Width: 40},
		{Title: "size", Width: 10},
		{Title: "share%", Width: 8},
	}
}

// ID implements ui.Screen.
func (d Disks) ID() string { return "disk" }

// Title implements ui.Screen.
func (d Disks) Title() string { return "Disks" }

// Hints implements ui.Screen.
func (d Disks) Hints() []key.Binding {
	if d.mode == "fs" {
		return []key.Binding{
			ui.Keys.Filter,
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "analyze")),
			ui.Keys.Refresh,
		}
	}
	return []key.Binding{
		ui.Keys.Filter,
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "drill down")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		ui.Keys.Refresh,
	}
}

// CapturingInput implements ui.Screen.
func (d Disks) CapturingInput() bool {
	return d.fsTbl.Filtering() || d.dirTbl.Filtering()
}

// Init loads the filesystem list.
func (d Disks) Init() tea.Cmd {
	return func() tea.Msg { return fsListMsg(disk.ListFilesystems()) }
}

// Update handles messages.
func (d Disks) Update(msg tea.Msg) (ui.Screen, tea.Cmd) {
	switch m := msg.(type) {
	case ui.SizeMsg:
		d.w, d.h = m.Width, m.Height
		d.layout()

	case fsListMsg:
		d.fss = m
		rows := make([]ui.Row, len(m))
		keys := make([]string, len(m))
		for i, f := range m {
			pct := itoa(int(f.UsedPercent)) + "%"
			rows[i] = ui.Row{
				f.Mountpoint,
				mutedSty.Render(ui.Truncate(f.Device, 18)),
				faintSty.Render(ui.Narrow(f.FSType)),
				sysinfo.FormatBytes(float64(f.Total)),
				lipgloss.NewStyle().Foreground(ui.StateColor(f.UsedPercent)).Render(pct),
			}
			keys[i] = f.Mountpoint
		}
		d.fsTbl.SetRowsTracked(rows, keys)

	case scanDoneMsg:
		d.busy = false
		d.cancel = nil
		if errors.Is(m.err, context.Canceled) {
			return d, nil // user cancelled; keep the current view
		}
		if m.err != nil {
			d.err = "cannot analyze " + filepath.Base(m.root)
			return d, ui.ErrToast(d.err)
		}
		d.err = ""
		d.items = m.res
		d.path = m.root
		d.syncItems()

	case spinner.TickMsg:
		if !d.busy {
			return d, nil // spinner chain retires when nothing is scanning
		}
		var cmd tea.Cmd
		d.spin, cmd = d.spin.Update(m)
		return d, cmd

	case tea.MouseMsg:
		if d.fsTbl.Filtering() || d.dirTbl.Filtering() {
			return d, nil
		}
		tbl := &d.dirTbl
		if d.mode == "fs" {
			tbl = &d.fsTbl
		}
		_, dbl := tbl.Mouse(m, 3)
		if dbl { // double-click = the enter gesture
			return d.handleKey(tea.KeyPressMsg{Code: tea.KeyEnter})
		}
		return d, nil

	case tea.KeyPressMsg:
		return d.handleKey(m)
	}
	return d, nil
}

func (d Disks) handleKey(m tea.KeyMsg) (ui.Screen, tea.Cmd) {
	if d.mode == "fs" && d.fsTbl.Filtering() {
		var cmd tea.Cmd
		d.fsTbl, cmd = d.fsTbl.Update(m)
		return d, cmd
	}
	if d.mode == "scan" && d.dirTbl.Filtering() {
		var cmd tea.Cmd
		d.dirTbl, cmd = d.dirTbl.Update(m)
		return d, cmd
	}

	switch m.String() {
	case "r":
		if d.mode == "fs" {
			return d, d.Init()
		}
		_, cmd := d.startScan(d.path)
		return d, cmd
	case "esc":
		if d.mode == "scan" {
			return d.backOut()
		}
	case "h":
		if d.mode == "scan" {
			return d.backOut()
		}
	case "enter":
		if d.mode == "fs" {
			if idx, ok := d.fsTbl.Selected(); ok && idx < len(d.fss) {
				return d.startScan(d.fss[idx].Mountpoint)
			}
			return d, nil
		}
		if idx, ok := d.dirTbl.Selected(); ok && d.items != nil && idx < len(d.items.Items) {
			it := d.items.Items[idx]
			if it.IsDir {
				d.stack = append(d.stack, d.path)
				return d.startScan(it.Path)
			}
			return d, nil
		}
	}

	tbl := &d.dirTbl
	if d.mode == "fs" {
		tbl = &d.fsTbl
	}
	var cmd tea.Cmd
	*tbl, cmd = tbl.Update(m)
	return d, cmd
}

func (d Disks) backOut() (ui.Screen, tea.Cmd) {
	if d.cancel != nil { // cancel an in-flight scan first
		d.cancel()
		d.cancel = nil
		d.busy = false
		if len(d.stack) == 0 {
			d.mode = "fs"
			return d, nil
		}
	}
	if len(d.stack) == 0 {
		d.mode = "fs"
		return d, nil
	}
	parent := d.stack[len(d.stack)-1]
	d.stack = d.stack[:len(d.stack)-1]
	return d.startScan(parent)
}

func (d Disks) startScan(root string) (ui.Screen, tea.Cmd) {
	if d.cancel != nil {
		d.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	d.cancel = cancel
	d.busy = true
	d.mode = "scan"
	return d, tea.Batch(
		func() tea.Msg { return d.spin.Tick() },
		func() tea.Msg {
			res, err := disk.ScanDir(ctx, root, func(bytes int64) {
				_ = bytes // progress kept coarse; duration shown on completion
			})
			return scanDoneMsg{root: root, res: res, err: err}
		},
	)
}

func (d *Disks) syncItems() {
	if d.items == nil {
		return
	}
	total := float64(d.items.TotalSize)
	rows := make([]ui.Row, 0, len(d.items.Items))
	keys := make([]string, 0, len(d.items.Items))
	for _, it := range d.items.Items {
		share := 0.0
		if total > 0 {
			share = float64(it.Size) / total * 100
		}
		name := it.Name
		if it.IsDir {
			name = lipgloss.NewStyle().Bold(true).
				Foreground(ui.Accent("disk")).Render(name)
		}
		rows = append(rows, ui.Row{
			name,
			sysinfo.FormatBytes(float64(it.Size)),
			lipgloss.NewStyle().Foreground(ui.StateColor(share)).Render(f1(share)),
		})
		keys = append(keys, it.Path)
	}
	d.dirTbl.SetRowsTracked(rows, keys)
}

func colorPtr(c color.Color) *color.Color { return &c }

func (d *Disks) layout() {
	wide, mainW, _ := splitGeom(d.w)
	tw := clampInt(d.w-2, 40, d.w)
	if wide {
		tw = mainW
	}
	th := clampInt(d.h-1, 5, d.h)
	d.fsTbl.SetSize(tw, th)
	d.dirTbl.SetSize(tw, th)
}

// View renders the filesystem list or analysis view with its preview.
func (d Disks) View() string {
	if d.w == 0 {
		return ""
	}
	wide, mainW, prevW := splitGeom(d.w)

	var head, main string
	if d.mode == "fs" {
		head = pageHead("Disks",
			fmt.Sprintf("%d mounted filesystems", len(d.fss)), d.w)
		main = d.fsTbl.View()
	} else {
		meta := ""
		if d.busy {
			meta = lipgloss.NewStyle().Foreground(ui.Accent("disk")).
				Render(d.spin.View()+" analyzing") +
				faintSty.Render(" - esc cancels")
		} else if d.items != nil {
			meta = fmt.Sprintf("%d entries - %s - %s", len(d.items.Items),
				sysinfo.FormatBytes(float64(d.items.TotalSize)),
				d.items.Duration.Round(100*time.Millisecond).String())
		} else if d.err != "" {
			meta = d.err
		}
		head = pageHead("Directory usage: "+crumbMeta(d.path), meta, d.w)
		main = d.dirTbl.View()
	}

	prev := renderPreview("disk", d.previewTitle(), "", d.previewBody(), prevW, d.h-1)

	if !wide { // stacked: clip list height so the preview fits below
		keep := clampInt(d.h-8, 6, d.h-4)
		lines := strings.Split(ui.ClipBlock(main, mainW), "\n")
		if len(lines) > keep {
			main = strings.Join(lines[:keep], "\n")
		}
	}
	body := joinPanesWide(wide, main, prev, mainW, d.w)

	out := head + "\n" + body
	lines := strings.Split(out, "\n")
	for len(lines) < d.h {
		lines = append(lines, "")
	}
	if len(lines) > d.h {
		lines = lines[:d.h]
	}
	return strings.Join(lines, "\n")
}

// previewBody renders the right-hand pane for either mode.
func (d Disks) previewBody() string {
	_, _, pw := splitGeom(d.w)
	if d.mode == "fs" {
		idx, ok := d.fsTbl.Selected()
		if !ok || idx >= len(d.fss) {
			return ""
		}
		f := d.fss[idx]
		bar := ui.Gauge(f.UsedPercent, clampInt(pw-14, 16, 44), colorPtr(ui.StateColor(f.UsedPercent)))
		kv := func(k, v string) string {
			return mutedSty.Render(padTo(k, 8)) + faintSty.Render(ui.Truncate(v, maxInt(pw-10, 4)))
		}
		return strings.Join([]string{
			bar,
			"",
			kv("device", f.Device),
			kv("type", f.FSType),
			kv("total", sysinfo.FormatBytes(float64(f.Total))),
			kv("used", sysinfo.FormatBytes(float64(f.Used))),
			kv("free", sysinfo.FormatBytes(float64(f.Free))),
		}, "\n")
	}

	idx, ok := d.dirTbl.Selected()
	if !ok || d.items == nil || idx >= len(d.items.Items) {
		return ""
	}
	it := d.items.Items[idx]
	total := float64(d.items.TotalSize)
	share := 0.0
	if total > 0 {
		share = float64(it.Size) / total * 100
	}
	kv := func(k, v string) string {
		return mutedSty.Render(padTo(k, 8)) + faintSty.Render(ui.Truncate(v, maxInt(pw-10, 4)))
	}
	return strings.Join([]string{
		ui.Gauge(share, clampInt(pw-14, 16, 44), colorPtr(ui.Accent("disk"))),
		"",
		kv("kind", dirFileWord(it.IsDir)),
		kv("size", sysinfo.FormatBytes(float64(it.Size))),
		kv("share", f1(share)+"% of scanned"),
		kv("path", it.Path),
	}, "\n")
}

func (d Disks) previewTitle() string {
	if d.mode == "fs" {
		if k, ok := d.fsTbl.SelectedKey(); ok {
			return truncCell(k, 24)
		}
		return "filesystem"
	}
	if k, ok := d.dirTbl.SelectedKey(); ok {
		return truncCell(filepathBase(k), 24)
	}
	return "entry"
}

// crumbMeta renders a path for the page-head meta slot: faint head,
// bold-accent final segment. Absolute paths render with one leading
// slash ("/tmp/x" -> "/tmp/" + "x").
func crumbMeta(path string) string {
	if path == "" {
		return mutedSty.Render("/")
	}
	i := strings.LastIndexByte(path, '/')
	head, last := path[:i+1], path[i+1:]
	if last == "" { // path ends in "/": root
		return mutedSty.Render(head)
	}
	return mutedSty.Render(head) +
		lipgloss.NewStyle().Bold(true).Foreground(ui.Palette.Peach).Render(last)
}
