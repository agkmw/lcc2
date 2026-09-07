# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

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
