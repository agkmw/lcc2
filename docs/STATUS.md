# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

Full-app audit done at `dc03bf7`: build/vet/test green. 3 critical, 6 high,
12 medium, 6 polish issues logged in `docs/backlog.md`. Handoff system
(AGENTS.md, docs/, scripts/check.sh) bootstrapped this session.

## In progress

Nothing.

## Next action

Fix [C1] same-dir paste destroys source file:
1. `internal/files/ops.go` — in `Copy()`, refuse when `filepath.Clean(dst) == filepath.Clean(src)`
2. `internal/files/ops_test.go` — regression: `Copy(f, dir)` errors and file bytes intact
3. Gate: `scripts/check.sh`
4. Ritual: update this file → tick backlog C1 → commit `fix(files): refuse same-path paste`
