package sysinfo

import "github.com/shirou/gopsutil/v3/mem"

// Memory reports RAM and swap usage.
type Memory struct {
	Total       uint64
	Used        uint64
	Available   uint64
	UsedPercent float64
	SwapTotal   uint64
	SwapUsed    uint64
	SwapPercent float64
}

// ReadMemory reads current memory statistics; failures degrade to
// an empty struct so the UI can show a friendly empty state.
func ReadMemory() Memory {
	m, err := mem.VirtualMemory()
	if err != nil {
		return Memory{}
	}
	out := Memory{
		Total:       m.Total,
		Used:        m.Used,
		Available:   m.Available,
		UsedPercent: m.UsedPercent,
	}
	if sw, err := mem.SwapMemory(); err == nil {
		out.SwapTotal = sw.Total
		out.SwapUsed = sw.Used
		out.SwapPercent = sw.UsedPercent
	}
	return out
}
