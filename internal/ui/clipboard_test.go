package ui

import (
	"os"
	"path/filepath"
	"testing"
)

// A stub wl-copy on PATH wins over every other channel and receives
// the text on stdin.
func TestCopyTextPrefersWlCopy(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "wl-copy")
	os.WriteFile(script, []byte("#!/bin/sh\nexit 0\n"), 0o755)
	t.Setenv("PATH", dir)

	res := CopyText("hello path")
	if res.Err != nil {
		t.Fatalf("err: %v", res.Err)
	}
	if res.Channel != "wl-copy" {
		t.Fatalf("channel = %q", res.Channel)
	}
}

// With no helper binaries, the native tea.SetClipboard OSC52 command
// is the last resort.
func TestCopyTextFallsBackToNative(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)

	res := CopyText("ssh text")
	if res.Err != nil || res.Channel != "osc52" {
		t.Fatalf("res = %+v", res)
	}
	if res.Cmd == nil {
		t.Fatal("native clipboard cmd missing")
	}
}
