#!/bin/bash
# Turtle Soup Bot - Common Constants and Functions
# Version: 1.0

#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Java Configuration
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

export JAVA_HOME="${JAVA_HOME:-/home/kapu/.sdkman/candidates/java/25.0.1-tem}"
export PATH="$JAVA_HOME/bin:$PATH"

#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Constants
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# shellcheck disable=SC2034
PID_FILE="turtle-soup-bot.pid"
LOG_FILE="logs/turtle-soup-bot.log"
APP_JAR="build/libs/turtle-soup-bot-all.jar"
APP_PATTERN="turtle-soup-bot.*\.jar"
DEFAULT_PORT=40257

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Functions
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# Check required command exists
check_required_cmd() {
    local missing=()

    for cmd in "$@"; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            missing+=("$cmd")
        fi
    done

    if [ ${#missing[@]} -gt 0 ]; then
        echo -e "${RED}[ERROR]${NC} Required command(s) not found: ${missing[*]}" >&2
        exit 1
    fi
}

# Load environment variables from .env file
load_env_file() {
    if [ -f .env ]; then
        echo -e "${GREEN}[INFO]${NC} Loading environment variables from .env"
        set -a
        # shellcheck disable=SC1091
        . ./.env
        set +a
        return 0
    else
        echo -e "${YELLOW}[WARN]${NC} .env file not found"
        return 1
    fi
}
