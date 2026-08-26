package proc

import (
	"os"
	"strconv"
	"strings"
)

// Counts is a point-in-time tally of system-wide process activity,
// cheap enough to sample every dashboard tick.
type Counts struct {
	Processes int // numeric entries under /proc
	Threads   int // scheduling entities (/proc/loadavg)
	Running   int // runnable entities
}

// ReadCounts samples process/thread/runnable counts from /proc.
func ReadCounts() Counts {
	var c Counts
	if data, err := os.ReadFile(procDir + "/loadavg"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) >= 4 {
			parts := strings.SplitN(fields[3], "/", 2)
			if len(parts) == 2 {
				c.Running, _ = strconv.Atoi(parts[0])
				c.Threads, _ = strconv.Atoi(parts[1])
			}
		}
	}
	if ents, err := os.ReadDir(procDir); err == nil {
		for _, e := range ents {
			if _, err := strconv.Atoi(e.Name()); err == nil {
				c.Processes++
			}
		}
	}
	return c
}
