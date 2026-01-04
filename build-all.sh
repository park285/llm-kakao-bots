#!/bin/bash
# build-all.sh: Go 서비스 버전 관리 및 Docker 이미지 빌드 스크립트
#
# 사용법:
#   ./build-all.sh                      # 모든 서비스 버전 bump + 빌드
#   ./build-all.sh --no-bump            # 버전 bump 없이 빌드만
#   ./build-all.sh hololive-bot         # 특정 서비스만 빌드

set -e

# 스크립트 위치 기준 절대 경로로 이동 (루트에 위치)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 버전 관리 대상 디렉토리
VERSION_DIRS=(
    "hololive-kakao-bot-go"
    "game-bot-go"
    "mcp-llm-server-go"
    "admin-dashboard/backend"
)

# 인자 파싱
NO_BUMP=false
TARGET_SERVICES=()

while [[ $# -gt 0 ]]; do
    case "$1" in
        --no-bump)
            NO_BUMP=true
            shift
            ;;
        *)
            TARGET_SERVICES+=("$1")
            shift
            ;;
    esac
done

# 범프 대상 확인 함수
should_bump() {
    local dir_path=$1
    if [ ${#TARGET_SERVICES[@]} -eq 0 ]; then
        return 0
    fi
    for target in "${TARGET_SERVICES[@]}"; do
        if [[ "$dir_path" == *"$target"* ]]; then
            return 0
        fi
    done
    return 1
}

# Step 1: 버전 범프
if [ "$NO_BUMP" = false ]; then
    echo "[BUMP] Bumping patch versions..."
    for dir in "${VERSION_DIRS[@]}"; do
        if should_bump "$dir"; then
            if [ -f "$dir/Makefile" ] && [ -f "$dir/VERSION" ]; then
                old_version=$(cat "$dir/VERSION" | xargs)
                make -C "$dir" bump-patch --no-print-directory > /dev/null
                new_version=$(cat "$dir/VERSION" | xargs)
                echo "  [OK] $dir: $old_version -> $new_version"
            else
                echo "  [WARN] $dir: Makefile or VERSION not found, skipping"
            fi
        fi
    done
    echo ""
else
    echo "[SKIP] Skipping version bump (--no-bump set)"
    echo ""
fi

# Step 2: Docker Compose 빌드
echo "[BUILD] Building Docker images..."

# VERSION 파일에서 환경변수 설정 (docker-compose build args로 전달)
export HOLO_BOT_VERSION=$(cat hololive-kakao-bot-go/VERSION 2>/dev/null | xargs || echo "dev")
export GAME_BOT_VERSION=$(cat game-bot-go/VERSION 2>/dev/null | xargs || echo "dev")
export MCP_LLM_VERSION=$(cat mcp-llm-server-go/VERSION 2>/dev/null | xargs || echo "dev")
export ADMIN_VERSION=$(cat admin-dashboard/backend/VERSION 2>/dev/null | xargs || echo "dev")

echo "  HOLO_BOT_VERSION=$HOLO_BOT_VERSION"
echo "  GAME_BOT_VERSION=$GAME_BOT_VERSION"
echo "  MCP_LLM_VERSION=$MCP_LLM_VERSION"
echo "  ADMIN_VERSION=$ADMIN_VERSION"
echo ""

if [ ${#TARGET_SERVICES[@]} -gt 0 ]; then
    echo "  Targets: ${TARGET_SERVICES[*]}"
    docker compose -f docker-compose.prod.yml up -d --build "${TARGET_SERVICES[@]}"
else
    echo "  Target: All Services"
    docker compose -f docker-compose.prod.yml up -d --build
fi

echo ""
echo "[DONE] Build complete!"

# Step 3: 버전 리포트
echo ""
echo "[VERSIONS] Current versions:"
for dir in "${VERSION_DIRS[@]}"; do
    if [ -f "$dir/VERSION" ]; then
        printf "  %-30s : %s\n" "$dir" "$(cat "$dir/VERSION" | xargs)"
    fi
done
