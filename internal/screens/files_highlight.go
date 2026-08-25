package screens

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"lcc2/internal/ui"
)

// highlightCode syntax-highlights a text sample for languages chroma
// recognizes. It returns nil whenever highlighting does not apply —
// unknown language, NO_COLOR, or a non-truecolor terminal — and the
// caller falls back to plain numbered lines. Output line count always
// equals the input's.
func highlightCode(name string, lines []string) []string {
	if len(lines) == 0 || os.Getenv("NO_COLOR") != "" {
		return nil
	}
	if lipgloss.DefaultRenderer().ColorProfile() != termenv.TrueColor {
		return nil
	}
	lexer := lexers.Match(name)
	if lexer == nil {
		return nil
	}
	style := styles.Get("catppuccin-mocha")
	if style == nil {
		style = styles.Fallback
	}
	formatter := formatters.Get("terminal16m")
	if formatter == nil {
		return nil
	}
	it, err := lexer.Tokenise(nil, strings.Join(lines, "\n"))
	if err != nil {
		return nil
	}
	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, it); err != nil {
		return nil
	}
	out := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(out) != len(lines) {
		return nil
	}
	for i := range out { // trailing resets per line are pure noise here
		out[i] = strings.TrimSuffix(out[i], "\x1b[0m")
		out[i] = ui.Narrow(out[i])
	}
	return out
}

// previewBodyForFile assembles highlighted-or-plain numbered lines.
func previewBodyForFile(name string, lines []string, first, hit int, truncated bool) string {
	body := lines
	if hl := highlightCode(name, lines); hl != nil {
		body = hl
	}
	rendered := numberLines(body, first, hit)
	if truncated {
		rendered += "\n" + faintSty.Render(".. truncated")
	}
	return rendered
}

var _ = filepath.Join // keep path import anchored if gates change
