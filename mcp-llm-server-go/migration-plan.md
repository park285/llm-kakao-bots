# MCP LLM Server: Python → Go 마이그레이션 계획

## 개요

Python (FastAPI + LangChain) 기반 LLM 서버를 Go로 재작성하여 GIL 문제 해결 및 성능 향상.

---

## 현재 시스템 분석

### 1. 아키텍처

```
┌─────────────────────────────────────────────────────────────────┐
│                    HTTP Server (FastAPI + Hypercorn h2c)        │
├─────────────────────────────────────────────────────────────────┤
│  Routes                                                          │
│  ├── /health, /health/ready, /health/models                     │
│  ├── /api/llm/* (chat, stream, structured, usage)               │
│  ├── /api/sessions/* (create, chat, end, info)                  │
│  ├── /api/guard/* (evaluate, checks)                            │
│  ├── /api/usage/* (daily, recent, total)                        │
│  ├── /api/twentyq/* (hints, answer, verify, normalize, synonym) │
│  └── /api/turtle-soup/* (answer, hint, validate, reveal, ...)   │
├─────────────────────────────────────────────────────────────────┤
│  Infrastructure                                                  │
│  ├── GeminiClient (langchain-google-genai)                      │
│  ├── LangGraphSessionManager (Redis checkpointer)               │
│  ├── InjectionGuard (Aho-Corasick + Regex)                      │
│  ├── UsageRepository (PostgreSQL asyncpg)                       │
│  └── Callbacks (logging, metrics, DB upsert)                    │
├─────────────────────────────────────────────────────────────────┤
│  External Dependencies                                           │
│  ├── Google Gemini API                                          │
│  ├── Redis Stack (세션 히스토리)                                   │
│  └── PostgreSQL (토큰 사용량 추적)                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 2. 파일 구조 및 코드 규모

| 디렉토리 | 파일 수 | 주요 역할 |
|----------|---------|----------|
| `routes/` | 10 | HTTP 엔드포인트 (health, llm, session, guard, usage, twentyq, turtle_soup) |
| `infra/` | 6 | 인프라스트럭처 (gemini_client, session_store, injection_guard, usage_repository, callbacks) |
| `domains/twentyq/` | 4+ | Twenty Questions 도메인 (models, prompts, YAML 프롬프트 5개) |
| `domains/turtle_soup/` | 5+ | Turtle Soup 도메인 (models, prompts, puzzle_loader, YAML 프롬프트 6개, puzzles) |
| `config/` | 3 | 설정 (settings, logging) |
| `models/` | 3 | 공통 모델 (error, guard, stream) |
| `utils/` | 5+ | 유틸리티 (text, session, prompts, decorators, unicode) |

**총 Python 파일**: ~50개, **총 라인 수**: ~8,000줄 (추정)

### 3. 핵심 기능 목록

#### 3.1 LLM Core (`/api/llm/*`)
| 엔드포인트 | 메서드 | 기능 |
|-----------|--------|------|
| `/api/llm/chat` | POST | 단일 채팅 (stateless) |
| `/api/llm/chat-with-usage` | POST | 채팅 + 토큰 사용량 반환 |
| `/api/llm/structured` | POST | JSON 스키마 기반 구조화된 출력 |
| `/api/llm/usage` | GET | 인메모리 메트릭 |
| `/api/llm/usage/total` | GET | DB 기반 총 사용량 |
| `/api/llm/metrics` | GET | 집계된 LLM 메트릭 |

> **Note**: SSE/스트리밍 엔드포인트 (`/api/llm/stream`, `/api/llm/stream-events`) 불필요로 제외.

#### 3.2 Session Management (`/api/sessions/*`)
| 엔드포인트 | 메서드 | 기능 |
|-----------|--------|------|
| `/api/sessions` | POST | 세션 생성 |
| `/api/sessions/{id}/messages` | POST | 세션 기반 채팅 |
| `/api/sessions/{id}` | DELETE | 세션 종료 |
| `/api/sessions/{id}` | GET | 세션 정보 조회 |

#### 3.3 Injection Guard (`/api/guard/*`)
| 엔드포인트 | 메서드 | 기능 |
|-----------|--------|------|
| `/api/guard/evaluations` | POST | 입력 평가 (score, hits, threshold) |
| `/api/guard/checks` | POST | 빠른 악성 여부 체크 |

#### 3.4 Usage Tracking (`/api/usage/*`)
| 엔드포인트 | 메서드 | 기능 |
|-----------|--------|------|
| `/api/usage/daily` | GET | 오늘 사용량 |
| `/api/usage/recent` | GET | 최근 N일 사용량 |
| `/api/usage/total` | GET | N일간 총 사용량 |

#### 3.5 Twenty Questions (`/api/twentyq/*`)
| 엔드포인트 | 메서드 | 기능 |
|-----------|--------|------|
| `/api/twentyq/hints` | POST | 힌트 생성 |
| `/api/twentyq/answers` | POST | 예/아니오 질문 답변 |
| `/api/twentyq/verifications` | POST | 정답 검증 (ACCEPT/CLOSE/REJECT) |
| `/api/twentyq/normalizations` | POST | 질문 정규화 |
| `/api/twentyq/synonym-checks` | POST | 동의어 체크 |

**프롬프트 파일**: `hints.yml`, `answer.yml`, `verify-answer.yml`, `normalize.yml`, `synonym-check.yml`

#### 3.6 Turtle Soup (`/api/turtle-soup/*`)
| 엔드포인트 | 메서드 | 기능 |
|-----------|--------|------|
| `/api/turtle-soup/answers` | POST | 플레이어 질문에 답변 |
| `/api/turtle-soup/hints` | POST | 힌트 생성 (레벨 1-3) |
| `/api/turtle-soup/validations` | POST | 정답 검증 (YES/NO/CLOSE) |
| `/api/turtle-soup/reveals` | POST | 정답 공개 (드라마틱 내레이션) |
| `/api/turtle-soup/puzzles` | POST | 새 퍼즐 생성 |
| `/api/turtle-soup/rewrites` | POST | 퍼즐 재작성 |
| `/api/turtle-soup/puzzles` | GET | 전체 퍼즐 목록 |
| `/api/turtle-soup/puzzles/random` | GET | 랜덤 퍼즐 |
| `/api/turtle-soup/puzzles/{id}` | GET | ID로 퍼즐 조회 |
| `/api/turtle-soup/puzzles/reload` | POST | 퍼즐 핫 리로드 |

**프롬프트 파일**: `answer.yml`, `hint.yml`, `validate.yml`, `reveal.yml`, `generate.yml`, `rewrite.yml`

#### 3.7 Health & Config
| 엔드포인트 | 메서드 | 기능 |
|-----------|--------|------|
| `/health` | GET | 헬스 체크 (deep checks) |
| `/health/ready` | GET | 레디니스 프로브 |
| `/health/models` | GET | 모델 설정 스냅샷 |

### 4. 인프라스트럭처 컴포넌트

#### 4.1 GeminiClient
- LangChain `ChatGoogleGenerativeAI` 래핑
- 기능: chat, stream, chat_structured
- Thinking 설정 (Gemini 3: level, Gemini 2.5: budget)
- LLM 캐싱 (모델+태스크별)
- Tool calling 지원

#### 4.2 SessionStoreManager
- Valkey/InMemory 저장소 (히스토리 저장)
- 세션 메타데이터 Valkey 저장
- 만료 세션 자동 정리
- TTL 기반 세션 관리

#### 4.3 InjectionGuard
- Aho-Corasick 구문 매칭
- Regex 패턴 매칭
- 자모만 입력 차단 (한국어 공격 벡터)
- 이모지 차단
- 결과 캐싱 (TTLCache)

#### 4.4 UsageRepository
- PostgreSQL asyncpg
- 일별 토큰 사용량 upsert
- 배치 플러싱 (옵션)
- 집계 쿼리

#### 4.5 Callbacks
- `LoggingCallbackHandler`: 구조화된 로깅
- `MetricsCallbackHandler`: 토큰 사용량 추적 + DB 기록

### 5. 설정 구조

| 설정 그룹 | 환경 변수 예시 |
|-----------|---------------|
| **Gemini** | `GOOGLE_API_KEY(S)`, `GEMINI_MODEL`, `GEMINI_TEMPERATURE`, `GEMINI_THINKING_*` |
| **Session** | `MAX_SESSIONS`, `SESSION_TTL_MINUTES`, `SESSION_HISTORY_MAX_PAIRS` |
| **Session Store** | `SESSION_STORE_URL`, `SESSION_STORE_ENABLED`, `SESSION_STORE_REQUIRED`, `SESSION_STORE_CONNECT_*` |
| **Guard** | `GUARD_ENABLED`, `GUARD_THRESHOLD` |
| **Logging** | `LOG_LEVEL`, `LOG_DIR`, `LOG_FILE_MAX_SIZE_MB`, `LOG_FILE_MAX_BACKUPS`, `LOG_FILE_MAX_AGE_DAYS`, `LOG_FILE_COMPRESS` |
| **HTTP** | `HTTP_HOST`, `HTTP_PORT`, `HTTP2_ENABLED` |
| **Auth** | `HTTP_API_KEY` |
| **RateLimit** | `HTTP_RATE_LIMIT_RPM` |
| **Database** | `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD` |

> **Note**: 단위 테스트(miniredis)는 RESP2-only라 valkey-go가 `DisableCache=true`를 요구함. 서버 기본 설정은 cache 활성(DisableCache=false)이며, 테스트에서만 config 주입으로 끈다.

---

## 마이그레이션 전략

### 접근법: **직접 구현 (google-genai SDK + valkey-go)**

**선정 이유:**
1. 세션 그래프 계층의 실제 활용도 낮음 (passthrough only)
2. Eino 과도함 (불필요한 추상화)
3. 최소 의존성으로 완전한 제어

### 목표 아키텍처

```
┌─────────────────────────────────────────────────────────────────┐
│                    HTTP Server (Gin + h2c)                      │
├─────────────────────────────────────────────────────────────────┤
│  Handlers (기존 routes 대응)                                     │
│  ├── health.go                                                  │
│  ├── llm.go                                                     │
│  ├── session.go                                                 │
│  ├── guard.go                                                   │
│  ├── usage.go                                                   │
│  ├── twentyq.go                                                 │
│  └── turtle_soup.go                                             │
├─────────────────────────────────────────────────────────────────┤
│  Services                                                        │
│  ├── gemini/client.go (google.golang.org/genai)                 │
│  ├── session/manager.go (Valkey)                                │
│  ├── guard/guard.go (Aho-Corasick)                              │
│  └── usage/repository.go (GORM + PostgreSQL)                    │
├─────────────────────────────────────────────────────────────────┤
│  Domain                                                          │
│  ├── twentyq/ (models, prompts)                                 │
│  └── turtlesoup/ (models, prompts, puzzles)                     │
├─────────────────────────────────────────────────────────────────┤
│  Infrastructure                                                  │
│  ├── Config (godotenv)                                          │
│  ├── Logging (slog+tint+lumberjack)                             │
│  └── DI (Wire)                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Go 의존성 (현재 go.mod 기준)

```go
require (
    github.com/alicebob/miniredis/v2 v2.35.0
    github.com/cloudflare/ahocorasick v0.0.0-20240916140611-054963ec9396
    github.com/gin-gonic/gin v1.11.0
    github.com/go-playground/validator/v10 v10.30.0
    github.com/goccy/go-json v0.10.5
    github.com/google/wire v0.7.0
    github.com/joho/godotenv v1.5.1
    github.com/valkey-io/valkey-go v1.0.69
    go.uber.org/zap v1.27.1
    golang.org/x/net v0.48.0
    golang.org/x/sync v0.19.0
    golang.org/x/text v0.32.0
    google.golang.org/genai v1.40.0
    gopkg.in/yaml.v3 v3.0.1
    gorm.io/driver/postgres v1.6.0
    gorm.io/gorm v1.31.1
)
```

**정책/구현 메모**
- Gemini 3.x only (2.5 계열 미지원)
- thinking_budget 미지원 (thinking_level만 사용)
- JSON: `github.com/goccy/go-json` 사용
- **TOON 포맷 사용**: 외부 의존성 없이 `internal/toon` 사용

---

## 마이그레이션 단계

### Phase 1: 핵심 인프라 

| 작업 | 설명 | 우선순위 |
|------|------|---------|
| **1.1 프로젝트 초기화** | Go 모듈, 디렉토리 구조, 기본 설정 | P0 |
| **1.2 Config** | 환경 변수 파싱 (기존과 동일한 변수명) | P0 |
| **1.3 Wire 설정** | Provider 정의, wire.go 생성, 의존성 그래프 | P0 |
| **1.4 GeminiClient** | google.golang.org/genai(v1.40.0) 래핑, chat/stream/structured | P0 |
| **1.5 HTTP Server** | Gin 라우터, CORS, 에러 핸들링, h2c | P0 |
| **1.6 Health 엔드포인트** | `/health`, `/health/ready`, `/health/models` | P0 |

### Phase 2: LLM 코어 

| 작업 | 설명 | 우선순위 |
|------|------|---------|
| **2.1 Chat 엔드포인트** | `/api/llm/chat`, `/api/llm/chat-with-usage` | P0 |
| **2.2 Stream 엔드포인트** | `/api/llm/stream`, `/api/llm/stream-events` (SSE) | P0 |
| **2.3 Structured 엔드포인트** | `/api/llm/structured` (아래 상세) | P1 |

#### 2.3 Structured Output 상세

| 작업 | 설명 | 우선순위 |
|------|------|---------|
| **2.3.1 정적 스키마** | 도메인별 Output 타입 (HintsOutput, AnswerOutput 등) 수동 정의 | P0 |
| **2.3.2 jsonschema 연동** | `invopop/jsonschema`로 Go struct → JSON Schema 생성 | P1 |
| **2.3.3 변환 유틸리티** | JSON Schema → genai.Schema 변환 (`internal/gemini/structured.go`) | P1 |
| **2.3.4 엔드포인트 구현** | 클라이언트 JSON Schema 수신 → genai.Schema 변환 → 응답 | P1 |


### Phase 3: 세션 관리 

| 작업 | 설명 | 우선순위 |
|------|------|---------|
| **3.1 SessionManager** | Redis 기반 세션 저장/조회/삭제 | P0 |
| **3.2 Session 엔드포인트** | `/api/sessions/*` | P0 |
| **3.3 History 관리** | 대화 히스토리 저장/조회 | P0 |

### Phase 4: 보안 & 모니터링 

| 작업 | 설명 | 우선순위 |
|------|------|---------|
| **4.1 InjectionGuard** | Aho-Corasick + Regex, YAML rulepack 로딩 | P1 |
| **4.2 Guard 엔드포인트** | `/api/guard/*` | P1 |
| **4.3 UsageRepository** | PostgreSQL 연동, 토큰 사용량 기록 | P1 |
| **4.4 Usage 엔드포인트** | `/api/usage/*` | P1 |

### Phase 5: Twenty Questions 도메인 

| 작업 | 설명 | 우선순위 |
|------|------|---------|
| **5.1 프롬프트 로더** | YAML 파일 로딩, 템플릿 치환 | P0 |
| **5.2 Models** | HintsOutput, AnswerOutput, VerifyResult 등 | P0 |
| **5.3 Handlers** | hints, answer, verify, normalize, synonym | P0 |

### Phase 6: Turtle Soup 도메인 

| 작업 | 설명 | 우선순위 |
|------|------|---------|
| **6.1 퍼즐 로더** | YAML 퍼즐 파일 로딩, 핫 리로드 | P0 |
| **6.2 프롬프트 로더** | YAML 프롬프트 로딩 | P0 |
| **6.3 Models** | AnswerResult, HintResult, ValidateResult 등 | P0 |
| **6.4 Handlers** | answer, hint, validate, reveal, generate, rewrite, puzzles | P0 |

### Phase 7: 미들웨어 & 완성도 

| 작업 | 설명 | 우선순위 |
|------|------|---------|
| **7.1 인증 미들웨어** | API Key 검증 | P1 |
| **7.2 레이트 리밋** | Fixed-window per minute | P2 |
| **7.3 로깅** | 구조화된 로깅 (slog+tint+lumberjack) | P1 |
| **7.4 Graceful Shutdown** | lifespan 관리 | P1 |

### Phase 8: 테스트 & 배포 

| 작업 | 설명 | 우선순위 |
|------|------|---------|
| **8.0 Lint** | golangci-lint 도입 (game-bot-go 설정 재사용) | P1 |
| **8.1 단위 테스트** | 핵심 로직 테스트 | P1 |
| **8.2 통합 테스트** | API 엔드포인트 테스트 | P1 |
| **8.3 Dockerfile** | 멀티스테이지 빌드 | P0 |
| **8.4 기존 Python 서버와 병렬 운영** | 점진적 전환 | P2 |

---

## 파일 매핑

| Python | Go |
|--------|-----|
| `http_server.py` | `cmd/server/main.go` |
| `config/settings.py` | `internal/config/config.go` |
| `routes/health.py` | `internal/handler/health.go` |
| `routes/llm.py` | `internal/handler/llm.go` |
| `routes/session.py` | `internal/handler/session.go` |
| `routes/guard.py` | `internal/handler/guard.go` |
| `routes/usage.py` | `internal/handler/usage.go` |
| `routes/twentyq.py` | `internal/handler/twentyq.go` |
| `routes/turtle_soup.py` | `internal/handler/turtlesoup.go` |
| `infra/gemini_client.py` | `internal/gemini/client.go` |
| `infra/session_store.py` | `internal/session/manager.go` |
| `infra/injection_guard.py` | `internal/guard/guard.go` |
| `infra/usage_repository.py` | `internal/usage/repository.go` |
| `domains/twentyq/*` | `internal/domain/twentyq/*` |
| `domains/turtle_soup/*` | `internal/domain/turtlesoup/*` |

---

## 디렉토리 구조 (Go)

```
mcp-llm-server-go/
├── cmd/
│   └── server/
│       └── main.go              # 진입점
├── internal/
│   ├── config/
│   │   └── config.go            # 환경 변수 설정
│   ├── di/
│   │   ├── app.go               # App 컨테이너
│   │   ├── providers.go         # Wire provider
│   │   ├── wire.go              # Wire 정의 (+build wireinject)
│   │   └── wire_gen.go          # 자동 생성 (wire)
│   ├── gemini/
│   │   ├── client.go            # Gemini SDK 래핑
│   ├── health/
│   │   └── health.go            # 헬스 수집 로직
│   ├── llm/
│   │   └── types.go             # LLM 공통 타입
│   ├── metrics/
│   │   └── metrics.go           # 인메모리 메트릭
│   ├── server/
│   │   └── http.go              # HTTP 서버 구성
│   ├── session/
│   │   ├── manager.go           # 세션 관리
│   │   └── store.go             # Redis 저장소
│   ├── guard/
│   │   ├── guard.go             # 인젝션 가드
│   │   └── rulepack.go          # YAML 룰팩 로더
│   ├── usage/
│   │   └── repository.go        # PostgreSQL 저장소
│   ├── domain/
│   │   ├── twentyq/
│   │   │   ├── models.go
│   │   │   ├── prompts.go
│   │   │   └── prompts/         # YAML 복사
│   │   └── turtlesoup/
│   │       ├── models.go
│   │       ├── prompts.go
│   │       ├── puzzle_loader.go
│   │       ├── prompts/          # YAML 복사
│   │       └── puzzles/          # YAML 복사
│   ├── handler/
│   │   ├── health.go
│   │   ├── llm.go
│   │   ├── session.go
│   │   ├── guard.go
│   │   ├── usage.go
│   │   ├── twentyq.go
│   │   └── turtlesoup.go
│   └── middleware/
│       ├── auth.go
│       ├── ratelimit.go
│       └── logging.go
├── rulepacks/                    # 복사
├── Dockerfile
├── Dockerfile.prod
├── go.mod
├── go.sum
└── README.md
```

---

## 예상 일정

| Phase | 작업 | 예상 시간 |
|-------|------|----------|
| 1 | 핵심 인프라 | 1-2일 |
| 2 | LLM 코어 | 1일 |
| 3 | 세션 관리 | 1일 |
| 4 | 보안 & 모니터링 | 0.5일 |
| 5 | Twenty Questions | 1일 |
| 6 | Turtle Soup | 1일 |
| 7 | 미들웨어 & 완성도 | 0.5일 |
| 8 | 테스트 & 배포 | 1일 |
| **합계** | | **7-8일** |

---

## 진행 현황 (2025-12-22)

### ✅ Phase 1: 핵심 인프라 (100% 완료)
- 1.1 프로젝트 초기화 (`go.mod`, 디렉토리 구조)
- 1.2 Config (`internal/config/config.go`)
- 1.3 Wire 설정 (`internal/di/`)
- 1.4 GeminiClient (`internal/gemini/client.go` - Chat/Structured)
- 1.5 HTTP Server (`internal/server/http.go` - Gin + h2c)
- 1.6 Health 엔드포인트 (`/health`, `/health/ready`, `/health/models`)
- golangci-lint 설정 적용 (`.golangci.yml`)

### ✅ Phase 2: LLM 코어 (완료)
- 2.1 Chat: `POST /api/llm/chat`, `POST /api/llm/chat-with-usage`
- 2.2 Structured: `POST /api/llm/structured` (클라이언트 JSON Schema 직접 사용)
- 인메모리 메트릭: `GET /api/llm/usage`, `GET /api/llm/metrics`

> **Note**: SSE/스트리밍 불필요로 제외 (`/api/llm/stream`, `/api/llm/stream-events`).

> **Note**: genai SDK는 `ResponseJsonSchema` (map[string]any)를 직접 지원하여 클라이언트 JSON Schema를 그대로 사용 가능.
> TwentyQ 도메인 스키마는 map 기반으로 적용 완료.

### ✅ Phase 3: 세션 관리 (완료)
- 3.1 SessionStore (`internal/session/store.go`) - Valkey 필수 연결
- 3.2 SessionManager (`internal/session/manager.go`)
- 3.3 Session 핸들러 (`internal/handler/session.go`)
- 엔드포인트: `POST /api/sessions`, `GET /api/sessions/:id`, `POST /api/sessions/:id/messages`, `DELETE /api/sessions/:id`

### ✅ Phase 4: 보안/모니터링 (완료)
- 4.1 InjectionGuard (Aho-Corasick + Regex, YAML rulepacks)
- 4.2 Guard 엔드포인트 (`/api/guard/evaluations`, `/api/guard/checks`)
- 4.3 UsageRepository (GORM + PostgreSQL, 배치 옵션 포함)
- 4.4 Usage 엔드포인트 (`/api/usage/daily`, `/api/usage/recent`, `/api/usage/total`)
- `/api/llm/usage/total` DB 집계 연동 완료
- HTTP API Key/RateLimit/Request-ID 미들웨어 추가 (기본 비활성)

### ✅ Phase 5: Twenty Questions 도메인 (완료)
- 5.1 프롬프트 로더 (`internal/prompt/*`, `internal/domain/twentyq/prompts/*.yml`)
- 5.2 Models (`internal/domain/twentyq/models.go`)
- 5.3 핸들러/라우터/DI (`internal/handler/twentyq.go`, `/api/twentyq/*`)

### ✅ Phase 6: Turtle Soup 도메인 (완료)
- 6.1 퍼즐 로더 (`internal/domain/turtlesoup/puzzle_loader.go`, `internal/domain/turtlesoup/puzzles/*.json`)
- 6.2 프롬프트 로더 (`internal/domain/turtlesoup/prompts.go`, `internal/domain/turtlesoup/prompts/*.yml`)
- 6.3 Models/스키마 (`internal/domain/turtlesoup/models.go`)
- 6.4 핸들러/라우터/DI (`internal/handler/turtlesoup.go`, `/api/turtle-soup/*`, wire 반영)

### ✅ Phase 7: 미들웨어 & 완성도 (완료)
- 7.1 API Key 인증 미들웨어 (`internal/middleware/auth.go`)
- 7.2 레이트 리밋 미들웨어 (`internal/middleware/ratelimit.go`)
- 7.3 구조화 로깅 (slog+tint+lumberjack, `logs/server.log`)
- 7.4 Graceful Shutdown (SIGTERM/SIGINT, `cmd/server/main.go`)

---

## 마이그레이션 체크리스트

### 기능 패리티
- [x] `/health`, `/health/ready`, `/health/models`
- [x] `/api/llm/chat`, `/api/llm/chat-with-usage`, `/api/llm/structured`
- [x] `/api/sessions/*` ✅
- [x] `/api/guard/*`
- [x] `/api/usage/*`
- [x] `/api/twentyq/*`
- [x] `/api/turtle-soup/*`

### 인프라
- [x] Gemini API 연동 (Gemini 3.x only, thinking_level, structured output)
- [x] Valkey 세션 저장 ✅
- [x] PostgreSQL 토큰 추적
- [x] Aho-Corasick 인젝션 가드

### 설정
- [x] 환경 변수 Config 골격 반영
- [x] TwentyQ YAML 프롬프트 호환
- [x] Turtle Soup YAML 프롬프트/퍼즐 파일 호환
- [x] Gemini 3.x only (GEMINI_THINKING_BUDGET_* 미지원)

### 품질
- [x] golangci-lint 적용 (game-bot-go 설정 재사용)
- [x] go test ./... ✅ (unit tests 추가 + miniredis 기반 session store 테스트 포함)

### 코드 품질 개선 (2025-12-22)
- [x] `internal/handler/shared` 패키지 생성 및 공통 함수 추출
  - `response.go`: WriteError, BindJSON, BindJSONAllowEmpty
  - `session.go`: ResolveSessionID, BuildRecentQAHistoryContext, ValueOrEmpty
  - `parse.go`: ParseStringField, ParseStringSlice, SerializeDetails, TrimRunes
  - `logging.go`: LogError
  - `constants.go`: 한국어 메시지 상수 (MsgSafetyBlock, MsgInvalidQuestion 등)
- [x] `twentyq.go` 리팩토링 (740줄 → 636줄, -104줄)
- [x] `turtlesoup.go` 리팩토링 (shared 패키지 사용)
- [x] 인터페이스 추출 (테스트 용이성 개선)
  - `session/storage.go`: Storage 인터페이스
  - `usage/store.go`: UsageStore 인터페이스
  - `guard/interface.go`: Guard 인터페이스
  - `gemini/interface.go`: LLM 인터페이스
- [x] shared 패키지 테스트 추가 (커버리지 70.1%)
- [x] 테스트 커버리지 개선 (config 27%→45%, httperror 53%→77%)

### 배포
- [x] Docker 이미지 빌드 (mcp-llm-server:latest, 80.1MB)
- [x] Makefile docker-build/docker-push 타겟 추가
- [ ] docker-compose.prod.yml 업데이트

---

## 위험 요소 및 완화 방안

| 위험 요소 | 영향 | 완화 방안 |
|-----------|------|----------|
| ~~Gemini Go SDK 기능 누락~~ | ~~중간~~ | ✅ 해결 - `ResponseJsonSchema` 직접 지원 확인 |
| ~~스트리밍 구현 복잡도~~ | ~~중간~~ | ✅ 해결 - 스트리밍 요구사항 제거 (/api/llm/stream 미구현) |
| Structured Output 스키마 변환 | 낮음 | map 기반 스키마로 TwentyQ 적용 완료 |
| 프롬프트 템플릿 호환성 | 낮음 | internal/prompt.FormatTemplate 사용, 기존 YAML 그대로 유지 |
| 배포 중 다운타임 | 높음 | 블루-그린 배포, 점진적 트래픽 전환 |

---

## 완료 상태

1. ~~Phase 3 세션 관리~~ ✅ 완료
2. ~~Phase 5 Twenty Questions 도메인~~ ✅ 완료
3. ~~Phase 6 Turtle Soup 도메인~~ ✅ 완료
4. ~~Phase 7 미들웨어 & 완성도~~ ✅ 완료
5. ~~코드 품질 개선~~ ✅ 완료
   - shared 패키지 추출 (104줄 감소)
   - 인터페이스 추출 (4개 패키지)
   - 상수 분리
6. ~~Phase 8 Docker 빌드~~ ✅ 완료
   - Docker 이미지 빌드 (80.1MB)
   - Makefile 타겟 추가

## 다음 단계

- [ ] docker-compose.prod.yml 업데이트
- [ ] 기존 서버와 병렬 운영 테스트
- [ ] 프로덕션 배포

---

*최종 업데이트: 2025-12-22*
*버전: 2.0*

