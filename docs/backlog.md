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

### H6 · closed (5f2e20f, amended in staged-files commit)
Directory copied into its own subtree recursed unbounded.
Fix: `nestingErr` prefix check in `internal/files/ops.go`.
Test: `ops_test.go::TestCopyIntoOwnSubtreeRefused`.
Amendment (2026-08-24): `nestingErr` inverted its `rel == "."` case, so
copying a dir into *itself* (`Copy(sub, sub)`) slipped through and still
recursed. Logic rewritten; caught by `stage_test.go::TestStageValidation`.

## Medium

### M1 · closed (polish-batch commit)
`wordWrap` slices bytes, corrupts UTF-8 cmdlines. Ptr: `internal/screens/processes.go:393-395`.

### M2 · closed (canvas redesign, ADR-0009)
Detail-pane keys differed per screen (proc `esc/enter/q`; services
swallowed `q`; users let `q` through). Detail modals are gone entirely:
previews are persistent panes that never capture input; `esc` only
cancels transient states, `q` always quits.

### M3 · closed (canvas redesign, ADR-0009)
`FilterTable.SetRowsTracked`/`SetRows` now clip every cell to its
fitted column width centrally (`internal/ui/table.go` `clipCells`).

### M4 · closed (polish-batch commit)
Disks footer shows two conflicting `enter` hints ("select" + "analyze").
Ptr: `internal/screens/disks.go:84-89`.

### M5 · closed (polish-batch commit)
`x` (cut) missing from Files hints/footer/help — invisible feature.
Ptr: `internal/screens/files.go:70-81`.

### M6 · closed (polish-batch commit)
Services `SizeMsg` branch reimplements `layout()` with different clamp.
Ptr: `internal/screens/services.go:100-103` vs `254-257`.

### M7 · closed (polish-batch + tables commits)
Filtered counts and scroll position now shown per table via the reserved
position line (`internal/ui/table.go` View).

### M8 · closed (polish-batch commit)
Comment claims case-insensitive sort; code is byte-order.
Ptr: `internal/files/ops.go:62,92-97`.

### M9 · closed (polish-batch commit)
Broken unused `Chown` API resolved by removal — see [M12].

### M10 · closed (polish-batch commit)
Error toasts live 3 s like success and lack context ("permission denied" — for what?).
Ptrs: `internal/app/root.go:101-107`, `files.go:321-323`.

### M11 · closed (fix(disk) commit, 2026-08-23)
Unreadable scan root yielded silent "0 entries · 0 B" instead of an error.
Root walk error is now fatal (`internal/disk/scan.go`); UI shows analyze error.
Test: `internal/disk/scan_test.go::TestScanDirUnreadableRoot`.

### M12 · partial
Dead code removed: `ui.ErrorDialog/NewError`, `files.Chown` (broken API),
`disk.LargestFiles`, `ui.SelectedRow`, `ui.toastTickMsg`.
Kept intentionally: `proc.Filter` (tested provider helper).
Still open: [M9] note folded here — Chown removed rather than fixed.

### L7 · open
File ops have no byte progress or cancel (coarse busy guard only, see H4).
Ptrs: `internal/screens/files.go`, pattern `internal/disk/scan.go`.

### N1 · closed (feat(ui) notify commit)

### S1 · closed (feat(screens) panes commit)
Detail/meta panes used fixed or full-height boxes regardless of content.
Fix: `paneHeight(contentLines, avail)` helper (`internal/screens/styles.go`)
applied to proc detail viewport, files meta pane, services and users
detail panes; services content now wraps instead of single-line truncation.

### U1 · closed (fix(screens) split commit, 2026-08-24)
Detail panes overflowed their width budget at common sizes and bled
through the app frame — read as "screens rendered over each other"
(svc-detail@76: 81>76; users-detail@84: 89>84; files-meta similar).
Fix: `ui.Split` exact-width columns (`internal/ui/split.go`); all four
detail modes on per-screen `detailGeom()`; `joinPanes` deleted.
Regression: `internal/screens/detail_fit_test.go` (5 sizes × 4 screens).

