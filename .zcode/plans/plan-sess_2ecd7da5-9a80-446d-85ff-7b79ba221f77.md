# Permission & Access Management for lcc

## Locked-in requirements (from your answers)
- Privileges: app runs as root (`sudo`); privileged actions error with a clear toast when euid != 0. No sudo/polkit machinery.
- User CRUD: standard admin set (create/edit/delete, lock/unlock, password, groups; system vs human accounts). No UID overrides or chage-level shadow editing except a simple expiry field.
- Permission scopes: admin/sudo rights (group membership only, **no sudoers editing**), account status, file ownership (chown), existing chmod support. **No ACLs.**
- UI: extends the existing Users & Groups screen (section 6). File permission editing stays in Files.
- Unanswered follow-ups resolved by my defaults: full guard rails, both-direction membership editing, all four Files upgrades (phased).

## Architecture (per ADR-0002/0001/0004/0006/0010/0011)
Providers shell out to standard admin tools (`useradd`, `usermod`, `userdel`, `groupadd`, `groupdel`, `gpasswd`, `chpasswd`, `chage`) — never edit /etc files directly. New seam `internal/accounts`: `var execer = func(name string, args []string, stdin io.Reader) (string, error)` so tests assert exact argv/stdin without running real tools. Same pattern as proc's `procDir` seam.

## Part 1 — Provider: `internal/accounts/mutate.go`
- `IsRoot()` (`os.Geteuid()==0`), sentinel `ErrNotRoot` returned by every mutation; `geteuid` is a package var for tests.
- Users: `CreateUser(UserOpts{Name, Shell, Home, PrimaryGroup, Groups, System})` → `useradd -m -s ... [-r] [-g] [-G] -- name`; `DeleteUser(name, removeHome)` → `userdel [-r] --`; `ModifyUser(name, shell, home)` → `usermod -s -d`; `SetGroups(name, primary, supp)` → `usermod -g -G` (full replace); `LockUser/UnlockUser` → `usermod -L/-U`; `SetPassword` → `chpasswd` with stdin `name:pw`; `SetExpiry(date)` → `chage -E` (blank = `-1`).
- Groups: `CreateGroup(name, gid)`, `DeleteGroup(name)`, `AddMember/RemoveMember(group, user)` → `gpasswd -a/-d`.
- Guard rails (provider-level, testable):
  - `validName`: `^[a-z_][a-z0-9_-]*$?...` regex + `--` before every name arg (blocks arg injection).
  - Refuse `DeleteUser`/`LockUser` on uid 0 and uid<1000 (sentinel `ErrSystemAccount`).
  - `SetGroups`/`RemoveMember` refuse when the change would strip admin-group membership from uid<1000; primary-group change on system users refused.
  - `DeleteGroup` refuses gid 0 and gid<1000, and any group that is a user's primary group.
  - Create checks name/GID collisions against loaded `/etc/passwd`+`/etc/group` first (friendlier than raw tool errors).
- Pure helper `AdminGroup(groups) string` (first of wheel/sudo) + `HasAdmin(user)` for the sudo indicator.

## Part 2 — UI component: `internal/ui/form.go` (+tests)
Multi-field modal form, the missing design-system piece (FilterTable/ConfirmDialog exist; forms don't). Fields: label, `bubbles/textinput`, password echo mode, optional prefill. Keys: tab/up/down cycle, enter submit, esc cancel. `Values() []string`. Rendered via `Panel()` + section accent — consumed by both the users screen and the Files chown dialog.

## Part 3 — Users & Groups screen (`internal/screens/users.go`)
State: `root bool` (from Init), `confirm *ui.ConfirmDialog`, `form *ui.Form` + `formKind`, `pending` op struct. All new keys sit behind the existing ADR-0004 filter guard; `CapturingInput()` extended for dialog/form. Every mutating key first checks `root`, else `ui.ErrToast("needs root - restart with sudo")` and no dialog.
- Users tab: `n` new user form (name, shell, home, primary group, groups, system y/n); `e` edit (shell/home/groups prefilled; confirm shows +/- membership diff); `p` password (2 password fields); `L`/`u` lock/unlock (confirm); `x` delete (confirm, home kept) + `X` delete incl. home (extra confirm); `E` expiry form (`chage -E`, blank clears).
- Groups tab: `n` new group (name, gid); `x` delete (confirm); `a` add member / `d` remove member (name form; remove confirms).
- Root-owned keys (`s`, `r`, filter, table nav) untouched. After each success: refresh cmd (re-Init) + `ui.OkToast`; failures → `ui.ErrToast` with context (M10 convention). Dialogs render via `overlayCenter` like Services (services.go:479).
- Preview additions: "can sudo: yes/no (wheel)" on user cards; admin-group flag on group cards. `Hints()` gains the new bindings (status bar auto-renders them). Head line shows "read-only (not root)" when unprivileged.

## Part 4 — Files screen permission upgrades (phased)
1. **OpChown**: `UID/GID int` fields on `files.Op` (-1 = unchanged); `files.Chown` via `os.Chown`; Stage validation, `ApplyOp` case, `Label()` ("chown user:group file"), staged glyph color. Name→id resolution via `os/user`. `O` key opens the shared Form (owner/group prefilled; blank = keep), stages per target; gated on root.
2. **Special bits + octal**: 4th row (setuid/setgid/sticky) in the `P` editor grid (`PermBits.Special` already modeled, `Octal()` already 4-digit); digit keys 0-7/backspace build an octal buffer in the editor, enter applies it.
3. **Recursive + multi-target + group column**: `P`/`O` apply to the marked set (`f.targets()`); recursive toggle `r` inside the perm editor — on stage, a dir expands to one op per entry (`fs.WalkDir`, symlinks skipped for chmod) with a count-confirm dialog; group column added to listings (GID + `GroupName` already exist), group shown in `entryMetaLine`.

## Part 5 — Docs & tests
- **ADR-0013** (new, append-only): "Account/file-ownership mutations shell out to admin tools behind an exec seam; mutating UI gates on root; system accounts are guarded."
- Tests: guard-rail table tests + argv/stdin assertions via the seam (accounts); screen key-gating (not-root → toast, no dialog), form/confirm flows via the `feed()` helper; `ui.Form` navigation; OpChown stage/apply/label; octal buffer; recursive expansion over `t.TempDir()` trees; special-bit toggles. Existing gauntlets (glyph ASCII, chrome_wellformed, resize) cover new views automatically.
- Backlog entries for deferred items (checkbox-list membership editor, progress UI for large recursive batches extending L7, undocumented `P` binding fix). Ritual per standing rules: `scripts/check.sh` green → STATUS.md → commit per slice.

## Commit sequence
1. `feat(ui): form modal component`
2. `feat(accounts): root check, name validation, admin-group helpers`
3. `feat(accounts): user/group mutations via admin tools with guard rails`
4. `feat(users): user create/edit/delete, password, lock, expiry UI`
5. `feat(users): group CRUD + both-direction membership + sudo indicator`
6. `feat(files): staged OpChown + chown dialog`
7. `feat(files): special-bit rows + octal entry in perm editor`
8. `feat(files): recursive/multi-target perms + group column`
9. `docs: ADR-0013 + STATUS + backlog`

Parts 1–5 are the headline feature; 6–8 land Files-side power incrementally, each green on `scripts/check.sh` before commit.