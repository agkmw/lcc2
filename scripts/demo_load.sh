#!/usr/bin/env bash
# scripts/demo_load.sh — Generate controlled, safe, temporary activity for Overview graphs
set -euo pipefail

DURATION="${2:-8}"
ACTION="${1:-pulse}"
LOCK_FILE="/tmp/lcc_demo_load.pid"

stop_load() {
    if [ -f "$LOCK_FILE" ]; then
        PID="$(cat "$LOCK_FILE" 2>/dev/null || true)"
        if [ -n "$PID" ] && kill -0 "$PID" 2>/dev/null; then
            kill -9 "$PID" 2>/dev/null || true
        fi
        rm -f "$LOCK_FILE"
    fi
    pkill -f "lcc-load-pulse" 2>/dev/null || true
}

run_pulse() {
    stop_load
    echo "$$" > "$LOCK_FILE"

    echo "Pulsing system activity for ${DURATION}s..."

    # 1. CPU load generator: 2 threads spinning on sha256
    (
        exec -a lcc-load-pulse-cpu bash -c "
            END=\$((SECONDS + $DURATION))
            while [ \$SECONDS -lt \$END ]; do
                echo \$RANDOM | sha256sum >/dev/null
            done
        "
    ) &
    PID_CPU1=$!

    (
        exec -a lcc-load-pulse-cpu bash -c "
            END=\$((SECONDS + $DURATION))
            while [ \$SECONDS -lt \$END ]; do
                echo \$RANDOM | sha256sum >/dev/null
            done
        "
    ) &
    PID_CPU2=$!

    # 2. Disk I/O generator: writes small blocks to /tmp and removes them
    (
        exec -a lcc-load-pulse-io bash -c "
            END=\$((SECONDS + $DURATION))
            TMP_FILE=\$(mktemp /tmp/lcc_load_XXXXXX)
            while [ \$SECONDS -lt \$END ]; do
                dd if=/dev/urandom of=\"\$TMP_FILE\" bs=64k count=16 conv=fdatasync status=none 2>/dev/null || true
                sleep 0.2
            done
            rm -f \"\$TMP_FILE\"
        "
    ) &
    PID_IO=$!

    # 3. Localhost network activity: ping burst to loopback
    (
        exec -a lcc-load-pulse-net bash -c "
            ping -i 0.1 -c \$(( $DURATION * 10 )) 127.0.0.1 >/dev/null 2>&1 || true
        "
    ) &
    PID_NET=$!

    # Wait for duration and cleanup
    sleep "$DURATION"
    kill "$PID_CPU1" "$PID_CPU2" "$PID_IO" "$PID_NET" 2>/dev/null || true
    rm -f "$LOCK_FILE"
    echo "Activity pulse completed."
}

case "$ACTION" in
    pulse|start)
        if [ "$#" -gt 0 ] && [ "${3:-}" = "bg" ]; then
            run_pulse >/dev/null 2>&1 &
        else
            run_pulse
        fi
        ;;
    stop)
        stop_load
        echo "Load generators stopped."
        ;;
    *)
        echo "Usage: $0 [pulse|start|stop] [duration_seconds]"
        exit 1
        ;;
esac
