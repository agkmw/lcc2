# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

- Trash + terminal-restore batch (G1-G4, 2026-09-10): SIGHUP now
  shuts down gracefully (terminal restores on SIGINT/SIGTERM/SIGHUP;
  SIGKILL is uncatchable); `R` inside the trash stages restore via
  .trashinfo records, `esc`/`h` return to the pre-trash directory,
  and previews show each entry's origin. Same-name trash collisions
  (`foobar.2`) are per freedesktop spec - origin now visible.
- Kill-flow review follow-up (2026-09-09): double-y no longer fires a
  second signal (dialog cleared at confirmation time); killDoneMsg
  carries its own signal and matches on pid, so a stale completion
  can neither dismiss a fresh confirm dialog nor steal the toast verb
  ("terminated" vs "killed" pinned by regression tests). Debug
  harness zz_kill_test.go deleted; lcc/lcc2 binaries untracked +
  ignored; test victims now killed via t.Cleanup. Recorded the gofmt
  gate gap (backlog G1: check.sh never runs gofmt, 9 files flagged).
- Terminate/kill fix (F1, 2026-09-09): x and K were silent no-ops
  (askSignal's dialog screen was discarded at both call sites - broken
  since the first commit). Dialog now opens, failed signals explain
  why (EPERM gets a sudo hint), and a successful kill refreshes the
  list immediately.
- Find UX batch (E1-E2, 2026-09-08): `enter` on a search result lands
  the cursor ON the entry in the target listing (directories still
  open; the grep hit-line preview survives); the query-bar tally shows
  `1000+` / `500+` when the result cap truncated the search.
- Paste + create-file batch (D1-D2, 2026-09-08): same-dir paste
  dedupes the name (stem.2.ext, claim-aware across queued copies)
  instead of erroring; copy/move ops now carry full destination paths.
  New `n` key stages an empty-file create with phantom-row preview;
  mkdir is dirs-only and the review queue labels are distinct.
- Refresh convention (C1, 2026-09-08): `r` refreshes on every screen;
  Services restart moved to `R`, Files gained `r` (re-list + preview
  refetch). Recorded N3: ellipsis truncation residue after shrinking
  then growing the window - user opted to leave it.
- User bug batch 2 (B1-B7, 2026-09-07): session restore lands on the
  section the user actually quit in (save-after-flip + q saves);
  Disks drills with enter/l like Files; ctrl+o/ctrl+i always in help;
  `fd` falls back to `fdfind` at runtime; `t` opens the trash as a
  listing (deletes inside it are permanent purges); head meta says
  `view hidden`; the editor chain never opens a pager — $EDITOR,
  $VISUAL, then vi/nvim/nano, else a toast, and install.sh adds nano
  when no editor exists.
- Help-content sync batch (backlog H4-H8 + N1/N2, 2026-09-07): the
  help panel no longer advertises keys the active screen ignores.
  Globals are exactly what root binds; the j/k row shows only on
  FilterTable screens; enter/esc verbs come from each screen's hints
  (Files gained enter/l, h, conditional esc; Services refresh
  rebound to R after review). While help is open, q closes it and
  tab/1-6 switch sections with the panel following. Manual globals
  section split into global vs list keys.
- Automated installer & README Installation overhaul (2026-09-07):
  Added `scripts/install.sh` supporting automated multi-distro dependency
  installation (apt/dnf/pacman/zypper/apk), Debian `fdfind` symlink
  resolution, CGO-free build, and path configuration. Updated `README.md`
  with comprehensive prerequisites, platform limitations, and a single
  copy-and-run installation command.
- Help window batch (backlog H1-H3): overlay splices terminate the
  base row's SGR state, so floating panel text no longer inherits
  bold/bright from underneath (also fixes the files perm editor /
  chown form / review overlay). The help window is a fixed-height
  viewport, identical on every screen, with a `n-m of N` position
  line; while open it owns keyboard (ctrl+d/ctrl+u half page, j/k,
  pgup/pgdn, reset on reopen) and mouse (wheel scrolls the list,
  clicks do nothing) — the background no longer scrolls behind it.

Earlier: user manual + demo runbook (`docs/manual.md`,
`docs/showcase.md`); permission & access management (ADR-0013) and the
UX clarity batch (U1-U8: state-only status bar, staged-change review
via `w`, staged-aware previews, phantom rows).

Full suite green (`scripts/check.sh`).

## In progress

Nothing.

## Next action

procfs time.Tick leak (last L6 remnant,
`internal/proc/procfs.go:172`) — replace with a ticker that stops.
Then first real rehearsal of `docs/showcase.md` (which doubles as the
live-pass of the help window and staged-changes batches).
