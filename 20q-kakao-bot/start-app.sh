#!/bin/bash
source .env 2>/dev/null || true
SPRING_PROFILES_ACTIVE="${SPRING_PROFILES_ACTIVE:-prod}"
JAVA_OPTS_EFFECTIVE="${JAVA_OPTS:-"-Xmx8g -Dfile.encoding=UTF-8"} -Dspring.profiles.active=${SPRING_PROFILES_ACTIVE}"
echo "[20Q-BOT] Active profile: ${SPRING_PROFILES_ACTIVE}"
java $JAVA_OPTS_EFFECTIVE -jar build/libs/20q-kakao-bot-0.0.1-SNAPSHOT.jar
