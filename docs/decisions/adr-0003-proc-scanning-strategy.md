# ADR-0003: Direct /proc scanning; gopsutil only for on-demand Inspect

Date: 2026-08-23 (retroactive) · Status: Accepted

## Context

Process list refreshes every 3 s over all PIDs. The heavyweight gopsutil
path allocates too much for a full-scan-per-tick workload.

## Decision

`proc.Collector.Snapshot` reads `/proc/<pid>/stat` + `/status` directly
(two small reads per PID) with delta-based CPU accounting between
snapshots; kernel threads filtered by `ppid == 2`. gopsutil is used only
in `Inspect(pid)` for single-process detail views.

## Consequences

Refresh cost close to `ps`. Heuristic filters children of kthreadd only;
memTotal cached with a 30 s refresher goroutine (see BACKLOG-H3 race).
