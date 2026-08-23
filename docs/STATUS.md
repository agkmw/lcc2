# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

Backlog burn-down under way. [C1], [C2] closed; ADR-0004 Accepted.
Gate green.

## In progress

Nothing — next up is C3.

## Next action

Fix [C3] services confirm dialog stuck after "yes":
1. `internal/screens/services.go` handleKey: set `s.confirm = nil` in the `done && yes` branch
2. Also clear in the `svcActionDoneMsg` handler (mirror `processes.go:142`)
3. Test in `internal/screens/filter_guard_test.go` style: 's' → 'y' → svcActionDoneMsg → assert confirm nil (both err and ok paths)
4. Gate: `scripts/check.sh` → ritual → commit `fix(screens): dismiss services confirm on action start (C3)`
