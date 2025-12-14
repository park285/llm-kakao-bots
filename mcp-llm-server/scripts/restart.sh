#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(dirname "$0")"

"$SCRIPT_DIR/stop.sh"
sleep 1
"$SCRIPT_DIR/start.sh"
