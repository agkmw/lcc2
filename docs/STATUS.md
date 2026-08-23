# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

App-shell redesign landed (3 commits, ADR-0008): square-only borders,
nav/status bars, bordered rail, pageHead pattern, overview card grid,
strip-below tables, context slot in status bar. Renderer pitfalls
(Style.Render inflation, ANSI-unsafe truncation, JoinHorizontal
normalization) fixed and pinned by tests. Gate green.

## In progress

Nothing — session wrapped.

## Next action

Live pass: `go run ./cmd/lcc2` — judge nav/rail/dashboard on a real
terminal at ~90 and ~130 cols. Log anything off to `docs/backlog.md`
with a pointer. Open items: M2 M3, L1-L7.
