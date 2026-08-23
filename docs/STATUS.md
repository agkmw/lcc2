# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

Full-app audit done at `dc03bf7`: build/vet/test green. 3 critical, 6 high,
12 medium, 6 polish issues logged in `docs/backlog.md`. Handoff system
(AGENTS.md, docs/, scripts/check.sh) bootstrapped this session.

## In progress

[C1] guard landed in `internal/files/ops.go` (`Copy()` refuses cleaned
dst == src). Regression test NOT yet written; fix unverified.

## Next action

Finish [C1]:
1. `internal/files/ops_test.go` — regression: write file, `Copy(src, dir)` must error and bytes stay intact
2. Gate: `scripts/check.sh`
3. Ritual: update this file → backlog C1 → `closed (commit <sha>)` → commit `test(files): self-copy regression closes C1`
