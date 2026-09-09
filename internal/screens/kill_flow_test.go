package screens

import (
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"lcc/internal/proc"
	"lcc/internal/ui"
)

// spawnVictim starts a real sleep process and registers its kill as
// cleanup: an early t.Fatal must not leave the victim alive for the
// full 300 s.
func spawnVictim(t *testing.T) int32 {
	t.Helper()
	victim := exec.Command("sleep", "300")
	if err := victim.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = victim.Process.Kill() }) // no-op once dead
	go func() { _ = victim.Wait() }()               // reap to avoid zombie
	time.Sleep(50 * time.Millisecond)               // let it settle in /proc
	return int32(victim.Process.Pid)
}

// toastFrom digs the ui.ToastMsg out of a command, batching included.
func toastFrom(t *testing.T, cmd tea.Cmd) ui.ToastMsg {
	t.Helper()
	if cmd == nil {
		t.Fatal("want a command, got nil")
	}
	msgs := []tea.Msg{cmd()}
	if batch, ok := msgs[0].(tea.BatchMsg); ok {
		msgs = msgs[:0]
		for _, c := range batch {
			msgs = append(msgs, c())
		}
	}
	for _, m := range msgs {
		if tm, ok := m.(ui.ToastMsg); ok {
			return tm
		}
	}
	t.Fatalf("no ui.ToastMsg among %d message(s)", len(msgs))
	return ui.ToastMsg{}
}

// procGone polls until pid no longer exists in /proc (reaped); false
// if it survives the budget.
func procGone(t *testing.T, pid int32) bool {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if proc.Signal(pid, 0) != nil { // signal 0: existence probe
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

// The full terminate gesture must actually signal: x opens the
// confirm, y sends it, and the victim dies. Regression: the x/K call
// sites discarded askSignal's returned screen, so the dialog never
// opened and terminate/force-kill did nothing at all.
func TestTerminateFlowSignals(t *testing.T) {
	pid := spawnVictim(t)

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
	if p.pendingPID != 0 {
		t.Fatalf("pendingPID = %d after the kill completed", p.pendingPID)
	}
	if !procGone(t, pid) {
		t.Fatal("victim still alive 2s after terminate")
	}
}

// A second y while the first signal is in flight must not fire a
// second signalCmd: the dialog is cleared at confirmation time, not
// in killDoneMsg. Regression: ConfirmDialog re-answers y on every
// press, so double-y double-signalled.
func TestSecondConfirmDoesNotRefire(t *testing.T) {
	p := NewProcesses()
	p.all = []proc.Process{{PID: 4242, Name: "sleep"}}
	p.syncTable()

	sc, _ := p.Update(keyRunes("x"))
	p = sc.(Processes)
	sc, first := p.Update(keyRunes("y"))
	p = sc.(Processes)
	if first == nil {
		t.Fatal("first y did not dispatch the signal")
	}
	if _, ok := first().(killDoneMsg); !ok {
		t.Fatalf("first y produced %T, want killDoneMsg", first())
	}
	if p.confirm != nil {
		t.Fatal("dialog survived its own confirmation")
	}

	sc, second := p.Update(keyRunes("y"))
	p = sc.(Processes)
	if second != nil {
		t.Fatalf("second y produced %v, want nil (no second signal)", second())
	}
	if p.confirm != nil {
		t.Fatal("second y reopened a dialog")
	}
}

// killDoneMsg must match on pid: a stale completion (user already
// opened a newer confirm while the older signal was in flight) leaves
// the fresh dialog and pending state alone; a matching one clears the
// state and toasts the verb of ITS OWN signal.
func TestKillDoneMatchedByPID(t *testing.T) {
	p := NewProcesses()
	p.all = []proc.Process{{PID: 111, Name: "a"}, {PID: 222, Name: "b"}}
	p.syncTable()

	sc, _ := p.Update(keyRunes("x")) // ask terminate on 111
	p = sc.(Processes)
	sc, cmd := p.Update(keyRunes("y")) // confirm: signal 111 in flight
	p = sc.(Processes)
	if cmd == nil || p.confirm != nil || p.pendingPID != 111 {
		t.Fatal("confirming y did not dispatch cleanly")
	}

	sc, _ = p.Update(keyRunes("j")) // cursor onto 222
	p = sc.(Processes)
	sc, _ = p.Update(keyRunes("x")) // fresh dialog targeting 222
	p = sc.(Processes)
	if p.confirm == nil || p.pendingPID != 222 {
		t.Fatalf("fresh dialog: confirm=%v pendingPID=%d",
			p.confirm != nil, p.pendingPID)
	}

	// stale completion for 111 arrives while 222's dialog is open
	sc, staleCmd := p.Update(killDoneMsg{pid: 111, sig: syscall.SIGTERM})
	p = sc.(Processes)
	if staleCmd != nil {
		t.Fatal("stale killDoneMsg produced a command")
	}
	if p.confirm == nil || p.pendingPID != 222 {
		t.Fatal("stale killDoneMsg dismissed the fresh dialog")
	}

	// matching completion for a SIGKILL: verb from the message's signal
	sc, cmd = p.Update(killDoneMsg{pid: 222, sig: syscall.SIGKILL})
	p = sc.(Processes)
	if p.confirm != nil || p.pendingPID != 0 {
		t.Fatalf("matching msg did not clear state: confirm=%v pendingPID=%d",
			p.confirm != nil, p.pendingPID)
	}
	if got := toastFrom(t, cmd); got.Text != "killed 222" {
		t.Fatalf("toast = %q, want %q", got.Text, "killed 222")
	}
}

// The success verb comes from the signal the message carried:
// SIGTERM reports "terminated", SIGKILL "killed" - verified end to
// end against real victims so proc.Signal actually succeeds.
func TestKillDoneVerbsMatchSignal(t *testing.T) {
	confirmKill := func(t *testing.T, key string) (Processes, killDoneMsg) {
		t.Helper()
		p := NewProcesses()
		p.all = []proc.Process{{PID: spawnVictim(t), Name: "sleep"}}
		p.syncTable()
		sc, _ := p.Update(keyRunes(key))
		p = sc.(Processes)
		sc, cmd := p.Update(keyRunes("y"))
		p = sc.(Processes)
		done, ok := cmd().(killDoneMsg)
		if !ok || done.err != nil {
			t.Fatalf("%s signal result = %v (ok=%v), want clean killDoneMsg", key, done, ok)
		}
		return p, done
	}

	p, done := confirmKill(t, "x")
	_, cmd := p.Update(done)
	if got := toastFrom(t, cmd); got.Text != "terminated "+itoa(int(done.pid)) {
		t.Fatalf("terminate toast = %q", got.Text)
	}

	p2, done2 := confirmKill(t, "K")
	_, cmd2 := p2.Update(done2)
	if got := toastFrom(t, cmd2); got.Text != "killed "+itoa(int(done2.pid)) {
		t.Fatalf("kill toast = %q", got.Text)
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
