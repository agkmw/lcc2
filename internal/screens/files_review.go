package screens

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	tea "charm.land/bubbletea/v2"

	"lcc2/internal/files"
	"lcc2/internal/ui"
)

// reviewPane is the staged-change list behind `w`: every queued op
// with old -> new detail, scrollable, shown before anything touches
// disk. It replaces the old apply-immediately save.
type reviewPane struct {
	top int // first visible op line
}

func (f Files) openReview() (ui.Screen, tea.Cmd) {
	if f.stager.Len() == 0 {
		return f, nil
	}
	f.review = &reviewPane{}
	return f, nil
}

func (f Files) handleReviewKeys(m tea.KeyMsg) (ui.Screen, tea.Cmd) {
	switch m.String() {
	case "n", "esc":
		f.review = nil
		return f, nil
	case "y", "enter":
		f.review = nil
		return f, f.startSave()
	case "u":
		if _, ok := f.stager.Undo(); !ok {
			f.review = nil
			return f, nil
		}
		f.syncTable()
		if f.stager.Len() == 0 {
			f.review = nil
		}
		return f, ui.InfoToast("undid last staged op")
	case "U":
		n := f.stager.Len()
		f.stager.Clear()
		f.review = nil
		f.syncTable()
		return f, ui.InfoToast(fmt.Sprintf("discarded %d change%s", n, plural(n)))
	case "j", "down":
		f.review.top++
	case "k", "up":
		if f.review.top > 0 {
			f.review.top--
		}
	}
	return f, nil
}

// reviewLine renders one op with its before -> after values resolved
// against the on-disk listing.
func reviewLine(f Files, op files.Op) string {
	switch op.Kind {
	case files.OpChmod:
		newMode := files.ParsePermBits(op.Mode).Octal()
		if e, ok := f.entryByPath(op.Path); ok {
			old := files.ParsePermBits(e.Mode).Octal()
			return fmt.Sprintf("chmod  %s -> %s  %s", old, newMode, filepathBase(op.Path))
		}
		return fmt.Sprintf("chmod  -> %s  %s", newMode, filepathBase(op.Path))
	case files.OpChown:
		if e, ok := f.entryByPath(op.Path); ok {
			old := files.UserName(e.UID) + ":" + files.GroupName(e.GID)
			return fmt.Sprintf("chown  %s -> %s  %s", old, op.Arg, filepathBase(op.Path))
		}
		return fmt.Sprintf("chown  -> %s  %s", op.Arg, filepathBase(op.Path))
	default:
		return op.Label()
	}
}

func reviewView(f Files) string {
	ops := f.stager.Ops()
	bodyH := clampInt(f.h-8, 6, 22)
	maxTop := len(ops) - bodyH
	if maxTop < 0 {
		maxTop = 0
	}
	top := f.review.top
	if top > maxTop {
		top = maxTop
	}
	f.review.top = top

	var lines []string
	head := lipgloss.NewStyle().Bold(true).Foreground(ui.Accent("files")).
		Render("files") + faintSty.Render(fmt.Sprintf("  %d staged change%s", len(ops), plural(len(ops))))
	lines = append(lines, head, "")
	end := top + bodyH
	if end > len(ops) {
		end = len(ops)
	}
	for i := top; i < end; i++ {
		lines = append(lines, "  "+ui.Truncate(reviewLine(f, ops[i]), f.w-10))
	}
	if len(ops) == 0 {
		lines = append(lines, faintSty.Render("  (nothing staged)"))
	}
	lines = append(lines, "")
	foot := faintSty.Render("[y]") + " save all   " +
		faintSty.Render("[u]") + " undo last   " +
		faintSty.Render("[U]") + " discard all   " +
		faintSty.Render("[j/k]") + " scroll   " +
		faintSty.Render("[esc]") + " back"
	lines = append(lines, foot)
	if top > 0 || end < len(ops) {
		lines = append(lines, faintSty.Render(fmt.Sprintf("showing %d-%d of %d", top+1, end, len(ops))))
	}
	return ui.Panel().BorderForeground(ui.Accent("files")).
		Width(clampInt(f.w-8, 50, 80)).
		Padding(1, 2).
		Render(strings.Join(lines, "\n"))
}
