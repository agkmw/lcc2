# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

[C1] [C2] [C3] closed. Criticals clear. Gate green.

## In progress

Nothing — next up is H1.

## Next action

Fix [H1]+[M11] disk scan correctness:
1. `internal/disk/scan.go`: return `ctx.Err()` on cancel (no partial results); root-unreadable walk error aborts with error instead of silent empty
2. `internal/screens/disks.go` scanDoneMsg: ignore `context.Canceled` silently
3. Tests in `internal/disk/scan_test.go`: cancelled ctx errors; unreadable root errors
4. Gate → ritual → commit `fix(disk): honest scan cancellation and root errors (H1,M11)`
