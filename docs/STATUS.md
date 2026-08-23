# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

Audit done; backlog seeded (3C/6H/12M/6L). [C1] fixed and regression-tested
(`5ba6d67` + test commit): `Copy()` refuses same-path paste. Gate green.

## In progress

Nothing.

## Next action

Fix [C2] filter-mode key interception (see `docs/decisions/adr-0004-*`, Proposed):
1. `internal/ui/table.go` — no change needed if screens gate: in each screen's `handleKey`, early-return into `tbl.Update` when `Filtering()`
2. Touch points: `internal/screens/{processes,files,services,disks,users}.go` handleKey switches
3. Tests: extend `internal/screens/layout_test.go`-style feeding — type `/dat` on Files, assert no dialog/prompt state
4. Gate: `scripts/check.sh`
5. Ritual: update this file → ADR-0004 → Accepted → backlog C2 → `closed (<sha>)` → commit
