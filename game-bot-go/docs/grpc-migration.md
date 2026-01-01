# gRPC 마이그레이션 가이드

> 최종 업데이트: 2025-12-31

## 개요

봇(game-bot-go)과 LLM 서버(mcp-llm-server-go) 간의 내부 통신이 **HTTP/REST에서 gRPC로 전환**되었습니다. 이 문서는 마이그레이션의 배경, 구현 세부사항, 운영 가이드를 제공합니다.

---

## 1. 마이그레이션 배경

### 1.1 기존 문제점

| 문제 | 설명 |
|------|------|
| **프로토콜 불일치** | HTTP와 gRPC 간 동작 차이로 인한 버그 발생 가능성 |
| **설정 실수** | `http://` 스킴 사용 시 HTTP로 통신되어 의도치 않은 동작 |
| **런타임 오류** | 잘못된 설정이 서비스 시작 후에야 발견됨 |
| **레거시 네이밍** | `LLM_REST_*` 환경 변수가 gRPC 통신을 제어하는 모순 |

### 1.2 마이그레이션 목표

1. **Fail-Fast**: 잘못된 설정을 시작 단계에서 즉시 차단
2. **프로토콜 일관성**: 모든 내부 통신을 gRPC로 통일
3. **명확한 네이밍**: 프로토콜 중립적 환경 변수명 사용
4. **데드 코드 제거**: 미사용 HTTP-only 메서드 정리

---

## 2. 아키텍처 변경

### 2.1 통신 흐름 (Before vs After)

```
[Before - HTTP/REST]
┌─────────────┐   HTTP/1.1 or h2c   ┌──────────────────┐
│ twentyq-bot │ ─────────────────── │ mcp-llm-server   │
│ turtlesoup  │    POST /api/...    │ (HTTP :40527)    │
└─────────────┘                     └──────────────────┘

[After - gRPC]
┌─────────────┐      gRPC/h2c       ┌──────────────────┐
│ twentyq-bot │ ─────────────────── │ mcp-llm-server   │
│ turtlesoup  │   LLMService RPC    │ (gRPC :40528)    │
└─────────────┘                     └──────────────────┘
```

### 2.2 포트 구성

| 서비스 | 포트 | 프로토콜 | 용도 |
|--------|------|----------|------|
| mcp-llm-server | 40527 | HTTP/1.1, h2c | Health Check, Admin API |
| mcp-llm-server | 40528 | gRPC (plaintext) | 봇 ↔ LLM 내부 통신 |
| twentyq-bot | 30081 | HTTP | Health Check, Debug API |
| turtle-soup-bot | 30082 | HTTP | Health Check, Debug API |

---

## 3. 환경 변수 변경

### 3.1 리네이밍 (레거시 → 신규)

| 기존 (레거시) | 신규 | 설명 |
|--------------|------|------|
| `LLM_REST_BASE_URL` | `LLM_BASE_URL` | LLM 서버 주소 (스킴 포함) |
| `LLM_REST_TIMEOUT_SECONDS` | `LLM_TIMEOUT_SECONDS` | 요청 타임아웃 |
| `LLM_REST_CONNECT_TIMEOUT_SECONDS` | `LLM_CONNECT_TIMEOUT_SECONDS` | 연결 타임아웃 |
| `LLM_REST_REQUIRE_GRPC` | `LLM_REQUIRE_GRPC` | gRPC 강제 여부 |
| `LLM_REST_HTTP2_ENABLED` | `LLM_HTTP2_ENABLED` | HTTP/2 활성화 |
| `LLM_REST_RETRY_MAX_ATTEMPTS` | `LLM_RETRY_MAX_ATTEMPTS` | 최대 재시도 횟수 |
| `LLM_REST_RETRY_DELAY_MS` | `LLM_RETRY_DELAY_MS` | 재시도 지연 (ms) |
| `LLM_REST_API_KEY` | `LLM_API_KEY` | API 인증 키 |

### 3.2 핵심 설정

```bash
# BaseURL은 항상 grpc:// 스킴을 사용해야 합니다 (HTTP fallback 제거됨)
LLM_BASE_URL=grpc://mcp-llm-server:40528
```

### 3.3 스킴별 동작

| 스킴 | 동작 |
|------|------|
| `grpc://` | ✅ gRPC 통신 |
| `http://` | ❌ 시작 실패 (Fail-Fast) |
| `https://` | ❌ 시작 실패 (Fail-Fast) |
| `grpcs://` | ❌ 미지원 (TLS 비활성) |

> **Note**: HTTP fallback 코드가 2025-12-31에 제거되어, grpc:// 스킴만 허용됩니다.

