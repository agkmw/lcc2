# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

Canvas-shell redesign landed end-to-end (ADR-0009, supersedes
0007/0008; 5 commits). App owns an opaque background (`ui.Canvas`,
transparent-terminal safe), sidebar replaced by a bufferline tab strip
with Tab/Shift+Tab cycling, all six screens use the borderless
main|preview pane pattern. Overview is btop-style (stacked cpu/mem/
net/disk sections, per-core graphs, autoscaled net). Files is oil-style:
multi-select + staged ops (`files.Stager`), applied only on `w`, with
live dir/text/binary preview. Gate green.

## In progress

Nothing — session wrapped.

## Next action

Live pass: `go run ./cmd/lcc2` at ~90 and ~130 cols on a real terminal.
Check: canvas opacity on the transparent setup, tab-strip truncation
(L9), net-scale pinning after a big transfer (L8), staged-save flow on
real files. Log to `docs/backlog.md`. Open items: L3, L4(partial),
L6, L7, L8, L9.
