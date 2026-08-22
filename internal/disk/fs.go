// Package disk provides filesystem statistics and directory size
// analysis. Expensive scans run in the caller's goroutine (a tea.Cmd)
// and can be cancelled through context.
package disk

import (
	"sort"

	"github.com/shirou/gopsutil/v3/disk"
)

// Filesystem describes one mounted filesystem.
type Filesystem struct {
	Device      string
	Mountpoint  string
	FSType      string
	Total       uint64
	Used        uint64
	Free        uint64
	UsedPercent float64
}

// virtualFS types that add noise rather than insight.
var virtualFS = map[string]bool{
	"tmpfs": true, "devtmpfs": true, "squashfs": true,
	"proc": true, "sysfs": true, "devpts": true, "securityfs": true,
	"cgroup": true, "cgroup2": true, "pstore": true, "bpf": true,
	"debugfs": true, "tracefs": true, "configfs": true, "fusectl": true,
	"hugetlbfs": true, "efivarfs": true, "mqueue": true, "ramfs": true,
	"overlay": true,
}

// ListFilesystems returns mounted real filesystems sorted by mountpoint.
// Virtual pseudo-filesystems are filtered out.
func ListFilesystems() []Filesystem {
	parts, err := disk.Partitions(false)
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var out []Filesystem
	for _, p := range parts {
		if p.Device == "" || p.Device == "none" || seen[p.Mountpoint] {
			continue
		}
		if virtualFS[p.Fstype] {
			continue
		}
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}
		seen[p.Mountpoint] = true
		out = append(out, Filesystem{
			Device:      p.Device,
			Mountpoint:  p.Mountpoint,
			FSType:      p.Fstype,
			Total:       usage.Total,
			Used:        usage.Used,
			Free:        usage.Free,
			UsedPercent: usage.UsedPercent,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Mountpoint < out[j].Mountpoint })
	return out
}

// RootUsage returns usage of the root filesystem, if present.
func RootUsage() (Filesystem, bool) {
	fss := ListFilesystems()
	for _, f := range fss {
		if f.Mountpoint == "/" {
			return f, true
		}
	}
	return Filesystem{}, false
}
