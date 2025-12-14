#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/common.sh"

if [[ ! -d "$LOG_DIR" ]]; then
    log_warn "Log directory not found"
    exit 0
fi

count=$(find "$LOG_DIR" -name "*.gz" -type f 2>/dev/null | wc -l)
if [[ $count -gt 0 ]]; then
    find "$LOG_DIR" -name "*.gz" -type f -delete
    log_info "Deleted $count compressed log files"
else
    log_info "No compressed logs to clean"
fi
