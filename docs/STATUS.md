# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

N1 done: nvim-notify-style floating windows, non-destructive compositing
(ADR-0006). Gate green. All C/H closed; M2 M3 M7 and L-tier open.

## In progress

Nothing — next up is T1 (cursor tracking).

## Next action

[T1] identity-based cursor tracking:
1. `internal/ui/table.go`: `keys []string` parallel to rows; `SetRowsTracked(rows, keys)`; `applyFilter` restores cursor by key (nearest-index fallback)
2. Wire keys in all five screens: files=path, processes=PID (drop syncTable PID hack), services=name, users=name, disks=mountpoint/path
3. Tests in `internal/ui/table_test.go`: delete-shift keeps selection; filter narrow/widen keeps it
4. Gate → ritual → commit `feat(ui): identity-based cursor tracking (T1)`
Then [S1] content-sized detail panes.
