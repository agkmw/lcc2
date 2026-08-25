# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

Polish batch landed (2026-08-25, 9 commits): files sort cycling,
session memory across runs, $EDITOR/$PAGER opening (files+services),
clipboard path copy (wl-copy/xclip/OSC52), trash-instead-of-delete
with permanent-fallback labeling, dashboard disk I/O rates, mouse
support (clicks/wheel/tabs), and chroma syntax-highlighted previews
(new dep). CLI gained --version/--help + NO_COLOR. Gate green.

## In progress

Nothing — awaiting user's live pass on this batch.

## Next action

User exercises the new surface: sorts, editor open, clipboard over
ssh/tmux, trash-then-undo, mouse gestures, highlighted previews.
THEN: dedicated bubbletea/lipgloss/bubbles v2 migration session (own
ADR — module paths, color types, termenv→colorprofile rework of
canvas/painter/profile tests, ghosting recalibration; swap clipboard
to native tea.SetClipboard).

## Next action

User verifies: net graph shape/scale feel, cursor highlight and tab
chip on real hardware colors, ctrl+o/ctrl+i traversal while browsing,
F content search from a scoped directory. Then candidates: T6
(replace bubbles table), L3 symlink metadata consistency, mouse (L6).
