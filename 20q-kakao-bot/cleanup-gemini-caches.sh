#!/bin/bash
# Gemini Context Cache Cleanup Script
# 현재 Gemini API 서버에 남아있는 orphan 캐시들을 정리

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}   GEMINI CONTEXT CACHE CLEANUP${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# .env 로드
if [ ! -f .env ]; then
    echo -e "${RED}[ERROR]${NC} .env file not found"
    exit 1
fi

source .env

# API 키 추출 (GOOGLE_API_KEY 또는 GOOGLE_API_KEYS의 첫 번째 키)
API_KEY=""
if [ -n "$GOOGLE_API_KEY" ]; then
    API_KEY="$GOOGLE_API_KEY"
elif [ -n "$GOOGLE_API_KEYS" ]; then
    # 쉼표, 공백, 개행으로 구분된 첫 번째 키 추출
    API_KEY=$(echo "$GOOGLE_API_KEYS" | tr ',\n' ' ' | awk '{print $1}')
fi

if [ -z "$API_KEY" ]; then
    echo -e "${RED}[ERROR]${NC} No API key found (GOOGLE_API_KEY or GOOGLE_API_KEYS)"
    exit 1
fi

echo -e "${GREEN}[INFO]${NC} Using API key: ${API_KEY:0:8}..."

# 1. 현재 캐시 목록 조회
echo -e "\n${YELLOW}[STEP 1/3]${NC} Fetching cached contents from Gemini API..."

CACHE_LIST=$(curl -s -X GET \
    "https://generativelanguage.googleapis.com/v1beta/cachedContents?key=$API_KEY")

# jq 없이 파싱 (cachedContents.name 추출)
CACHE_NAMES=$(echo "$CACHE_LIST" | grep -o '"name":"[^"]*"' | cut -d'"' -f4 || true)

if [ -z "$CACHE_NAMES" ]; then
    echo -e "${GREEN}[SUCCESS]${NC} No caches found. Nothing to clean up."
    exit 0
fi

# 20q- 프리픽스 필터링 (안전장치)
FILTERED_CACHES=$(echo "$CACHE_NAMES" | grep "cachedContents/20q-" || true)

if [ -z "$FILTERED_CACHES" ]; then
    echo -e "${GREEN}[SUCCESS]${NC} No 20Q-related caches found."
    exit 0
fi

CACHE_COUNT=$(echo "$FILTERED_CACHES" | wc -l)
echo -e "${YELLOW}[INFO]${NC} Found ${CACHE_COUNT} cache(s) to delete:"
echo "$FILTERED_CACHES" | while read -r cache_name; do
    CACHE_ID=$(echo "$cache_name" | sed 's|cachedContents/||')
    echo -e "  - ${CACHE_ID}"
done

# 2. 삭제 확인
echo -e "\n${YELLOW}[STEP 2/3]${NC} Confirmation required"
echo -e "${RED}[WARNING]${NC} This will delete ${CACHE_COUNT} cache(s) from Gemini API"
read -p "Continue? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}[CANCELLED]${NC} Operation cancelled by user"
    exit 0
fi

# 3. 삭제 실행
echo -e "\n${YELLOW}[STEP 3/3]${NC} Deleting caches..."

SUCCESS_COUNT=0
FAIL_COUNT=0

echo "$FILTERED_CACHES" | while read -r cache_name; do
    CACHE_ID=$(echo "$cache_name" | sed 's|cachedContents/||')
    
    DELETE_RESPONSE=$(curl -s -X DELETE \
        "https://generativelanguage.googleapis.com/v1beta/${cache_name}?key=$API_KEY")
    
    # 빈 응답 = 성공 (204 No Content)
    if [ -z "$DELETE_RESPONSE" ] || [ "$DELETE_RESPONSE" = "{}" ]; then
        echo -e "  ${GREEN}✓${NC} Deleted: ${CACHE_ID}"
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
    else
        echo -e "  ${RED}✗${NC} Failed: ${CACHE_ID}"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
done

echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}[SUMMARY]${NC}"
echo -e "  Success: ${SUCCESS_COUNT}"
echo -e "  Failed:  ${FAIL_COUNT}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

if [ $FAIL_COUNT -gt 0 ]; then
    exit 1
fi
