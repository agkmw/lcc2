package proc

import "testing"

func TestReadCounts(t *testing.T) {
	c := ReadCounts()
	if c.Processes <= 0 {
		t.Fatalf("processes = %d, want > 0", c.Processes)
	}
	if c.Running < 0 || c.Threads < c.Running {
		t.Fatalf("inconsistent counts: %+v", c)
	}
}
