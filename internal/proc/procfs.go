package proc

import (
	"bytes"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// procDir is the mount point scanned for processes (a var for tests).
var procDir = "/proc"

// clockTicks is USER_HZ on Linux; used to convert CPU ticks to seconds.
const clockTicks = 100

// Collector computes per-process CPU% from the delta between two
// snapshots, like top does. Snapshots read /proc directly with two
// small file reads per process instead of going through a heavyweight
// library, keeping refresh cost close to `ps`.
type Collector struct {
	last  map[int32][2]uint64 // pid -> utime, stime ticks
	t     time.Time
	first bool
}

// NewCollector returns a ready-to-use collector.
func NewCollector() *Collector {
	return &Collector{last: map[int32][2]uint64{}, first: true}
}

// Snapshot lists all userland processes. The first call reports CPU
// as 0; subsequent calls report usage since the previous call.
func (c *Collector) Snapshot() ([]Process, error) {
	now := time.Now()
	dt := now.Sub(c.t).Seconds()
	if c.first || dt <= 0 {
		dt = 0
	}

	dirEntries, err := os.ReadDir(procDir)
	if err != nil {
		return nil, err
	}
	totalMem := totalMemOnce()

	cur := make(map[int32][2]uint64, len(dirEntries))
	out := make([]Process, 0, len(dirEntries))
	for _, de := range dirEvents(dirEntries) {
		pid := de.pid
		raw, err := os.ReadFile(procDir + "/" + de.name + "/stat")
		if err != nil {
			continue // raced with exit or permission denied
		}
		pr, ticks, ok := parseStat(raw, pid)
		if !ok {
			continue
		}
		if pr.PPID == 2 {
			continue // kernel threads add noise, not insight
		}
		if uid := readUID(pid); uid != noUID {
			pr.User = userName(uid)
		} else {
			pr.User = "-"
		}
		if totalMem > 0 {
			pr.MemPercent = float64(pr.RSS) / float64(totalMem) * 100
		}
		cur[pid] = ticks
		if prev, ok := c.last[pid]; ok && dt > 0 {
			dU := int64(ticks[0]) - int64(prev[0])
			dS := int64(ticks[1]) - int64(prev[1])
			if dU < 0 {
				dU = 0
			}
			if dS < 0 {
				dS = 0
			}
			pr.CPUPercent = (float64(dU+dS) / clockTicks / dt) * 100
		}
		out = append(out, pr)
	}

	c.last = cur
	c.t = now
	c.first = false
	return out, nil
}

type pidEntry struct {
	name string
	pid  int32
}

func dirEvents(entries []os.DirEntry) []pidEntry {
	out := make([]pidEntry, 0, len(entries))
	for _, de := range entries {
		n := de.Name()
		if pid64, err := strconv.ParseInt(n, 10, 32); err == nil && pid64 > 0 {
			out = append(out, pidEntry{name: n, pid: int32(pid64)})
		}
	}
	return out
}

// parseStat parses /proc/<pid>/stat: "pid (comm) state ppid ...".
// Field offsets are relative to the LAST ')' because comm may contain
// spaces and parentheses. After it: state(0) ppid(1) … utime(11)
// stime(12) … rss pages(21).
func parseStat(raw []byte, pid int32) (Process, [2]uint64, bool) {
	open := bytes.IndexByte(raw, '(')
	close := bytes.LastIndexByte(raw, ')')
	if open < 0 || close <= open || close+2 >= len(raw) {
		return Process{}, [2]uint64{}, false
	}
	var pr Process
	pr.PID = pid
	pr.Name = string(raw[open+1 : close])
	rest := strings.Fields(string(raw[close+2:]))
	if len(rest) < 22 {
		return Process{}, [2]uint64{}, false
	}
	pr.State = rest[0]
	if pp, err := strconv.Atoi(rest[1]); err == nil {
		pr.PPID = int32(pp)
	}
	u, _ := strconv.ParseUint(rest[11], 10, 64)
	s, _ := strconv.ParseUint(rest[12], 10, 64)
	rssPages, _ := strconv.ParseUint(rest[21], 10, 64)
	pr.RSS = rssPages * uint64(os.Getpagesize())
	pr.Command = pr.Name // full cmdline is fetched on demand in Inspect
	return pr, [2]uint64{u, s}, true
}

var (
	memOnceMu sync.Mutex
	memTotal  uint64
)

// totalMemOnce reads /proc/meminfo at most every 30s.
func totalMemOnce() uint64 {
	memOnceMu.Lock()
	defer memOnceMu.Unlock()
	return memTotal
}

func init() { refreshTotalMem(); go memRefresher() }

func refreshTotalMem() {
	b, err := os.ReadFile(procDir + "/meminfo")
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			f := strings.Fields(line)
			if len(f) >= 2 {
				if kb, err := strconv.ParseUint(f[1], 10, 64); err == nil {
					memTotal = kb * 1024
				}
			}
			return
		}
	}
}

func memRefresher() {
	for range time.Tick(30 * time.Second) {
		refreshTotalMem()
	}
}

// readUID returns the real uid of a pid from /proc/<pid>/status.
func readUID(pid int32) uint32 {
	b, err := os.ReadFile(procDir + "/" + strconv.Itoa(int(pid)) + "/status")
	if err != nil {
		return noUID
	}
	return parseUIDLine(b)
}

const noUID = ^uint32(0)

func parseUIDLine(b []byte) uint32 {
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "Uid:") {
			f := strings.Fields(strings.TrimPrefix(line, "Uid:"))
			if len(f) >= 1 {
				if uid, err := strconv.ParseUint(f[0], 10, 32); err == nil {
					return uint32(uid)
				}
			}
			break
		}
	}
	return noUID
}
