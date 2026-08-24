# Status

Volatile — rewrite freely. Exactly three sections, always.

## Current state

Grounding pass landed (ADR-0010, 6 commits) after the user's first
live tmux session. EAW-Ambiguous glyphs banned (the services-bleed /
misalignment root cause), pane divider column added, help is a dim
backdrop painted by the final canvas pass, tmux resize forces full
repaints with a byte-equality ghosting test. Overview: braille line
graphs default (`g` toggles block, `LCC2_GRAPH` seeds), labeled net
graphs with pow2 scale steps, right-flush disk rows. Files got its
structural pass; disks main pane is now a pure table. Gate green.

## In progress

Nothing — awaiting the user's second live pass.

## Next action

User verifies in real tmux: bleed gone, divider reads clearly, help
dims fully edge-to-edge, braille renders on their font (`g` fallback),
zoom/split no longer ghosts. Then L8/L9 remain open; candidates next:
netPeak decay, tab-strip priority truncation.
