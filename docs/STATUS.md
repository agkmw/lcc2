# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

User manual + demo runbook landed (2026-09-06):

- `docs/manual.md`: install/run, frame layout, global keys, all six
  sections with per-screen key tables, the Files staged-changes model
  (staging, glyphs, review via `w`, trash semantics, perm editor),
  troubleshooting. Written from the source, key-by-key verified.
- `docs/showcase.md`: 12-minute live-demo runbook (7 beats with
  SAY/DO/POINT AT/IF-IT-FAILS), sandbox setup script, 5-minute
  lightning cut, Q&A prep, presentation craft.
- Backlog hygiene: pruned stale duplicate `L6/L8/L9 open` blocks; the
  live entries (L6 partial, L8/L9 closed) remain.

Full suite green (`scripts/check.sh`).

## In progress

Nothing.

## Next action

procfs time.Tick leak (last L6 remnant,
`internal/proc/procfs.go:172`) — replace with a ticker that stops.
Then first real rehearsal of `docs/showcase.md` to shake out timing.
