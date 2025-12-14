#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/common.sh"

lines="${1:-50}"

if [[ ! -f "$ERROR_LOG" ]]; then
    log_warn "Error log not found: $ERROR_LOG"
    exit 1
fi

echo -e "${RED}=== Last $lines errors ===${NC}"
tail -n "$lines" "$ERROR_LOG"
