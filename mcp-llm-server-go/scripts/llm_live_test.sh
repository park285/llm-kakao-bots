#!/usr/bin/env bash
set -euo pipefail

ENV_FILE="${1:-/home/kapu/gemini/llm/.env}"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing command: $1" >&2
    exit 1
  fi
}

pick_port() {
  if command -v python3 >/dev/null 2>&1; then
    python3 - <<'PY'
import socket
s = socket.socket()
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
    return
  fi

  echo "40527"
}

if [[ ! -f "$ENV_FILE" ]]; then
  echo "env file not found: $ENV_FILE" >&2
  exit 1
fi

require_cmd go
require_cmd curl

set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

if [[ -z "${GOOGLE_API_KEY:-}" && -z "${GOOGLE_API_KEYS:-}" ]]; then
  echo "missing GOOGLE_API_KEY(S) in env" >&2
  exit 1
fi

PORT="$(pick_port)"
BASE_URL="http://127.0.0.1:$PORT"
LOG_PATH="/tmp/mcp-llm-server-go.llm_live.log"

API_HEADERS=()
if [[ -n "${HTTP_API_KEY:-}" ]]; then
  API_HEADERS=(-H "X-API-Key: ${HTTP_API_KEY}")
fi

start_server() {
  SESSION_STORE_ENABLED="false" \
  SESSION_STORE_REQUIRED="false" \
  SESSION_STORE_URL="redis://localhost:6379" \
  HTTP_HOST="127.0.0.1" \
  HTTP_PORT="$PORT" \
  HTTP_RATE_LIMIT_RPM="0" \
  LOG_DIR="" \
  go run ./cmd/server >"$LOG_PATH" 2>&1 &
  echo $!
}

wait_ready() {
  for _ in $(seq 1 80); do
    if curl -fsS "$BASE_URL/health/ready" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.25
  done
  return 1
}

req() {
  local title="$1"; shift
  echo "### ${title}"
  curl -sS -w "\n(status=%{http_code})\n" "$@"
  echo
}

PID="$(start_server)"
cleanup() {
  kill -TERM "$PID" >/dev/null 2>&1 || true
  wait "$PID" >/dev/null 2>&1 || true
}
trap cleanup EXIT

if ! wait_ready; then
  echo "server did not become ready; log:" >&2
  tail -n 200 "$LOG_PATH" >&2 || true
  exit 1
fi

req "GET /health/models" "$BASE_URL/health/models"

req "POST /api/llm/chat" \
  -X POST "$BASE_URL/api/llm/chat" \
  "${API_HEADERS[@]}" \
  -H "Content-Type: application/json" \
  -d '{"prompt":"Respond only with the single word OK."}'

req "POST /api/llm/chat-with-usage" \
  -X POST "$BASE_URL/api/llm/chat-with-usage" \
  "${API_HEADERS[@]}" \
  -H "Content-Type: application/json" \
  -d '{"prompt":"Respond only with the single word OK."}'

req "POST /api/llm/structured" \
  -X POST "$BASE_URL/api/llm/structured" \
  "${API_HEADERS[@]}" \
  -H "Content-Type: application/json" \
  -d '{"prompt":"Return a JSON object with a single string field named answer. answer must be OK.","json_schema":{"type":"object","properties":{"answer":{"type":"string"}},"required":["answer"]}}'

req "POST /api/twentyq/hints" \
  -X POST "$BASE_URL/api/twentyq/hints" \
  "${API_HEADERS[@]}" \
  -H "Content-Type: application/json" \
  -d '{"target":"노트북","category":"사물"}'

req "GET /api/llm/usage" \
  -X GET "$BASE_URL/api/llm/usage" \
  "${API_HEADERS[@]}"

req "GET /api/llm/usage/total" \
  -X GET "$BASE_URL/api/llm/usage/total" \
  "${API_HEADERS[@]}"

req "GET /api/usage/daily" \
  -X GET "$BASE_URL/api/usage/daily" \
  "${API_HEADERS[@]}"

req "GET /api/usage/total" \
  -X GET "$BASE_URL/api/usage/total?days=30" \
  "${API_HEADERS[@]}"

echo "server log: $LOG_PATH"
tail -n 50 "$LOG_PATH" || true
