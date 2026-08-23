# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

UI polish pass under way. Commit 1/4 done: design-system foundations —
`TitledBox`, `KeyBadge`, `SelectedRow` surface highlight, `StateColor`,
shaded fractional gauges (`internal/ui/theme.go`, `meter.go`).
Gate green.

## In progress

Commit 2/4: chrome restyle.

## Next action

Finish commit 2 (`feat(app): chrome restyle`): pill header, glyph sidebar,
badge footer + rule, titled screen frame in `internal/app/root.go`; update
footer string assertions; gate → ritual → commit. Then commits 3-4 per plan
in session context: tables/data display, modals polish.
