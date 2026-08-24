# ADR-0011: Escape-safe table cells, minimum geometry, resize gauntlet

Date: 2026-08-24
Status: Accepted
Supersedes: nothing (extends ADR-0010's renderer-hazard discipline)

## Context

User report: "resizing the terminal corrupts the UI." Root cause found
in the vendored renderer contract, not our layout math:

bubbles/table re-truncates every cell on render with
`runewidth.Truncate(value, col.Width, "…")` — RUNE count, not display
cells. ANSI escapes count as runes. Our styled cells (bold dirs,
accent names/locations, colored percentages) are display-clipped by
`clipCells` but their escape bytes inflate rune counts. The moment a
terminal shrink refits columns narrower, bubbles slices cells
mid-escape; orphaned `\x1b[` fragments then make the terminal
misparse everything after them. Static frame tests never see it:
they measure our View output before bubbles' internal truncation.

Second finding: no floor below which we stop rendering — `contentArea`
degraded to 10x4 and every screen produced garbage instead of a
refusal.

## Decision

1. **Cell invariant**: every cell set on a FilterTable must fit its
   column in BOTH display width and rune count (`ui/table.go`
   `fitCell`). Styled cells that cannot satisfy both drop styling.
   bubbles' rune-based truncation thereby becomes a strict no-op — it
   can never slice escapes or emit its banned `…`.
2. **Minimum geometry 64x16** (`app.MinW/MinH`). Below the floor the
   app renders an exactly-frame-sized centered notice instead of
   screens; sizes keep flowing so recovery on grow is instant. Chosen
   lenient over a btop-strict 80x24 per user preference.
3. **Resize broadcast**: WindowSizeMsg forwards computed SizeMsg to
   all screens, not just the active one — inactive tabs never carry
   stale geometry into their next visit.
4. **Verification is automated**: shrink/grow cycles across sections,
   modal states, data-fed tables assert exact line count, width
   compliance and glyph cleanliness after every step
   (`app/resize_gauntlet_test.go`). User directive: no tmux eyeballing.

## Consequences

- Any future cell source must route through `clipCells` (i.e.
  SetRows/SetRowsTracked); raw bubbles tables would re-expose T6.
- Below-floor rendering is intentionally dumb text; no effort is spent
  making layouts degrade gracefully under 64x16.
- The gauntlet is the regression wall for the entire "resize broke it"
  bug class; extend it whenever a new modal or layout mode lands.
