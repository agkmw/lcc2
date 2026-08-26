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

// A file fully inside the window is NOT truncated — the flag used to
// fire on every such file because newline bytes were never counted.
func TestReadPreviewCompleteNotFlagged(t *testing.T) {
	p := filepath.Join(t.TempDir(), "small.txt")
	os.WriteFile(p, []byte("hello\nworld\n"), 0644)

	prev, err := ReadPreview(p, 10, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if len(prev.Lines) != 2 || prev.Truncated {
		t.Fatalf("lines=%d truncated=%v, want 2/false", len(prev.Lines), prev.Truncated)
	}
}

// Hitting maxLines only counts as truncation when more lines follow.
func TestReadPreviewExactLineCount(t *testing.T) {
	p := filepath.Join(t.TempDir(), "exact.txt")
	os.WriteFile(p, []byte(strings.Repeat("line\n", 5)), 0644)

	prev, err := ReadPreview(p, 5, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if len(prev.Lines) != 5 || prev.Truncated {
		t.Fatalf("lines=%d truncated=%v, want 5/false", len(prev.Lines), prev.Truncated)
	}
}

// The byte cap still flags truncation when it bites mid-file.
func TestReadPreviewByteCapFlagsTruncated(t *testing.T) {
	p := filepath.Join(t.TempDir(), "wide.txt")
	os.WriteFile(p, []byte(strings.Repeat("x", 100)+"\n"+strings.Repeat("y", 100)+"\n"), 0644)

	prev, err := ReadPreview(p, 10, 150)
	if err != nil {
		t.Fatal(err)
	}
	if !prev.Truncated || len(prev.Lines) == 0 {
		t.Fatalf("lines=%d truncated=%v, want >0/true", len(prev.Lines), prev.Truncated)
	}
}
