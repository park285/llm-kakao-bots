#!/bin/bash
# 20Q Kakao Bot - Stop Script with Gemini Cache Cleanup
# Version: 3.0

set -eo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$PROJECT_DIR"

# shellcheck source=/dev/null
. "$PROJECT_DIR/bot-common.sh"

#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Load environment variables
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

load_env_vars() {
    if ! load_env_file; then
        echo -e "${YELLOW}[WARN]${NC} .env file not found, skipping cache cleanup"
        return 1
    fi

    # Extract API key
    if [ -n "${GOOGLE_API_KEY:-}" ]; then
        API_KEY="$GOOGLE_API_KEY"
    elif [ -n "${GOOGLE_API_KEYS:-}" ]; then
        API_KEY=$(echo "$GOOGLE_API_KEYS" | tr ',\n' ' ' | awk '{print $1}')
    else
        echo -e "${YELLOW}[WARN]${NC} No API key found, skipping cache cleanup"
        return 1
    fi

    # Extract Cache (Valkey) config
    CACHE_HOST="${CACHE_HOST:-localhost}"
    CACHE_PORT="${CACHE_PORT:-6379}"

    return 0
}

#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Cleanup Gemini Context Caches
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

cleanup_gemini_caches() {
    echo -e "${BLUE}[CACHE]${NC} Starting Gemini cache cleanup..."

    # Load environment
    if ! load_env_vars; then
        echo -e "${YELLOW}[CACHE]${NC} Skipping cache cleanup"
        return 0
    fi

    # Query Redis for cache IDs
    local CACHE_CMD="redis-cli -h $CACHE_HOST -p $CACHE_PORT"
    if [ -n "${CACHE_PASSWORD:-}" ]; then
        CACHE_CMD="$CACHE_CMD -a $CACHE_PASSWORD"
    fi

    local CACHE_KEYS
    CACHE_KEYS=$($CACHE_CMD --raw KEYS "20q:ai:gemini-cache-id:*" 2>/dev/null || echo "")

    if [ -z "$CACHE_KEYS" ]; then
        echo -e "${GREEN}[CACHE]${NC} No caches to clean up"
        return 0
    fi

    local CACHE_COUNT
    CACHE_COUNT=$(echo "$CACHE_KEYS" | wc -l)
    echo -e "${BLUE}[CACHE]${NC} Found $CACHE_COUNT cache(s)"

    local SUCCESS=0
    local FAILED=0

    # Delete each cache
    for KEY in $CACHE_KEYS; do
        local CACHE_ID
        CACHE_ID=$($CACHE_CMD --raw GET "$KEY" 2>/dev/null || echo "")

        if [ -z "$CACHE_ID" ]; then
            continue
        fi

        # Delete from Gemini API
        local HTTP_CODE
        HTTP_CODE=$(curl -s -m 5 -o /dev/null -w "%{http_code}" -X DELETE \
            "https://generativelanguage.googleapis.com/v1beta/cachedContents/$CACHE_ID?key=$API_KEY" \
            2>/dev/null || echo "000")

        if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
            # Delete from Redis
            $CACHE_CMD DEL "$KEY" >/dev/null 2>&1 || true
            SUCCESS=$((SUCCESS + 1))
            echo -e "${GREEN}[CACHE]${NC} Deleted: $(echo "$KEY" | sed 's/20q:ai:gemini-cache-id://')"
        elif [ "$HTTP_CODE" = "404" ]; then
            # Cache already deleted or expired
            $CACHE_CMD DEL "$KEY" >/dev/null 2>&1 || true
            SUCCESS=$((SUCCESS + 1))
            echo -e "${BLUE}[CACHE]${NC} Already deleted: $(echo "$KEY" | sed 's/20q:ai:gemini-cache-id://')"
        else
            FAILED=$((FAILED + 1))
            echo -e "${YELLOW}[CACHE]${NC} Failed to delete: $CACHE_ID (HTTP $HTTP_CODE)"
        fi
    done

    echo -e "${BLUE}[CACHE]${NC} Cleanup complete: success=$SUCCESS, failed=$FAILED"
}

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

echo -e "${GREEN}[20Q-BOT]${NC} Stopping Iris 20 Questions Service..."

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

    # Cleanup caches even without PID
    cleanup_gemini_caches

    exit 0
fi

# Read PID
PID=$(cat "$PID_FILE")

# Check if process is running
if ! ps -p "$PID" > /dev/null 2>&1; then
    echo -e "${YELLOW}[WARN]${NC} Process (PID: $PID) is not running"
    rm -f "$PID_FILE"
    echo -e "${GREEN}[INFO]${NC} Cleaned up stale PID file"

    # Cleanup caches even if process is dead
    cleanup_gemini_caches

    exit 0
fi

# Graceful shutdown (SIGTERM)
echo -e "${GREEN}[INFO]${NC} Sending SIGTERM to process (PID: $PID)..."
kill_process "$PID" 15

# Cleanup Gemini caches immediately (while process is shutting down)
cleanup_gemini_caches

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
