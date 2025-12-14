#!/usr/bin/env bash
# 공통 설정 및 함수

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
VENV_DIR="$PROJECT_DIR/.venv"
LOG_DIR="$PROJECT_DIR/logs"
PID_FILE="$LOG_DIR/mcp-llm-server.pid"
LOG_FILE="$LOG_DIR/mcp-llm-server.log"
ERROR_LOG="$LOG_DIR/mcp-llm-server.error.log"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

init_dirs() { mkdir -p "$LOG_DIR"; }

activate_venv() {
    if [[ ! -d "$VENV_DIR" ]]; then
        log_error "Virtual environment not found: $VENV_DIR"
        exit 1
    fi
    source "$VENV_DIR/bin/activate"
}

get_pid() { [[ -f "$PID_FILE" ]] && cat "$PID_FILE" || echo ""; }

is_running() {
    local pid=$(get_pid)
    [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null
}
