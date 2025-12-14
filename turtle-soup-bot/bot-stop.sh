#!/bin/bash
# Turtle Soup Bot - Stop Script
# Version: 1.0

set -eo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$PROJECT_DIR"

# shellcheck source=/dev/null
. "$PROJECT_DIR/bot-common.sh"

#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Kill process helper (allows failure)
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

kill_process() {
    local pid=$1
    local signal=$2

    kill "-$signal" "$pid" 2>/dev/null || true
}

#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Main
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

echo -e "${GREEN}[TURTLE-SOUP]${NC} Stopping Turtle Soup Bot..."

# Check PID file
if [ ! -f "$PID_FILE" ]; then
    echo -e "${YELLOW}[WARN]${NC} PID file not found. Service may not be running."

    # Cleanup orphan processes using pgrep
    ORPHAN_PIDS=$(pgrep -f "$APP_PATTERN" || true)

    if [ -n "$ORPHAN_PIDS" ]; then
        echo -e "${YELLOW}[INFO]${NC} Found orphan process(es): $ORPHAN_PIDS"
        echo -e "${YELLOW}[INFO]${NC} Killing orphan processes..."

        # Try graceful shutdown first
        for pid in $ORPHAN_PIDS; do
            kill_process "$pid" 15
        done

        # Wait for termination
        ORPHAN_WAIT=0
        MAX_ORPHAN_WAIT=10
        while [ $ORPHAN_WAIT -lt $MAX_ORPHAN_WAIT ]; do
            STILL_RUNNING=$(pgrep -f "$APP_PATTERN" || true)
            if [ -z "$STILL_RUNNING" ]; then
                break
            fi
            sleep 1
            ORPHAN_WAIT=$((ORPHAN_WAIT + 1))
        done

        # Force kill if still running
        STILL_RUNNING=$(pgrep -f "$APP_PATTERN" || true)
        if [ -n "$STILL_RUNNING" ]; then
            echo -e "${RED}[WARN]${NC} Orphan graceful shutdown timeout. Force killing..."
            for pid in $STILL_RUNNING; do
                kill_process "$pid" 9
            done
        fi

        echo -e "${GREEN}[SUCCESS]${NC} Orphan process(es) terminated"
    else
        echo -e "${GREEN}[INFO]${NC} No running process found"
    fi

    exit 0
fi

# Read PID
PID=$(cat "$PID_FILE")

# Check if process is running
if ! ps -p "$PID" > /dev/null 2>&1; then
    echo -e "${YELLOW}[WARN]${NC} Process (PID: $PID) is not running"
    rm -f "$PID_FILE"
    echo -e "${GREEN}[INFO]${NC} Cleaned up stale PID file"
    exit 0
fi

# Graceful shutdown (SIGTERM)
echo -e "${GREEN}[INFO]${NC} Sending SIGTERM to process (PID: $PID)..."
kill_process "$PID" 15

# Wait for process termination
WAIT_COUNT=0
MAX_WAIT=20

while ps -p "$PID" > /dev/null 2>&1; do
    if [ $WAIT_COUNT -ge $MAX_WAIT ]; then
        echo -e "${RED}[WARN]${NC} Graceful shutdown timeout. Force killing..."
        kill_process "$PID" 9
        sleep 1
        break
    fi
    echo -e "${YELLOW}[INFO]${NC} Waiting for shutdown... ($WAIT_COUNT/$MAX_WAIT)"
    sleep 1
    WAIT_COUNT=$((WAIT_COUNT + 1))
done

# Cleanup PID file
rm -f "$PID_FILE"

# Final status
if ps -p "$PID" > /dev/null 2>&1; then
    echo -e "${RED}[ERROR]${NC} Failed to stop process (PID: $PID)"
    exit 1
else
    echo -e "${GREEN}[SUCCESS]${NC} Service stopped successfully"
fi
