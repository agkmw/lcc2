// Package proc provides process enumeration with delta-based CPU
// accounting, inspection details and signal delivery.
package proc

import (
	"fmt"
	"os"
	"os/user"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

// Process is one row of the process list.
type Process struct {
	PID        int32
	PPID       int32
	Name       string
	User       string
	State      string // single letter: R, S, D, Z, T…
	CPUPercent float64
	MemPercent float64
	RSS        uint64
	Command    string
}

// Details carries extended information about a single process.
type Details struct {
	Process
	Executable  string
	CWD         string
	Nice        int32
	Threads     int32
	FDs         int32
	StartedUnix int64
}

// Inspect fetches extended information about pid. Only called for a
// single process on demand, so the heavyweight library path is fine.
func Inspect(pid int32) (Details, error) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return Details{}, err
	}
	d := Details{Process: Process{PID: pid}}
	if n, err := p.Name(); err == nil {
		d.Name = n
	}
	if u, err := p.Username(); err == nil {
		d.User = u
	}
	if st, err := p.Status(); err == nil && len(st) > 0 {
		d.State = st[0]
	}
	if pp, err := p.Ppid(); err == nil {
		d.PPID = pp
	}
	if mi, err := p.MemoryInfo(); err == nil {
		d.RSS = mi.RSS
	}
	if mp, err := p.MemoryPercent(); err == nil {
		d.MemPercent = float64(mp)
	}
	if cp, err := p.CPUPercent(); err == nil {
		d.CPUPercent = cp
	}
	if cl, err := p.Cmdline(); err == nil {
		d.Command = cl
	} else {
		d.Command = d.Name
	}
	if v, err := p.Exe(); err == nil {
		d.Executable = v
	}
	if v, err := p.Cwd(); err == nil {
		d.CWD = v
	}
	if v, err := p.Nice(); err == nil {
		d.Nice = v
	}
	if v, err := p.NumThreads(); err == nil {
		d.Threads = v
	}
	if v, err := p.NumFDs(); err == nil {
		d.FDs = v
	}
	if ct, err := p.CreateTime(); err == nil {
		d.StartedUnix = ct
	}
	return d, nil
}

// Signal sends a signal to a process.
func Signal(pid int32, sig syscall.Signal) error {
	p, err := os.FindProcess(int(pid))
	if err != nil {
		return err
	}
	return p.Signal(sig)
}

// SortKey identifies a sortable column of the process list.
type SortKey string

// Supported sort keys.
const (
	SortCPU SortKey = "cpu%"
	SortMem SortKey = "mem%"
	SortPID SortKey = "pid"
	SortNam SortKey = "name"
	SortUsr SortKey = "user"
)

// SortKeys is the cycle order used by the UI.
var SortKeys = []SortKey{SortCPU, SortMem, SortPID, SortNam, SortUsr}

// NextSortKey returns the key following cur in SortKeys.
func NextSortKey(cur SortKey) SortKey {
	for i, k := range SortKeys {
		if k == cur {
			return SortKeys[(i+1)%len(SortKeys)]
		}
	}
	return SortKeys[0]
}

// Sort sorts processes by key descending except pid/name which are
// ascending, matching common expectations.
func Sort(procs []Process, key SortKey) {
	sort.SliceStable(procs, func(i, j int) bool {
		a, b := procs[i], procs[j]
		switch key {
		case SortMem:
			return a.MemPercent > b.MemPercent
		case SortPID:
			return a.PID < b.PID
		case SortNam:
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		case SortUsr:
			return a.User < b.User
		default: // cpu%
			return a.CPUPercent > b.CPUPercent
		}
	})
}

// Filter keeps processes whose name, user, command or PID matches q
// case-insensitively. An empty query keeps everything.
func Filter(procs []Process, q string) []Process {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return procs
	}
	out := make([]Process, 0, len(procs))
	for _, p := range procs {
		if strings.Contains(strings.ToLower(p.Name), q) ||
			strings.Contains(strings.ToLower(p.User), q) ||
			strings.Contains(strings.ToLower(p.Command), q) ||
			strconv.Itoa(int(p.PID)) == q {
			out = append(out, p)
		}
	}
	return out
}

// FormatAge renders a start time relative to now.
func FormatAge(startedUnixMs int64) string {
	if startedUnixMs == 0 {
		return "?"
	}
	d := time.Since(time.Unix(startedUnixMs/1000, 0))
	switch {
	case d >= 24*time.Hour:
		return fmt.Sprintf("%dd%dh", int(d.Hours()/24), int(d.Hours())%24)
	case d >= time.Hour:
		return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
	default:
		return fmt.Sprintf("%02d:%02d", int(d.Minutes()), int(d.Seconds())%60)
	}
}

// --- uid -> username memoization shared by the /proc scanner ---

var (
	uidMu    sync.Mutex
	uidCache = map[uint32]string{}
)

func userName(uid uint32) string {
	uidMu.Lock()
	defer uidMu.Unlock()
	if n, ok := uidCache[uid]; ok {
		return n
	}
	n := "#" + strconv.Itoa(int(uid))
	if u, err := user.LookupId(strconv.Itoa(int(uid))); err == nil {
		n = u.Username
	}
	uidCache[uid] = n
	return n
}
