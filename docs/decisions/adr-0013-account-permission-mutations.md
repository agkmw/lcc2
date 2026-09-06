# ADR-0013: Account & ownership mutations via admin tools behind an exec seam

Date: 2026-09-06 · Status: Accepted

## Context

The users & groups section was read-only (parses of `/etc/passwd` and
`/etc/group`), and file ownership (chown) was absent. Adding permission
and access management means the app performs its first genuinely
privileged mutations: user/group CRUD, membership, passwords, account
expiry, and chown. Nothing in the codebase detects root or executes
privileged tools, so the privilege model, the mutation mechanism and the
safety rules were all open decisions.

## Decision

- **The app runs as root** (`sudo go run ./cmd/lcc2`). There is no
  sudo/polkit/password plumbing inside the app. Every mutating entry
  point in `internal/accounts` re-checks `IsRoot()` and fails with
  `ErrNotRoot`; screens gate first so unprivileged users see a toast
  instead of a dialog.
- **Mutations shell out to standard admin tools** — `useradd`, `userdel`,
  `usermod`, `groupadd`, `groupdel`, `gpasswd`, `chpasswd`, `chage` —
  through a package-level `execer` seam (`internal/accounts/exec.go`)
  that tests replace to assert exact argv/stdin. `/etc/passwd` and
  `/etc/group` are never written directly; ownership goes through
  `os.Chown` in `internal/files`.
- **Guard rails live in the provider** (testable without a UI):
  names must match the portable regex and every exec passes them after
  a `--` separator (blocks argument injection); uid/gid 0 and any id
  < 1000 is refused for deletion, locking, group stripping and primary
  -group changes (`ErrSystemAccount`); removing a group that is some
  user's primary group is refused; `chpasswd` receives the password on
  stdin, never as an argument; membership changes are diffs applied as
  individual `gpasswd -a/-d` calls so each exec is unambiguous.
- **UI actions confirm before they run**: destructive actions go through
  `ui.ConfirmDialog`, multi-field input through the new `ui.Form`;
  results surface as toasts (ADR-0006). File-side chown/chmod reuse the
  staged-ops pipeline — staged batches (recursive chmod > 100 paths)
  confirm before joining the queue and stay undoable until save.

## Consequences

- No password/credential handling inside the app; `chpasswd` stdin is
  transient and never logged.
- The exec seam makes argv regression-testable; the guards run before
  any exec, so refused paths are observable as "zero exec calls" in
  tests.
- Root-required UX is a clear limitation: running unprivileged yields a
  read-only users screen. A future ADR could add polkit or per-op sudo,
  superseding this one.
- Deferred: ACLs (getfacl/setfacl), sudoers.d editing, shadow-field
  editing beyond expiry. These are out of scope by design.
