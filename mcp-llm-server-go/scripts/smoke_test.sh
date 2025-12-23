#!/usr/bin/env bash
set -euo pipefail

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

start_server() {
  local port="$1"
  local rate_limit_rpm="${2:-0}"
  local api_key="${3:-secret}"

  GOOGLE_API_KEY="dummy" \
  SESSION_STORE_ENABLED="false" \
  SESSION_STORE_REQUIRED="false" \
  HTTP_HOST="127.0.0.1" \
  HTTP_PORT="$port" \
  HTTP_API_KEY="$api_key" \
  HTTP_RATE_LIMIT_RPM="$rate_limit_rpm" \
  LOG_LEVEL="info" \
  LOG_DIR="" \
  go run ./cmd/server >/tmp/mcp-llm-server-go.test.log 2>&1 &

  echo $!
}

wait_ready() {
  local base_url="$1"
  for _ in $(seq 1 50); do
    if curl -fsS "$base_url/health" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.1
  done
  return 1
}

http_code() {
  curl -s -o /dev/null -w "%{http_code}" "$@"
}

require_cmd go
require_cmd curl

PORT="$(pick_port)"
BASE_URL="http://127.0.0.1:$PORT"

PID="$(start_server "$PORT" 0 secret)"
cleanup() {
  kill -TERM "$PID" >/dev/null 2>&1 || true
  wait "$PID" >/dev/null 2>&1 || true
}
trap cleanup EXIT

if ! wait_ready "$BASE_URL"; then
  echo "server did not become ready; log:" >&2
  tail -n 200 /tmp/mcp-llm-server-go.test.log >&2 || true
  exit 1
fi

code="$(http_code "$BASE_URL/health")"
if [[ "$code" != "200" ]]; then
  echo "/health expected 200, got $code" >&2
  exit 1
fi

code="$(http_code "$BASE_URL/health/ready")"
if [[ "$code" != "200" ]]; then
  echo "/health/ready expected 200, got $code" >&2
  exit 1
fi

unauth_code="$(http_code -X POST "$BASE_URL/api/guard/checks" -H "Content-Type: application/json" -d '{"input_text":"hello"}')"
if [[ "$unauth_code" != "401" ]]; then
  echo "guard checks without key expected 401, got $unauth_code" >&2
  exit 1
fi

guard_body="$(curl -fsS -X POST "$BASE_URL/api/guard/checks" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: secret" \
  -d '{"input_text":"hello"}')"
echo "$guard_body" | grep -q '"malicious":false' || {
  echo "unexpected guard checks response: $guard_body" >&2
  exit 1
}

session_resp="$(curl -fsS -X POST "$BASE_URL/api/sessions" -H "X-API-Key: secret" -H "Content-Type: application/json" -d '{}')"
if command -v python3 >/dev/null 2>&1; then
  SESSION_ID="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])' <<<"$session_resp")"
else
  SESSION_ID="$(echo "$session_resp" | sed -n 's/.*"id":"\\([^"]*\\)".*/\\1/p')"
fi
if [[ -z "${SESSION_ID:-}" ]]; then
  echo "failed to parse session id from: $session_resp" >&2
  exit 1
fi

get_resp="$(curl -fsS -X GET "$BASE_URL/api/sessions/$SESSION_ID" -H "X-API-Key: secret")"
echo "$get_resp" | grep -q "\"id\":\"$SESSION_ID\"" || {
  echo "unexpected session get response: $get_resp" >&2
  exit 1
}

del_code="$(http_code -X DELETE "$BASE_URL/api/sessions/$SESSION_ID" -H "X-API-Key: secret")"
if [[ "$del_code" != "200" ]]; then
  echo "delete session expected 200, got $del_code" >&2
  exit 1
fi

trap - EXIT
cleanup

PORT2="$(pick_port)"
BASE_URL2="http://127.0.0.1:$PORT2"
PID2="$(start_server "$PORT2" 1 secret)"
cleanup2() {
  kill -TERM "$PID2" >/dev/null 2>&1 || true
  wait "$PID2" >/dev/null 2>&1 || true
}
trap cleanup2 EXIT

if ! wait_ready "$BASE_URL2"; then
  echo "rate-limit server did not become ready; log:" >&2
  tail -n 200 /tmp/mcp-llm-server-go.test.log >&2 || true
  exit 1
fi

first="$(http_code -X POST "$BASE_URL2/api/guard/checks" -H "X-API-Key: secret" -H "Content-Type: application/json" -d '{"input_text":"hello"}')"
second="$(http_code -X POST "$BASE_URL2/api/guard/checks" -H "X-API-Key: secret" -H "Content-Type: application/json" -d '{"input_text":"hello"}')"
if [[ "$first" != "200" || "$second" != "429" ]]; then
  echo "rate-limit expected 200 then 429, got $first then $second" >&2
  exit 1
fi

echo "OK"
