#!/usr/bin/env bash
# scripts/demo_setup.sh — Prepare a clean, deterministic demo environment for LCC2
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEMO_DIR="${DEMO_DIR:-$HOME/lcc2-demo}"
CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/lcc2"
STATE_FILE="$CONFIG_DIR/state.json"
BAK_FILE="$CONFIG_DIR/state.json.demo-bak"
PID_FILE="$DEMO_DIR/.demo_worker.pid"

# Colors for terminal output
BOLD="\033[1m"
GREEN="\033[32m"
YELLOW="\033[33m"
CYAN="\033[36m"
RED="\033[31m"
RESET="\033[0m"

echo -e "${BOLD}${CYAN}=== LCC2 Controlled Demo Environment Setup ===${RESET}"

# 1. Check prerequisites and compile binary if needed
echo -e "\n${BOLD}[1/5] Checking tools and building LCC2...${RESET}"

if ! command -v go >/dev/null 2>&1; then
    echo -e "${RED}[ERROR] 'go' is required to compile LCC2.${RESET}"
    exit 1
fi

mkdir -p "$ROOT_DIR/bin"
echo "Building bin/lcc2..."
go build -trimpath -o "$ROOT_DIR/bin/lcc2" "$ROOT_DIR/cmd/lcc2"
echo -e "${GREEN}[OK] Binary compiled at bin/lcc2${RESET}"

# Check optional accelerators
check_tool() {
    local name="$1"
    local desc="$2"
    if command -v "$name" >/dev/null 2>&1; then
        echo -e "  ${GREEN}✓${RESET} $name ($desc)"
    else
        echo -e "  ${YELLOW}!${RESET} $name ($desc) - optional; will degrade gracefully"
    fi
}

echo "Checking optional accelerators:"
check_tool "fd" "Files screen fast name search (f)"
check_tool "rg" "Files screen fast grep search (F)"
check_tool "gio" "Freedesktop trash backend"
check_tool "systemctl" "Services management screen (5)"

# Check terminal geometry
TERM_COLS="$(tput cols 2>/dev/null || echo 80)"
TERM_LINES="$(tput lines 2>/dev/null || echo 24)"
echo -e "Current terminal geometry: ${TERM_COLS}x${TERM_LINES}"
if [ "$TERM_COLS" -lt 64 ] || [ "$TERM_LINES" -lt 16 ]; then
    echo -e "${RED}[WARNING] Terminal is below minimum 64x16! Resize before starting demo.${RESET}"
elif [ "$TERM_COLS" -lt 100 ] || [ "$TERM_LINES" -lt 30 ]; then
    echo -e "${YELLOW}[NOTE] Terminal works best at ≥120x35 for wide split-pane layouts.${RESET}"
fi

# 2. Kill any stale worker from a previous run
echo -e "\n${BOLD}[2/5] Managing demo background processes...${RESET}"
if [ -f "$PID_FILE" ]; then
    OLD_PID="$(cat "$PID_FILE" 2>/dev/null || true)"
    if [ -n "$OLD_PID" ] && kill -0 "$OLD_PID" 2>/dev/null; then
        echo "Terminating stale demo worker (PID $OLD_PID)..."
        kill -9 "$OLD_PID" 2>/dev/null || true
    fi
    rm -f "$PID_FILE"
fi

# Also pkill any lingering lcc2-demo-worker
pkill -f "lcc2-demo-worker" 2>/dev/null || true

# Spawn deterministic background worker process for Processes screen demo
bash -c 'exec -a lcc2-demo-worker sleep 7200' &
WORKER_PID=$!
echo -e "${GREEN}[OK] Spawned demo target process:${RESET} 'lcc2-demo-worker' (PID: $WORKER_PID)"

# 3. Create sandbox directory structure and fixtures
echo -e "\n${BOLD}[3/5] Setting up sandbox fixtures in ${DEMO_DIR}...${RESET}"
rm -rf "$DEMO_DIR"
mkdir -p "$DEMO_DIR"/{src/nested/deep/tree,docs,config,scripts,assets}
echo "$WORKER_PID" > "$PID_FILE"

# Fixture 1: Go file (shows Chroma syntax highlighting)
cat << 'EOF' > "$DEMO_DIR/src/server.go"
package main

import (
	"fmt"
	"net/http"
	"time"
)

// Server handles health checks and demo status endpoints.
func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "status: ok - %s\n", time.Now().Format(time.RFC3339))
	})

	fmt.Println("LCC2 demo web service listening on :8080...")
	_ = http.ListenAndServe(":8080", mux)
}
EOF