---

## 4. 코드 변경 사항

### 4.1 타입 리네이밍

```go
// Before
type LlmRestConfig struct { ... }
func ReadLlmRestConfigFromEnv() (LlmRestConfig, error)
cfg.LlmRest

// After
type LlmConfig struct { ... }
func ReadLlmConfigFromEnv() (LlmConfig, error)
cfg.Llm
```

### 4.2 클라이언트 초기화 (llmrest/client.go)

```go
func New(cfg Config) (*Client, error) {
    // grpc:// 스킴 파싱 (HTTP fallback 제거됨)
    if !strings.HasPrefix(strings.ToLower(baseURL), "grpc://") {
        return nil, fmt.Errorf("grpc scheme required: base url must start with grpc://")
    }

    // gRPC 클라이언트 초기화
    conn, err := grpc.NewClient(
        host,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithUnaryInterceptor(interceptor),
    )
    client.grpcConn = conn
    client.grpcClient = llmv1.NewLLMServiceClient(conn)
}
```

### 4.3 gRPC 전용 메서드 (항상 gRPC 사용)

```go
func (c *Client) TwentyQAnswerQuestion(ctx context.Context, ...) (*TwentyQAnswerResponse, error) {
    if c.grpcClient == nil {
        return nil, ErrGRPCClientRequired
    }

    callCtx, cancel := c.grpcCallContext(ctx)
    defer cancel()

    resp, err := c.grpcClient.TwentyQAnswerQuestion(callCtx, &llmv1.TwentyQAnswerQuestionRequest{
        ChatId:    &chatID,
        Namespace: &namespace,
        Target:    target,
        Category:  category,
        Question:  question,
        Details:   detailsPb,
    })
    if err != nil {
        return nil, fmt.Errorf("grpc twentyq answer failed: %w", err)
    }

    return &TwentyQAnswerResponse{
        Scale:   resp.Scale,
        RawText: resp.RawText,
    }, nil
}
```

### 4.4 제거된 코드 (2025-12-31)

| 항목 | 파일 | 이유 |
|------|------|------|
| HTTP fallback 코드 | common.go, twentyq.go, turtlesoup.go | gRPC 전용으로 전환 |
| `RequireGRPC` 필드 | config/types.go | 더 이상 필요 없음 (grpc 스킴 필수) |
| `HTTP2Enabled` 필드 | config/types.go | HTTP 제거로 불필요 |
| `RetryMaxAttempts`, `RetryDelay` | config/types.go | HTTP 재시도 로직 제거 |
| `Get`, `Post`, `Delete` 메서드 | client.go | HTTP fallback 제거로 불필요 |
| HTTP 테스트 | client_test.go | HTTP 코드 제거로 불필요 |
| `CreateSession` 메서드 | common.go | 봇에서 미사용 (서버 측 자체 생성) |
| `GetUsage` 메서드 | common.go | `GetTotalUsage`로 대체됨 |

**제거된 코드 양**: ~1,259줄 (-1,994 삭제 / +735 추가)

---

## 5. gRPC 서비스 정의

### 5.1 Proto 파일 위치

```
mcp-llm-server-go/internal/grpcserver/pb/llm/v1/llm_service.proto
game-bot-go/internal/common/llmrest/pb/llm/v1/llm_service.pb.go
```

### 5.2 주요 RPC 메서드

