# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

v2 migration landed (ADR-0012): bubbletea/lipgloss/bubbles at
charm.land v2 paths, colorprofile replacing termenv, declarative View
fields (altscreen/mouse), native clipboard fallback, own Column/Row
types. All gauntlets recalibrated and green on the new renderer.

## In progress

Nothing — awaiting user's live pass on the v2 build.

Nothing — awaiting user's live pass on this batch.

## Next action

User runs the v2 binary for a real session: confirm mouse gestures,
clipboard over ssh/tmux, resize behavior (decide whether the
ClearScreen-on-resize workaround can be dropped), and general frame
stability. Then: procfs time.Tick leak (last L6 remnant).