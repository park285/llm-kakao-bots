# OpenTelemetry 분산 추적 통합 설계서

> **문서 버전**: 2.0 (2026-01-01)  
> **변경 사항**: 리뷰 피드백 반영 - ParentBased 샘플러, Valkey Consumer 계측, Gemini Client otelhttp, otelslog 브릿지, Graceful Shutdown, GenAI Semantic Conventions

## 개요

본 문서는 `mcp-llm-server-go`, `game-bot-go`, `hololive-kakao-bot-go` 서비스에 OpenTelemetry 기반 분산 추적(Distributed Tracing)을 통합하기 위한 설계서입니다.

### 목표

1. **End-to-End 요청 추적**: 하나의 요청이 여러 서비스를 거치는 흐름을 시각화
2. **Trace Context 전파**: W3C TraceContext 표준을 따르는 trace_id/span_id 전파
3. **로그 상관관계**: 기존 slog 로그에 trace_id/span_id 추가로 로그-트레이스 연계
4. **최소 침습적 통합**: 기존 코드 변경 최소화
5. **LLM 특화 메트릭**: GenAI Semantic Conventions를 통한 토큰 사용량 추적

### 현재 상태

| 항목 | 현재 상태 | 비고 |
|------|----------|------|
| 로깅 | `slog` + `tint` 핸들러 | 프로덕션에서는 `slog.JSONHandler` 전환 권장 |
| Request ID | HTTP(Gin), gRPC 미들웨어 구현됨 | 서비스 간 전파는 수동 |
| OTel 의존성 | `mcp-llm-server-go`에 간접 의존성 존재 | `go.opentelemetry.io/otel` v1.39.0 |
| 서비스 통신 | gRPC (TCP + UDS Dual-Mode) | `game-bot-go` → `mcp-llm-server-go` |
| 메시지 큐 | Valkey Streams (Consumer Group) | `StreamConsumer.Run()` 진입점 |

---

## 아키텍처

### 전체 구조

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              User Request (Kakao)                           │
└─────────────────────────────────────────────────────────────────────────────┘
                                       │
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           game-bot-go (Client)                              │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐  │
│  │ Valkey Consumer │  │ otelgrpc Client │  │ slog + TraceContext Handler │  │
│  │ (Entry Point)   │──│ StatsHandler    │──│ trace_id, span_id 주입      │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘  │
└──────────────────────────────┬──────────────────────────────────────────────┘
                               │ gRPC (W3C TraceContext in metadata)
                               ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                        mcp-llm-server-go (Server)                           │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐  │
│  │ otelgrpc Server │  │ Gemini Client   │  │ slog + TraceContext Handler │  │
│  │ StatsHandler    │──│ (HTTP Trace)    │──│ trace_id, span_id 주입      │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘  │
└──────────────────────────────┬──────────────────────────────────────────────┘
                               │ OTLP (gRPC or HTTP)
                               ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Tracing Backend (Jaeger / Grafana Tempo)                 │
│                         http://localhost:16686 (Jaeger UI)                  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Trace Context 전파 흐름

```
1. [진입점] game-bot-go Valkey Consumer
   ├─ StreamConsumer.handleMessage()에서 Root Span 생성
   ├─ (미래 확장) 메시지 헤더에서 부모 TraceContext Extract 가능
   └─ 생성된 ctx를 비즈니스 로직에 전파

2. [클라이언트→서버] gRPC 호출
   ├─ otelgrpc.NewClientHandler()가 TraceContext를 gRPC metadata에 주입
   └─ otelgrpc.NewServerHandler()가 metadata에서 TraceContext 추출

3. [외부 API] Gemini HTTP 호출
   ├─ otelhttp.NewTransport()가 HTTP 헤더에 TraceContext 주입
   └─ 응답 시간, 상태 코드, 토큰 사용량 기록

4. 응답 반환
   └─ 각 span 종료 및 OTLP로 비동기 export
```

> **⚠️ 핵심 원칙**: 모든 Entry Point(Valkey Consumer, HTTP Handler, gRPC Handler)에서 Context 생성/추출이 필수입니다. 누락 시 Trace가 끊깁니다.

---

## 구현 상세

