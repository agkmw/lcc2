package ui

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// BG returns the application canvas color. The app paints its own
// opaque background so it looks identical on every terminal, including
// transparent ones.
func BG() lipgloss.Color { return appBG }

// BGDim returns the dimmed canvas shade used behind modal overlays
// (help): everything recedes one step darker while the overlay floats.
func BGDim() lipgloss.Color { return dimBG }

var resetSeqRe = regexp.MustCompile("\x1b\\[0?m")

// bgSeqFor returns the SGR truecolor background sequence for c, or ""
// when the active color profile cannot render it.
func bgSeqFor(c lipgloss.Color) string {
	if lipgloss.DefaultRenderer().ColorProfile() == termenv.Ascii {
		return ""
	}
	hex := string(c)
	if len(hex) != 7 || hex[0] != '#' {
		return ""
	}
	r, err1 := strconv.ParseUint(hex[1:3], 16, 8)
	g, err2 := strconv.ParseUint(hex[3:5], 16, 8)
	b, err3 := strconv.ParseUint(hex[5:7], 16, 8)
	if err1 != nil || err2 != nil || err3 != nil {
		return ""
	}
	return "\x1b[48;2;" + strconv.FormatUint(r, 10) + ";" +
		strconv.FormatUint(g, 10) + ";" + strconv.FormatUint(b, 10) + "m"
}

// PaintBlock fills every cell of block — clipped and padded to w
// columns — with background color c. Inner ANSI resets are followed by
// a fresh background sequence so styled spans cannot punch transparent
// holes into the fill.
func PaintBlock(block string, w int, c lipgloss.Color) string {
	seq := bgSeqFor(c)
	lines := strings.Split(block, "\n")
	for i, l := range lines {
		l = clipLine(l, w)
		if seq != "" {
			if strings.Contains(l, "\x1b") {
				l = seq + resetSeqRe.ReplaceAllString(l, "\x1b[0m"+seq) + "\x1b[0m"
			} else {
				l = seq + l + "\x1b[0m"
			}
		}
		lines[i] = l
	}
	return strings.Join(lines, "\n")
}

// Canvas paints frame as the application's own opaque background,
// exactly w cells wide and h lines tall. It must be the last step of
// View: anything composited earlier that lacks an explicit fill ends
// up on the canvas instead of the terminal's own background. bg
// selects the shade (BG normally, BGDim behind modal overlays).
func CanvasWith(frame string, w, h int, bg lipgloss.Color) string {
	lines := strings.Split(frame, "\n")
	for len(lines) < h {
		lines = append(lines, "")
	}
	if len(lines) > h {
		lines = lines[:h]
	}
	return PaintBlock(strings.Join(lines, "\n"), w, bg)
}

// Canvas paints frame onto the standard app background.
func Canvas(frame string, w, h int) string {
	return CanvasWith(frame, w, h, appBG)
}
