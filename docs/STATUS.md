# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

Second live-pass batch landed (7 commits). Net dashboard graphs plot
real history now (they drew a single point before) with scale
hysteresis — no more continuous shrinking under ordinary load. Box
tops are exact rectangles; help is a bordered card. The canvas pass
is an SGR state machine, so intentional backgrounds survive: selected
rows are bold-on-surface full-line highlights everywhere, the active
tab is an inverted chip, dialog bands keep their color. Search UX:
arrows-only navigation in aux modes (letters type into the query), rg
is case-insensitive with a searching indicator, and ctrl+o / ctrl+i
walk directory history. Semantic coloring shipped for perms, uid/gid,
shells, service sub-states, pids and disk fields. Gate green.

## In progress

Nothing — awaiting user's third live pass.

## Next action

User verifies: net graph shape/scale feel, cursor highlight and tab
chip on real hardware colors, ctrl+o/ctrl+i traversal while browsing,
F content search from a scoped directory. Then candidates: T6
(replace bubbles table), L3 symlink metadata consistency, mouse (L6).
