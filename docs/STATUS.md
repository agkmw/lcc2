# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

Fifth pass landed (2026-08-25): services details pane is a structured
kv card from `systemctl show` (state/boot/since/pid/memory/cpu/
restarts), with a red FAILED banner, journal tail on every selection,
amber restart badge in the meta line and the raw highlighted dump as
fallback for not-found units. Gate green.

## In progress

Nothing — awaiting user's live pass on this batch.

## Next action

User verifies: net graph shape/scale feel, cursor highlight and tab
chip on real hardware colors, ctrl+o/ctrl+i traversal while browsing,
F content search from a scoped directory. Then candidates: T6
(replace bubbles table), L3 symlink metadata consistency, mouse (L6).
