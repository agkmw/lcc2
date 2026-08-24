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

### T2 · closed (glyph-ban commit, ADR-0010)
Services preview bled into the table; dashboard bars drifted; files
rows wandered. Root cause: EAW-Ambiguous glyphs (● ▸ ▾ · › …) count as
1 cell in lipgloss but render 2 in tmux/CJK locales. Fixed via ui.Narrow
sanitizer + ASCII chrome + denylist test (`internal/app/glyph_test.go`).
Discovery recorded in `docs/experiments.md`.

### T3 · closed (canvas paint-order commit, ADR-0010)
Help overlay left a transparent strip right of the panel on transparent
terminals: the splice dropped the remainder of each line after the
panel. Canvas now paints last, guaranteeing full coverage; help is a
dim backdrop (#11111B) with no card.

### T4 · closed (ClearScreen commit, ADR-0010)
tmux split/zoom ghosted frames from inactive screens. Any dimension
change now batches tea.ClearScreen; regression gauntlet
`ghosting_test.go` requires byte-equality with fresh renders across
resize/tab sequences.

### T5 · closed (files structural pass commit)
Files screen was unreadable: cramped marker column, cryptic glyphs, no
pane titles. Marks folded into name cells, breadcrumb pane header added,
staged-ops legend appears whenever ops are queued, previews numbered/
indented.

## 2026-08-24 UI overhaul audit (user verdict: "the app is shit")

Batch-discovered while rebuilding the dashboard btop-style and the list
screens yazi-style. All closed same session unless noted.

### C4 · closed
Phantom green "+" staged-create glyph on EVERY Files row:
`stagedAt[path]` on unstaged rows returned the zero OpKind (= OpMkdir).
Ptr: `internal/screens/files.go` syncTable; regression
`files_glyph_test.go::TestUnstagedRowsCarryNoGlyph`.

### H7 · closed
Resize corrupted the live frame: bubbles/table re-truncates cells by
RUNE count (`runewidth.Truncate`, v1.0.0 table.go:435), ANSI escapes
count as runes; shrinking refits columns and sliced styled cells
mid-escape -> orphaned `\x1b[` fragments garbled the terminal. Fix:
`clipCells` now enforces display-width AND rune-count fit, styled
overflow falls back to plain text (`internal/ui/table.go` `fitCell`).
Regression: `table_test.go::TestStyledCellsSurviveShrink` +
app-level `resize_gauntlet_test.go`. User directive: verify via such
automated gauntlets, not tmux eyeballing.

### U4 · closed
Confirm/prompt/perm modals replaced the whole screen with a centered
panel on an empty background — total context loss mid-action. Fix:
`overlayCenter` splices panels over the live body
(`internal/screens/pane.go`); services confirm uses a neutral band for
non-destructive actions (`ui.ConfirmDialog.Danger`).

### U5 · closed
Users screen "tab switches lists" was dead: root pins tab to section
cycling. Rebound to `s`; regression `users_switch_test.go`.

### U6 · closed
Breadcrumb rendered "/ / tmp / x" for absolute paths
(`crumbMeta` double slash). Rewritten head+last form; hardcoded
#FAB387 switched to Palette.Peach.

### M13 · closed
Files table sized one row too tall for its own pane header: position
strip / last data row clipped every frame; staged legend made it two.
Legend deleted (glyph meanings live in `?` help), sizing fixed
(h-2/h-3 per chrome rows). Stacked split rebalanced 3/5 vs 2/5.

### M14 · closed
App violated its own ADR-0010 in chrome: `●` badge, `·` stats sep,
`…` filter placeholder. All ASCII now; denylist hole closed by scanning
fed-data + filter states (`TestTableChromeGlyphClean`) and files badge.

### M15 · closed
Processes preview said "select a process.." while a row was selected,
waiting on async inspect. Preview now renders instantly from the list
row (`processCard`), enriches when Details arrive. Filler preview
bodies ("j/k move...", "select a unit..") removed across screens;
blank panes instead.

### M16 · closed
Status-bar clock froze on tick-less screens; root schedules its own
minute tick. Notify window caption assumed 6-cell width. Dead code
(titleCase) removed; `x` cut restored to Files hints.

### R1 · closed (min-size gate commit)
No floor below which the UI renders garbage: `contentArea` clamped to
10x4. Now MinW=64 MinH=16; below shows an exactly-frame-sized notice
(ADR-0011). Resize broadcasts SizeMsg to ALL screens (no stale
inactive geometry). Gauntlet: `TestResizeCyclesKeepFrame`,
`TestResizeBelowFloorShowsNotice`.

### L8 · closed (rolling peak)
netPeak was monotonic per session; now max of a 60-sample rolling
window (`peakWin`). Regression `TestOverviewNetPeakDecays`.

### L9 · closed (tab-strip degradation)
Narrow strips degrade by priority — badges off, numbers off, inactive
labels shrink; active label last. `TestTabStripDegradesWithinWidth`.

### T6 · open
bubbles renderRow also emits "…" (banned glyph) whenever IT truncates;
our invariant makes that unreachable for FilterTable rows, but any
future direct use of bubbles tables re-exposes it. Consider vendoring
or replacing bubbles table long-term.
Ptr: bubbles v1.0.0 table.go:422,435.

### T7 · open
fd/rg integration assumes POSIX vimgrep parsing (SplitN on ':');
paths containing ':' would mis-parse. Fine on this codebase's Linux
scope; revisit if portability ever matters. Ptr: `internal/files/grep.go`.

## 2026-08-24 second live-pass batch (user feedback after overhaul)

### U7 · closed
Dashboard box tops were w-1 cells: the ┐ sat one column left of every
body row (fill = iw-3-H-1 instead of iw-2-H-S). Fixed in ui.Section;
`TestSectionRectangle` pins widths and corner columns.
Ptrs: `internal/ui/section.go`, `section_test.go`.

### U8 · closed
Canvas PaintBlock replaced EVERY reset with reset+canvas-bg,
erasing intentional backgrounds (selection highlight, dialog bands).
Replaced with an SGR state machine: spans push/pop, pops re-synthesize
the enclosing style, base level resumes the canvas fill
(`internal/ui/canvas.go`). Enables SelectedRow bold-on-Surface and the
active-tab chip.

### N2 · closed
Net dashboard graphs plotted pctOfScaled(currentRate) — ONE point per
tick; rxHist/txHist existed but were never wired in. Now plots the
byte-rate history normalized against the scale. Scale basis gains
hysteresis: grows instantly with the rolling peak, steps down only
after it drops below ¼ of the basis (floor 64 KiB/s) — no more
continuous shrinking under ordinary load (user report). Tests:
`TestOverviewNetHistoryPlotted`, `TestOverviewNetPeakHysteresis`,
`TestOverviewNetPeakDecays`.
Ptr: `internal/screens/overview.go`.

### X1 · closed
Help overlay was borderless text on dim backdrop — read as stray
content. Now a bordered card (ADR-0010 no-card clause superseded for
help only). Ptr: `app/root.go helpPanel`.

### X2 · closed
Aux search stole j/k/g/G for navigation, making such queries
impossible; arrows-only now steer results, letters type literally.
rg also ran smart-case ('Todo' matched nothing) — now always -i, plus
a 'searching..' indicator. Ptrs: `files_aux.go handleAuxKey`,
`files/grep.go`.

### X3 · closed
Directory traversal had no forward history: h went up, but you could
not return to where you left. ctrl+o / ctrl+i walk back/forward
stacks; entering dirs pushes, new descent clears forward.
Tests: `TestDirectoryBackForwardHistory`. Ptr: `files.go navigate`.

### X4 · closed
Active tab indicated only by bold+accent — too subtle. Now an
inverted chip (surface bg, accent text), enabled by U8's painter.
Ptr: `app/root.go viewTabStrip`.

## 2026-08-25 polish pass

### U9 · closed
Net charts sat behind a 6-column gutter, narrower by the rate suffix
— misaligned with the cpu chart's full-width plot. Direction labels
moved to their own lines (rate right-aligned); both plots now span
the whole box interior with identical edges. Budget base h-14-cRows.
Test: `TestNetBoxAlignedLikeCpu`. Ptr: `internal/screens/overview.go`.

### S2 · closed
Remaining plain table cells gained semantic tone: files symlinks teal
with "@" tail (`entryNameCell`, shared with dir previews), disks scan
dirs accent-bold, kernel-thread commands dimmed, failed service units
red-tinted names, group names class-toned like users.
Ptrs: files/disks/processes/services/users syncTable paths.

## 2026-08-25 third pass (zsh-style density)

### S3 · closed
Files: archives/distributions red, executables green, size units
dimmed (`4.0` `MB`), permission triples fully colored (r teal,
w yellow, x green). Services: unit-type suffix dimmed. Users: missing
home dirs red (os.Stat check at row build). Processes: args after the
executable faint; mem% own thresholds 40/70.
Ptrs: files.go entryNameCell/sizeCell, styles.go modeCell,
services.go unitCell, users.go homeCell, processes.go cmdCell/memPctCell.

### D1 · closed
Dashboard density extras, each gated to w>80: mem rows lead stats
with `avail`, page head carries platform/version + kernel, cpu box
grows a procs/threads/running strip (new `proc.ReadCounts()` sampling
/proc/loadavg + numeric /proc dirs), per-core columns justify across
the interior. Narrow layouts unchanged; memLine's fit-check still
drops right-hand stats that cannot fit.
Test: `TestOverviewWideExtrasGate`, `TestReadCounts`.
Ptrs: overview.go headMeta/cpuBox/coreSeps/memBox, proc/counts.go.

## Open items carried from earlier

### L3 · open

