# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

Two batches landed 2026-09-06:

- Help window batch (backlog H1-H3): overlay splices terminate the
  base row's SGR state, so floating panel text no longer inherits
  bold/bright from underneath (also fixes the files perm editor /
  chown form / review overlay). The help window is a fixed-height
  viewport, identical on every screen, with a `n-m of N` position
  line; while open it owns keyboard (ctrl+d/ctrl+u half page, j/k,
  pgup/pgdn, reset on reopen) and mouse (wheel scrolls the list,
  clicks do nothing) — the background no longer scrolls behind it.
- User manual + demo runbook: `docs/manual.md` (install/run, frame
  layout, per-screen key tables, Files staged-changes model,
  troubleshooting) and `docs/showcase.md` (12-minute demo runbook,
  sandbox script, lightning cut, Q&A prep). Backlog hygiene: pruned
  stale duplicate `L6/L8/L9 open` blocks.

Earlier today: permission & access management (ADR-0013) and the UX
clarity batch (U1-U8: state-only status bar, staged-change review via
`w`, staged-aware previews, phantom rows).

Full suite green (`scripts/check.sh`).

## In progress

Nothing.

## Next action

procfs time.Tick leak (last L6 remnant,
`internal/proc/procfs.go:172`) — replace with a ticker that stops.
Then first real rehearsal of `docs/showcase.md` (which doubles as the
live-pass of the help window and staged-changes batches); check
`docs/manual.md` key tables against the reworked help window keys.
