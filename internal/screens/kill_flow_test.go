package screens

import (
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"lcc/internal/proc"
)

// The full terminate gesture must actually signal: x opens the
// confirm, y sends it, and the victim dies. Regression: the x/K call
// sites discarded askSignal's returned screen, so the dialog never
// opened and terminate/force-kill did nothing at all.
func TestTerminateFlowSignals(t *testing.T) {
	victim := exec.Command("sleep", "300")
	if err := victim.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- victim.Wait() }() // no Release: reap to detect exit
	pid := int32(victim.Process.Pid)
	time.Sleep(50 * time.Millisecond) // let it settle in /proc

	p := NewProcesses()
	p.all = []proc.Process{{PID: pid, Name: "sleep"}}
	p.syncTable()

	sc, _ := p.Update(keyRunes("x"))
	p = sc.(Processes)
	if p.confirm == nil {
		t.Fatal("x did not open the confirm dialog")
	}

	sc, cmd := p.Update(keyRunes("y"))
	p = sc.(Processes)
	if cmd == nil {
		t.Fatal("y did not produce the signal command")
	}
	msg, ok := cmd().(killDoneMsg)
	if !ok || msg.err != nil {
		t.Fatalf("signal cmd result = %v (ok=%v), want clean killDoneMsg", msg, ok)
	}
	sc, _ = p.Update(msg)
	p = sc.(Processes)
	if p.confirm != nil {
		t.Fatal("dialog stayed open after the signal")
	}

	select {
	case <-done: // process exited: terminated
	case <-time.After(2 * time.Second):
		t.Fatal("victim still alive 2s after terminate")
	}
}

// Force kill (shift-k) stages SIGKILL with its own confirm.
func TestForceKillOpensConfirm(t *testing.T) {
	p := NewProcesses()
	p.all = []proc.Process{{PID: 4242, Name: "sleeper"}}
	p.syncTable()

	sc, _ := p.Update(keyRunes("K"))
	p = sc.(Processes)
	if p.confirm == nil {
		t.Fatal("K did not open the confirm dialog")
	}
	if !strings.Contains(p.confirm.Body, "SIGKILL") {
		t.Fatalf("confirm body = %q, want SIGKILL", p.confirm.Body)
	}
	if p.pendingSig != syscall.SIGKILL {
		t.Fatalf("pending signal = %v", p.pendingSig)
	}
}
