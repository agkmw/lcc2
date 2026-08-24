// Package sysinfo provides point-in-time readings of general system
// information: host identity, load average, CPU, memory and network.
// All functions are safe to call repeatedly from tea.Cmd values.
package sysinfo

import (
	"time"

	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
)

// Host describes the machine itself.
type Host struct {
	Hostname        string
	OS              string // "linux"
	Platform        string // e.g. "ubuntu"
	PlatformVersion string
	KernelArch      string
	KernelVersion   string
	Uptime          time.Duration
}

// ReadHost gathers static-ish host information.
func ReadHost() (Host, error) {
	info, err := host.Info()
	if err != nil {
		return Host{}, err
	}
	return Host{
		Hostname:        info.Hostname,
		OS:              info.OS,
		Platform:        info.Platform,
		PlatformVersion: info.PlatformVersion,
		KernelArch:      info.KernelArch,
		KernelVersion:   info.KernelVersion,
		Uptime:          time.Duration(info.Uptime) * time.Second,
	}, nil
}

// Load holds the classic three load averages.
type Load struct {
	One, Five, Fifteen float64
}

// ReadLoad reads load averages; failures degrade to zeroes.
func ReadLoad() Load {
	l, err := load.Avg()
	if err != nil {
		return Load{}
	}
	return Load{One: l.Load1, Five: l.Load5, Fifteen: l.Load15}
}
