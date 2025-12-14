#!/bin/bash
# Turtle Soup Bot - Status Script
# Version: 1.0

set -eo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$PROJECT_DIR"

# shellcheck source=/dev/null
. "$PROJECT_DIR/bot-common.sh"

# Load env to respect overrides such as SERVER_PORT
load_env_file >/dev/null 2>&1 || true

#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Main
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}   TURTLE SOUP BOT STATUS${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Check PID file
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

if [ ! -f "$PID_FILE" ]; then
    echo -e "${RED}Status:${NC} NOT RUNNING (no PID file)"

    # Check for zombie process
    ZOMBIE_PIDS=$(pgrep -f "$APP_PATTERN" || true)
    if [ -n "$ZOMBIE_PIDS" ]; then
        echo -e "${YELLOW}Warning:${NC} Found orphan process(es): $ZOMBIE_PIDS"
        echo -e "${YELLOW}Action:${NC} Run './bot-stop.sh' to clean up"
    fi
    exit 1
fi

PID=$(cat "$PID_FILE")
echo -e "${GREEN}PID File:${NC} $PID_FILE"
echo -e "${GREEN}Process ID:${NC} $PID"

#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Check process running
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

if ! ps -p "$PID" > /dev/null 2>&1; then
    echo -e "${RED}Status:${NC} NOT RUNNING (stale PID file)"
    echo -e "${YELLOW}Action:${NC} Run './bot-start.sh' to restart"
    exit 1
fi

echo -e "${GREEN}Status:${NC} RUNNING"

#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Process details
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

PROCESS_INFO=$(ps -p "$PID" -o pid,ppid,user,%cpu,%mem,etime,args | tail -n 1)
echo -e "\n${BLUE}Process Details:${NC}"
echo "$PROCESS_INFO" | awk '{
    printf "  CPU Usage:    %s%%\n", $4
    printf "  Memory Usage: %s%%\n", $5
    printf "  Uptime:       %s\n", $6
    printf "  User:         %s\n", $3
}'

#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Health check (Ktor)
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

TARGET_PORT="${SERVER_PORT:-$DEFAULT_PORT}"

echo -e "\n${BLUE}Health Status:${NC}"
HEALTH_CODE=$(curl -s -o /dev/null -w "%{http_code}" "localhost:$TARGET_PORT/health" 2>/dev/null || echo "000")

if [ "$HEALTH_CODE" = "200" ]; then
    echo -e "  HTTP Health:  ${GREEN}UP${NC}"
else
    echo -e "  HTTP Health:  ${YELLOW}UNKNOWN${NC} (HTTP $HEALTH_CODE)"
fi

#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Network status
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

PORT_INFO=$(ss -tlnp 2>/dev/null | grep ":$TARGET_PORT" || echo "")
if [ -n "$PORT_INFO" ]; then
    echo -e "  Port $TARGET_PORT:   ${GREEN}LISTENING${NC}"
else
    echo -e "  Port $TARGET_PORT:   ${YELLOW}NOT LISTENING${NC}"
fi

#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Log file info
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

STDOUT_LOG="${LOG_FILE%.log}.stdout.log"
if [ -f "$STDOUT_LOG" ]; then
    LOG_SIZE=$(du -h "$STDOUT_LOG" | cut -f1)
    LOG_LINES=$(wc -l < "$STDOUT_LOG")
    echo -e "\n${BLUE}Log File:${NC}"
    echo -e "  Path:         $STDOUT_LOG"
    echo -e "  Size:         $LOG_SIZE"
    echo -e "  Lines:        $LOG_LINES"

    # Check for errors
ERROR_COUNT=$(grep -c "ERROR" "$STDOUT_LOG" 2>/dev/null || true)
ERROR_COUNT=${ERROR_COUNT:-0}
    if [ "$ERROR_COUNT" -gt 0 ]; then
        echo -e "  Errors:       ${RED}$ERROR_COUNT${NC}"
        echo -e "\n${YELLOW}Recent Errors (last 3):${NC}"
        grep "ERROR" "$STDOUT_LOG" | tail -n 3 | sed 's/^/  /'
    else
        echo -e "  Errors:       ${GREEN}0${NC}"
    fi
fi

#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Footer
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}Commands:${NC}"
echo -e "  View logs:    tail -f $STDOUT_LOG"
echo -e "  Restart:      ./bot-restart.sh"
echo -e "  Stop:         ./bot-stop.sh"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
