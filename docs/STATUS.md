# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

Full UI overhaul landed (2026-08-24 session, ADR-0011). Dashboard is
btop-style boxed sections (cpu/mem/net/disk with embedded titles,
prefilled history so charts span immediately, rolling 60-sample net
peak). Files gained fd find (`f`) and rg grep (`F`) aux modes with
fzf-style live results and jump-to-line preview highlight; tab switch
on Users moved to `s` (tab is section-cycling). The resize-corruption
root cause is fixed (bubbles rune-truncation slicing styled cells;
escape-safe `clipCells` invariant) plus a 64x16 minimum-size gate with
a friendly notice; resizes broadcast to all screens. Modals overlay
live content instead of blanking the screen; process previews render
instantly from list rows; phantom staged glyphs, breadcrumb double
slashes, mode-column type bits, ADR-0010 chrome violations — all fixed
and pinned by tests. Verification policy per user: automated gauntlets
(resize/glyph/chrome), no tmux eyeballing. Gate green.

## In progress

Nothing — awaiting user's live pass on this batch.

## Next action

User verifies in their terminal of choice: resize up/down at will
(should never garble; <64x16 shows notice), dashboard boxes read like
btop, `f`/`F` search feel right, `s` switches users/groups. Then
candidates: T6 (replace bubbles table), L7 (byte progress/cancel),
mouse support (L6).
