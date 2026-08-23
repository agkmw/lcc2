# Experiments — negative & surprising results

Every failed experiment, dead end or surprising measurement gets a row here.
A month saved is a month saved. Strike through superseded rows, never delete.

| date | what was tried | result | pointer |
|---|---|---|---|
| 2026-08-23 (pre-audit) | trusting bubbles table with zero-row SetRows | clamps cursor to -1 → panics on enter, viewport corrupts, "retyping never matches again"; fixed by normalizing cursor in applyFilter | `internal/ui/table.go:186-199`, `table_test.go:36-74` |
