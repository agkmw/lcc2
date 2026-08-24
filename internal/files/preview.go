package files

import (
	"bytes"
	"os"
)

// Preview holds bounded content for the preview pane.
type Preview struct {
	Lines     []string
	Binary    bool // NUL byte seen: show metadata instead
	Truncated bool
	Size      int64
}

// ReadPreview returns the head of a text file, bounded by maxLines and
// maxBytes. Files containing NUL bytes in the sampled window are
// reported as Binary so the caller can fall back to metadata.
func ReadPreview(path string, maxLines, maxBytes int) (Preview, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Preview{}, err
	}
	if info.IsDir() {
		return Preview{}, errNotFile
	}
	f, err := os.Open(path)
	if err != nil {
		return Preview{}, err
	}
	defer f.Close()

	buf := make([]byte, maxBytes)
	n, err := f.Read(buf)
	if err != nil && n == 0 {
		return Preview{}, err
	}
	chunk := buf[:n]
	p := Preview{Size: info.Size()}
	if bytes.IndexByte(chunk, 0) >= 0 {
		p.Binary = true
		return p, nil
	}
	lines := splitLines(string(chunk))
	if len(lines) > maxLines || int64(n) == int64(maxBytes) && info.Size() > int64(maxBytes) {
		p.Truncated = true
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	p.Lines = lines
	return p, nil
}

var errNotFile = previewError("not a regular file")

type previewError string

func (e previewError) Error() string { return string(e) }

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			end := i
			if end > start && s[end-1] == '\r' {
				end--
			}
			out = append(out, s[start:end])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	if len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return out
}