### U2 · closed (fix(screens) split commit, 2026-08-24)
Ghosting guard: chrome well-formedness pinned — every section view must
be exactly h lines and ≤ w cells at 4 window sizes, plus modal states.
Test: `internal/app/chrome_wellformed_test.go`.

### U3 · closed (ui shell commits, 2026-08-24)
Post-polish user verdict: still messy — mixed border corners, frame-in-frame
nesting, chip-heavy nav, floating rule, off-center empty states.
Fix: full app-shell redesign per ADR-0008 (square-only borders, nav/status
bars, rail with border, pageHead pattern, overview card grid, strip-below,
true centering). Renderer hazards found & fixed en route:
- `Style.Render` pads every line to widest input → whole-frame inflation
  when one line overflowed (status-bar hints); now clipped post-render
  (`internal/app/root.go`) + hints always budget-clamped
- hand-rolled rune Truncate sliced ANSI escapes → replaced with `x/ansi`
  (`internal/ui/theme.go`)
- `JoinHorizontal` width normalization swallowed gutters → overview grid
  assembled via exact-width `ui.Split`

### T1 · closed (feat(ui) tracking commit)
Cursor did not follow the selected item across refreshes/filters —
selection "jumped to top" after any rebuild.
Fix: `FilterTable.SetRowsTracked(rows, keys)` + key-based cursor restore
in every `applyFilter` path (`internal/ui/table.go`). Keys wired in all
five screens: files=path, processes=PID (syncTable PID hack deleted),
services=unit, users=name, disks=mountpoint/path.
Tests: `internal/ui/table_test.go` (delete-shift, filter narrow/widen),
`internal/screens/lifecycle_test.go::TestFilesCursorTracksAcrossRefresh`.

Toast redesign to nvim-notify style after user feedback: old overlay washed
the whole frame and truncated footer hints.
New: `internal/ui/notify.go` (NotifyStack + CompositeNotes), root wiring in
`internal/app/root.go`; ADR-0006.

## Polish

### L1 · closed (overview rebuild commit, canvas redesign)
`f1` truncation replaced with `strconv.FormatFloat` rounding; Gauge
label now rounds (`int(v+0.5)`); the wasted label column went away with
the old Gauge layout in the btop sections. SegGauge renders label-free
by design (value shown beside).

### L2 · closed (canvas chrome commit)
Help panel rewritten: globals are exactly what root binds
(tab/shift+tab, 1-6, j/k, /, enter, esc, ?, q). The phantom `h/l`
globals are gone.

### L3 · open
Symlink rows show lstat metadata except when Info() errors — inconsistent.
Ptr: `internal/files/ops.go:75-89`. (Entry.Link now captured for previews.)

### L4 · partial (canvas shell)
App paints its own opaque background (`ui.Canvas`) so transparent and
light terminals get a consistent surface; palette is still hardcoded
Catppuccin Mocha — no theme switching or NO_COLOR-specific palette.
Ptr: `internal/ui/theme.go`, `internal/ui/canvas.go`.

### L5 · closed (overview rebuild commit)
Network graphs auto-scale to the session peak with a 64 KiB/s floor;
scale label rendered in section title. Ptr: `internal/screens/overview.go`.

### L6 · open
No `--version`/`--help`; no mouse support; `time.Tick` leak.
Ptrs: `cmd/lcc2/main.go`, `internal/proc/procfs.go:170`.

### L7 · open
File saves have per-op progress + stop-on-error (staged pipeline), but
still no byte-level progress or cancel within a single big copy/move.
Ptrs: `internal/screens/files.go` `runStageStep`, pattern `internal/disk/scan.go`.

### L8 · open
Overview netPeak is monotonic per session — one huge transfer pins the
graph scale forever after. Needs decay or rolling window.
Ptr: `internal/screens/overview.go` `observe`.

### L9 · open
Tab strip truncates mid-segment below ~70 cols (ClipBlock cuts the
rightmost tabs first, no priority logic).
Ptr: `internal/app/root.go` `viewTabStrip`.
