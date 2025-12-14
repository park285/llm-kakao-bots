#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/common.sh"

if is_running; then
    pid=$(get_pid)
    log_info "Running (PID: $pid)"
    ps -p "$pid" -o pid,user,%cpu,%mem,start,time 2>/dev/null || true
    echo ""
    [[ -f "$LOG_FILE" ]] && log_info "Log: $LOG_FILE ($(du -h "$LOG_FILE" | cut -f1))"
else
    log_warn "Not running"
    exit 1
fi
