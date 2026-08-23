# ADR-0004: FilterTable consumes keys before screen handlers

Date: 2026-08-23 · Status: Proposed

## Context

Audit found (BACKLOG-C2): while the filter input is focused, screen-level
`handleKey` switches intercept keys first — typing `d`/`x`/`K`/`s/t/r/e/D`
fires destructive actions and `enter` navigates instead of committing the
filter.

## Decision

When a FilterTable owns focus (`Filtering() == true`), it gets the key
before any screen-specific handler; screens gate their action switches on
it. Becomes Accepted when the C2 fix lands across all five list screens.

## Consequences

Typing a filter is always safe. Screens lose access to intercepted
letters only during filtering — acceptable because filter mode is modal.
