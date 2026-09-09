package screens

import (
	"testing"

	"lcc/internal/proc"
)

func TestDebugKillFlow(t *testing.T) {
	p := NewProcesses()
	p.all = []proc.Process{{PID: 4242, Name: "sleep"}}
	p.syncTable()

	k := keyRunes("x")
	t.Logf("key=%q", k.String())

	sc, cmd := p.Update(k)
	p2 := sc.(Processes)
	t.Logf("after x: confirm=%v cmd=%v", p2.confirm != nil, cmd != nil)

	sc, cmd = p2.handleKey(k)
	p3 := sc.(Processes)
	t.Logf("direct handleKey: confirm=%v cmd=%v", p3.confirm != nil, cmd != nil)

	idx, ok := p.tbl.Selected()
	t.Logf("selected idx=%v ok=%v len(all)=%d", idx, ok, len(p.all))
}
