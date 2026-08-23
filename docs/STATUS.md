# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

[C1-C3], [H1], [H2], [H3], [M11] closed. Gate green.

## In progress

Nothing — next up is H4.

## Next action

Fix [H4] file-op busy guard + indicator:
1. `internal/screens/files.go`: add `opCount *atomic.Int32`; `doThenRefresh` increments around fn; mutating keys (`d m R y x p P a enter l h`) ignored while >0
2. Head line shows faint "working…" while busy
3. Test: delete flow in tempdir → cmd runs → busy clears; second action key ignored mid-op
4. Gate → ritual → commit `fix(files): busy guard and indicator (H4)`
