package sysinfo

import (
	"regexp"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/disk"
)

// DiskIO is aggregate read/write throughput across all block devices.
type DiskIO struct {
	ReadPerSec  float64
	WritePerSec float64
	ReadTotal   uint64
	WriteTotal  uint64
}

// IOMonitor diffs gopsutil disk counters between calls, mirroring
// NetMonitor's contract (reset-tolerant, ~1s cadence).
type IOMonitor struct {
	last    map[string]disk.IOCountersStat
	t       time.Time
	hasLast bool
}

// Rates samples aggregate disk I/O since the previous call.
func (m *IOMonitor) Rates() DiskIO {
	counters, err := disk.IOCounters()
	out := DiskIO{}
	if err != nil || len(counters) == 0 {
		return out
	}
	curRead, curWrite := sumAll(counters)
	defer func() {
		m.last = counters
		m.t = time.Now()
		m.hasLast = true
	}()
	if !m.hasLast {
		out.ReadTotal, out.WriteTotal = curRead, curWrite
		return out
	}
	dt := time.Since(m.t).Seconds()
	if dt <= 0 {
		dt = 1
	}
	var dRead, dWrite int64
	for name, c := range counters {
		if !usableDevice(name) {
			continue
		}
		prev, ok := m.last[name]
		if !ok { // device appeared mid-window; count from zero
			dRead += int64(c.ReadBytes)
			dWrite += int64(c.WriteBytes)
			continue
		}
		dRead += int64(c.ReadBytes) - int64(prev.ReadBytes)
		dWrite += int64(c.WriteBytes) - int64(prev.WriteBytes)
	}
	if dRead < 0 || dWrite < 0 { // counters reset (reboot, device swap)
		out.ReadTotal, out.WriteTotal = curRead, curWrite
		return out
	}
	out.ReadPerSec = float64(dRead) / dt
	out.WritePerSec = float64(dWrite) / dt
	out.ReadTotal, out.WriteTotal = curRead, curWrite
	return out
}

// partitionRe matches partition-shaped device names whose bytes the
// kernel also reports on their parent (sda1 → sda): summing both
// double-counts real throughput.
var partitionRe = regexp.MustCompile(
	`^((sd|hd|vd|xvd)[a-z]+[0-9]+|nvme[0-9]+n[0-9]+p[0-9]+|mmcblk[0-9]+p[0-9]+)$`)

// usableDevice filters out loop/ram devices and partition children:
// noise or duplicates for a usage dashboard.
func usableDevice(name string) bool {
	if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") ||
		strings.HasPrefix(name, "zram") {
		return false
	}
	return !partitionRe.MatchString(name)
}

// sumAll totals read/write bytes across usable counter entries.
func sumAll(counters map[string]disk.IOCountersStat) (r, w uint64) {
	for name, c := range counters {
		if !usableDevice(name) {
			continue
		}
		r += c.ReadBytes
		w += c.WriteBytes
	}
	return r, w
}
