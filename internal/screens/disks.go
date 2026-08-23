package screens

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
// directory usage analysis with drill-down navigation.
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

func fsCols() []table.Column {
	return []table.Column{
		{Title: "mount", Width: 16},
		{Title: "device", Width: 20},
		{Title: "type", Width: 7},
		{Title: "size", Width: 9},
		{Title: "used", Width: 9},
		{Title: "free", Width: 9},
		{Title: "use%", Width: 6},
	}
}

func itemCols() []table.Column {
	return []table.Column{
		{Title: "name", Width: 44},
		{Title: "size", Width: 11},
		{Title: "% of dir", Width: 24},
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
			ui.Keys.Select,
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "analyze")),
			ui.Keys.Refresh,
			ui.Keys.Filter,
		}
	}
	return []key.Binding{
		ui.Keys.Back,
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "drill down")),
		ui.Keys.Refresh,
		ui.Keys.Filter,
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
		rows := make([]table.Row, len(m))
		for i, f := range m {
			rows[i] = table.Row{
				f.Mountpoint, f.Device, f.FSType,
				sysinfo.FormatBytes(float64(f.Total)),
				sysinfo.FormatBytes(float64(f.Used)),
				sysinfo.FormatBytes(float64(f.Free)),
				itoa(int(f.UsedPercent)),
			}
		}
		d.fsTbl.SetRows(rows)

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
		var cmd tea.Cmd
		d.spin, cmd = d.spin.Update(m)
		return d, cmd

	case tea.KeyMsg:
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

	if d.mode == "fs" {
		var cmd tea.Cmd
		d.fsTbl, cmd = d.fsTbl.Update(m)
		return d, cmd
	}
	var cmd tea.Cmd
	d.dirTbl, cmd = d.dirTbl.Update(m)
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
		d.spin.Tick,
		func() tea.Msg {
			res, err := disk.ScanDir(ctx, root, func(bytes int64) {
				_ = bytes // progress kept coarse; duration shown on completion
			})
			return scanDoneMsg{root: root, res: res, err: err}
		},
	)
}

func isChildDir(child, parent string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "."
}

func (d *Disks) syncItems() {
	if d.items == nil {
		return
	}
	total := float64(d.items.TotalSize)
	rows := make([]table.Row, 0, len(d.items.Items))
	for _, it := range d.items.Items {
		pct := 0.0
		if total > 0 {
			pct = float64(it.Size) / total * 100
		}
		bar := ui.Gauge(pct, 22, colorPtr(ui.Accent("disk")))
		name := it.Name
		if it.IsDir {
			name += "/"
		}
		rows = append(rows, table.Row{
			truncCell(name, 44),
			sysinfo.FormatBytes(float64(it.Size)),
			bar,
		})
	}
	d.dirTbl.SetRows(rows)
}

func colorPtr(c lipgloss.Color) *lipgloss.Color { return &c }

func (d *Disks) layout() {
	th := clampInt(d.h-4, 5, d.h)
	d.fsTbl.SetSize(clampInt(d.w-2, 40, d.w), th)
	d.dirTbl.SetSize(clampInt(d.w-2, 40, d.w), clampInt(th-1, 4, th))
}

// View renders the filesystem list or the directory analysis view.
func (d Disks) View() string {
	if d.w == 0 {
		return ""
	}
	if d.mode == "fs" {
		head := faintSty.Render("mounted filesystems")
		return head + "\n" + d.fsTbl.View()
	}

	crumbs := renderCrumbs(d.path, d.w)
	status := ""
	if d.busy {
		status = "\n" + lipgloss.NewStyle().Foreground(ui.Accent("disk")).
			Render(d.spin.View()+" analyzing ") +
			faintSty.Render(ui.Truncate(d.path, clampInt(d.w-16, 10, d.w))) +
			faintSty.Render("  ·  esc to cancel")
	} else if d.err != "" {
		status = "\n" + badSty.Render(d.err)
	} else if d.items != nil {
		status = faintSty.Render("\n" + itoa(len(d.items.Items)) + " entries · " +
			sysinfo.FormatBytes(float64(d.items.TotalSize)) + " · scanned in " +
			d.items.Duration.Round(100*time.Millisecond).String())
	}
	return crumbs + status + "\n" + d.dirTbl.View()
}

func renderCrumbs(path string, w int) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	out := accentDiskSty.Render("/") + " "
	for i, p := range parts {
		if p == "" {
			continue
		}
		sty := mutedSty
		if i == len(parts)-1 {
			sty = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAB387"))
		}
		out += sty.Render(p)
		if i < len(parts)-1 {
			out += faintSty.Render(" > ")
		}
	}
	return " " + ui.Truncate(out, w-2) + "\n"
}

var accentDiskSty = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAB387"))
