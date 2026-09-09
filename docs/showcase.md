# Showcasing lcc — a 12-minute live demo runbook

This is a timed talk track for presenting lcc in front of an audience:
what to say, what to type, what to point at, and what to do when the demo
gods misbehave. A 5-minute lightning cut is at the end.

The one sentence to land: **lcc is one keyboard-first terminal app that
replaces the six tools you keep opening — htop, ncdu, a file manager,
systemctl, passwd — with consistent keys and a file manager that never
touches the disk until you review and save.**

The demo's emotional arc: *pretty dashboard* → *fast control* → **the
staged-files wow moment** → *trust* (confirms, trash, root gates) →
*polish* (session restore).

## 0. Before you present

Run this the day before AND an hour before:

```sh
go build -o lcc ./cmd/lcc && ./lcc --version   # builds, prints version
which fd rg gio systemctl                          # note what's missing
echo $EDITOR                                       # set it if empty
```

Prepare a sandbox home for the Files demo — never your real home:

```sh
mkdir -p ~/lcc-demo/{src,docs,assets}
cd ~/lcc-demo
echo "func main() {}" > src/main.go
echo "# demo" > docs/readme.md
echo "TODO: present lcc" > notes.txt            # grep target
dd if=/dev/urandom of=assets/blob.bin bs=1k count=64
mkdir -p src/deep/nested/tree                    # recursive-chmod target
touch src/deep/nested/tree/{a,b,c}.go
sleep 100000 & echo $!                           # process to kill later
```

Stage checks:

- Terminal: ≥120×35, truecolor, a Nerd-font-free monospace (lcc is
  deliberately glyph-safe), notifications silenced.
- `./lcc` once, walk all six tabs, quit. Session restore now opens Files
  in your sandbox — set it up before the audience arrives.
- If you'll show Users mutations, decide the machine is disposable; the
  runbook below stays read-only there and makes that a feature.
- Rehearse the Files beat twice. It is the demo.

## 1. The runbook

Timings assume 12 minutes plus Q&A. Keys are exact; SAY lines are cues,
not scripts — use your own words.

### Beat 1 · Hook (0:00–1:00)

DO: launch `./lcc`. Let the Overview draw. Then press `?`.

SAY: "This is lcc — every Linux admin's morning routine in one TUI.
Six sections, and every key you'll ever need is one keypress away —
that's the help overlay, per screen, always current."

POINT AT: the tab strip ("a window manager for system tasks"), the
status bar ("hints always visible; the bar never overflows"), the help
overlay ("I never have to memorize — press ? anywhere").

DO: `esc` to close.

### Beat 2 · Overview (1:00–2:30)

SAY: "A btop-style dashboard, refreshed every second, in pure Go."

DO: let the CPU and network graphs run for a few seconds. Generate load
in your spare terminal (`yes > /dev/null &`) and point at the CPU graph
reacting; `kill %1` it. Press `g` to switch braille → block graphs.

