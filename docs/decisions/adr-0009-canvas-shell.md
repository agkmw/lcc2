# ADR-0009: canvas shell — owned background, borderless panes, staged file manager

Date: 2026-08-24 · Status: Accepted · Supersedes ADR-0007, ADR-0008

## Context

User verdict on the ADR-0008 shell: still box-heavy and busy. Two
hard constraints came with it:

1. The user runs a **transparent terminal**. Pre-ADR8 components set
   their own backgrounds while the app painted none
   (`internal/ui/theme.go` `Base()` deliberately skipped bg), so fills
   floated as ugly patches over whatever was behind the window.
2. The aesthetic target is minimal TUIs (yazi, opencode): whitespace
   for separation, foreground color for state, no frame-in-frame.

Separately, the Files screen applied every mutation to disk the moment
a dialog was confirmed — no undo, no batch review.

## Decision

- **The app owns an opaque canvas.** `ui.Canvas` is the last step of
  root `View()`: every line clipped, padded, and filled with the app
  background; inner ANSI resets are re-opened so styled spans cannot
  punch transparent holes (`internal/ui/canvas.go`). Component rule:
  **foregrounds only by default**; background fills are reserved for
  intentional surfaces — cursor/dialog/toast/help layers.
- **Borderless panes.** No more bordered cards/boxes. Screens render a
  persistent **main | preview** split (shared scaffold
  `internal/screens/pane.go`: accent title, thin rule, exact-width
  clipping; stacks below 96 cols). Selection highlight is bold
  foreground + marker, not a fill row.
- **Navigation = nvim bufferline.** Sidebar removed. Top strip shows
  one segment per screen (faint digit, glyph, label); active is
  accent+bold; optional per-screen badge (`ui.BadgeSource`). Tab /
  Shift+Tab cycle with wrap-around; 1–6 still jump. Clock moved to the
  status bar's right slot beside the context hint.
- **Overview follows btop**: stacked full-width sections cpu / mem /
  net / disk with multi-row area graphs (`ui.Graph`), per-core sparkline
  grid, segmented memory bar (`ui.SegGauge`, cache overlay), auto-scaled
  network graphs keyed to session peak.
- **Files adopts the oil.nvim model.** Every mutation stages into a
  pure provider queue (`files.Stager`, ADR-0002-clean) shown as row
  glyphs plus a tab badge; `w` applies op-by-op with stop-on-error,
  `u`/`U` unstage. Multi-select via space; clipboard carries many
  paths. Preview pane lists dirs, previews text heads
  (`files.ReadPreview`, NUL-sniffed binary → metadata card).
- Services/processes/disks/users previews always follow the cursor;
  services auto-refresh every 15 s; process preview re-inspects on
  movement and tick. `r=restart` on Services moves to `R`; `r` means
  refresh everywhere else.

## Consequences

Renderer hazards from ADR-0008 remain pinned by tests (frame
well-formedness, exact-width splits, ANSI-safe truncation) and two new
ones are covered: canvas fill must survive inner resets, and must
degrade to no-op under NO_COLOR/Ascii profiles. The old visual pins
(TitledBox format, KeyBadge) were deleted with their code. Frame height
budget shrank by one line (no blank separator row). Staged saves give
per-op progress but still no byte-level progress/cancel (backlog L7).
