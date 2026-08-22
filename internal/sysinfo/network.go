package sysinfo

import (
	"time"

	"github.com/shirou/gopsutil/v3/net"
)

// NetRates holds network throughput in bytes per second plus totals.
type NetRates struct {
	RecvPerSec float64
	SentPerSec float64
	RecvTotal  uint64
	SentTotal  uint64
}

// NetMonitor computes rates from the delta between successive calls.
// The zero value is ready to use; the first Rates call returns zeroes.
type NetMonitor struct {
	last    net.IOCountersStat
	t       time.Time
	hasLast bool
}

// Rates measures aggregate network throughput since the previous call.
func (m *NetMonitor) Rates() NetRates {
	stats, err := net.IOCounters(false)
	out := NetRates{}
	if err != nil || len(stats) == 0 {
		return out
	}
	cur := stats[0]
	defer func() {
		m.last = cur
		m.t = time.Now()
		m.hasLast = true
	}()
	if !m.hasLast {
		out.RecvTotal, out.SentTotal = cur.BytesRecv, cur.BytesSent
		return out
	}
	dt := time.Since(m.t).Seconds()
	if dt <= 0 {
		dt = 1
	}
	dRecv := float64(cur.BytesRecv) - float64(m.last.BytesRecv)
	dSent := float64(cur.BytesSent) - float64(m.last.BytesSent)
	if dRecv < 0 || dSent < 0 { // counters reset (reboot, iface change)
		out.RecvTotal, out.SentTotal = cur.BytesRecv, cur.BytesSent
		return out
	}
	out.RecvPerSec = dRecv / dt
	out.SentPerSec = dSent / dt
	out.RecvTotal = cur.BytesRecv
	out.SentTotal = cur.BytesSent
	return out
}
