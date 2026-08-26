# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

2026-08-26 audit batch landed: 15 findings (1 critical, 3 high,
5 medium, 6 low) fixed in 15 one-bug commits, each with a regression
test. Headline: trash no longer falls back to permanent deletion —
cross-device deletes refuse with an explicit error (C5); session
restore initializes the restored section (H9); tab-strip clicks work
(H10). Full suite + gauntlets green (`scripts/check.sh`).

## In progress

Nothing — awaiting user's live pass on the audit build.

## Next action

User runs `go run ./cmd/lcc2` for a real session: confirm mouse
gestures end-to-end (tab clicks, committed-filter clicks), a session
restored onto a non-overview tab, and trashing a file on a second
filesystem (expect the refusal toast, not deletion). Then: procfs
time.Tick leak (last L6 remnant).
