package files

import (
	"bufio"
	"context"
	"os/exec"
	"strconv"
	"strings"
)

// Match is one rg hit: the file, 1-based line, column and text.
type Match struct {
	Path string
	Line int
	Col  int
	Text string
}

// Grep searches file contents under root with ripgrep (smart-case,
// vimgrep output). Hidden files follow the flag; results stop at max,
// killing the search process.
func Grep(ctx context.Context, root, query string, hidden bool, max int) ([]Match, error) {
	if strings.TrimSpace(query) == "" {
		return []Match{}, nil
	}
	bin, err := findTool("rg")
	if err != nil {
		return nil, err
	}
	args := []string{"--vimgrep", "--smart-case", "--no-heading",
		"--color", "never", "--max-columns", "300"}
	if hidden {
		args = append(args, "--hidden")
	}
	args = append(args, "--", query, root)
	cmd := exec.CommandContext(ctx, bin, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var matches []Match
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 0, 64<<10), 512<<10)
	for sc.Scan() {
		if m, ok := parseVimgrep(sc.Text()); ok {
			matches = append(matches, m)
			if len(matches) >= max {
				break
			}
		}
	}
	if cmd.Process != nil {
		_ = cmd.Process.Kill() // no-op once rg exited on its own
	}
	_ = cmd.Wait()
	return matches, nil
}

func parseVimgrep(line string) (Match, bool) {
	parts := strings.SplitN(line, ":", 4)
	if len(parts) < 4 {
		return Match{}, false
	}
	ln, err1 := strconv.Atoi(parts[1])
	col, err2 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || ln < 1 {
		return Match{}, false
	}
	return Match{Path: parts[0], Line: ln, Col: col,
		Text: strings.ReplaceAll(parts[3], "\t", "  ")}, true
}
