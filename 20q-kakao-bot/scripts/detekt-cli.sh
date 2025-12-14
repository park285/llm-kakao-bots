#!/usr/bin/env bash
# Detekt Standalone CLI Runner
# Kotlin 2.3.0-RC 호환 버전 사용

set -euo pipefail

DETEKT_VERSION="2.0.0-alpha.1"  # Kotlin 2.3.0-RC 호환 alpha
DETEKT_CLI_JAR="detekt-cli-${DETEKT_VERSION}-all.jar"
DETEKT_DIR=".detekt"
DETEKT_PATH="${DETEKT_DIR}/${DETEKT_CLI_JAR}"

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Detekt Standalone CLI Runner${NC}"
echo "Version: ${DETEKT_VERSION}"
echo ""

# Detekt CLI 다운로드 (최초 1회)
if [ ! -f "${DETEKT_PATH}" ]; then
    echo -e "${YELLOW}Detekt CLI JAR 다운로드 중...${NC}"
    mkdir -p "${DETEKT_DIR}"
    
    DOWNLOAD_URL="https://github.com/detekt/detekt/releases/download/v${DETEKT_VERSION}/${DETEKT_CLI_JAR}"
    
    if command -v wget &> /dev/null; then
        wget -q --show-progress -O "${DETEKT_PATH}" "${DOWNLOAD_URL}"
    elif command -v curl &> /dev/null; then
        curl -L --progress-bar -o "${DETEKT_PATH}" "${DOWNLOAD_URL}"
    else
        echo -e "${RED}wget 또는 curl이 필요합니다${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}다운로드 완료: ${DETEKT_PATH}${NC}"
    echo ""
fi

# Detekt 실행
echo -e "${GREEN}Detekt 정적 분석 시작...${NC}"
echo "Config: config/detekt/detekt.yml"
echo "Baseline: config/detekt/detekt-baseline.xml"
echo ""

java -jar "${DETEKT_PATH}" \
    --config config/detekt/detekt.yml \
    --input src/main/kotlin \
    --report html:build/reports/detekt/detekt.html \
    --report xml:build/reports/detekt/detekt.xml \
    --build-upon-default-config \
    --jvm-target 24 \
    --language-version 2.3 \
    "$@"

EXIT_CODE=$?

echo ""
if [ ${EXIT_CODE} -eq 0 ]; then
    echo -e "${GREEN}Detekt 통과!${NC}"
    echo "Report: build/reports/detekt/detekt.html"
else
    echo -e "${RED}Detekt 실패 (Exit Code: ${EXIT_CODE})${NC}"
    echo "Report: build/reports/detekt/detekt.html"
    echo ""
    echo -e "${YELLOW}Tip: Baseline 업데이트가 필요하면:${NC}"
    echo "   ./scripts/detekt-cli.sh --create-baseline --baseline config/detekt/detekt-baseline.xml"
fi

exit ${EXIT_CODE}
