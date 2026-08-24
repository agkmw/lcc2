# Experiments — negative & surprising results

Every failed experiment, dead end or surprising measurement gets a row here.
A month saved is a month saved. Strike through superseded rows, never delete.

| date | what was tried | result | pointer |
|---|---|---|---|
| 2026-08-23 (pre-audit) | trusting bubbles table with zero-row SetRows | clamps cursor to -1 → panics on enter, viewport corrupts, "retyping never matches again"; fixed by normalizing cursor in applyFilter | `internal/ui/table.go:186-199`, `table_test.go:36-74` |
| 2026-08-24 | Ambiguous-width glyphs (●▸▾·›…) in chrome + systemctl output | tmux/locales render EAW-Ambiguous codepoints double-width while lipgloss counts 1 cell → column drift ("services bleed", dashboard misalignment). Fix: ui.Narrow sanitizer + ASCII-only chrome; denylist test `internal/app/glyph_test.go`. Residual risk kept: block elements █░▌ + box drawing + braille are TUI-standard; revisit only if bars still drift on user setup | `internal/ui/theme.go`, ADR-0010 |