# Fixture 2: Python worker script
cat << 'EOF' > "$DEMO_DIR/src/worker.py"
#!/usr/bin/env python3
"""LCC2 demo background task worker."""
import sys
import time

def process_tasks():
    print("Worker pool initialized with 4 threads.")
    while True:
        time.sleep(60)

if __name__ == "__main__":
    process_tasks()
EOF

# Fixture 3: Markdown architecture document
cat << 'EOF' > "$DEMO_DIR/docs/architecture.md"
# LCC2 System Architecture

LCC2 is a keyboard-first Linux utility TUI engineered for speed, safety, and reliability.

## Key Architectural Principles

1. **Pure Data Providers**: `internal/{files,proc,services,disk,accounts,sysinfo}` never import UI packages.
2. **Screens Own All UI State**: Section models own their Bubble Tea state; root chrome routes sections.
3. **Oil.nvim Staged Changes**: File mutations are never written immediately to disk; changes are queued, inspected, and committed as an atomic batch.
4. **Strict Glyphs & Geometry**: ASCII chrome only to eliminate East Asian Width (EAW) drift in tmux. Minimum 64x16 terminal floor.
EOF

# Fixture 4: JSON configuration
cat << 'EOF' > "$DEMO_DIR/config/service.json"
{
  "service": "lcc2-demo-worker",
  "environment": "demo-presentation",
  "log_level": "info",
  "limits": {
    "max_memory_mb": 512,
    "max_threads": 8
  },
  "features": {
    "staged_changes": true,
    "auto_refresh": true
  }
}
EOF

# Fixture 5: Executable shell script
cat << 'EOF' > "$DEMO_DIR/scripts/deploy.sh"
#!/usr/bin/env bash
# Demonstration deployment script
set -euo pipefail
echo "Deploying LCC2 demo services..."
exit 0
EOF
chmod +x "$DEMO_DIR/scripts/deploy.sh"

# Fixture 6: Text file with grep search targets & rename candidate
cat << 'EOF' > "$DEMO_DIR/notes.txt"
Presentation Runbook Notes
==========================
- Overview: show braille & block sparklines, toggle with 'g'
- Processes: filter for 'lcc2-demo-worker' and terminate safely
- Disks: show ncdu drilldown and graceful esc cancel
- Files: stage mkdir, rename notes.txt, recursive chmod, stage trash, then 'w' review!
- TODO: deploy v0.3.0 to production cluster
- Remember: nothing touches disk until you review and save!
EOF

# Fixture 7: Binary file (target for staged trash delete)
dd if=/dev/urandom of="$DEMO_DIR/assets/cache.bin" bs=1024 count=64 status=none

# Fixture 8: Deep tree files (target for recursive chmod blast-radius demo)
touch "$DEMO_DIR/src/nested/deep/tree/alpha.go"
touch "$DEMO_DIR/src/nested/deep/tree/beta.go"
touch "$DEMO_DIR/src/nested/deep/tree/gamma.go"

echo -e "${GREEN}[OK] Fixtures populated in ${DEMO_DIR}${RESET}"

# 4. Preset LCC2 session state for a deterministic initial screen
echo -e "\n${BOLD}[4/5] Configuring session state...${RESET}"
mkdir -p "$CONFIG_DIR"

if [ -f "$STATE_FILE" ] && [ ! -f "$BAK_FILE" ]; then
    cp "$STATE_FILE" "$BAK_FILE"
    echo "Backed up existing state to $BAK_FILE"
fi

cat << EOF > "$STATE_FILE"
{
 "screen": 0,
 "cwd": "$DEMO_DIR",
 "hidden": false,
 "sortKey": "name",
 "sortDesc": false
}
EOF
echo -e "${GREEN}[OK] Session preset: Start screen = 0 (Overview), Files CWD = ${DEMO_DIR}${RESET}"

# 5. Ready!
echo -e "\n${BOLD}[5/5] Setup complete!${RESET}"
echo -e "\n${BOLD}${GREEN}Ready for presentation:${RESET}"
echo -e "  1. Run: ${BOLD}./bin/lcc2${RESET}"
echo -e "  2. (Optional) Generate live CPU/Net activity in a side terminal:"
echo -e "     ${BOLD}./scripts/demo_load.sh pulse${RESET}"
echo -e "  3. Clean up after demo:"
echo -e "     ${BOLD}./scripts/demo_cleanup.sh${RESET}\n"
