// Package services inspects and controls systemd units by shelling
// out to systemctl. All command execution is isolated here.
package services

import (
	"errors"
	"os/exec"
	"strings"
)

// Unit is one systemd service as shown in the UI.
type Unit struct {
	Name        string // "sshd.service"
	Description string
	Load        string // "loaded", "not-found", …
	Active      string // "active", "inactive", "failed", …
	Sub         string // "running", "exited", …
	Enabled     string // "enabled", "disabled", "static", …
}

// ErrUnavailable reports that no service manager was found.
var ErrUnavailable = errors.New("systemctl not available on this system")

// Available reports whether systemctl exists on PATH.
func Available() bool {
	_, err := exec.LookPath("systemctl")
	return err == nil
}

func run(args ...string) (string, error) {
	cmd := exec.Command("systemctl", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// List enumerates all units and merges in their enablement state.
func List() ([]Unit, error) {
	if !Available() {
		return nil, ErrUnavailable
	}
	out, err := run("list-units", "--type=service", "--all", "--no-legend", "--no-pager")
	units := parseListUnits(out)
	if err != nil && len(units) == 0 {
		return nil, wrapErr(err, out)
	}
	if en, _ := run("list-unit-files", "--type=service", "--no-legend", "--no-pager"); en != "" {
		states := parseUnitFiles(en)
		for i := range units {
			if st, ok := states[units[i].Name]; ok {
				units[i].Enabled = st
			}
		}
	}
	return units, nil
}

// parseListUnits parses `systemctl list-units` output lines:
//
//	proc-sys-fs-binfmt_misc.automount loaded active running Arbitrary Executable File Formats File System Automount Point
func parseListUnits(out string) []Unit {
	var units []Unit
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 || !strings.HasSuffix(fields[0], ".service") {
			continue
		}
		u := Unit{
			Name:   fields[0],
			Load:   fields[1],
			Active: fields[2],
			Sub:    fields[3],
		}
		if idx := strings.Index(line, fields[3]); idx >= 0 {
			u.Description = strings.TrimSpace(line[idx+len(fields[3]):])
		}
		units = append(units, u)
	}
	return units
}

// parseUnitFiles maps unit name -> enablement state from
// `systemctl list-unit-files` output lines: "name.service state preset".
func parseUnitFiles(out string) map[string]string {
	m := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		m[fields[0]] = fields[1]
	}
	return m
}

// Actions supported by the UI.
var Actions = []string{"start", "stop", "restart", "enable", "disable"}

// Action performs a management action on a unit, returning the raw
// output so the UI can surface details on failure (e.g. permissions).
func Action(unit, action string) error {
	if !Available() {
		return ErrUnavailable
	}
	out, err := run(action, unit)
	if err != nil {
		return wrapErr(err, out)
	}
	return nil
}

// StatusDetail returns the full status text of a unit for the
// details pane.
func StatusDetail(unit string) string {
	out, err := run("status", unit, "--no-pager", "-l")
	if err != nil {
		if out == "" {
			return err.Error()
		}
	}
	return out
}

func wrapErr(err error, out string) error {
	msg := strings.TrimSpace(out)
	if msg == "" {
		return err
	}
	if i := strings.Index(msg, "\n"); i > 0 {
		msg = msg[:i]
	}
	return errors.New(msg)
}
