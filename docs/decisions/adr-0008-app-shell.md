# ADR-0008: App-shell layout — quiet chrome, loud data

Date: 2026-08-24 · Status: Accepted

## Context

After the polish pass the UI still read as cluttered: rounded/square
borders mixed, screen frame wrapped inner cards (frame-in-frame), nav
pills used loud background chips, sidebar floated without separation,
the status rule floated apart from hints. User feedback: panes messy,
dashboard/disk/files/users/processes all unpleasant.

## Decision

Treat the TUI like a web app shell:
- Nav bar and status bar are real bars with their own square rules;
  the standalone rule line and the outer screen TitledBox are gone.
- Section nav = plain text links in the nav bar (active: accent +
  underline); a bordered left rail mirrors it; hidden below 84 cols.
- Content sits directly on the page: shared `pageHead(title, meta)`
  per section, then square `Card`s on an equal-height grid
  (overview: 2×2 + full-width disk card).
- Table position strip renders BELOW its table; filter prompt above.
- Square borders only (`NormalBorder`); accent reserved for active nav,
  card titles, selected-row marker, key badges.
- Status bar right slot shows per-screen context (`ui.ContextSource`)
  — selected item name/pid/unit.
- Airy rhythm: one blank row between cards; page margins 2 cols.

## Consequences

`contentArea` grew (+4 cols from dropped frame). Two renderer pitfalls
now guarded by tests: lipgloss `Style.Render` pads ALL lines to the
widest input line (status-bar overflow once inflated the whole frame),
and `JoinHorizontal` normalizes block widths (swallowed gutters) —
overview rows assemble via exact-width `ui.Split`. ANSI-safe truncation
(`x/ansi`) is mandatory everywhere; hand-rolled rune slicing leaked
escape fragments.
