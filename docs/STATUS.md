# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

UI bug batch fixed (U1/U2): detail panes rebuilt on exact-width
`ui.Split` (side-by-side ≥100 cols, stacked below), frame bleed-through
eliminated, chrome well-formedness pinned by tests. Gate green.

## In progress

Nothing — session wrapped.

## Next action

Live pass recommended: `go run ./cmd/lcc2` on a real terminal, check
detail panes at ~80 and ~120 cols and section switching for ghosting.
If anything remains, log to `docs/backlog.md` with a pointer. Open
items: M2 M3, L1-L7.