### Phase 1: 공통 Telemetry 패키지

#### 1.1 패키지 구조

```
internal/common/telemetry/
├── config.go      # 설정 타입 정의
├── provider.go    # TracerProvider 초기화
├── handler.go     # slog Handler wrapper
└── propagator.go  # Context propagation 유틸리티
```

#### 1.2 설정 타입 (`config.go`)

```go
package telemetry

// Config: OpenTelemetry 설정입니다.
type Config struct {
    // Enabled: true면 트레이싱을 활성화합니다.
    Enabled bool

    // ServiceName: 서비스 식별자입니다 (예: "mcp-llm-server").
    ServiceName string

    // ServiceVersion: 서비스 버전입니다 (예: "1.0.0").
    ServiceVersion string

    // Environment: 배포 환경입니다 (예: "production", "development").
    Environment string

    // OTLPEndpoint: OTLP collector/exporter 주소입니다.
    // 예: "http://jaeger:4317" (gRPC) 또는 "http://jaeger:4318" (HTTP)
    OTLPEndpoint string

    // OTLPInsecure: true면 TLS 없이 연결합니다. 내부망에서만 사용하세요.
    OTLPInsecure bool

    // SampleRate: 샘플링 비율입니다 (0.0 ~ 1.0). 1.0이면 전체 트레이싱.
    // 프로덕션에서는 0.1 ~ 0.5 권장.
    SampleRate float64
}

// DefaultConfig: 기본 설정을 반환합니다.
func DefaultConfig() Config {
    return Config{
        Enabled:      false,
        SampleRate:   1.0,
        OTLPInsecure: true,
    }
}
```

#### 1.3 TracerProvider 초기화 (`provider.go`)

```go
package telemetry

import (
    "context"
    "fmt"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

// Provider: OpenTelemetry provider를 관리합니다.
type Provider struct {
    tracerProvider *sdktrace.TracerProvider
}

// NewProvider: TracerProvider를 초기화하고 글로벌로 설정합니다.
func NewProvider(ctx context.Context, cfg Config) (*Provider, error) {
    if !cfg.Enabled {
        return &Provider{}, nil
    }

    // Resource: 서비스 메타데이터 정의
    res, err := resource.Merge(
        resource.Default(),
        resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceName(cfg.ServiceName),
            semconv.ServiceVersion(cfg.ServiceVersion),
            semconv.DeploymentEnvironmentName(cfg.Environment),
        ),
    )
    if err != nil {
        return nil, fmt.Errorf("create resource: %w", err)
    }

    // OTLP Exporter 설정
    var exporterOpts []otlptracegrpc.Option
    exporterOpts = append(exporterOpts, otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint))
    if cfg.OTLPInsecure {
        exporterOpts = append(exporterOpts,
            otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
        )
    }

    exporter, err := otlptracegrpc.New(ctx, exporterOpts...)
    if err != nil {
        return nil, fmt.Errorf("create exporter: %w", err)
    }

    // Sampler 설정
    // [중요] ParentBased로 감싸서 부모가 샘플링했으면 자식도 무조건 샘플링
    // 이렇게 하지 않으면 분산 추적에서 Trace가 끊길 수 있음
    var rootSampler sdktrace.Sampler
    if cfg.SampleRate >= 1.0 {
        rootSampler = sdktrace.AlwaysSample()
    } else if cfg.SampleRate <= 0 {
        rootSampler = sdktrace.NeverSample()
    } else {
        rootSampler = sdktrace.TraceIDRatioBased(cfg.SampleRate)
    }
    // ParentBased: 부모가 샘플링 결정을 했으면 그 결정을 따름
    // 부모가 없으면(Root Span) rootSampler 사용
    sampler := sdktrace.ParentBased(rootSampler)

    // TracerProvider 생성
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(res),
        sdktrace.WithSampler(sampler),
    )

    // 글로벌 설정
    otel.SetTracerProvider(tp)
    otel.SetTextMapPropagator(
        propagation.NewCompositeTextMapPropagator(
            propagation.TraceContext{}, // W3C Trace Context
            propagation.Baggage{},      // W3C Baggage
        ),
    )

    return &Provider{tracerProvider: tp}, nil
}

// Shutdown: TracerProvider를 정리합니다. 애플리케이션 종료 시 호출하세요.
func (p *Provider) Shutdown(ctx context.Context) error {
    if p.tracerProvider == nil {
        return nil
    }
    return p.tracerProvider.Shutdown(ctx)
}
```

