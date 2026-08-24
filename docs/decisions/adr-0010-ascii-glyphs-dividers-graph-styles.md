# ADR-0010: ASCII-safe glyphs, pane dividers, graph styles

Date: 2026-08-24 · Status: Accepted · Supersedes ADR-0009

## Context

First live pass on a transparent terminal inside tmux surfaced five
failure clusters: services preview text bled into the table column,
dashboard bars drifted out of alignment, the Files screen read as
noise, the help overlay left a transparent strip on its right, and
tmux split/zoom produced ghost frames from inactive screens.

Root-causing tied three of these to one mechanism: chrome and provider
text carried **East-Asian-Ambiguous** codepoints (● ▸ ▾ · › …). Our
cell math counts them as one column; tmux and CJK-locale terminals
render two. Every following cell shifts — alignment dies in ways that
look like unrelated layout bugs.

Separately: pure-whitespace pane separation did not read as separation;
the help card's Surface fill looked heavy next to its unpainted right
strip; and the block area-charts lacked the btop smoothness the project
targets.

## Decision

- **Glyph policy (tiered).** EAW-Ambiguous codepoints are banned from
  chrome and scrubbed from provider text via `ui.Narrow` (systemctl's
  ● → `*`, etc.). Tabs become numbered labels; dirs get `/` suffixes;
  service states are colored words; `-` replaces `·`. Block elements,
  box drawing and braille stay: they are the TUI lingua franca and
  render narrow in practice. A denylist test fails any future
  regression (`internal/app/glyph_test.go`).
- **Pane divider.** Main|preview panes are separated by a Surface-
  colored `│` column (horizontal rule when stacked) instead of bare
  whitespace.
- **Help = dim backdrop.** View order becomes frame → overlay splice →
  `ui.CanvasWith` last → toasts. The final canvas pass guarantees zero
  unpainted cells (fixes the transparent strip structurally). With help
  open the canvas dims to `BGDim` (#11111B); the key list floats
  cardless.
- **Graph styles.** New braille line-chart renderer
  (`ui.GraphBraille`, 2×4 dots per cell, EAW-neutral) is the default
  for time series; `g` toggles braille⇄block live, `LCC2_GRAPH=block`
  seeds the startup default. Bars/gauges remain blocks regardless.
- **Resize repaint.** Any terminal dimension change batches
  `tea.ClearScreen` so tmux split/zoom cannot leave stale cells; a
  regression gauntlet asserts each frame equals a fresh model render.

## Consequences

The renderer is now honest about width on every terminal we know of,
but "narrow in practice" for blocks/box-drawing remains an empirical
claim — if bar drift ever reappears on exotic setups, the fallback is
ASCII bars, not more glyph triage. Braille needs reasonable font
support; the toggle is the escape hatch. Frame geometry tests now pin
divider columns and dim-backdrop paths alongside the ADR-0008-era
well-formedness guards.
