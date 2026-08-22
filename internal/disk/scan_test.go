package disk

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestScanDir(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, size int) {
		if err := os.WriteFile(filepath.Join(dir, name), make([]byte, size), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("big.bin", 4096)
	write("small.txt", 2)
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "inner.dat"), make([]byte, 1000), 0644); err != nil {
		t.Fatal(err)
	}

	res, err := ScanDir(context.Background(), dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.TotalSize != 4096+2+1000 {
		t.Fatalf("total = %d, want %d", res.TotalSize, 4096+2+1000)
	}
	var sub *Item
	for i := range res.Items {
		it := res.Items[i]
		if it.Name == "sub" {
			sub = &res.Items[i]
		}
		if it.Name == "big.bin" && it.Size != 4096 {
			t.Fatalf("big.bin size = %d", it.Size)
		}
	}
	if sub == nil || !sub.IsDir || sub.Size != 1000 {
		t.Fatalf("sub missing or wrong: %+v", sub)
	}
}

func TestScanDirCancel(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// Cancellation yields a best-effort partial result, not an error.
	res, err := ScanDir(ctx, dir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatal("expected partial result")
	}
}
