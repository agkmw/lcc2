# lcc2 — agent entry point

Keyboard-first Linux system utility TUI (Go, Bubble Tea). Run: `go run ./cmd/lcc2`.

## Standing rules

1. **End-of-session ritual — non-negotiable, in order:** `scripts/check.sh` green
   → update `docs/STATUS.md` → file decisions / failures / noticed-but-unfixed
   → `git add -A && git commit`. An interrupted session must leave a coherent
   repo: these steps happen before walking away, never after "finishing".
2. **ADRs are append-only.** Supersede, never edit (`docs/decisions/`).
3. **Record failed experiments** in `docs/experiments.md`. Negative results are
   the institutional memory most likely to save a month.
4. **Problem noticed but not fixed → `docs/backlog.md` immediately**, with a
   `file:line` pointer. Fixing is optional; recording is not.
5. **Commit everything, always.** Git history is the second copy of every
   pattern above.
6. **Compression keeps paths back.** Summaries keep URLs; digests keep
   `file:line`. Nothing disappears without a route back to it.
7. **Volatile content stays out of this file.** Current state lives in
   `docs/STATUS.md`; never here.
8. **Brevity is good.** Applies to code, comments and commit messages.
   Don't write a novel.

## Read-order (cold session)

`AGENTS.md` → `docs/STATUS.md` → `docs/backlog.md` → `docs/decisions/` → `docs/experiments.md`

## Package map

| path | role |
|---|---|
| `cmd/lcc2` | entrypoint; wires the six screens into `app.New` |
| `internal/app` | root model: chrome, section routing, toasts, help overlay |
| `internal/screens` | section models (overview, proc, disk, files, services, users); own all UI state |
| `internal/ui` | design system: FilterTable, ConfirmDialog, theme, keymap |
| `internal/{files,proc,services,disk,accounts,sysinfo}` | pure data providers; never import UI |

Build/test gate: `scripts/check.sh`.
