#!/bin/bash
# Start bot (integrated: webhook + alarm + YouTube scheduler)
set -e

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$PROJECT_ROOT"

# Defaults (can be overridden by env or .env)
MIN_COUNT=${CORE_MEMBER_HASH_SOFT_MIN_COUNT:-50}
TIMEOUT_SEC=${CORE_MEMBER_HASH_SOFT_TIMEOUT_SECONDS:-45}

# Load .env if present
if [[ -f ./.env ]]; then
  set -a; . ./.env; set +a
fi

# Start unified bot
./scripts/start-bot.sh

# Wait for readiness: prefer ready flag, fallback to HLEN threshold
start_ts=$(date +%s)
while true; do
  # Prefer ready flag
  if docker exec holo-valkey valkey-cli EXISTS hololive:members:ready 2>/dev/null | grep -q "^1$"; then
    echo "[READY] hololive:members:ready flag detected"; break
  fi
  # Fallback: HLEN threshold
  count=$(docker exec holo-valkey valkey-cli HLEN hololive:members 2>/dev/null | tr -d '') || count=0
  if [[ "$count" =~ ^[0-9]+$ ]] && [ "$count" -ge "$MIN_COUNT" ]; then
    echo "[READY] hololive:members count >= $MIN_COUNT (=$count)"; break
  fi
  now=$(date +%s); elapsed=$((now - start_ts))
  if [ $elapsed -ge $TIMEOUT_SEC ]; then
    echo "[WARN] Readiness not reached in ${TIMEOUT_SEC}s (flag missing, count=$count). Proceeding anyway."
    break
  fi
  sleep 1
done

echo "[OK] Bot started (webhook + alarm + YouTube scheduler)"
