# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

[C1-C3], [H1]-[H4], [M11] closed. Gate green.

## In progress

Nothing — next up is H5+H6.

## Next action

Fix [H5]+[H6] safe file ops:
1. `internal/files/ops.go`: Rename refuses existing target; copyFile refuses existing dst; Move checks before rename/copy
2. H6: Copy/Move refuse when dstDir lies inside src subtree (`filepath.Rel` prefix check)
3. Tests in `internal/files/ops_test.go`: overwrite refused ×3; dir-into-own-child refused
4. Gate → ritual → commit `fix(files): refuse overwrites and self-nesting copies (H5,H6)`
