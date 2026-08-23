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

### H1 · closed (fix(disk) commit, 2026-08-23)
Cancelled disk scan returned partial results as complete.
Contract reversed: `ScanDir` now returns `ctx.Err()` on cancel
(`internal/disk/scan.go`); Disks ignores `context.Canceled` silently
(`internal/screens/disks.go`). Old partial-result behavior recorded in
git history (`internal/disk/scan_test.go` before this commit).

### H2 · closed (fix(screens) commit, 2026-08-23)
Refresh tick chains multiplied or died across section switches.
Fix: epoch-guarded chains — `Init` bumps a shared `*atomic.Uint64`; stale
ticks return nil (Overview, Processes). Spinner chains retire when idle
(Disks) or loaded (Services; spinner now actually animates via Init).
Tests: `internal/screens/lifecycle_test.go`.

### H3 · closed (5f2e20f)
Data race: `refreshTotalMem()` wrote `memTotal` unlocked while readers
held `memOnceMu`. Write side now locks (`internal/proc/procfs.go`).

### H4 · closed (fix(files) commit, 2026-08-23)
File ops had no busy state or in-flight guard; repeated `p` stacked copies.
Fix: `opCount` guard locks Files input while ops run; "working…" indicator
in head (`internal/screens/files.go`). Coarse (no byte progress, no cancel)
— progress/cancel tracked as enhancement, see L7.

### H5 · closed (fix(files) commit, 2026-08-23)
Silent overwrites via Rename/copyFile/Move. All three now refuse with
"%s already exists" (`internal/files/ops.go`).
Tests: `ops_test.go::TestOverwritesRefused`.

### H6 · closed (fix(files) commit, 2026-08-23)
Directory copied into its own subtree recursed unbounded.
Fix: `nestingErr` prefix check in `internal/files/ops.go`.
Test: `ops_test.go::TestCopyIntoOwnSubtreeRefused`.

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

### M11 · closed (fix(disk) commit, 2026-08-23)
Unreadable scan root yielded silent "0 entries · 0 B" instead of an error.
Root walk error is now fatal (`internal/disk/scan.go`); UI shows analyze error.
Test: `internal/disk/scan_test.go::TestScanDirUnreadableRoot`.

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
