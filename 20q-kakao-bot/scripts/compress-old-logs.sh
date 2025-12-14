#!/bin/bash
# Archive Old Logs - Compress each non-current log file individually
# Version: 2.0

set -eo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
LOG_DIR="$PROJECT_DIR/logs"

TODAY=$(date +%Y-%m-%d)
ARCHIVE_DIR="$LOG_DIR/archived"

# Create archived directory if it doesn't exist
mkdir -p "$ARCHIVE_DIR"

echo "[INFO] Compressing old log files individually (excluding today: $TODAY)"

# Find log files to compress (exclude current and today's logs)
LOG_FILES=$(find "$ARCHIVE_DIR" -maxdepth 1 -type f \
    -name "*.20[0-9][0-9]-[0-1][0-9]-[0-3][0-9].*.log" \
    ! -name "*$TODAY*" \
    2>/dev/null || true)

if [ -z "$LOG_FILES" ]; then
    echo "[INFO] No old log files to compress"
    exit 0
fi

# Count files
FILE_COUNT=$(echo "$LOG_FILES" | wc -l)
echo "[INFO] Found $FILE_COUNT log file(s) to compress"

COMPRESSED_COUNT=0
TOTAL_SIZE_SAVED=0

# Compress each file individually
while IFS= read -r log_file; do
    if [ ! -f "$log_file" ]; then
        continue
    fi

    ORIGINAL_SIZE=$(stat -c%s "$log_file" 2>/dev/null || stat -f%z "$log_file" 2>/dev/null)

    tar -czf "$log_file.tar.gz" -C "$(dirname "$log_file")" "$(basename "$log_file")"

    if [ -f "$log_file.tar.gz" ]; then
        rm -f "$log_file"
        COMPRESSED_SIZE=$(stat -c%s "$log_file.tar.gz" 2>/dev/null || stat -f%z "$log_file.tar.gz" 2>/dev/null)
        SIZE_SAVED=$((ORIGINAL_SIZE - COMPRESSED_SIZE))
        TOTAL_SIZE_SAVED=$((TOTAL_SIZE_SAVED + SIZE_SAVED))
        COMPRESSED_COUNT=$((COMPRESSED_COUNT + 1))
        echo "[SUCCESS] $(basename "$log_file") → $(basename "$log_file.tar.gz")"
    else
        echo "[ERROR] Failed to compress: $log_file"
    fi
done <<< "$LOG_FILES"

if [ $COMPRESSED_COUNT -gt 0 ]; then
    SAVED_MB=$((TOTAL_SIZE_SAVED / 1024 / 1024))
    echo "[COMPLETE] Compressed $COMPRESSED_COUNT file(s), saved ~${SAVED_MB}MB"
else
    echo "[INFO] No files were compressed"
fi
