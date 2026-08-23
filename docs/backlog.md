# Backlog

Problems noticed but not fixed. Append the moment something is noticed;
fixing is optional, recording is not. Every entry carries a code pointer.

ID scheme: `C/H/M/L<n>` = critical / high / medium / polish. State:
`open` → `partial` → `closed (commit <sha>)`. Discovered: 2026-08-23 full-app audit unless noted.

## Critical

### C1 · closed (5ba6d67)
Same-directory paste zeroes the source file: `Copy()` opens dst with
`O_TRUNC` where dst == src.
Ptr: `internal/files/ops.go:127-129`; regression `internal/files/ops_test.go` `TestCopyOntoItselfRefused`.

### C2 · closed (fix(screens) commit, 2026-08-23)
Typing in a table filter triggered screen actions (`d` delete, `x/K` kill,
`s/t/r/e/D` service ops, `enter` navigates instead of committing).
Fix: every screen's `handleKey` delegates to the active FilterTable while
`Filtering()` (ADR-0004 Accepted). Guards in `processes.go`, `files.go`,
`services.go`, `disks.go` (top of handleKey), `users.go`.
Regression: `internal/screens/filter_guard_test.go`.

### C3 · closed (fix(screens) commit, 2026-08-23)
Services confirm dialog never cleared after "yes" → modal trapped input.
Fix: `s.confirm = nil` on the yes branch and in `svcActionDoneMsg`
(`internal/screens/services.go`). Test: `filter_guard_test.go::TestServicesConfirmDismissedOnActionStart`.

## High

### H1 · open
Cancelled disk scan returns partial result as complete (ctx.Err swallowed).
Ptr: `internal/disk/scan.go:69-71`.

### H2 · open
Refresh tick chains multiply or die on section switches: root re-calls
`Init()` per switch, ticks swallowed by inactive screens.
Ptrs: `internal/app/root.go:132-139`, `overview.go:83-91`, `processes.go:94-100`.

### H3 · open
Data race: `refreshTotalMem()` writes `memTotal` unlocked; readers hold mutex.
Ptr: `internal/proc/procfs.go:149-173`.

### H4 · open
File ops (copy/move/delete) have no busy state, progress or in-flight guard;
repeated `p` stacks concurrent copies.
Ptr: `internal/screens/files.go:319-331`; working pattern to reuse: `disks.go:217-234`.

### H5 · open
Silent overwrite: `os.Rename` and `copyFile` clobber existing targets.
Ptrs: `internal/files/ops.go:115-117,167`.

### H6 · open
Copying a directory into its own descendant recurses unbounded (no cycle check).
Ptr: `internal/files/ops.go:141-146`.

## Medium

### M1 · open
`wordWrap` slices bytes, corrupts UTF-8 cmdlines. Ptr: `internal/screens/processes.go:393-395`.

### M2 · open
Detail-pane keys differ per screen: proc `esc/enter/q`; services swallows `q`
(`services.go:193-202`); users lets `q` quit app (`users.go:90-92`); files needs `esc`.

### M3 · open
Cells not clipped to fitted column widths → misalignment on narrow terminals.
Ptrs: `internal/screens/processes.go:280`, `internal/ui/table.go:68-98`.

### M4 · open
Disks footer shows two conflicting `enter` hints ("select" + "analyze").
Ptr: `internal/screens/disks.go:84-89`.

### M5 · open
`x` (cut) missing from Files hints/footer/help — invisible feature.
Ptr: `internal/screens/files.go:70-81`.

### M6 · open
Services `SizeMsg` branch reimplements `layout()` with different clamp.
Ptr: `internal/screens/services.go:100-103` vs `254-257`.

### M7 · open
List headers show totals while filter active; no filtered/total count.
Ptrs: all screens' head renderers (e.g. `services.go:264`).

### M8 · open
Comment claims case-insensitive sort; code is byte-order.
Ptr: `internal/files/ops.go:62,92-97`.

### M9 · open
`Chown` passes username to `LookupId` (wants UID); unused broken API.
Ptr: `internal/files/ops.go:184-199`.

### M10 · open
Error toasts live 3 s like success and lack context ("permission denied" — for what?).
Ptrs: `internal/app/root.go:101-107`, `files.go:321-323`.

### M11 · open
Unreadable scan root yields "0 entries · 0 B" instead of an error.
Ptr: `internal/disk/scan.go:46-48`.

### M12 · open
Dead code: `ui.ErrorDialog/NewError`, `files.Chown`, `disk.LargestFiles`,
`ui.SelectedRow()`, `proc.Filter`, `ui.toastTickMsg` (shadowed by root's own).
Ptrs: `internal/ui/dialog.go:70-129`, `theme.go:93-95`, `messages.go:32` vs `app/root.go:370`.

## Polish

### L1 · open
Rounding/truncation nits: `f1` truncates (99.96→"99.9"), Gauge label truncates,
one wasted label column. Ptrs: `overview.go:273-276`, `meter.go:15,35`, `format.go:37-44`.

### L2 · open
Help panel claims global j/k/h/l that do nothing on Overview.
Ptr: `internal/app/root.go:324-329`.

### L3 · open
Symlink rows show lstat metadata except when Info() errors — inconsistent.
Ptr: `internal/files/ops.go:75-89`.

### L4 · open
Hardcoded dark palette; poor contrast under NO_COLOR/light terminals.
Ptr: `internal/ui/theme.go:22-33`.

### L5 · open
Network sparkline fixed scale 1 MiB/s saturates on fast links.
Ptr: `internal/screens/overview.go:105-106`.

### L6 · open
No `--version`/`--help`; no mouse support; `time.Tick` leak.
Ptrs: `cmd/lcc2/main.go`, `internal/proc/procfs.go:170`.
