#!/bin/bash
# 20Q Kakao Bot - Start Script
# Version: 3.0

set -eo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$PROJECT_DIR"

# shellcheck source=/dev/null
. "$PROJECT_DIR/bot-common.sh"

echo -e "${GREEN}[20Q-BOT]${NC} Starting 20 Questions Kakao Bot..."

#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Check if already running
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

if [ -f "$PID_FILE" ]; then
    OLD_PID=$(cat "$PID_FILE")
    if ps -p "$OLD_PID" > /dev/null 2>&1; then
        echo -e "${RED}[ERROR]${NC} Service is already running (PID: $OLD_PID)"
        echo -e "${YELLOW}[TIP]${NC} Use './bot-stop.sh' to stop it first"
        exit 1
    else
        echo -e "${YELLOW}[WARN]${NC} Stale PID file found. Removing..."
        rm -f "$PID_FILE"
    fi
fi

#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Build if JAR not found
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

if [ ! -f "$APP_JAR" ]; then
    echo -e "${YELLOW}[INFO]${NC} JAR file not found. Building..."
    ./gradlew clean build

    if [ $? -ne 0 ]; then
        echo -e "${RED}[ERROR]${NC} Build failed!"
        exit 1
    fi
fi

#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Load environment variables
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

load_env_file || true

#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Setup Java (sdkman is optional)
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

if [ -f "$HOME/.sdkman/bin/sdkman-init.sh" ]; then
    # shellcheck disable=SC1091
    source "$HOME/.sdkman/bin/sdkman-init.sh"
    sdk use java 25.0.1-tem >/dev/null 2>&1 || true
else
    echo -e "${YELLOW}[WARN]${NC} sdkman not found, using system java"
fi

# Verify Java is available
check_required_cmd java

#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Prepare JAVA_OPTS
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

DEFAULT_JAVA_OPTS=(
    # JVM Modules & Access
    
    "--enable-native-access=ALL-UNNAMED"
    "--add-opens=java.base/java.util=ALL-UNNAMED"
    "--add-opens=java.base/jdk.internal.loader=ALL-UNNAMED"
    "--add-opens=java.base/java.lang=ALL-UNNAMED"
    
    # Heap & Memory
    "-Xmx2g"
    "-Xms2g"
    "-XX:+AlwaysPreTouch"
    "-XX:MetaspaceSize=256m"
    "-XX:MaxMetaspaceSize=768m"
    
    # GC (Shenandoah - Low Latency)
    "-XX:+UseShenandoahGC"
    "-XX:ShenandoahGCMode=generational"
    "-XX:ShenandoahGCHeuristics=adaptive"
    "-XX:+DisableExplicitGC"
    
    # JVM 25 Optimizations
    "-XX:+UseCompactObjectHeaders"
    "-XX:+UseStringDeduplication"
    
    # Code Cache
    "-XX:ReservedCodeCacheSize=512m"
    "-XX:InitialCodeCacheSize=256m"
    
    # JIT Compiler
    "-XX:CICompilerCount=4"
    
    # Encoding
    "-Dfile.encoding=UTF-8"
)

# Allow override via JAVA_OPTS env var
EXTRA_JAVA_OPTS=()
if [ -n "${JAVA_OPTS:-}" ]; then
    # shellcheck disable=SC2206
    EXTRA_JAVA_OPTS=($JAVA_OPTS)
fi

SPRING_PROFILES_ACTIVE="${SPRING_PROFILES_ACTIVE:-prod}"

JAVA_OPTS_EFFECTIVE=(
    "${DEFAULT_JAVA_OPTS[@]}"
    "${EXTRA_JAVA_OPTS[@]}"
    "-Dspring.profiles.active=${SPRING_PROFILES_ACTIVE}"
)

echo -e "${GREEN}[INFO]${NC} Active profile: ${SPRING_PROFILES_ACTIVE}"
echo -e "${GREEN}[INFO]${NC} Using JAVA_OPTS: ${JAVA_OPTS_EFFECTIVE[*]}"

#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Start application
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

echo -e "${GREEN}[INFO]${NC} Starting application..."

# Ensure logs directory exists
mkdir -p "$(dirname "$LOG_FILE")"

STDOUT_LOG="${LOG_FILE%.log}.stdout.log"
nohup java "${JAVA_OPTS_EFFECTIVE[@]}" -jar "$APP_JAR" \
    > "$STDOUT_LOG" \
    2> >(grep -v "incubator\|Commons Logging" >&2) &
APP_PID=$!

echo "$APP_PID" > "$PID_FILE"
echo -e "${GREEN}[SUCCESS]${NC} Service started (PID: $APP_PID)"

#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Wait for application startup
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

echo -e "${GREEN}[INFO]${NC} Waiting for application startup (timeout 30s)..."

READY=0
for i in {1..30}; do
    # Check if application started successfully via log
    if [ -f "$LOG_FILE" ] && grep -q "Started TwentyQKakaoBotApplicationKt" "$LOG_FILE"; then
        READY=1
        break
    fi

    # Exit early if process died during startup
    if ! ps -p "$APP_PID" > /dev/null 2>&1; then
        break
    fi

    sleep 1
done

#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Final status
#━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

if ps -p "$APP_PID" > /dev/null 2>&1; then
    if [ "$READY" -eq 1 ]; then
        echo -e "${GREEN}[SUCCESS]${NC} Application started successfully"
    else
        echo -e "${YELLOW}[WARN]${NC} Service is running but startup not confirmed in logs yet"
    fi
    echo -e "${GREEN}[INFO]${NC} Log file: $LOG_FILE"
    echo -e "${GREEN}[INFO]${NC} Use './bot-status.sh' to check status"
    echo -e "${GREEN}[INFO]${NC} Use 'tail -f $LOG_FILE' to view logs"
else
    echo -e "${RED}[ERROR]${NC} Service failed to start. Check $LOG_FILE for details"
    rm -f "$PID_FILE"
    exit 1
fi
