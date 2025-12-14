#!/bin/bash
# 20Q Kakao Bot - Common Constants and Functions
# Version: 3.0
# This file is sourced by all bot-*.sh scripts

#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Constants
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# shellcheck disable=SC2034
PID_FILE="20q-kakao-bot.pid"
LOG_FILE="logs/20q-kakao-bot.log"
APP_JAR="build/libs/20q-kakao-bot-0.0.1-SNAPSHOT.jar"
APP_PATTERN="20q-kakao-bot.*\.jar"
DEFAULT_PORT=30003

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Functions
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# Check required command exists (hard dependency)
# Usage: check_required_cmd <command_name> [<command_name> ...]
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
# Simple and safe approach using set -a
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
