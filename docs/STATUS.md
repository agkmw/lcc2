# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

All Critical + High items closed ([C1-C3], [H1-H6]); [M11] done. Gate green.

## In progress

Nothing — next up is the M batch.

## Next action

M-batch (small, one commit): M1 rune-safe wordWrap (`processes.go`);
M4 drop duplicate enter hint (`disks.go` fs hints); M5 add `x cut` hint
(`files.go`); M6 SizeMsg calls `layout()` (`services.go`); M8 case-insensitive
dir listing sort (`files/ops.go`, fix comment OR code — code); M10 error toasts
live 6 s (`app/root.go` tick duration by kind). Then M12 dead-code sweep in its
own commit. Gate → ritual → commit `fix(screens): medium-tier polish batch`.
