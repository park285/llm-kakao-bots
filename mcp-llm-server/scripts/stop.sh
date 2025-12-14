#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/common.sh"

if ! is_running; then
    log_warn "Not running"
    rm -f "$PID_FILE"
    exit 0
fi

pid=$(get_pid)
log_info "Stopping (PID: $pid)..."
kill "$pid" 2>/dev/null || true

for _ in {1..10}; do
    is_running || break
    sleep 1
done

if is_running; then
    log_warn "Force killing..."
    kill -9 "$pid" 2>/dev/null || true
fi

rm -f "$PID_FILE"
log_info "Stopped"
