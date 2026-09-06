# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

UX clarity batch landed (2026-09-06, backlog U1-U8), on top of the
access-management feature from earlier today:

- Status bar no longer overflows: StatusSource screens show state only
  (sort, marks, staged counts, root pointer) plus `? keys (n)`; the
  full key list lives in the ? overlay. Files shows sort/marks/review;
  unprivileged Users collapses to a single sudo pointer.
- Staged changes are now visible everywhere: `w` opens a scrollable
  review of the whole queue (old -> new per op) before anything
  applies; the preview overlays pending chmod/chown/delete/rename with
  `(staged ...)` tags; staged creates get phantom rows with their own
  preview. The >100-op stage-time confirm is gone - review is the one
  gate.
- Small clarity fixes: sort + direction in the files pane header (and
  fileSort actually initialized), marks-cleared toast, ASCII sort
  direction. Dialog danger-band audit found nothing to change.

Full suite + gauntlets green (`scripts/check.sh`).

## In progress

Nothing.

## Next action

User live-pass: stage several mixed ops in Files (mkdir, rename, chmod,
chown, trash), check glyphs/preview tags, then `w` — the review should
list everything with old -> new before save. Then confirm the status
bar reads cleanly at 80 and 120 columns. Afterwards: procfs time.Tick
leak (last L6 remnant).
