package sysinfo

import (
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
)

// CPUSample holds instantaneous CPU usage percentages (0-100).
type CPUSample struct {
	Cores   int
	PerCore []float64
	Total   float64
}

// SampleCPU measures usage across a short window. A zero interval
// uses the diff since the last call, which is what the dashboard
// does on every refresh tick.
func SampleCPU(interval time.Duration) (CPUSample, error) {
	perCore, err := cpu.Percent(interval, true)
	if err != nil {
		return CPUSample{}, err
	}
	total, err := cpu.Percent(0, false)
	if err != nil {
		return CPUSample{}, err
	}
	t := 0.0
	if len(total) > 0 {
		t = total[0]
	}
	return CPUSample{
		Cores:   len(perCore),
		PerCore: perCore,
		Total:   t,
	}, nil
}

// Counts returns physical and logical core counts.
func Counts() (physical, logical int, err error) {
	logical, err = cpu.Counts(true)
	if err != nil {
		return 0, 0, err
	}
	physical, err = cpu.Counts(false)
	return physical, logical, err
}
