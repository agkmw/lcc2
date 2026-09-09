# lcc user manual

lcc is a keyboard-first Linux system utility TUI: one terminal app with six
sections — Overview, Processes, Disks, Files, Services, Users & Groups.
Everything is reachable from the keyboard, destructive actions ask first, and
the Files section never touches the disk until you tell it to.

## 1. Install & run

Requirements:

- Linux, a terminal at least **64×16** characters (more is better; the layout
  goes wide above 80 columns and stacked below).
- Go to build: `go build -o lcc ./cmd/lcc`, then `./lcc`. For development,
  `go run ./cmd/lcc`.
- Optional helpers, used when present:
  - `fd` — recursive filename search (Files: `f`)
  - `rg` (ripgrep) — content search (Files: `F`)
  - `gio` — preferred trash backend (without it the home trash is used)
  - `systemctl` — the Services section; without it that section shows a
    friendly "not available here" notice
  - `$EDITOR` / `$VISUAL` / `$PAGER` — external opener (Files: `e`,
    Services: `E`); falls back to `less -R`

Flags: `lcc --version` prints the version. `NO_COLOR` / `CLICOLOR` are
honored; a truecolor terminal looks best.

Root: run as your normal user for monitoring. Actions that need uid 0
(service control, user administration, chown) tell you "needs root -
restart the app with sudo" instead of failing halfway; killing another
user's process as non-root simply fails with a "cannot signal" toast. The
Users section is read-only until you are root.

### Session restore

On quit (and every minute) lcc saves the last active screen plus the Files
working state (directory, hidden-files flag, sort) to
`~/.config/lcc/state.json`. The next launch reopens where you left off.
Delete the file to reset.

## 2. The frame

```
 lcc │ 1 Overview │ 2 Processes │ 3 Disks │ 4 Files │ ...
 ────────────────────────────────────────────────────────────
                    (section body)
 ────────────────────────────────────────────────────────────
 [s] sort: name  [w] review 2  [?] keys (19)     ~/demo - 14:03
```

- **Tab strip** (top): six numbered sections; the active one is a bright
  chip. The Files tab grows a `*N` badge while changes are staged.
- **Status bar** (bottom): live key hints for the current screen left,
  context and a clock right. Busy screens show a compact state summary and
  a `? keys (n)` pointer; the full key list lives in the help overlay.
- **Toasts** appear over the body: info for 3 s, errors for 6 s.
- **Help overlay**: press `?` anywhere. Same fixed height on every
  screen; shows the global keys plus every key the current screen
  accepts. Scroll it with the mouse wheel or `ctrl+d` / `ctrl+u`
  (half page), `j`/`k` (line), `pgup`/`pgdn`. While it is open the
  wheel and keys scroll only the help — the screen behind does not
  move, and clicks do nothing. `tab`, `shift+tab` and `1`–`6` still
  switch sections and the list follows the new screen; `?`, `esc` or
  `q` closes it.

### Global keys

Root binds exactly this set, on every screen:

| key | action |
|---|---|
| `1`–`6` | jump to a section |
| `tab` / `shift+tab` | next / previous section |
| `?` | help overlay |
| `q`, `ctrl+c` | quit (session is saved) |

### List keys

Every screen with a table answers to `j`/`k`, `g`/`G` and `/`; the
`enter` and `esc` gestures differ per screen. The help overlay shows
only what applies on the current screen, using each screen's own
wording.

| key | action |
|---|---|
| `j`/`k` or arrows | move selection |
| `g` / `G` | jump to top / bottom |
| `/` | filter the list (type to narrow) |
| `enter` | open / confirm (per-screen verb in help) |
| `esc` | back / cancel where the screen defines one |

Mouse works too: click tabs to switch, click rows to select, wheel scrolls,
double-click acts like `enter` on Files and Disks rows. While a filter or
dialog has focus, the keyboard belongs to it — screen shortcuts are
suppressed so typing `d` in a filter never deletes anything. With the
help overlay open, the wheel scrolls the help and clicks do nothing
until you close it.

## 3. Sections

### 1 · Overview

A btop-style dashboard refreshed every second: CPU total graph plus
per-core sparklines and load averages, RAM and swap gauges, network
down/up rate graphs with an auto-scaling axis, and mount usage bars with
disk read/write rates.

