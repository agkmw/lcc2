package files

import (
	"bufio"
	"os"
	"strings"
)

// Preview holds bounded content for the preview pane.
type Preview struct {
	Lines     []string
	First     int  // 1-based line number of Lines[0]
	Binary    bool // NUL byte seen: show metadata instead
	Truncated bool
	Size      int64
}

// ReadPreview returns the head of a text file, bounded by maxLines and
// maxBytes. Files containing NUL bytes in the sampled window are
// reported as Binary so the caller can fall back to metadata.
func ReadPreview(path string, maxLines, maxBytes int) (Preview, error) {
	return readWindow(path, 1, maxLines, maxBytes)
}

// ReadPreviewAt returns a window of at most maxLines lines centered on
// targetLine (1-based), so previews can jump to a search hit.
func ReadPreviewAt(path string, targetLine, maxLines, maxBytes int) (Preview, error) {
	return readWindow(path, targetLine-maxLines/2, maxLines, maxBytes)
}

// readWindow samples the file around firstLine (1-based; clamped).
func readWindow(path string, firstLine, maxLines, maxBytes int) (Preview, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Preview{}, err
	}
	if info.IsDir() {
		return Preview{}, errNotFile
	}
	if firstLine < 1 {
		firstLine = 1
	}
	f, err := os.Open(path)
	if err != nil {
		return Preview{}, err
	}
	defer f.Close()

	p := Preview{Size: info.Size(), First: firstLine}
	var all []string
	consumed := 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64<<10), 512<<10)

	// Truncated must mean "the window cut something off", decided by
	// why the read stopped — never by byte-size arithmetic (newline
	// accounting made every fully-read file claim truncation).
	truncated := false
	if skip := firstLine - 1; skip > 0 {
		for i := 0; i < skip && sc.Scan(); i++ { // skip to window start
		}
		truncated = true // window starts mid-file
	}
	byteCapped := false
	for len(all) < maxLines && sc.Scan() {
		l := sc.Text()
		if consumed+len(l) > maxBytes {
			byteCapped = true
			break
		}
		if strings.IndexByte(l, 0) >= 0 {
			p.Binary = true
			return p, nil
		}
		all = append(all, strings.ReplaceAll(l, "\t", "  ")) // tabs break cell math
		consumed += len(l)
	}
	if len(all) == maxLines && sc.Scan() { // probe: another line exists
		truncated = true
	}
	p.Lines = all
	p.Truncated = truncated || byteCapped || sc.Err() != nil
	return p, nil
}

var errNotFile = previewError("not a regular file")

type previewError string

func (e previewError) Error() string { return string(e) }
