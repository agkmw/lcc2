# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

Permission & access management landed (2026-09-06, ADR-0013):

- `internal/ui` gained `Form`, a multi-field modal (tab/enter/esc,
  password masking) alongside ConfirmDialog.
- `internal/accounts` gained a mutation layer: user/group CRUD,
  membership, lock, password (`chpasswd`, stdin), expiry (`chage`),
  all shelled out through an `execer` test seam with provider-level
  guard rails — root check, name regex + `--` argv separator,
  system accounts (uid/gid < 1000) and primary groups protected,
  membership applied as gpasswd diffs. Plus `AdminGroup`/`HasAdmin`.
- Users & Groups screen (6): `n` create, `e` edit, `p` password,
  `L`/`u` lock, `x`/`X` delete (± home), `E` expiry; groups tab
  `n`/`x`/`a`/`d`. Every action confirms, is root-gated (read-only
  badge when unprivileged), user cards show a sudo flag.
- Files screen: staged `OpChown` via `O` (owner/group form),
  perm editor (`P`) gained a special-bit row, typed-octal entry and
  a recursive toggle (dir → one op per entry, > 100 ops confirm),
  multi-target staging for marked sets, and a group column.

Full suite + gauntlets green (`scripts/check.sh`).

## In progress

Nothing.

## Next action

User live-pass of the feature under `sudo go run ./cmd/lcc2`:
create/lock/delete a scratch user, toggle wheel membership, chown +
recursive chmod in Files, then save. Root-gated toasts should appear
when running without sudo. Afterwards: procfs `time.Tick` leak (last
L6 remnant), then P1 (membership checkbox editor) if the comma-list
form feels too raw in practice.