#### 1.4 slog 공식 브릿지 사용 (`otelslog`)

> **⚠️ 중요**: 직접 Handler를 구현하는 대신 **공식 브릿지 패키지**를 사용합니다.  
> 유지보수 부담을 줄이고 OTel 업데이트에 자동으로 대응합니다.

```bash
go get go.opentelemetry.io/contrib/bridges/otelslog
```

```go
package main

import (
    "log/slog"
    "os"

    "go.opentelemetry.io/contrib/bridges/otelslog"
)

func setupLogger(serviceName string, otelEnabled bool) *slog.Logger {
    if otelEnabled {
        // OTel 공식 브릿지: trace_id, span_id 자동 주입 + LogRecord export
        return slog.New(otelslog.NewHandler(serviceName))
    }
    // OTel 비활성화 시 기본 핸들러
    return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}
```

**`otelslog` 장점:**
- trace_id, span_id 자동 주입
- OTel LogProvider와 통합 (로그도 OTLP로 export 가능)
- 공식 지원으로 호환성 보장

---

### Phase 2: gRPC 서버 통합 (mcp-llm-server-go)

#### 2.1 Config 확장

```go
// 기존 config/types.go에 추가

// TelemetryConfig: OpenTelemetry 설정입니다.
type TelemetryConfig struct {
    Enabled        bool
    ServiceName    string
    ServiceVersion string
    Environment    string
    OTLPEndpoint   string
    OTLPInsecure   bool
    SampleRate     float64
}

// Config 구조체에 필드 추가
type Config struct {
    // ... 기존 필드들 ...
    Telemetry TelemetryConfig
}
```

#### 2.2 환경 변수 매핑

```go
// config/env.go에 추가

// TelemetryEnvVars: 텔레메트리 관련 환경 변수입니다.
var TelemetryEnvVars = map[string]string{
    "Enabled":        "OTEL_ENABLED",
    "ServiceName":    "OTEL_SERVICE_NAME",
    "ServiceVersion": "OTEL_SERVICE_VERSION",
    "Environment":    "OTEL_ENVIRONMENT",
    "OTLPEndpoint":   "OTEL_EXPORTER_OTLP_ENDPOINT",
    "OTLPInsecure":   "OTEL_EXPORTER_OTLP_INSECURE",
    "SampleRate":     "OTEL_SAMPLE_RATE",
}
```

#### 2.3 gRPC 서버 수정 (`grpcserver/server.go`)

```go
import (
    // ... 기존 imports ...
    "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

func NewServer(cfg *config.Config, logger *slog.Logger) (*grpc.Server, net.Listener, net.Listener, error) {
    // ... 기존 코드 ...

    // gRPC 서버 옵션에 OTel StatsHandler 추가
    serverOpts := []grpc.ServerOption{
        grpc.MaxRecvMsgSize(maxRecvMsgSizeBytes),
        grpc.ChainUnaryInterceptor(
            unaryInterceptor(logger, apiKey, apiKeyRequired),
            errorMapperInterceptor(),
        ),
    }

    // OTel이 활성화된 경우 StatsHandler 추가
    if cfg.Telemetry.Enabled {
        serverOpts = append(serverOpts, grpc.StatsHandler(otelgrpc.NewServerHandler()))
    }

    server := grpc.NewServer(serverOpts...)
    return server, tcpLis, udsLis, nil
}
```

#### 2.4 Gemini Client에 otelhttp 적용 (`internal/gemini/client.go`)

> **⚠️ 핵심**: LLM API 호출은 전체 지연 시간의 대부분을 차지합니다.  
> `otelhttp`로 계측하지 않으면 병목 원인을 분석할 수 없습니다.

