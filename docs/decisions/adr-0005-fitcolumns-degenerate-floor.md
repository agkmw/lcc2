# ADR-0005: fitColumns floor accepts overflow on degenerate terminals

Date: 2026-08-23 (retroactive) · Status: Accepted

## Context

bubbles/table wraps rows when column totals exceed table width, corrupting
the frame. Extremely narrow terminals cannot fit minimum-width columns.

## Decision

`FilterTable.fitColumns` shrinks widest-first down to a floor of 3 cells
per column; below that budget it accepts overflow rather than hiding
columns or wrapping.

## Consequences

Normal terminals never wrap. Degenerate ones (<~30 cols) render overflowing
lines by design. Revisit only if a hard requirement appears.
