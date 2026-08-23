# ADR-0007: Visual language — pills, badges, titled frames, unicode-safe glyphs

Date: 2026-08-23 · Status: Accepted

## Context

The UI read as plain text next to btop/yazi/lazygit: no panel titles, no
focus framing, foreground-only row selection, unbadged hints.

## Decision

One polish pass, UI-layer only:
- Header: filled `lcc2` chip + segmented section pills (active = accent
  background); narrow mode reuses it as the tab strip.
- Sidebar: unicode-safe glyph per section, active `▍` edge marker.
- Footer: full-width surface rule + `[key] desc` badge hints.
- Every screen renders inside a `TitledBox` frame (`╭─ Title ─╮`); chrome
  budget in `contentArea` accounts for its border.
- Table selection is a full-line surface background (bubbles pads cells,
  so the highlight is contiguous with no extra work).
- Each table block reserves one top line: live position (`n/m · pct%`)
  or the filter prompt.
- State colors via `StateColor` thresholds; shaded fractional gauges;
  service state dots; process state letters colored; dir names bold
  accent with `▸`.
- Catppuccin Mocha only, unicode-safe glyphs only (no Nerd Fonts).

## Consequences

`contentArea` reserves 4 cols / 2 rows more than before — screens size to
the frame interior. Filter-height compensation moved out of Processes into
FilterTable. NO_COLOR/light-terminal support stays open as backlog L4 by
explicit choice.
