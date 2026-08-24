package files

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadPreviewTextTruncation(t *testing.T) {
	p := filepath.Join(t.TempDir(), "t.txt")
	content := strings.Repeat("line\n", 100)
	os.WriteFile(p, []byte(content), 0644)

	prev, err := ReadPreview(p, 10, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if len(prev.Lines) != 10 || !prev.Truncated {
		t.Fatalf("lines=%d truncated=%v, want 10/true", len(prev.Lines), prev.Truncated)
	}
}

func TestReadPreviewBinaryDetected(t *testing.T) {
	p := filepath.Join(t.TempDir(), "b.bin")
	os.WriteFile(p, []byte("ok\x00binary"), 0644)

	prev, err := ReadPreview(p, 10, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if !prev.Binary {
		t.Fatal("NUL byte not detected as binary")
	}
}

func TestReadPreviewMissing(t *testing.T) {
	if _, err := ReadPreview(filepath.Join(t.TempDir(), "nope"), 5, 100); err == nil {
		t.Fatal("expected error for missing file")
	}
}