```go
import (
    "net/http"

    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
    "google.golang.org/genai"
)

// selectClient 내부에서 genai.NewClient 생성 시 적용
func (c *Client) selectClient(ctx context.Context) (*genai.Client, error) {
    // ... 기존 코드 ...

    // OTel이 적용된 HTTP Transport 생성
    httpClient := &http.Client{
        Transport: otelhttp.NewTransport(
            http.DefaultTransport,
            otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
                return "Gemini." + r.URL.Path
            }),
        ),
    }

    client, err := genai.NewClient(context.WithoutCancel(ctx), &genai.ClientConfig{
        APIKey:  key,
        Backend: genai.BackendGeminiAPI,
        HTTPOptions: genai.HTTPOptions{
            Timeout:    genai.Ptr(timeout),
            HTTPClient: httpClient, // OTel HTTP Client 주입
        },
    })
    // ...
}
```

#### 2.5 GenAI Semantic Conventions (토큰 사용량 추적)

> **⚠️ LLM 특화**: 단순 성공 여부보다 **토큰 사용량**이 운영에 훨씬 중요합니다.

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/trace"
)

// recordUsage 또는 generate 함수 내에서 Span에 속성 추가
func (c *Client) recordLLMSpanAttributes(ctx context.Context, model string, resp *genai.GenerateContentResponse) {
    span := trace.SpanFromContext(ctx)
    if !span.IsRecording() {
        return
    }

    span.SetAttributes(
        // GenAI Semantic Conventions
        attribute.String("gen_ai.system", "gemini"),
        attribute.String("gen_ai.request.model", model),
    )

    if resp != nil && resp.UsageMetadata != nil {
        span.SetAttributes(
            attribute.Int("gen_ai.response.input_tokens", int(resp.UsageMetadata.PromptTokenCount)),
            attribute.Int("gen_ai.response.output_tokens", int(resp.UsageMetadata.CandidatesTokenCount)),
            attribute.Int("gen_ai.response.cached_tokens", int(resp.UsageMetadata.CachedContentTokenCount)),
        )
    }
}
```

#### 2.6 Graceful Shutdown 패턴 (`cmd/server/main.go`)

> **⚠️ 주의**: `os.Exit()`은 `defer` 구문을 실행하지 않습니다.  
> 마지막 트레이스 데이터가 유실될 수 있으므로 정상 종료 흐름을 사용합니다.

```go
func main() {
    // 에러 코드를 저장할 변수
    var exitCode int
    defer func() {
        os.Exit(exitCode)
    }()

    app, err := di.InitializeApp()
    if err != nil {
        log.Printf("failed to initialize app: %v", err)
        exitCode = 1
        return // defer os.Exit(1) 실행
    }
    defer app.Close() // 이제 항상 실행됨

    // OpenTelemetry 초기화
    otelProvider, err := telemetry.NewProvider(ctx, cfg.Telemetry)
    if err != nil {
        app.Logger.Error("otel_init_failed", "err", err)
        exitCode = 1
        return // defer들이 순서대로 실행됨
    }
    defer func() {
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        if err := otelProvider.Shutdown(shutdownCtx); err != nil {
            app.Logger.Error("otel_shutdown_failed", "err", err)
        }
    }()

    // ... 나머지 코드 ...

    if err != nil && !errors.Is(err, http.ErrServerClosed) {
        exitCode = 1
        return // defer들이 실행되어 OTel flush 보장
    }
}
```

---

### Phase 3: gRPC 클라이언트 및 Valkey Consumer 통합 (game-bot-go)

#### 3.1 Valkey Consumer 계측 (`internal/common/mq/streams.go`)

> **⚠️ 핵심**: 이 부분이 없으면 **분산 추적이 완전히 끊깁니다**.  
> Consumer에서 Root Span을 생성해야 이후 gRPC 호출이 연결됩니다.

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/trace"
)

func (c *StreamConsumer) handleMessage(
    ctx context.Context,
    cfg StreamConsumerConfig,
    msg XMessage,
    handler func(ctx context.Context, msg XMessage) error,
) {
    tracer := otel.Tracer("game-bot-go/valkey-consumer")
    propagator := otel.GetTextMapPropagator()

    // 1. 메시지 헤더에서 부모 Context 추출 (미래 확장용)
    // 현재는 메시지에 TraceContext가 없으므로 새 Trace 시작
    carrier := propagation.MapCarrier(msg.Values)
    parentCtx := propagator.Extract(ctx, carrier)

    // 2. Root Span 시작 (SpanKindConsumer로 표시)
    spanCtx, span := tracer.Start(parentCtx, "Valkey.ProcessMessage",
        trace.WithSpanKind(trace.SpanKindConsumer),
        trace.WithAttributes(
            attribute.String("messaging.system", "valkey"),
            attribute.String("messaging.destination", cfg.Stream),
            attribute.String("messaging.message_id", msg.ID),
            attribute.String("messaging.consumer_group", cfg.Group),
        ),
    )
    defer span.End()

    // 3. 생성된 spanCtx를 비즈니스 로직에 전달
    handleErr := handler(spanCtx, msg)
    if handleErr != nil {
        span.RecordError(handleErr)
        span.SetStatus(codes.Error, handleErr.Error())
        c.logger.ErrorContext(spanCtx, "message_handler_failed",
            "err", handleErr,
            "stream", cfg.Stream,
            "id", msg.ID,
        )
        if !cfg.AckOnError {
            return
        }
    } else {
        span.SetStatus(codes.Ok, "")
    }

    if errAck := c.ackWithRetry(spanCtx, cfg, msg.ID); errAck != nil {
        c.logger.WarnContext(spanCtx, "xack_failed",
            "err", errAck,
            "stream", cfg.Stream,
            "id", msg.ID,
        )
    }
}
```

