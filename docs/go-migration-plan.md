# Go 마이그레이션 계획

> 작성일: 2025-12-14  
> 대상: `20q-kakao-bot`, `turtle-soup-bot`  
> 상태: 초안 (Draft)

---

## 1. 개요

### 1.1 배경
- LLM 레이어가 `mcp-llm-server` (Python FastAPI)로 완전히 이관 완료
- 봇들은 이제 순수 HTTP REST 클라이언트로 동작
- Kotlin + Spring/Ktor 스택의 복잡도 대비 실제 역할이 단순화됨

### 1.2 마이그레이션 목표
- [ ] 런타임 리소스 최적화 (메모리, 시작 시간)
- [ ] 배포 단순화 (단일 바이너리, CGO 없음)
- [ ] 개발/유지보수 통일 (Go 단일 스택)
- [ ] 기존 기능 100% 호환

### 1.3 현재 스택

| 컴포넌트 | 현재 스택 | 코드 규모 |
|----------|-----------|-----------|
| `20q-kakao-bot` | Kotlin 2.3 + Spring Boot 4.0 (WebFlux) + R2DBC + Redis | ~186 파일 (main) |
| `turtle-soup-bot` | Kotlin 2.3 + Ktor 3.3 + Koin + Redis | ~60 파일 (main) |
| `mcp-llm-server` | Python 3.13 + FastAPI + LangChain | 유지 (마이그레이션 대상 아님) |

---

## 2. 현재 아키텍처 분석

### 2.1 20q-kakao-bot (스무고개봇)

```
party.qwer.twentyq/
├── api/              # 외부 API 클라이언트 (LLM 서버 호출)
├── bridge/           # 카카오톡 브릿지 핸들러
├── config/           # Spring 설정, AppProperties
├── logging/          # 로깅 설정
├── mcp/              # MCP 관련 (미사용?)
├── model/            # 도메인 모델
├── mq/               # 메시지 큐 (Redis Streams?)
├── redis/            # Redis 세션/캐시
├── repository/       # R2DBC 리포지토리 (PostgreSQL)
├── rest/             # REST 컨트롤러
├── security/         # 보안 설정
├── service/          # 비즈니스 로직 (~40 파일)
└── util/             # 유틸리티
```

**주요 의존성:**
- Spring Boot 4.0 WebFlux (리액티브 HTTP)
- Spring Data R2DBC (PostgreSQL)
- Redisson (Redis 클라이언트)
- Ktor Client (LLM 서버 호출)
- Jackson (JSON 처리)

### 2.2 turtle-soup-bot (바다거북스프봇)

```
io.github.kapu.turtlesoup/
├── api/              # 외부 API 클라이언트
├── bridge/           # 카카오톡 브릿지
├── config/           # Koin 모듈, 설정
├── models/           # 도메인 모델
├── mq/               # 메시지 큐
├── redis/            # Redis 세션/캐시
├── rest/             # REST 라우팅
├── security/         # 보안
├── service/          # 비즈니스 로직
└── utils/            # 유틸리티
```

**주요 의존성:**
- Ktor 3.3 Server (Netty)
- Ktor Client (LLM 서버 호출)
- Koin 4.0 (DI)
- Redisson (Redis)
- kotlinx.serialization (JSON)

---

## 3. Go 마이그레이션 설계

### 3.1 제안 Go 스택

| 영역 | Kotlin | Go(1.25+) 대체 |
|------|--------|---------|
| HTTP 서버 | Spring WebFlux / Ktor | `net/fasthttp` + Chi 또는 Fiber |
| HTTP 클라이언트 | Ktor Client | `net/fasthttp` (표준 라이브러리) |
| DI | Spring / Koin | Fx (Uber) - 라이프사이클 내장 |
| JSON | Jackson / kotlinx.serialization | `encoding/json` 또는 sonic |
| Redis | Redisson | go-redis/redis v9 |
| PostgreSQL | R2DBC | GORM + pgx (ORM) |
| 설정 | Spring properties / HOCON | Viper 또는 envconfig |
| 로깅 | Logback / kotlin-logging | slog (Go 1.25+) 또는 zerolog |
| 테스트 | JUnit5 + MockK | testing + testify |

