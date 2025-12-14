#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/common.sh"

lines="${1:-100}"

if [[ ! -f "$LOG_FILE" ]]; then
    log_warn "Log file not found: $LOG_FILE"
    exit 1
fi

echo -e "${CYAN}=== Last $lines lines ===${NC}"
tail -n "$lines" "$LOG_FILE"
