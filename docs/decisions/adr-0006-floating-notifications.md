# ADR-0006: Notifications as non-destructive floating windows

Date: 2026-08-23 · Status: Accepted

## Context

Toasts rendered through full-frame `lipgloss.Place` overlays, repainting
every line's whitespace on each appearance — the whole screen visibly
washed/shifted. They also shared the footer with key hints, truncating
them.

## Decision

nvim-notify-style windows (`internal/ui/notify.go`): a top-right stack of
bordered panels (blue/green/red by kind), newest first, max 4, per-note
expiry ticks carrying ids (info/ok 3 s, err 6 s). Compositing splices
window lines into the frame ANSI-aware (`x/ansi.Truncate`) — content
outside each window rectangle is byte-identical; header and footer rows
are never covered.

## Consequences

Footer hints never truncate for toasts; notifications survive above the
help overlay (composited last). Cost: frame lines are width-normalized
only inside window rectangles; help overlay remains intentionally modal
and unchanged.
