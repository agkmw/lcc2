# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

[C1]-[C3], [H1], [M11] closed. Gate green.

## In progress

Nothing — next up is H3 (tiny), then H2.

## Next action

Fix [H3] memTotal data race:
1. `internal/proc/procfs.go`: hold `memOnceMu` around the `memTotal` write in `refreshTotalMem`
2. Gate → ritual → commit `fix(proc): lock memTotal writes (H3)`
Then [H2]: epoch-guard tick chains in Overview+Processes, spinner stop guards in Disks/Services (`d.spin` chain dies when `!busy`; services when loaded).