**핵심 포인트:**
1. `trace.SpanKindConsumer`로 메시지 소비자임을 명시
2. `spanCtx`를 handler에 전달하여 이후 gRPC 호출과 연결
3. `span.RecordError()`로 에러를 Trace에 기록
4. `slog.ErrorContext(spanCtx, ...)`로 로그와 Trace 연계

#### 3.2 llmrest.Client 수정 (`internal/common/llmrest/client.go`)

```go
import (
    // ... 기존 imports ...
    "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

// Config에 OTel 옵션 추가
type Config struct {
    BaseURL        string
    APIKey         string
    Timeout        time.Duration
    ConnectTimeout time.Duration
    EnableOTel     bool // 추가: OTel 계측 활성화
}

func New(cfg Config) (*Client, error) {
    // ... 기존 코드 ...

    baseOpts := []grpc.DialOption{
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithUnaryInterceptor(interceptor),
        grpc.WithDefaultCallOptions(
            grpc.MaxCallRecvMsgSize(grpcMaxMsgSizeBytes),
            grpc.MaxCallSendMsgSize(grpcMaxMsgSizeBytes),
        ),
    }

    // OTel StatsHandler 추가
    if cfg.EnableOTel {
        baseOpts = append(baseOpts, grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
    }

    // ... 나머지 코드 ...
}
```

---

### Phase 4: 인프라 구성

#### 4.1 Docker Compose 추가 (`docker-compose.prod.yml`)

```yaml
services:
  # ... 기존 서비스들 ...

  # Jaeger All-in-One (개발/스테이징용)
  jaeger:
    image: jaegertracing/jaeger:2.5
    container_name: jaeger
    restart: unless-stopped
    ports:
      - "16686:16686"  # Jaeger UI
      - "4317:4317"    # OTLP gRPC
      - "4318:4318"    # OTLP HTTP
    environment:
      - COLLECTOR_OTLP_ENABLED=true
    networks:
      - bot-network
    labels:
      - "deunhealth.restart.on.unhealthy=true"
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:16686"]
      interval: 10s
      timeout: 5s
      retries: 3

  # Grafana Tempo (프로덕션 대안)
  # tempo:
  #   image: grafana/tempo:2.7
  #   ...
```

#### 4.2 환경 변수 예시 (`.env.example`)

```env
# OpenTelemetry 설정
OTEL_ENABLED=true
OTEL_SERVICE_NAME=mcp-llm-server
OTEL_SERVICE_VERSION=1.0.0
OTEL_ENVIRONMENT=production
OTEL_EXPORTER_OTLP_ENDPOINT=jaeger:4317
OTEL_EXPORTER_OTLP_INSECURE=true
OTEL_SAMPLE_RATE=0.1
```

---

## 의존성

### 추가 필요 패키지