| key | action |
|---|---|
| `r` | refresh immediately |
| `g` | toggle graph style: braille ↔ block |

`LCC_GRAPH=block` starts with block graphs.

### 2 · Processes

Process list (pid, user, cpu%, mem%, state, command) with a live detail
pane on the right for the cursor's process: parent, threads, file
descriptors, nice, start time, executable, working directory, full
command line. Refreshes every 3 s.

| key | action |
|---|---|
| `s` | cycle sort: cpu% → mem% → pid → name → user |
| `x` | terminate — sends SIGTERM after a confirm dialog |
| `K` | force kill — sends SIGKILL after a louder confirm |
| `r` | refresh now |

PID 1 is refused. Colours: cpu% goes amber ≥70 / red ≥90; mem% goes amber
≥40 / red ≥70; state letters highlight (R running, D disk wait, Z zombie).

### 3 · Disks

Two modes. **Filesystem list**: mount, device, type, size, use%. Press
`enter` or `l` on a mount to **analyze** it — an ncdu-style directory
scan that lists children by size and share %. Drill into directories
with `enter` / `l`; `esc` or `h` goes back up and also cancels an
in-flight scan; `r` re-scans. The preview pane shows a usage gauge and facts for the selected
filesystem or entry.

### 4 · Files

A two-pane file manager: listing left (name, size, mode, owner, group),
live preview right (wide terminals) or below (narrow). Directories carry a
`/`, symlinks a `@`, archives render red, executables green. The preview
shows the first 60 lines of a text file with line numbers, or the top
entries of a directory, or a metadata card for binary files.

#### Navigation

| key | action |
|---|---|
| `enter` / `l` | enter directory |
| `h` | parent directory |
| `ctrl+o` / `ctrl+i` | back / forward through visited directories |
| `t` | open the trash as a listing (once it exists) |
| `a` | view hidden files (persisted; head shows `view hidden`) |
| `s` / `S` | cycle sort (name → size → mtime) / reverse; dirs always first |
| `e` | open file in `$EDITOR`/`$VISUAL`, else a found vi/nvim/nano (lcc suspends cleanly) |
| `r` | re-list the current directory and refresh the preview |
| `f` | find files by name under the current dir (uses `fd`) |
| `F` | grep file contents (uses `rg`, case-insensitive) |
| `space` | mark / unmark entry (multi-select; esc clears) |
| `Y` | copy absolute path(s) to the system clipboard |

In find/grep mode a query bar sits above live results: type to search
(debounced), arrows steer the result cursor while every other key feeds
the query, the preview follows the cursor (grep highlights the hit line),
`enter` jumps to the hit — directories open, files and grep hits land the
cursor on the entry in the listing (the grep hit-line preview is kept) —
and `esc` exits the search. Results are capped (1000 filenames / 500
grep hits); the tally shows `1000+` when the cap truncated the list.

#### The staged-changes model

lcc borrows oil.nvim's idea: **nothing touches the disk until you
review and save.** Every mutation is queued; the queue is visible,
editable, and reversible at any time.

| key | action |
|---|---|
| `d` | stage trash for cursor / marked entries |
| `m` | stage directory create (a phantom row appears immediately) |
| `n` | stage empty-file create (same phantom model; `m` makes dirs only) |
| `R` | stage rename (prompt pre-filled with the current name) |
| `y` / `x` | copy / cut to the internal clipboard |
| `p` | stage paste (copy or move) of the clipboard into this directory; a
  name collision dedupes instead of erroring (`report`, `report.2`, …) |
| `P` | permission editor (below) |
| `O` | owner form — chown owner and/or group (root only) |
| `u` | undo the most recent staged op |
| `U` | discard the whole queue |
| `w` | review the queue, then save |

Queued changes are never a mystery:

- each affected row gets a glyph — `+` create, `-` delete, `>` rename,
  `~` chmod/chown — colour-coded by kind;
- the preview overlays pending state, e.g. `644 -> 600 (staged)`,
  `(staged trash)`, `(staged rename -> notes.txt)`;
- the Files tab shows a `*N` badge and the page head `N pending`;
- staged creates appear as bold phantom rows so they can be selected and
  undone like any other op.

`w` opens a **review**: every queued op on one scrollable list with
old → new detail (`chmod 644 -> 600 file`, `chown root:root file`). Then:

| key | action |
|---|---|
| `y` / `enter` | save all — ops apply one by one with a progress counter |
| `u` | undo last staged op (stay in review) |
| `U` | discard all |
| `j` / `k` | scroll |
| `n` / `esc` | back without saving |

Saving stops at the first failing op with an error toast; the not-yet-
applied remainder stays staged so you can fix and retry. Deleted files go
to the freedesktop trash (gio when present, else `~/.local/share/Trash`
with proper `.trashinfo` records) and are restorable from any file
manager. `t` opens the trash as a normal listing; deleting an entry
**inside** the trash purges it permanently. Inside the trash `R`
stages restore (the entry returns to its recorded original path,
recreating vanished directories; refuses to overwrite), and `esc` or
`h` returns straight to the directory you came from. The preview
shows where a trashed entry came from — two files trashed from
different places may both appear (the second as `foobar.2`: the
freedesktop trash requires unique names, records keep the originals)
— and the tally of a capped search shows `1000+` / `500+`. Cross-filesystem deletes are refused rather than silently
destroying data.

#### Permission editor (`P`)

An interactive rwx matrix for user/group/other plus a special-bits row
(setuid, setgid, sticky), with a live octal readout:

- `h`/`j`/`k`/`l` or arrows move; `space` toggles a bit;
- typing digits builds a mode in octal (`644`, `4755`) which overrides
  the grid; backspace edits it;
- `r` makes the change recursive — a directory expands to one staged
  chmod per contained entry, so the review shows the full blast radius;
- `enter` stages, `esc` cancels.

### 5 · Services

systemd unit list (unit, active, sub, boot, description) with the unit's
live status beside it: a structured card (state, since, pid, memory, cpu,
restart count) plus the last journal lines; raw `systemctl status` output
with highlighted tokens as fallback. Failed units render red and get a
FAILED banner. Auto-refreshes every 15 s.

| key | action |
|---|---|
| `s` / `t` / `R` | start / stop / restart (confirm dialog) |
| `e` / `D` | enable / disable (confirm dialog) |
| `E` | edit the unit file in your editor (toast reminds you about
  `systemctl daemon-reload`) |
| `r` | refresh the unit list now |

Stop / disable / restart get the danger-styled confirm. Requires
`systemctl`; elsewhere the section degrades to a notice.

### 6 · Users & Groups

One section, two lists — `s` switches between them. Users list name, uid,
gid, home and shell; the preview shows the effective class (human with
login, system account, or no login shell), sudo membership, and every
group the user belongs to. Groups list gid and members, with the sudo
group marked. System accounts (uid < 1000, except root) are dimmed; root
stays loud.

| key | (users tab) |
|---|---|
| `n` | new user (form: name, shell, home, primary group, extra groups, system flag) |
| `e` | edit shell / home / primary / supplementary groups |
| `p` | set password (typed twice, must match) |
| `L` / `u` | lock / unlock account |
| `x` | delete user, home kept |
| `X` | delete user **and** home directory |
| `E` | set expiry (YYYY-MM-DD; empty clears) |

| key | (groups tab) |
|---|---|
| `n` | new group (optional gid) |
| `x` | delete group |
| `a` / `d` | add / remove member |

Every mutation is root-gated and confirm- or form-driven. Without root the
section is read-only: the mutating hints collapse into a single `!` press
that explains you need sudo, instead of advertising keys that would only
error.

## 4. Troubleshooting

| symptom | meaning |
|---|---|
| "terminal too small" | resize to at least 64×16 |
| find/grep toast about a missing tool | install `fd` / `rg`, or search from a directory where they matter |
| "different filesystem — not trashed" | the file lives on another filesystem than the home trash; nothing was deleted |
| "needs root - restart the app with sudo" | that action requires uid 0 |
| Services says systemctl not found | this machine doesn't run systemd |
| graphs frozen at a huge scale | one giant transfer pins the network scale for the session; restart to reset |

Known gaps (tracked in `docs/backlog.md`): no cancel inside a single large
copy/move, and one giant transfer can pin the Overview network scale until
restart.

—
Sources of truth: `internal/screens/*.go` (keys and behaviour),
`internal/app/root.go` (chrome), `internal/files/*.go` (staging, trash,
permissions). If this manual and the code disagree, the code wins — then
file it in `docs/backlog.md`.