### 3.2 제안 패키지 구조

```
cmd/
├── twentyq/           # 스무고개봇 엔트리포인트
│   └── main.go
└── turtlesoup/        # 바다거북봇 엔트리포인트
    └── main.go

internal/
├── common/            # 공유 코드
│   ├── config/        # Viper 설정
│   ├── http/          # HTTP 클라이언트 래퍼
│   ├── redis/         # Redis 클라이언트
│   ├── mq/            # 메시지 큐 (Redis Streams)
│   └── security/      # 보안 유틸리티
│
├── twentyq/           # 스무고개 전용
│   ├── handler/       # HTTP 핸들러
│   ├── service/       # 비즈니스 로직
│   ├── model/         # 도메인 모델
│   ├── bridge/        # 카카오톡 브릿지
│   └── repository/    # PostgreSQL 리포지토리
│
└── turtlesoup/        # 바다거북 전용
    ├── handler/
    ├── service/
    ├── model/
    └── bridge/

pkg/                   # 공개 가능한 유틸리티 (필요시)
```

### 3.3 모놀리스

**옵션 A: 단일 바이너리 (권장)**
- 장점: 배포 단순화, 리소스 공유, 코드 재사용
- 단점: 개별 스케일링 불가
- 구현: `cmd/bot/main.go`에서 두 봇 모두 구동

---

## 4. 마이그레이션 단계

### Phase 1: 기반 구축 

| 태스크 | 우선순위 | 설명 |
|--------|----------|------|
| Go 모듈 초기화 | P0 | `go mod init`, 디렉토리 구조 |
| 설정 로더 | P0 | Viper + .env 로딩 |
| 로깅 설정 | P0 | slog 구조화 로그 |
| Redis 클라이언트 | P0 | go-redis 연결, 풀 설정 |
| HTTP 클라이언트 | P0 | LLM 서버 호출 래퍼 |
| PostgreSQL 연결 | P1 | GORM + pgx, AutoMigrate |

### Phase 2: 공통 레이어 

| 태스크 | 우선순위 | 설명 |
|--------|----------|------|
| 메시지 큐 | P0 | Redis Streams 소비자/생산자 |
| 카카오 브릿지 | P0 | 웹훅 수신, 응답 포맷 |
| 보안 유틸 | P1 | 입력 검증, 레이트 리밋 |

### Phase 3: turtle-soup-bot 전환

> 더 작은 규모(~60 파일)로 먼저 시작

| 태스크 | 우선순위 | 설명 |
|--------|----------|------|
| 도메인 모델 | P0 | Kotlin → Go 구조체 변환 |
| 서비스 레이어 | P0 | 비즈니스 로직 포팅 |
| HTTP 핸들러 | P0 | Ktor 라우트 → Chi/Fiber |
| 통합 테스트 | P1 | 기존 동작 검증 |
| Docker 이미지 | P1 | 멀티스테이지 빌드 |

### Phase 4: 20q-kakao-bot 전환 

> 더 큰 규모(~186 파일, R2DBC 포함)

| 태스크 | 우선순위 | 설명 |
|--------|----------|------|
| 도메인 모델 | P0 | 모델 변환 |
| R2DBC → GORM | P0 | 리포지토리 패턴 유지, 모델 매핑 |
| 서비스 레이어 | P0 | Spring 서비스 → Go 서비스 |
| WebFlux → net/http | P0 | 리액티브 → 동기 (goroutine) |
| 캐시 레이어 | P1 | Caffeine → go-cache 또는 ristretto |
| 스케줄러 | P1 | @Scheduled → cron 라이브러리 |

---

## 5. 리스크 및 대응

