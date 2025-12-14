#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/common.sh"

init_dirs
if is_running; then
    log_warn "Already running (PID: $(get_pid))"
    exit 0
fi

if ! command -v uv >/dev/null 2>&1; then
    log_error "uv not found; install uv to start the server"
    exit 1
fi

PYTHON_BIN="$VENV_DIR/bin/python"
if ! command -v "$PYTHON_BIN" >/dev/null 2>&1; then
    log_error "Python virtualenv not found: $PYTHON_BIN (run 'uv sync' with python3.14)"
    exit 1
fi

export PYTHON_JIT="${PYTHON_JIT:-1}"
RUN_CMD=(uv run --python "$PYTHON_BIN" python -m mcp_llm_server.http_server)
log_info "Using uv runner ($PYTHON_BIN, JIT=${PYTHON_JIT})"

# .env 로드 (UDS/TCP 모드 설정)
if [[ -f "$PROJECT_DIR/.env" ]]; then
    set -a
    source "$PROJECT_DIR/.env"
    set +a
fi

log_info "Starting MCP LLM Server (h2c)..."
cd "$PROJECT_DIR"
# main() 함수 호출 - HTTP/2 (h2c) 모드
PYTHONPATH=src nohup "${RUN_CMD[@]}" >> "$LOG_FILE" 2>&1 &
echo $! > "$PID_FILE"
sleep 1

if is_running; then
    log_info "Started (PID: $(get_pid))"
else
    log_error "Failed to start"
    rm -f "$PID_FILE"
    exit 1
fi
