#!/bin/bash
# Show status of Ingestion bot (with integrated alarm checker)
set -e

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$PROJECT_ROOT"

status_one() {
  local name="$1" pidfile="$2"
  if [[ -f "$pidfile" ]]; then
    local pid=$(cat "$pidfile" 2>/dev/null || echo "")
    if [[ -n "$pid" ]] && ps -p "$pid" >/dev/null 2>&1; then
      echo "[RUNNING] $name (PID $pid)"
    else
      echo "[STOPPED] $name (stale PID file)"; rm -f "$pidfile" || true
    fi
  else
    echo "[STOPPED] $name"
  fi
}

status_one "Bot" ".bot.pid"

# Optional: member readiness (requires docker `holo-valkey`)
if docker ps 2>/dev/null | grep -q "holo-valkey"; then
  flag=$(docker exec holo-valkey valkey-cli EXISTS hololive:members:ready 2>/dev/null | tr -d '\r' || echo 0)
  count=$(docker exec holo-valkey valkey-cli HLEN hololive:members 2>/dev/null | tr -d '\r' || echo 0)
  if [[ "$flag" == "1" ]]; then echo "[READY] hololive:members:ready set"; else echo "[READY] flag not set"; fi
  echo "[COUNT] hololive:members = $count"
fi