| 리스크 | 영향 | 대응 |
|--------|------|------|
| R2DBC 리액티브 패턴 전환 어려움 | 낮음 | GORM은 Spring Data JPA와 유사한 패턴, Repository 구조 유지 가능 |
| Coroutines → Goroutines 패턴 차이 | 중간 | 채널 기반 동시성으로 대체, select 문 활용 |
| Redisson 고급 기능 재구현 | 중간 | go-redis 기본 기능 + 커스텀 래퍼 |
| 기존 테스트 케이스 손실 | 높음 | 통합 테스트 포팅 우선, 동작 검증 |
| 개발 일정 지연 | 중간 | Phase별 완료 기준 명확화, 점진적 전환 |

---

## 6. Go 코드 스니펫 (참고)

### 6.1 HTTP 서버 (Chi)

```go
package main

import (
    "log/slog"
    "net/http"
    "os"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
)

func main() {
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    slog.SetDefault(logger)

    r := chi.NewRouter()
    r.Use(middleware.RequestID)
    r.Use(middleware.RealIP)
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)

    r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("ok"))
    })

    r.Route("/api/webhook", func(r chi.Router) {
        r.Post("/kakao", handleKakaoWebhook)
    })

    slog.Info("Server starting", "port", 8080)
    http.ListenAndServe(":8080", r)
}
```

### 6.2 Redis Streams 소비자

```go
package mq

import (
    "context"
    "log/slog"

    "github.com/redis/go-redis/v9"
)

type StreamConsumer struct {
    client *redis.Client
    stream string
    group  string
    name   string
}

func (c *StreamConsumer) Consume(ctx context.Context, handler func(msg redis.XMessage) error) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        streams, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
            Group:    c.group,
            Consumer: c.name,
            Streams:  []string{c.stream, ">"},
            Count:    10,
            Block:    0,
        }).Result()

        if err != nil {
            slog.Error("XReadGroup failed", "error", err)
            continue
        }

        for _, stream := range streams {
            for _, msg := range stream.Messages {
                if err := handler(msg); err != nil {
                    slog.Error("Handler failed", "id", msg.ID, "error", err)
                    continue
                }
                c.client.XAck(ctx, c.stream, c.group, msg.ID)
            }
        }
    }
}
```

### 6.3 LLM 클라이언트

```go
package llm

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

type Client struct {
    baseURL    string
    httpClient *http.Client
}

type ChatRequest struct {
    SessionID string `json:"session_id"`
    Message   string `json:"message"`
}

type ChatResponse struct {
    Response string `json:"response"`
    Usage    *Usage `json:"usage,omitempty"`
}

func (c *Client) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
    body, _ := json.Marshal(req)

    httpReq, err := http.NewRequestWithContext(
        ctx,
        http.MethodPost,
        c.baseURL+"/api/llm/chat",
        bytes.NewReader(body),
    )
    if err != nil {
        return nil, err
    }
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
    }

    var result ChatResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    return &result, nil
}
```

---

## 7. 결정 필요 항목

- [x] **모놀리스**: 단일 바이너리 (결정됨)
- [x] **DI 프레임워크**: Fx (결정됨) - 라이프사이클 내장, Spring/Koin 유사
- [x] **마이그레이션 우선순위**: turtle-soup 먼저 (결정됨) - 규모 작아 검증 용이
- [x] **PostgreSQL ORM**: GORM + pgx (결정됨)

---

## 8. 참고 자료

- [Go by Example](https://gobyexample.com/)
- [go-redis/redis](https://github.com/redis/go-redis)
- [chi router](https://github.com/go-chi/chi)
- [GORM ORM](https://gorm.io/)
- [pgx PostgreSQL driver](https://github.com/jackc/pgx)
- [Viper configuration](https://github.com/spf13/viper)
- [Fx DI](https://github.com/uber-go/fx)

---

## 변경 이력

| 날짜 | 작성자 | 변경 내용 |
|------|--------|-----------|
| 2025-12-14 | - | 초안 작성 |
| 2025-12-14 | - | PostgreSQL 드라이버 pgx → GORM 변경 |
| 2025-12-14 | - | DI 프레임워크 Fx 확정 |
| 2025-12-14 | - | 모놀리스 확정, 마이그레이션 순서 turtle-soup 먼저 |
