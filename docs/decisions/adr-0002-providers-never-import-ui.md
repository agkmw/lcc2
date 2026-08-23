# ADR-0002: Data providers never import UI

Date: 2026-08-23 (retroactive) · Status: Accepted

## Context

Process/filesystem/service collection must stay headless-testable and
reusable, and must not depend on Bubble Tea types or render state.

## Decision

`internal/{files,proc,services,disk,accounts,sysinfo}` return plain
structs and errors. All `tea.Model`/`tea.Cmd` state lives in
`internal/screens`; providers never import `internal/ui`.

## Consequences

Providers test as pure functions (`*_test.go` files exist for each).
Cost: each screen re-implements refresh plumbing (tick chains), which is
where the lifecycle duplication of BACKLOG-H2 lives.
