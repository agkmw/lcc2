# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

UI polish pass: commits 1-2 done (design system, chrome). Commit 3/4
done: full-line row selection, per-table position line (`n/m · pct%`,
closes M7 visually), bordered filter row, process state/% tinting,
service state dots, disk use% thresholds, titled overview panels,
dir glyph in Files. Gate green.

## In progress

Commit 4/4: modals & overlays polish.

## Next action

Finish commit 4 (`feat(ui): modals polish`): confirm dialog danger band +
button row (`internal/ui/dialog.go`), help overlay keycap chips
(`internal/app/root.go` helpPanel), notification kind-word caption
(`internal/ui/notify.go`). Gate → ritual → commit → final STATUS sweep.
