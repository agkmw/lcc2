package ui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Field describes one form input line.
type Field struct {
	Label       string
	Value       string // prefill
	Placeholder string
	Password    bool // masked input
	Width       int  // input width; 0 = form width minus label
}

// Form is a labeled multi-field modal for mutating actions (create
// user, chown, ...). Tab/up/down cycle focus, enter submits, esc
// cancels; every other key goes to the focused text input.
type Form struct {
	Title  string
	fields []formField
	cur    int
	width  int
}

type formField struct {
	label string
	input textinput.Model
}

// NewForm builds a form with the first field focused.
func NewForm(title, accent string, fields []Field) Form {
	f := Form{Title: title, width: 60}
	st := textinput.DefaultStyles(true)
	st.Focused.Prompt = lipgloss.NewStyle().Foreground(Accent(accent))
	st.Blurred.Prompt = st.Focused.Prompt
	st.Focused.Text = lipgloss.NewStyle().Foreground(Palette.Text)
	st.Blurred.Text = st.Focused.Text
	for i, spec := range fields {
		ti := textinput.New()
		ti.SetStyles(st)
		ti.Placeholder = spec.Placeholder
		if spec.Width > 0 {
			ti.SetWidth(spec.Width)
		}
		if spec.Password {
			ti.EchoMode = textinput.EchoPassword
			ti.EchoCharacter = '*'
		}
		ti.SetValue(spec.Value)
		if i == 0 {
			f.cur = 0
			ti.Focus()
		}
		f.fields = append(f.fields, formField{label: spec.Label, input: ti})
	}
	return f
}

// SetWidth sets the dialog width.
func (f *Form) SetWidth(w int) { f.width = w }

// Values returns the current value of each field, in order.
func (f Form) Values() []string {
	out := make([]string, len(f.fields))
	for i, fl := range f.fields {
		out[i] = strings.TrimSpace(fl.input.Value())
	}
	return out
}

// Update handles form keys; submits on enter, cancels on esc.
func (f *Form) Update(msg tea.Msg) (submitted, done bool, cmd tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return false, false, nil
	}
	switch key.String() {
	case "esc":
		return false, true, nil
	case "enter":
		return true, true, nil
	case "tab", "down":
		f.focus((f.cur + 1) % len(f.fields))
		return false, false, nil
	case "up", "shift+tab":
		f.focus((f.cur + len(f.fields) - 1) % len(f.fields))
		return false, false, nil
	}
	f.fields[f.cur].input, cmd = f.fields[f.cur].input.Update(msg)
	return false, false, cmd
}

// focus moves the cursor to field i, swapping focus styles.
func (f *Form) focus(i int) tea.Cmd {
	f.fields[f.cur].input.Blur()
	f.cur = i
	return f.fields[i].input.Focus()
}

// View renders the form as a floating panel.
func (f Form) View() string {
	inner := f.width - 4
	var b strings.Builder
	band := lipgloss.NewStyle().Bold(true).
		Background(Palette.Surface).
		Foreground(Palette.Text).
		Width(inner).
		Render(" " + f.Title)
	b.WriteString(band)
	b.WriteString("\n\n")
	for _, fl := range f.fields {
		lbl := mutedSty.Render(pad(fl.label, 12))
		b.WriteString(lbl)
		b.WriteString(fl.input.View())
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(KeyBadgeStyle(Palette.Text).Render("[enter]") + faintSty.Render(" submit   ") +
		KeyBadgeStyle(Palette.Text).Render("[tab]") + faintSty.Render(" next   ") +
		KeyBadgeStyle(Palette.Text).Render("[esc]") + faintSty.Render(" cancel"))
	return Panel().
		BorderForeground(Palette.Overlay).
		Width(f.width).
		Padding(1, 2).
		Render(b.String())
}

// pad right-pads s with spaces to width w (plain, no ANSI).
func pad(s string, w int) string {
	for len(s) < w {
		s += " "
	}
	return s
}
