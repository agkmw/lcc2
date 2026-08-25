package ui

import (
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// CopyResult reports which channel carried the text, for toasts.
type CopyResult struct {
	Channel string // "wl-copy", "xclip", "osc52"
	Cmd     tea.Cmd
	Err     error
}

// CopyText places text on the system clipboard: Wayland first, then
// X11, finally bubbletea's native OSC52 command (works over ssh;
// tmux needs `set -g set-clipboard on`). The Cmd must be returned to
// the program for the native path to run.
func CopyText(text string) CopyResult {
	for _, c := range []struct {
		bin  string
		args []string
		name string
	}{
		{"wl-copy", nil, "wl-copy"},
		{"xclip", []string{"-selection", "clipboard"}, "xclip"},
	} {
		if _, err := exec.LookPath(c.bin); err != nil {
			continue
		}
		cmd := exec.Command(c.bin, c.args...)
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err == nil {
			return CopyResult{Channel: c.name}
		}
	}
	return CopyResult{Channel: "osc52", Cmd: tea.SetClipboard(text)}
}
