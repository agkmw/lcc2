package ui

import (
	"strconv"
	"strings"
	"unicode/utf8"

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
// columns — with background color c.
//
// Inner styled spans are handled by a small SGR state machine with
// nesting: each non-reset SGR pushes a merged style state, every reset
// pops and RE-SYNTHESIZES the enclosing style. That is what keeps an
// outer background (selected table row, dialog band) intact across the
// resets of inner foreground-only spans, while spans that genuinely end
// fall back to the canvas fill — no transparent holes, no erased bands.
func PaintBlock(block string, w int, c lipgloss.Color) string {
	seq := bgSeqFor(c)
	lines := strings.Split(block, "\n")
	for i, l := range lines {
		l = clipLine(l, w)
		if seq != "" {
			// Painting may drop truncated escape fragments, shifting
			// the visible width — re-fit afterwards so every line is
			// exactly w cells.
			l = clipLine(paintSGR(l, seq), w)
		}
		lines[i] = l
	}
	return strings.Join(lines, "\n")
}

// sgrState is one level of active styling: SGR parameter fragments for
// what is set, "" meaning terminal default.
type sgrState struct {
	bold bool
	fg   string
	bg   string
}

func (s sgrState) synth() string {
	var parts []string
	if s.bold {
		parts = append(parts, "1")
	}
	if s.fg != "" {
		parts = append(parts, s.fg)
	}
	if s.bg != "" {
		parts = append(parts, s.bg)
	}
	if len(parts) == 0 {
		return ""
	}
	return "\x1b[" + strings.Join(parts, ";") + "m"
}

// applySGR merges raw SGR params into st; returns true when the
// sequence was a full reset (leading 0 or empty body).
func applySGR(st *sgrState, body string) bool {
	params := strings.Split(body, ";")
	reset := len(params) == 0 || params[0] == "0" || params[0] == ""
	if reset {
		*st = sgrState{}
		params = params[1:] // a leading 0 may be followed by more params
	}
	for k := 0; k < len(params); k++ {
		p := params[k]
		n := atoiSafe(p)
		switch {
		case p == "1":
			st.bold = true
		case p == "22":
			st.bold = false
		case n == 39:
			st.fg = ""
		case n == 49:
			st.bg = ""
		case (n >= 30 && n <= 37) || (n >= 90 && n <= 97):
			st.fg = p
		case (n >= 40 && n <= 47) || (n >= 100 && n <= 107):
			st.bg = p
		case n == 38 || n == 48:
			if k+1 >= len(params) {
				continue
			}
			switch params[k+1] {
			case "5":
				if k+2 < len(params) {
					frag := p + ";" + params[k+1] + ";" + params[k+2]
					if n == 38 {
						st.fg = frag
					} else {
						st.bg = frag
					}
					k += 2
				}
			case "2":
				if k+4 < len(params) {
					frag := p + ";" + params[k+1] + ";" + params[k+2] + ";" +
						params[k+3] + ";" + params[k+4]
					if n == 38 {
						st.fg = frag
					} else {
						st.bg = frag
					}
					k += 4
				}
			}
		}
	}
	return reset && len(params) == 0
}

func atoiSafe(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return -1
		}
		n = n*10 + int(s[i]-'0')
	}
	return n
}

func isFinalByte(b byte) bool { return b >= 0x40 && b <= 0x7e }

// paintSGR rewrites line so that: plain text carries canvasSeq; nested
// styled spans push/pop states; pops re-synthesize the enclosing style;
// returning to the base level resumes the canvas fill.
func paintSGR(line, canvasSeq string) string {
	var b strings.Builder
	b.WriteString(canvasSeq)
	stack := []sgrState{{}}
	i := 0
	for i < len(line) {
		if line[i] != 0x1b {
			_, sz := utf8.DecodeRuneInString(line[i:])
			b.WriteString(line[i : i+sz])
			i += sz
			continue
		}
		j := i + 1
		if j < len(line) && line[j] == '[' {
			j++
		}
		start := j
		for j < len(line) && !isFinalByte(line[j]) {
			j++
		}
		if j >= len(line) {
			break // truncated escape at EOL: drop it silently
		}
		final := line[j]
		body := line[start:j]
		if final != 'm' {
			b.WriteString(line[i : j+1]) // non-SGR escape: passthrough
			i = j + 1
			continue
		}
		top := stack[len(stack)-1]
		next := top
		wasReset := applySGR(&next, body)
		if wasReset {
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				stack = append(stack, sgrState{})
				b.WriteString("\x1b[0m" + canvasSeq)
			} else if synth := stack[len(stack)-1].synth(); synth != "" {
				b.WriteString(synth)
			} else {
				b.WriteString("\x1b[0m" + canvasSeq)
			}
		} else {
			// Merge onto parent only when the sequence actually sets
			// something; pure no-ops still push so their reset pops.
			stack = append(stack, next)
			b.WriteString(line[i : j+1])
		}
		i = j + 1
	}
	// Close any unclosed spans so padding/text after us stays on canvas.
	for len(stack) > 1 {
		stack = stack[:len(stack)-1]
	}
	b.WriteString("\x1b[0m" + canvasSeq)
	return b.String()
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