```protobuf
service LLMService {
  // 공통
  rpc GetModelConfig(google.protobuf.Empty) returns (ModelConfigResponse);
  rpc EndSession(EndSessionRequest) returns (EndSessionResponse);
  rpc GuardIsMalicious(GuardIsMaliciousRequest) returns (GuardIsMaliciousResponse);
  
  // 사용량
  rpc GetTotalUsage(GetTotalUsageRequest) returns (UsageResponse);
  rpc GetDailyUsage(google.protobuf.Empty) returns (DailyUsageResponse);
  rpc GetRecentUsage(GetRecentUsageRequest) returns (UsageListResponse);
  
  // TwentyQ
  rpc TwentyQSelectTopic(TwentyQSelectTopicRequest) returns (TwentyQSelectTopicResponse);
  rpc TwentyQGetCategories(google.protobuf.Empty) returns (TwentyQGetCategoriesResponse);
  rpc TwentyQGenerateHints(TwentyQGenerateHintsRequest) returns (TwentyQGenerateHintsResponse);
  rpc TwentyQAnswerQuestion(TwentyQAnswerQuestionRequest) returns (TwentyQAnswerQuestionResponse);
  rpc TwentyQVerifyGuess(TwentyQVerifyGuessRequest) returns (TwentyQVerifyGuessResponse);
  rpc TwentyQNormalizeQuestion(TwentyQNormalizeQuestionRequest) returns (TwentyQNormalizeQuestionResponse);
  rpc TwentyQCheckSynonym(TwentyQCheckSynonymRequest) returns (TwentyQCheckSynonymResponse);
  
  // TurtleSoup
  rpc TurtleSoupGeneratePuzzle(TurtleSoupGeneratePuzzleRequest) returns (TurtleSoupGeneratePuzzleResponse);
  rpc TurtleSoupGetRandomPuzzle(TurtleSoupGetRandomPuzzleRequest) returns (TurtleSoupGetRandomPuzzleResponse);
  rpc TurtleSoupRewriteScenario(TurtleSoupRewriteScenarioRequest) returns (TurtleSoupRewriteScenarioResponse);
  rpc TurtleSoupAnswerQuestion(TurtleSoupAnswerQuestionRequest) returns (TurtleSoupAnswerQuestionResponse);
  rpc TurtleSoupValidateSolution(TurtleSoupValidateSolutionRequest) returns (TurtleSoupValidateSolutionResponse);
  rpc TurtleSoupGenerateHint(TurtleSoupGenerateHintRequest) returns (TurtleSoupGenerateHintResponse);
}
```

---

## 6. 운영 가이드

### 6.1 Docker Compose 설정 (프로덕션)

```yaml
mcp-llm-server:
  environment:
    GRPC_HOST: 0.0.0.0
    GRPC_PORT: 40528
    GRPC_ENABLED: "true"

twentyq-bot:
  environment:
    LLM_BASE_URL: grpc://mcp-llm-server:40528
    # LLM_REQUIRE_GRPC: "true"  # 기본값

turtle-soup-bot:
  environment:
    LLM_BASE_URL: grpc://mcp-llm-server:40528
```

### 6.2 헬스 체크 확인

```bash
# HTTP 헬스 체크 (mcp-llm-server)
curl http://localhost:40527/health

# gRPC 연결 확인 (grpcurl 필요)
grpcurl -plaintext localhost:40528 list

# 봇 컨테이너 내부에서 gRPC 연결 확인
docker exec twentyq-bot wget -q -O- http://mcp-llm-server:40527/health
```

### 6.4 로그 확인

```bash
# gRPC 서버 시작 확인
docker logs mcp-llm-server 2>&1 | grep grpc_server_start
# 출력 예: grpc_server_start addr=[::]:40528 tls_enabled=false

# gRPC 요청 로그
docker logs mcp-llm-server 2>&1 | grep grpc_request
```

---

## 7. 트러블슈팅

### 7.1 "grpc required" 오류

```
grpc required: base url scheme must be grpc, got "http"
```

**원인**: `LLM_BASE_URL`이 `grpc://`로 시작하지 않음 (HTTP fallback 제거됨)

**해결**: `LLM_BASE_URL=grpc://mcp-llm-server:40528`로 설정

### 7.2 gRPC 연결 실패

```
grpc twentyq answer failed: rpc error: code = Unavailable
```

**확인 사항**:
1. mcp-llm-server 컨테이너가 healthy 상태인지 확인
2. `GRPC_ENABLED=true` 환경 변수 확인
3. gRPC 서버 시작 로그 확인 (`grpc_server_start`)
4. 네트워크 연결성 확인 (Docker network)

### 7.3 인증 오류

```
rpc error: code = Unauthenticated desc = invalid api key
```

**해결**: `LLM_API_KEY` (또는 `HTTP_API_KEY`) 환경 변수가 mcp-llm-server의 `HTTP_API_KEY`와 일치하는지 확인

---

## 8. 마이그레이션 체크리스트

- [ ] 환경 변수 이름 변경 (`LLM_REST_*` → `LLM_*`)
- [ ] `LLM_BASE_URL`을 `grpc://` 스킴으로 변경 (**필수**)
- [ ] mcp-llm-server에서 `GRPC_ENABLED=true` 확인
- [ ] Docker 이미지 재빌드 (`--no-cache` 권장)
- [ ] 컨테이너 재시작 후 gRPC 서버 시작 로그 확인
- [ ] 게임 시작 테스트

---

## 변경 이력

| 날짜 | 버전 | 변경 내용 |
|------|------|----------|
| 2025-12-31 | 2.0 | HTTP fallback 코드 완전 제거, gRPC 전용으로 전환 |
| 2025-12-30 | 1.0 | 초기 gRPC 마이그레이션 완료 |
