package proc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadCounts(t *testing.T) {
	c := ReadCounts()
	if c.Processes <= 0 {
		t.Fatalf("processes = %d, want > 0", c.Processes)
	}
	if c.Running < 0 || c.Threads < c.Running {
		t.Fatalf("inconsistent counts: %+v", c)
	}
}

// The /proc location must come from the package's overridable seam so
// fixtures can drive the parser without a real /proc.
func TestReadCountsUsesProcDirSeam(t *testing.T) {
	dir := t.TempDir()
	old := procDir
	procDir = dir
	t.Cleanup(func() { procDir = old })

	os.WriteFile(filepath.Join(dir, "loadavg"),
		[]byte("0.10 0.20 0.30 3/4321 9999\n"), 0644)
	for _, pid := range []string{"11", "22", "33", "notapid"} {
		os.MkdirAll(filepath.Join(dir, pid), 0755)
	}

	c := ReadCounts()
	if c.Processes != 3 || c.Running != 3 || c.Threads != 4321 {
		t.Fatalf("counts = %+v, want procs=3 running=3 threads=4321", c)
	}
}
