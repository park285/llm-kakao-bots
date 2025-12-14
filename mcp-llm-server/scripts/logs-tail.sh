#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/common.sh"

if [[ ! -f "$LOG_FILE" ]]; then
    log_warn "Log file not found: $LOG_FILE"
    exit 1
fi

echo -e "${CYAN}=== Tailing $LOG_FILE (Ctrl+C to exit) ===${NC}"
tail -f "$LOG_FILE"
