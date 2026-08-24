# lcc2

A keyboard-first Linux system utility TUI. Six sections — monitoring dashboard,
processes, disks, files, services, users & groups — in a single Go binary built
on [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Screenshots

TODO

## Screens

| Section | What it does |
|---|---|
| **1 Overview** | btop-style dashboard: CPU cores, memory gauges, network graphs (rolling auto-scale) and filesystem usage on one canvas. Refreshes every second. |
| **2 Processes** | Live process list with detail preview pane. Cycle sort columns, filter, send signals via confirm dialogs. |
| **3 Disks** | Filesystem usage overview; `enter` runs a du-style size scan of a mountpoint, then drill down directory by directory. |
| **4 Files** | File manager rooted at `$HOME` with an oil.nvim-style staging model: every mutation (delete/mkdir/rename/copy/move/chmod) is staged and only applied on save (`w`). Optional `fd` find and `rg` grep modes with jump-to-line previews. |
| **5 Services** | systemd units with start/stop/restart/enable/disable (each confirmed). Requires `systemctl` on PATH. |
| **6 Users** | Users and groups side by side with a detail pane; system accounts (uid/gid < 1000, except root) render dimmed. |

## Requirements

- Linux (providers read `/proc`, `systemctl`, etc.)
- Go 1.26+ to build
- Terminal of at least 64x16 — below that a friendly notice appears
- Optional: [`fd`](https://github.com/sharkdp/fd) and
  [`ripgrep`](https://github.com/BurntSushi/ripgrep) enable the Files search
  modes; systemd enables the Services screen
- Privileges apply naturally: signalling other users' processes, service
  control and writing outside your home need the usual rights

## Install & Run

```sh
make run        # go run ./cmd/lcc2
make build      # binary lands in bin/lcc2
make install    # go install ./cmd/lcc2 (respects $GOBIN)
```

Or without make:

```sh
go run ./cmd/lcc2
go build -o lcc2 ./cmd/lcc2
```

## Keybindings

`?` always shows live help for the active screen.

### Global

| Key | Action |
|---|---|
| `tab` / `shift+tab` | next / previous section |
| `1`–`6` | jump to section |
| `j` / `k` | move selection |
| `/` | filter the focused list |
| `enter` | select / open |
| `esc` | back / cancel |
| `?` | help overlay |
| `q` | quit |

### Per section

| Key | Screen | Action |
|---|---|---|
| `r` | Overview | force refresh |
| `g` | Overview | toggle graph style (braille/block) |
| `s` | Processes | cycle sort column (cpu%, mem%, pid, name, user) |
| `x` | Processes | terminate (SIGTERM, confirmed) |
| `K` | Processes | force kill (SIGKILL, confirmed) |
| `enter` | Disks | analyze mountpoint / drill into directory |
| `h` `esc` | Disks | back out of a scan |
| `enter` `l` | Files | open directory / reveal search hit |
| `h` | Files | parent directory |
| `a` | Files | toggle hidden files |
| `space` | Files | mark / unmark (multi-select; `esc` clears) |
| `f` | Files | find mode (`fd`) — live results, `enter` reveals |
| `F` | Files | grep mode (`rg`) — live hits, `enter` jumps to line |
| `d` | Files | stage delete |
| `m` | Files | stage create directory |
| `R` | Files | stage rename |
| `y` / `x` / `p` | Files | copy / cut / paste (staged at current directory) |
| `P` | Files | permission editor (applied on save) |
| `u` / `U` | Files | undo last staged op / discard all staged |
| `w` | Files | save — apply everything staged |
| `s` / `t` / `r` | Services | start / stop / restart unit (confirmed) |
| `e` / `D` | Services | enable / disable unit (confirmed) |
| `s` | Users | switch users / groups list |

## Environment Variables

| Variable | Effect |
|---|---|
| `LCC2_GRAPH=block` | use block-style sparkline bars instead of the default braille graphs |

## Project Structure

```
cmd/lcc2          entrypoint; wires the six screens into app.New
internal/app      root model: chrome (tab strip/status bar), section
                  routing, toasts, help overlay
internal/screens  section models; own all Bubble Tea state
internal/ui       design system: FilterTable, ConfirmDialog, theme,
                  keymap, canvas
internal/files    file operations, staging, find/grep integration
internal/proc     process collection from procfs
internal/services systemctl wrapper
internal/disk     filesystem listing and directory-size scanning
internal/accounts users & groups
internal/sysinfo  cpu/memory/network/host sampling
```

Two rules hold the architecture together:

- **Providers never import UI** — the data packages above are pure; screens
  consume them through Elm-style messages.
- **Screens own all UI state** — the root model only routes sections and
  paints chrome around whatever the active screen renders.

Design decisions are recorded as append-only ADRs in `docs/decisions/`.

## Development

```sh
make check   # session gate: go vet ./... && go test ./...
make fmt     # go fmt ./...
make cover   # coverage report
```

Repository docs: `docs/STATUS.md` (current state),
`docs/backlog.md` (noticed-but-unfixed problems),
`docs/experiments.md` (failed experiments), `docs/decisions/` (ADRs).

## Limitations

- Linux only
- Fixed Catppuccin Mocha palette; no theme switching or `NO_COLOR` support yet
- No mouse support, no CLI flags (`--version`/`--help` missing)
- Large single-file copies show no byte-level progress and cannot be cancelled
  once started (whole staged batches are stop-on-error per operation)
