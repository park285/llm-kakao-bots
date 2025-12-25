# LLM 워크스페이스

[![Go Version](https://img.shields.io/badge/Go-1.25.5-00ADD8?logo=go)](https://go.dev/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-18-336791?logo=postgresql)](https://www.postgresql.org/)
[![Valkey](https://img.shields.io/badge/Valkey-9.0-DC382D?logo=redis)](https://valkey.io/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

LLM 기반 카카오톡 봇 서비스를 위한 모노레포 워크스페이스입니다.

## 기술 스택

| 분류 | 기술 | 버전 |
|------|------|------|
| **언어** | Go | 1.25.5 |
| **AI** | Google Gemini API | go-genai SDK |
| **메시지큐** | Valkey Streams | 9.0-alpine |
| **캐시** | Valkey (AOF) | 9.0-alpine |
| **데이터베이스** | PostgreSQL | 18-alpine |
| **HTTP** | h2c (HTTP/2 Cleartext) | - |

## 프로젝트 구조

```
llm/
├── mcp-llm-server-go/        # LLM 추론 서버
│   ├── internal/
│   │   ├── gemini/           # Gemini SDK 래퍼
│   │   ├── guard/            # 프롬프트 인젝션 가드
│   │   ├── session/          # 세션 관리
│   │   ├── handler/          # HTTP 핸들러
│   │   └── usage/            # 토큰 사용량 추적
│   └── Dockerfile.prod
│
├── game-bot-go/              # 게임 봇 (모노레포)
│   ├── cmd/
│   │   ├── twentyq/          # 스무고개 엔트리포인트
│   │   └── turtlesoup/       # 바다거북수프 엔트리포인트
│   ├── internal/
│   │   ├── common/           # 공통 유틸리티
│   │   │   ├── valkeyx/      # Valkey 클라이언트 헬퍼
│   │   │   ├── parser/       # 명령어 파서 기반
│   │   │   ├── httputil/     # HTTP 유틸리티
│   │   │   └── config/       # 공통 상수
│   │   ├── twentyq/          # 스무고개 로직
│   │   └── turtlesoup/       # 바다거북수프 로직
│   └── Dockerfile.prod
│
├── hololive-kakao-bot-go/    # 홀로라이브 정보 봇
│   ├── internal/
│   │   ├── command/          # 명령어 핸들러
│   │   ├── service/          # 비즈니스 로직
│   │   └── repository/       # 데이터 접근
│   └── Dockerfile
│
├── watchdog/                 # 컨테이너 헬스체크 모니터
│   └── Dockerfile
│
├── docker-compose.prod.yml   # 프로덕션 스택
├── .env                      # 환경 변수 (SSOT)
├── logs/                     # 로그 디렉터리
└── backups/                  # 백업 스크립트
```

## 서비스 구성

### 애플리케이션 서비스

| 서비스 | 컨테이너명 | 포트 | 메모리 | 설명 |
|--------|------------|------|--------|------|
| `mcp-llm-server` | mcp-llm-server | 40527 | 1GB | LLM 추론/가드/세션 |
| `twentyq-bot` | twentyq-bot | 30081 | 512MB | 스무고개 게임 봇 |
| `turtle-soup-bot` | turtle-soup-bot | 30082 | 512MB | 바다거북수프 게임 봇 |
| `hololive-bot` | hololive-kakao-bot-go | 30001 | 512MB | 홀로라이브 정보 봇 |
| `llm-watchdog` | llm-watchdog | 30002 | 512MB | 컨테이너 모니터링 |

### 인프라 서비스

| 서비스 | 컨테이너명 | 포트 | 메모리 | 설명 |
|--------|------------|------|--------|------|
| `postgres` | llm-postgres | 5432 | 512MB | 통합 PostgreSQL |
| `valkey-cache` | valkey-cache | 6379 | 512MB | 세션/캐시 (AOF) |
| `valkey-mq` | valkey-mq | 1833 | 256MB | Streams 메시지큐 |

## API 엔드포인트

### LLM Server (`mcp-llm-server:40527`)

| 메서드 | 경로 | 설명 |
|--------|------|------|
| GET | `/health/ready` | 준비 상태 확인 |
| GET | `/health/models` | 모델 설정 조회 |
| POST | `/api/sessions` | 세션 생성 |
| DELETE | `/api/sessions/:id` | 세션 삭제 |
| POST | `/api/guard/checks` | 인젝션 가드 체크 |
| POST | `/api/llm/twentyq/*` | 스무고개 LLM 호출 |
| POST | `/api/llm/turtlesoup/*` | 바다거북수프 LLM 호출 |
| GET | `/api/usage/*` | 토큰 사용량 조회 |

### Game Bots

| 봇 | 메서드 | 경로 | 설명 |
|----|--------|------|------|
| twentyq | GET | `/health` | 헬스체크 |
| twentyq | GET/POST | `/api/twentyq/*` | REST API |
| turtlesoup | GET | `/health` | 헬스체크 |
| turtlesoup | GET/POST | `/api/turtlesoup/*` | REST API |

## 빠른 시작

### 요구사항

- Docker 24.0+
- Docker Compose v2
- Make (선택)

### 환경 설정

```bash
# .env 파일 생성
cp .env.example .env

# 필수 값 설정
vi .env
```

### 빌드 및 실행

```bash
# 전체 빌드 (캐시 미사용)
docker compose -f docker-compose.prod.yml build --no-cache

# 서비스 기동
docker compose -f docker-compose.prod.yml up -d

# 상태 확인
docker compose -f docker-compose.prod.yml ps

# 헬스체크
curl http://localhost:40527/health/ready
curl http://localhost:8081/health
curl http://localhost:8082/health
```

### 특정 서비스 재기동

```bash
# 빌드 후 재기동
docker compose -f docker-compose.prod.yml build twentyq-bot turtle-soup-bot
docker compose -f docker-compose.prod.yml up -d twentyq-bot turtle-soup-bot

# 강제 재생성
docker compose -f docker-compose.prod.yml up -d --force-recreate mcp-llm-server
```

## 환경 변수

`.env` 파일이 모든 서비스의 환경 변수 **SSOT** (Single Source of Truth)입니다.

### 필수 설정

| 변수 | 설명 | 예시 |
|------|------|------|
| `GOOGLE_API_KEY` | Gemini API 키 | `AIza...` |
| `DB_PASSWORD` | PostgreSQL 비밀번호 | `secure_password` |

### Gemini 설정

| 변수 | 설명 | 기본값 |
|------|------|--------|
| `GEMINI_MODEL` | 기본 모델 | `gemini-3.0-flash` |
| `GEMINI_TEMPERATURE` | Temperature | `1.0` |
| `GEMINI_TIMEOUT_SECONDS` | 타임아웃 | `60` |
| `GEMINI_MAX_RETRIES` | 최대 재시도 | `3` |

### 보안 설정

| 변수 | 설명 | 기본값 |
|------|------|--------|
| `HTTP_API_KEY` | API 인증 키 | (비활성화) |
| `HTTP_RATE_LIMIT_RPM` | 분당 요청 제한 | (비활성화) |
| `GUARD_ENABLED` | 인젝션 가드 | `true` |
| `GUARD_THRESHOLD` | 가드 임계값 | `0.85` |

### 세션/캐시 설정

| 변수 | 설명 | 기본값 |
|------|------|--------|
| `SESSION_STORE_URL` | Valkey URL | `redis://valkey-cache:6379` |
| `SESSION_STORE_ENABLED` | 세션 활성화 | `true` |
| `SESSION_TTL_HOURS` | 세션 만료 시간 | `24` |

### 로깅 설정

| 변수 | 설명 | 기본값 |
|------|------|--------|
| `LOG_DIR` | 로그 디렉터리 | `/app/logs` |
| `LOG_LEVEL` | 로그 레벨 | `info` |
| `LOG_FILE_MAX_SIZE_MB` | 최대 파일 크기 | `100` |
| `LOG_FILE_MAX_BACKUPS` | 백업 개수 | `3` |

## 아키텍처

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                              카카오톡 앱                                       │
└──────────────────────────────────┬───────────────────────────────────────────┘
                                   │
                                   ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                          Valkey MQ (Streams) :1833                           │
└─────────┬───────────────────────┬────────────────────────┬───────────────────┘
          │                       │                        │
          ▼                       ▼                        ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────────────┐
│  TwentyQ Bot    │    │ TurtleSoup Bot  │    │     Hololive Bot        │
│    :30081       │    │    :30082       │    │       :30001            │
└────────┬────────┘    └────────┬────────┘    └────────────┬────────────┘
         │                      │                          │
         │    ┌─────────────────┴──────────────┐           │
         └───►│         LLM Server             │           │
              │           :40527               │           │
              │        (h2c + Gin)             │           │
              └───────────────┬────────────────┘           │
                              │                            │
    ┌─────────────────────────┼─────────────────────────┐  │
    │                         │                         │  │
    ▼                         ▼                         ▼  ▼
┌──────────────┐    ┌───────────────┐    ┌────────────────────┐
│ Valkey Cache │◄───┼───────────────┼────┤    PostgreSQL      │
│    :6379     │    │  Gemini API   │    │      :5432         │
│ (AOF 영속화)  │    │  (External)   │    │                    │
└──────────────┘    └───────────────┘    └────────────────────┘
        ▲                                          ▲
        │                                          │
        └──────────────┬───────────────────────────┘
                       │
              ┌────────┴────────┐
              │    Watchdog     │
              │     :30002      │
              │ (헬스체크/재시작) │
              └─────────────────┘
```

## 개발 가이드

### 로컬 개발

```bash
# game-bot-go 개발
cd game-bot-go
go build ./...
go test ./... -race -count=1

# mcp-llm-server-go 개발
cd mcp-llm-server-go
go build ./...
go test ./... -race -count=1
make lint
```

### 코드 품질

```bash
# 포맷팅
gofmt -w .
goimports -w .

# 린팅
go vet ./...
staticcheck ./...

# 테스트 커버리지
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### 공통 패키지 (game-bot-go/internal/common)

| 패키지 | 설명 |
|--------|------|
| `valkeyx` | Valkey 클라이언트, 키 빌더 |
| `parser` | 명령어 파서 기반 클래스 |
| `httputil` | JSON 응답, HTTP 상수 |
| `config` | 공통 상수 (TTL, 타임아웃 등) |
| `textutil` | 텍스트 청킹 유틸리티 |
| `llmrest` | LLM 서버 REST 클라이언트 |

## 운영 가이드

### 로그 확인

```bash
# 실시간 로그
docker compose -f docker-compose.prod.yml logs -f

# 특정 서비스
docker compose -f docker-compose.prod.yml logs -f mcp-llm-server

# 파일 로그 (호스트)
tail -f logs/server.log
tail -f logs/twentyq.log
tail -f logs/turtlesoup.log
```

### 서비스 중지

```bash
docker compose -f docker-compose.prod.yml down --remove-orphans
```

### 볼륨 정리 (주의!)

```bash
# 데이터 포함 전체 삭제
docker compose -f docker-compose.prod.yml down -v
```

### 백업

```bash
# PostgreSQL 백업
docker exec llm-postgres pg_dumpall -U twentyq_app > backups/pgdump_$(date +%Y%m%d).sql

# Valkey 스냅샷
docker exec valkey-cache valkey-cli BGSAVE
```

## 트러블슈팅

### 서비스가 시작되지 않음

```bash
# 로그 확인
docker compose -f docker-compose.prod.yml logs <서비스명>

# 헬스체크 상태
docker inspect <컨테이너명> --format='{{json .State.Health}}'
```

### LLM 호출 실패

```bash
# LLM 서버 헬스체크
curl http://localhost:40527/health/ready

# 모델 설정 확인
curl http://localhost:40527/health/models
```

### Valkey 연결 실패

```bash
# Valkey 상태 확인
docker exec valkey-cache valkey-cli ping
docker exec valkey-mq valkey-cli -p 1833 ping
```

---

**Last Updated**: 2025-12-25