POINT AT: per-core sparklines, the RAM/swap gauges, the network
auto-scaling axis ("the scale snaps to powers of two so the label is
always readable"), mount bars with live disk r/w rates.

IF IT FAILS: graphs empty → press `r`.

### Beat 3 · Processes (2:30–4:00)

DO: `2`. Type `/` and filter for `sleep`. Move to your `sleep 100000`.

SAY: "Filtering is one key, and while I'm typing, the filter owns the
keyboard — `d` here types a letter, it doesn't delete."

DO: `esc` to leave the filter. Point at the preview pane (parent, threads,
fds, nice, exe, cwd, full cmdline). Then `x`:

SAY: "Terminate asks first. Force kill asks louder."

DO: confirm. Toast: `terminated <pid>`.

POINT AT: sort cycling with `s` (cpu% → mem% → pid → name → user), and
mention: "PID 1 is hard-refused."

### Beat 4 · Files — the star (4:00–8:30)

DO: `4`. You are in `~/lcc-demo` (session restore). Preview follows the
cursor — walk a few rows.

SAY: "Now the part I actually care about. This file manager is built on
one rule: nothing touches the disk until you review and save. Every
change is staged, visible, and undoable — like git add for your
filesystem."

DO, narrating each step:

1. `m`, type `build`, `enter` → "I just staged a mkdir. See the phantom
   row — bold, with a green plus. It doesn't exist yet; the listing is
   showing me the *future*."
2. Move to `notes.txt`, `R`, edit the name to `todo.txt`, `enter` →
   "staged rename — yellow `>` glyph."
3. Move to `src`, `P` → "an interactive permission matrix, with the
   octal live. I'll toggle write for others… actually, let me do this
   recursively: `r`, then type `700`, `enter`."
   POINT AT the toast: "staged chmod 700 — N paths. A directory expands
   to one op per contained entry, so the review will show the full blast
   radius. No surprises."
4. Move to `assets/blob.bin`, `d` → "staged trash — red minus. And it's
   trash, not rm: recoverable from any file manager."
5. Stop. POINT AT: the `*N` badge on the Files tab, the `N pending` in
   the page head, and the preview overlays (`644 -> 700 (staged)`).

DO: `w` — the review.

SAY: "Before anything applies, here is every queued change, old to new,
scrollable. This is the one gate — there is no second confirm dialog,
because I've already seen everything."

DO: `u` once ("undo the last op — the trash is unstaged"), then `j`/`k`
to scroll, then `y`.

POINT AT: per-op progress in the header (`saving i/n`), then the toast
`saved N changes`, glyphs gone, `build/` real, `todo.txt` renamed.

DO: `F`, type `todo` → "content search via ripgrep, live as I type."
Move the cursor: "the preview opens at the hit line." `enter` reveals
the file's directory. `esc`.

IF IT FAILS: missing `rg`/`fd` → say "those are optional accelerators,
the core is pure Go" and skip to the review beat. Saving error → `u`
unstages; the queue keeps the rest.

### Beat 5 · Disks (8:30–9:45)

DO: `3`, `enter` on a mount (pick a small one, not `/` — or do `/` and
let it run while you talk, then `esc` to cancel).

SAY: "ncdu lives here too. Enter analyzes a mount; drill in with enter,
esc backs out — and esc cancels a running scan. Nothing locks up."

POINT AT: share-% gauges in the preview, the cancel story.

### Beat 6 · Services (9:45–11:00)

DO: `5`. Move to a harmless unit you picked in rehearsal (a user timer
or ssh on a VM). `R` for restart, confirm.

POINT AT: the live status card (state, since, pid, memory, restarts),
last journal lines under it, failed units in red, `E` opening the unit
file in your editor with a daemon-reload reminder toast.

SAY: "systemctl status, journalctl, and edit — without leaving the
keyboard. Dangerous ops get the danger-styled confirm."

IF IT FAILS: no systemd → say so and show the graceful notice itself:
"it degrades honestly."

### Beat 7 · Users & close (11:00–12:00)

DO: `6`. Point at the users list, system accounts dimmed, preview with
login-class and sudo membership. As non-root, press `n`:

SAY: "Mutations are root-gated — and the app doesn't advertise keys
that would only fail; the bar collapses to this single `!` reminder.
As root you get full administration: create, lock, expiry, delete with
or without home."

DO: `?` one more time, then `q`. Relaunch `./lcc`.

SAY: "It saved my screen, my directory, my sort. I'm back exactly where
I left off. That's lcc — one binary, six tools, nothing destructive
without a review."

## 2. The 5-minute lightning cut

Hook (30 s, with `?`) → Overview (45 s, `g` toggle) → **Files only**
(2:30: mkdir + recursive chmod + `w` review + `y` save, then `F` grep)
→ quit/relaunch for session restore (30 s). Drop Processes, Disks,
Services, Users; mention them in one breath over the tab strip.

## 3. Q&A preparation

- **Why staged changes instead of applying immediately?** Batch edits
  need a batch review. You stage ten changes across five directories and
  check them as one unit — and `u`/`U` undo before anything happens.
  rm has no undo; this model does.
- **What are the dependencies?** The core is pure Go reading /proc,
  /sys and /etc. `fd`, `rg`, `gio`, systemctl are optional accelerators;
  each absence degrades a feature, not the app.
- **Does it work over SSH?** It's a TUI in the alt-screen — that *is*
  the SSH workflow. Mouse support is there when the terminal has it.
- **How is this different from htop/btop/ranger/ncdu?** Those are six
  mental models. lcc is one: same nav keys, same filter, same preview
  pattern in every section, and cross-section work (find a file → see
  its owner → check their processes) without switching apps.
- **Safety?** Confirms on signals and service ops, danger styling on
  destructive ones, trash instead of delete, cross-filesystem deletes
  refused, overwrites refused, PID 1 refused, root gates, and the Files
  review gate.
- **Terminal requirements?** 64×16 floor, degrades by priority — badges
  drop first, then numbers, then labels shrink. No special fonts; the
  glyphs are chosen to survive tmux and CJK locales.
- **What's next?** Be honest: cancel inside a single big copy, and the
  Overview network scale can stay pinned after a huge transfer. They're
  tracked in the public backlog with pointers.

## 4. Presentation craft

- Type slowly and say the key out loud before pressing it. The audience
  is learning a keyboard, not watching a screen.
- Use `?` deliberately twice — opening and closing the demo — so the
  audience leaves knowing the escape hatch.
- Never touch your real home directory on stage; the sandbox is the
  story (you can even say so).
- Let toasts finish. They're part of the feedback loop — narrate them.
- If everything freezes, `ctrl+c` quits cleanly, session restores, and
  that recovery *is* a feature moment.
