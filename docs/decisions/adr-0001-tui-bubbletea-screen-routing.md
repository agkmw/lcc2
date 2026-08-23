# ADR-0001: Bubble Tea TUI with Screen-interface routing

Date: 2026-08-23 (retroactive; decision predates record) · Status: Accepted

## Context

Keyboard-first system utility with six sections, live refresh, modals,
toasts and a help overlay. Needed testable UI logic and consistent chrome.

## Decision

Bubble Tea (Elm architecture) + lipgloss. Every section implements
`ui.Screen` (`Init/Update/View/ID/Title/Hints/CapturingInput`);
`app.Root` owns header/sidebar/footer, number-key routing, toast and help
overlays, and forwards computed `ui.SizeMsg` to the active screen.

## Consequences

Single static binary; screens are pure-ish models testable without a tty;
shared components (FilterTable, ConfirmDialog) enforced by construction.
Cost: global shortcuts need explicit suppression via `CapturingInput`,
and screen lifecycle (Init on switch) is Root's responsibility — see
BACKLOG-H2 for a lifecycle bug this produced.
