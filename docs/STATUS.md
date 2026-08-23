# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

N1 (floating notifications) and T1 (identity cursor tracking) done.
Gate green. Open: S1, M2 M3 M7, L-tier.

## In progress

Nothing — next up is S1.

## Next action

[S1] content-sized detail panes:
1. `internal/screens/styles.go`: `paneHeight(contentLines, avail int) int` = clamp(lines,4,avail-2)+2 (border)
2. Apply: processes detail (`dvp.Height = paneHeight(detailLines, th)` in layout/setDetailContent), files metaPane (drop fixed 14), services + users detail panes (drop full-height clampInt(h,8,h))
3. Tests: helper unit test; lifecycle test asserting proc detail view height ≤ h with small content
4. Gate → ritual → commit `feat(screens): content-sized detail panes (S1)`
