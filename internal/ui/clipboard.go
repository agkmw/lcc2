package ui

import (
	"encoding/base64"
	"os"
	"os/exec"
	"strings"
)

// CopyResult reports which channel carried the text, for toasts.
type CopyResult struct {
	Channel string // "wl-copy", "xclip", "osc52"
	Err     error
}

// CopyText places text on the system clipboard: Wayland first, then
// X11, then the OSC52 escape sequence written straight to the tty
// (works over ssh; tmux needs `set -g set-clipboard on`). OSC52 is
// consumed silently by the terminal, so it cannot corrupt the frame.
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
	if err := osc52(text); err != nil {
		return CopyResult{Err: err}
	}
	return CopyResult{Channel: "osc52"}
}

// osc52 emits the base64 clipboard sequence on the controlling tty,
// falling back to stdout.
func osc52(text string) error {
	seq := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(text)) + "\a"
	if f, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0); err == nil {
		defer f.Close()
		_, err = f.WriteString(seq)
		return err
	}
	_, err := os.Stdout.WriteString(seq)
	return err
}
