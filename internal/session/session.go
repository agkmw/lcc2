// Package session persists small cross-run preferences: the last
// active screen and the Files screen's working state. Corruption is
// never fatal — a missing or broken file just means defaults.
package session

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// State is the persisted snapshot.
type State struct {
	Screen   int    `json:"screen"`
	Cwd      string `json:"cwd,omitempty"`
	Hidden   bool   `json:"hidden"`
	SortKey  string `json:"sortKey"`
	SortDesc bool   `json:"sortDesc"`
}

// Path returns the state file location ($XDG_CONFIG_HOME/lcc2/...).
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "lcc2", "state.json"), nil
}

// Load reads the state file; any error yields defaults.
func Load() State {
	st := State{SortKey: "name"}
	p, err := Path()
	if err != nil {
		return st
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return st
	}
	var saved State
	if json.Unmarshal(data, &saved) != nil {
		return st
	}
	if saved.Screen < 0 {
		saved.Screen = 0
	}
	if saved.SortKey == "" {
		saved.SortKey = "name"
	}
	return saved
}

// Save writes the atomically (tmp + rename); errors are the caller's
// problem but never worth crashing over.
func Save(st State) error {
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(st, "", " ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}
