package services

import (
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Detail is the structured facts of one unit, sampled via
// `systemctl show`. Zero values mean "unknown" and render as "-".
type Detail struct {
	Active    string // "active", "failed", ...
	Sub       string // "running", "exited", ...
	Boot      string // "enabled", "static", ...
	PID       int32
	MemBytes  uint64
	CPUNanos  uint64
	Since     time.Time
	Restarts  int
	Timestamp string // raw ExecMainStartTimestamp, shown dimmed
}

// showProps are the properties Show requests; keep in sync with
// parseShow.
var showProps = []string{
	"ActiveState", "SubState", "UnitFileState", "MainPID",
	"MemoryCurrent", "CPUUsageNSec", "ExecMainStartTimestamp",
	"NRestarts",
}

// Show samples structured unit facts with one systemctl call.
func Show(unit string) (Detail, error) {
	if !Available() {
		return Detail{}, ErrUnavailable
	}
	args := []string{"show", unit, "--no-pager"}
	for _, p := range showProps {
		args = append(args, "-p", p)
	}
	out, err := run(args...)
	if err != nil && !strings.Contains(out, "=") {
		return Detail{}, wrapErr(err, out)
	}
	return parseShow(out), nil
}

func parseShow(out string) Detail {
	kv := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		if i := strings.IndexByte(line, '='); i > 0 {
			kv[line[:i]] = strings.TrimSpace(line[i+1:])
		}
	}
	var d Detail
	d.Active = kv["ActiveState"]
	d.Sub = kv["SubState"]
	d.Boot = kv["UnitFileState"]
	if pid, err := strconv.Atoi(kv["MainPID"]); err == nil && pid > 0 {
		d.PID = int32(pid)
	}
	d.MemBytes = parseUint(kv["MemoryCurrent"])
	d.CPUNanos = parseUint(kv["CPUUsageNSec"])
	d.Restarts, _ = strconv.Atoi(kv["NRestarts"])
	d.Timestamp = kv["ExecMainStartTimestamp"]
	const layout = "Mon 2006-01-02 15:04:05 -0700"
	if t, err := time.Parse(layout, d.Timestamp); err == nil {
		d.Since = t
	}
	return d
}

func parseUint(s string) uint64 {
	s = strings.TrimPrefix(s, "\"")
	s = strings.TrimSuffix(s, "\"")
	n, _ := strconv.ParseUint(s, 10, 64)
	return n
}

// Journal returns the last n log lines of a unit, or nothing when
// journalctl is unavailable — never a hard failure for the UI.
func Journal(unit string, n int) []string {
	bin, err := exec.LookPath("journalctl")
	if err != nil {
		return nil
	}
	out, err := exec.Command(bin, "-u", unit, "-n", strconv.Itoa(n),
		"--no-pager", "-o", "cat").Output()
	if err != nil && len(out) == 0 {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	res := lines[:0]
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			res = append(res, l)
		}
	}
	return res
}