```bash
# mcp-llm-server-go
go get go.opentelemetry.io/otel@v1.39.0
go get go.opentelemetry.io/otel/sdk@v1.39.0
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc@v1.39.0
go get go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc@v0.64.0
go get go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp@v0.64.0  # Gemini HTTP 계측

# game-bot-go
go get go.opentelemetry.io/otel@v1.39.0
go get go.opentelemetry.io/otel/sdk@v1.39.0
go get go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc@v0.64.0
```

---

## 예상 결과

### 로그 출력 변화

**Before:**
```json
{
  "time": "2026-01-01T14:00:00Z",
  "level": "INFO",
  "msg": "grpc_request",
  "request_id": "abc123def456",
  "method": "/llm.v1.LLMService/TwentyQVerify",
  "latency": "1.234s"
}
```

**After:**
```json
{
  "time": "2026-01-01T14:00:00Z",
  "level": "INFO",
  "msg": "grpc_request",
  "request_id": "abc123def456",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "method": "/llm.v1.LLMService/TwentyQVerify",
  "latency": "1.234s"
}
```

### Jaeger UI 예시

```
Trace: 4bf92f3577b34da6a3ce929d0e0e4736
├── game-bot-go: ProcessMessage (12.5s)
│   ├── game-bot-go: AcquireLock (0.5ms)
│   ├── game-bot-go: LLMService.TwentyQVerify (client) (11.2s)
│   │   └── mcp-llm-server-go: LLMService.TwentyQVerify (server) (11.1s)
│   │       ├── mcp-llm-server-go: Gemini.GenerateContent (10.8s)
│   │       └── mcp-llm-server-go: ConsensusEngine.Evaluate (0.2s)
│   └── game-bot-go: SendResponse (0.8s)
```

---

## Request ID와 Trace ID 관계

| 항목 | Request ID | Trace ID |
|------|-----------|----------|
| **생성 시점** | 각 서비스 진입점 | 최초 서비스에서만 생성, 이후 전파 |
| **범위** | 단일 서비스 내 | 전체 요청 흐름 (서비스 간) |
| **목적** | 서비스 내 로그 상관관계 | 분산 시스템 전체 추적 |
| **표준** | 자체 정의 | W3C TraceContext |

### 공존 전략

기존 `request_id`를 유지하면서 `trace_id`를 추가합니다:

```go
// gRPC 인터셉터에서
ctx = context.WithValue(ctx, ctxKey(requestIDKey), requestID)

// OTel은 별도로 trace context를 관리
// spanCtx := trace.SpanContextFromContext(ctx)
```

---

## 성능 고려사항

### 오버헤드

| 항목 | 예상 오버헤드 | 비고 |
|------|-------------|------|
| CPU | ~1-3% | 샘플링 비율에 따라 변동 |
| 메모리 | ~10-20 MB | 배치 버퍼 크기에 따라 변동 |
| 네트워크 | ~1-5 KB/요청 | OTLP 프로토콜 오버헤드 |
| 지연시간 | ~0.1-0.5 ms | 비동기 export로 최소화 |

### 권장 샘플링 전략

> **⚠️ 중요**: `ParentBased` 샘플러를 사용하므로, 아래 비율은 **Root Span**에만 적용됩니다.  
> 부모 Trace가 샘플링되면 자식도 무조건 샘플링됩니다.

| 환경 | SampleRate | 근거 |
|------|-----------|------|
| 개발 | 1.0 | 모든 요청 추적 필요 |
| 스테이징 | 0.5 | 충분한 샘플 + 비용 절감 |
| 프로덕션 | 0.1 | 비용 효율성 + 대표 샘플 |

---

## 마이그레이션 계획

### 단계별 롤아웃

1. **Week 1**: `mcp-llm-server-go`에만 적용 (OTEL_ENABLED=false로 배포 후 활성화)
2. **Week 2**: `game-bot-go`에 클라이언트 계측 추가
3. **Week 3**: `hololive-kakao-bot-go` 통합
4. **Week 4**: 프로덕션 샘플링 조정 및 대시보드 구성

### 롤백 계획

환경 변수 `OTEL_ENABLED=false`로 즉시 비활성화 가능합니다. 코드 변경 없이 롤백됩니다.

