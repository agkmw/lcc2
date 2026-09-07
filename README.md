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

## Requirements & Prerequisites

### Platform Requirements & Limitations
- **Linux only**: `lcc2` directly inspects Linux `/proc` (CPU, memory, load, processes) and `/sys` (disk I/O rates), interacts with `systemd`, and executes standard Linux account management utilities. macOS, Windows, and BSDs are not supported.
- **Terminal geometry**: Minimum size of **64×16** characters (a graceful fallback notice is displayed below this floor). A truecolor terminal is recommended.
- **Permissions**:
  - Unprivileged user: Monitoring (Overview dashboard, Processes, Disks, Files browsing/staging, read-only Users view).
  - Elevated (`sudo` / root): Required for service management (`systemctl`), user & group administration (`useradd`, `usermod`, etc.), and sending signals to processes owned by other users.

### Dependencies
- **Build toolchain**: Go 1.26+ (compiles as a single pure Go binary with `CGO_ENABLED=0`, no C runtime dependencies).
- **Helper tools (optional, recommended for full features)**:
  - [`fd`](https://github.com/sharkdp/fd): Enables fast filename search (Files screen `f`). On Debian/Ubuntu the binary is `fdfind` — the app falls back to it automatically, and the installer also links `fd`.
  - [`ripgrep`](https://github.com/BurntSushi/ripgrep) (`rg`): Enables fast regex/text content search (Files screen `F`).
  - `gio` (via GLib): Enables cross-filesystem Freedesktop trash support (Files screen `d`). Without `gio`, deletions fall back to `~/.local/share/Trash` within the same filesystem.
  - `wl-clipboard` (`wl-copy`) or `xclip`: System clipboard support (Files screen `Y`). Falls back to terminal OSC 52 if missing.
  - `systemctl` & `journalctl` (systemd): Powers the Services screen. Without systemd, the screen displays a notice that service inspection is unavailable.
  - `$EDITOR` / `$VISUAL`: External file and unit editor (Files `e`, Services `E`). Without them the app probes `nvim`/`vim`/`vi`/`nano`/`editor`; with none installed it shows a hint instead of opening a read-only pager.

## Installation

### Automated Installation (Recommended)

To install everything required (missing package dependencies, helper tools, and the compiled `lcc2` binary) in a single command, run:

```sh
git clone https://github.com/agkmw/lcc2.git && cd lcc2 && ./scripts/install.sh
```

Or if you have already cloned the repository:

```sh
./scripts/install.sh
```

**What the automated installer does:**
1. Verifies the host is running Linux and checks architecture.
2. Detects the system package manager (`apt`, `dnf`, `pacman`, `zypper`, `apk`).
3. Automatically installs missing dependencies (`go`, `fd`/`fd-find`, `ripgrep`, `gio`/GLib, clipboard utilities `wl-clipboard`/`xclip`, `less`, `nano` when no editor exists).
4. Automatically resolves the Debian/Ubuntu `fdfind` binary name by linking `fd` in the binary path.
5. Builds a standalone, trimmed binary (`CGO_ENABLED=0`) and installs it into `~/.local/bin/lcc2` (or `/usr/local/bin` if run as root).
6. Verifies your `$PATH` and suggests shell configuration if needed.

**Installer options:**
```sh
./scripts/install.sh --help              # Show usage and options
./scripts/install.sh --bin-dir /usr/bin  # Custom binary destination
./scripts/install.sh --no-deps           # Build and install without invoking package manager
./scripts/install.sh -y                  # Non-interactive mode
```

### Manual Installation

If you prefer installing dependencies manually:

1. **Install required packages for your distribution:**

   - **Debian / Ubuntu**:
     ```sh
     sudo apt update && sudo apt install -y golang fd-find ripgrep libglib2.0-bin wl-clipboard xclip less
     # Ensure 'fd' is callable (Debian names it 'fdfind'):
     mkdir -p ~/.local/bin && ln -sf "$(which fdfind)" ~/.local/bin/fd
     ```
   - **Fedora / RHEL**:
     ```sh
     sudo dnf install -y golang fd-find ripgrep glib2 wl-clipboard xclip less
     ```
   - **Arch Linux**:
     ```sh
     sudo pacman -S --needed go fd ripgrep glib2 wl-clipboard xclip less
     ```
   - **openSUSE**:
     ```sh
     sudo zypper install -y go fd ripgrep glib2 wl-clipboard xclip less
     ```
   - **Alpine Linux**:
     ```sh
     sudo apk add go fd ripgrep glib-tools wl-clipboard xclip less
     ```

2. **Build and install the binary:**

   ```sh
   # Using Make:
   make build      # outputs to bin/lcc2
   make install    # installs to $GOBIN

   # Or directly using Go:
   go build -trimpath -o ~/.local/bin/lcc2 ./cmd/lcc2
   ```

### Running lcc2

```sh
# Normal mode (monitoring, browsing, staging changes)
lcc2

# Elevated mode (service control, account/group management, killing other users' processes)
sudo lcc2
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

- Linux only (relies on `/proc`, `/sys`, systemd, and standard Linux account tools)
- Fixed Catppuccin Mocha palette; no runtime theme switching yet (`NO_COLOR` and `CLICOLOR` are supported)
- Large single-file copies show no byte-level progress and cannot be cancelled
  once started (whole staged batches are stop-on-error per operation)
