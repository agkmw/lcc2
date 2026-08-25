# ADR-0012: Migrate to Bubble Tea / Lip Gloss / Bubbles v2

Date: 2026-08-25
Status: Accepted (implemented)

## Context

The Charm ecosystem's current line is v2, published under vanity paths
(`charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`,
`charm.land/bubbles/v2`) with `colorprofile` replacing termenv. We were
on the v1 line. The polish batch (sort cycling, session memory,
editor/pager, clipboard, trash, disk I/O, mouse, chroma previews) was
deliberately built on v1 first so this migration would land against a
larger test surface — per the sequencing decision recorded before the
batch.

## Decision

Migrate the whole UI stack to v2 in one compile-green sweep
(M1), followed by small follow-ups (native clipboard; cleanup/docs).

Key mappings applied:

| v1 | v2 |
|---|---|
| `tea.KeyMsg{Type,Runes}` literals | `tea.KeyPressMsg{Code,Text}` (+`Mod` for shift/ctrl) |
| `msg.Type == KeyX` | concrete msg types (`MouseClickMsg`, `MouseWheelMsg`, …) or `Code` runes |
| space `" "` | `"space"` (plus literal `" "` kept where a raw space can arrive) |
| `View() string` | `View() tea.View` with `AltScreen=true`, `MouseMode=CellMotion` set declaratively |
| `WithAltScreen/WithMouseCellMotion` options | removed (View fields) |
| `lipgloss.Color("hex")` type/value | value stays; types become `color.Color`; palette parsed once via `ui.C(hex)` |
| termenv profile checks | `colorprofile.Detect(os.Stdout, os.Environ())` cached at init + `ui.SetProfileOverride` for tests (colorprofile honors NO_COLOR natively, so main.go's manual block is gone) |

Own-type decisions carried over: `ui.Column`/`ui.Row` replaced the
bubbles v1 table types entirely (no dual-module); FilterTable.Mouse
takes tea msgs but delegates through geometry that keeps table unit
tests tea-free.

## Calibration notes (v1 → v2 renderer deltas we had to absorb)

- Resets are emitted as bare `\x1b[m` (v1: `\x1b[0m`). Our SGR state
  painter already treats empty-body SGR as reset ✓; ui_test's rune
  filters were rewritten to skip full CSI sequences instead of a
  hand-list of characters.
- Everything else (truecolor SGR shapes, box glyphs, braille) renders
  byte-compatibly for our assertions after the above.

## Consequences

- Clipboard: native `tea.SetClipboard` replaces the hand-rolled OSC52
  tty writer as final fallback (wl-copy/xclip remain primary).
- ClearScreen-on-resize retained until live testing proves v2's full
  repaint makes it redundant (review in next hardening pass).
- bubbles v1 and termenv are fully out of go.mod; the only remaining
  v1-line dependency class was table types, already removed by H8.
