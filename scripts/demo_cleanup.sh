#!/usr/bin/env bash
# scripts/demo_cleanup.sh — Safely clean up LCC2 demo environment and restore host state
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEMO_DIR="${DEMO_DIR:-$HOME/lcc2-demo}"
CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/lcc2"
STATE_FILE="$CONFIG_DIR/state.json"
BAK_FILE="$CONFIG_DIR/state.json.demo-bak"
PID_FILE="$DEMO_DIR/.demo_worker.pid"

BOLD="\033[1m"
GREEN="\033[32m"
YELLOW="\033[33m"
CYAN="\033[36m"
RESET="\033[0m"

echo -e "${BOLD}${CYAN}=== LCC2 Demo Environment Cleanup ===${RESET}"

# 1. Terminate background demo worker process
echo -e "\n${BOLD}[1/4] Terminating demo background processes...${RESET}"
if [ -f "$PID_FILE" ]; then
    PID="$(cat "$PID_FILE" 2>/dev/null || true)"
    if [ -n "$PID" ] && kill -0 "$PID" 2>/dev/null; then
        echo "Killing demo worker (PID $PID)..."
        kill -9 "$PID" 2>/dev/null || true
    fi
fi
pkill -f "lcc2-demo-worker" 2>/dev/null || true

# 2. Stop any active demo_load activity
"$ROOT_DIR/scripts/demo_load.sh" stop 2>/dev/null || true
echo -e "${GREEN}[OK] Demo processes terminated.${RESET}"

# 3. Restore session state
echo -e "\n${BOLD}[2/4] Restoring session state...${RESET}"
if [ -f "$BAK_FILE" ]; then
    mv "$BAK_FILE" "$STATE_FILE"
    echo -e "${GREEN}[OK] Restored original state from $BAK_FILE${RESET}"
else
    # Remove state if it was created specifically by demo setup
    rm -f "$STATE_FILE"
    echo -e "${YELLOW}[NOTE] Removed demo state.json (no prior backup existed).${RESET}"
fi

# 4. Remove sandbox files
echo -e "\n${BOLD}[3/4] Cleaning sandbox files...${RESET}"
if [ -d "$DEMO_DIR" ]; then
    rm -rf "$DEMO_DIR"
    echo -e "${GREEN}[OK] Removed $DEMO_DIR${RESET}"
fi

# 5. Done
echo -e "\n${BOLD}[4/4] Cleanup complete!${RESET}"
echo -e "${GREEN}Host system is clean and restored to its original state.${RESET}\n"