---

## 대안 비교

### Tracing Backend 선택

| 솔루션 | 장점 | 단점 | 권장 시나리오 |
|-------|------|------|-------------|
| **Jaeger** | 성숙, 풍부한 UI | 스토리지 별도 필요 (Cassandra/ES) | 즉시 시작, 개발/스테이징 |
| **Grafana Tempo** | 비용 효율적 스토리지 (S3) | Grafana 필요 | 장기 보존, 대규모 |
| **Zipkin** | 간단, 가벼움 | 기능 제한적 | 소규모 팀 |

### 구현 접근 방식

| 방식 | 장점 | 단점 |
|------|------|------|
| **otelgrpc StatsHandler (권장)** | 자동 계측, 유지보수 용이 | 세밀한 커스터마이징 어려움 |
| **수동 Interceptor** | 완전한 제어 | 구현/유지보수 비용 높음 |
| **Agent 기반 (eBPF)** | 코드 수정 없음 | 인프라 복잡도 증가 |

---

## 추가 고려사항

### 보안

- OTLP 엔드포인트는 내부망에서만 접근 가능하도록 설정
- 민감한 데이터(API 키, 사용자 정보 등)를 span attributes에 포함하지 않도록 주의
- Baggage에 민감정보 전파 금지

### 모니터링

- Jaeger/Tempo 자체 헬스체크 필수
- `deunhealth` 라벨로 자동 재시작 설정
- Export 실패 시 로그 경고 (slog로 출력)

---

## 참고 자료

- [OpenTelemetry Go SDK](https://github.com/open-telemetry/opentelemetry-go)
- [OpenTelemetry Go Contrib](https://github.com/open-telemetry/opentelemetry-go-contrib)
- [otelgrpc Instrumentation](https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main/instrumentation/google.golang.org/grpc/otelgrpc)
- [otelhttp Instrumentation](https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main/instrumentation/net/http/otelhttp)
- [otelslog Bridge](https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main/bridges/otelslog)
- [W3C Trace Context](https://www.w3.org/TR/trace-context/)
- [GenAI Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/gen-ai/)
- [Jaeger Documentation](https://www.jaegertracing.io/docs/)
- [Grafana Tempo](https://grafana.com/docs/tempo/latest/)

---

## 문서 변경 이력

| 버전 | 날짜 | 변경 사항 |
|------|------|----------|
| 1.0 | 2026-01-01 | 최초 작성 - 기본 gRPC 계측 설계 |
| 2.0 | 2026-01-01 | 리뷰 피드백 반영: ParentBased 샘플러, Valkey Consumer 계측, Gemini otelhttp, otelslog 브릿지, Graceful Shutdown, GenAI Semantic Conventions |

### v2.0 주요 수정 사항

#### 1. 샘플링 전략 수정 (Critical)
- **문제**: `TraceIDRatioBased` 단독 사용 시 분산 추적에서 Trace 단절 발생 가능
- **해결**: `sdktrace.ParentBased(ratioSampler)`로 감싸서 부모 결정 존중

#### 2. Valkey Consumer 계측 추가 (Critical)
- **문제**: Entry Point에서 Span을 생성하지 않으면 이후 모든 호출이 새 Trace로 시작
- **해결**: `StreamConsumer.handleMessage()`에서 Root Span 생성 및 `ctx` 전파

#### 3. Gemini Client otelhttp 적용 (High)
- **문제**: LLM API 호출에 대한 계측 누락으로 병목 원인 분석 불가
- **해결**: `otelhttp.NewTransport()` 래핑 및 GenAI Semantic Conventions 속성 추가

#### 4. slog 공식 브릿지 사용 (Medium)
- **문제**: 직접 구현한 `TraceContextHandler`의 유지보수 부담
- **해결**: `go.opentelemetry.io/contrib/bridges/otelslog` 공식 패키지 사용

#### 5. Graceful Shutdown 패턴 (High)
- **문제**: `os.Exit()`이 `defer`를 실행하지 않아 마지막 트레이스 데이터 유실
- **해결**: `exitCode` 변수 + `defer os.Exit(exitCode)` 패턴으로 변경

