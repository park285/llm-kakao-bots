#!/bin/bash
# 20Q Kakao Bot - Status Script
# Version: 3.0

set -eo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$PROJECT_DIR"

# shellcheck source=/dev/null
. "$PROJECT_DIR/bot-common.sh"

#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Check required dependencies
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

check_required_cmd jq curl

#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Main
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}   IRIS 20Q SERVICE STATUS${NC}"
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

echo -e "${GREEN}Status:${NC} RUNNING ✓"

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
# Health check (Actuator)
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

TARGET_PORT="${TARGET_PORT:-$DEFAULT_PORT}"

echo -e "\n${BLUE}Health Status:${NC}"
HEALTH_JSON=$(curl -s "localhost:$TARGET_PORT/actuator/health" 2>/dev/null || echo '{"status":"DOWN"}')
HEALTH_STATUS=$(echo "$HEALTH_JSON" | jq -r '.status // "DOWN"')

if [ "$HEALTH_STATUS" = "UP" ]; then
    echo -e "  Overall:      ${GREEN}UP${NC}"
else
    echo -e "  Overall:      ${RED}${HEALTH_STATUS:-DOWN}${NC}"
fi

# Component Health (Redis, Gemini)
REDIS_STATUS=$(echo "$HEALTH_JSON" | jq -r '.components.redis.status // ""')
GEMINI_STATUS=$(echo "$HEALTH_JSON" | jq -r '.components.gemini.status // ""')

if [ -n "$REDIS_STATUS" ]; then
    if [ "$REDIS_STATUS" = "UP" ]; then
        echo -e "  Redis:        ${GREEN}UP${NC}"
    else
        echo -e "  Redis:        ${RED}${REDIS_STATUS}${NC}"
    fi
fi

if [ -n "$GEMINI_STATUS" ]; then
    if [ "$GEMINI_STATUS" = "UP" ]; then
        echo -e "  Gemini API:   ${GREEN}UP${NC}"
    else
        echo -e "  Gemini API:   ${RED}${GEMINI_STATUS}${NC}"
    fi
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

if [ -f "$LOG_FILE" ]; then
    LOG_SIZE=$(du -h "$LOG_FILE" | cut -f1)
    LOG_LINES=$(wc -l < "$LOG_FILE")
    echo -e "\n${BLUE}Log File:${NC}"
    echo -e "  Path:         $LOG_FILE"
    echo -e "  Size:         $LOG_SIZE"
    echo -e "  Lines:        $LOG_LINES"

    # Check for errors
    ERROR_COUNT=$(grep "ERROR" "$LOG_FILE" 2>/dev/null | wc -l || true)
    if [ "$ERROR_COUNT" -gt 0 ]; then
        echo -e "  Errors:       ${RED}$ERROR_COUNT${NC}"
        echo -e "\n${YELLOW}Recent Errors (last 3):${NC}"
        grep "ERROR" "$LOG_FILE" | tail -n 3 | sed 's/^/  /'
    else
        echo -e "  Errors:       ${GREEN}0${NC}"
    fi
fi

#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Footer
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}Commands:${NC}"
echo -e "  View logs:    tail -f $LOG_FILE"
echo -e "  Restart:      ./bot-restart.sh"
echo -e "  Stop:         ./bot-stop.sh"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
