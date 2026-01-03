# Traces API Documentation (Jaeger Integration)

## Overview

Traces API는 Jaeger Query API를 프록시하여 Admin UI에서 분산 트레이싱 데이터를 조회할 수 있게 합니다. 
Backend Proxy 패턴을 통해 기존 Admin 세션 인증을 재사용하고 보안을 유지합니다.

> **구현 완료** (2026-01-01): Backend API 구현 완료. Frontend (Admin UI Traces Tab) 구현 완료.

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Admin UI      │────▶│  admin-dashboard  │────▶│     Jaeger      │
│  (React/TS)     │     │   (Go Backend)  │     │  Query API      │
│                 │     │                 │     │  :16686         │
│ /dashboard/     │     │ /admin/api/     │     │                 │
│   traces        │     │   traces/*      │     │ /api/services   │
└─────────────────┘     └─────────────────┘     └─────────────────┘
       │                        │                       │
       └── 인증: admin_session ─┘                       │
                               └── HTTP Client ────────┘
```

### Why Backend Proxy?

| 옵션 | 설명 | 채택 여부 |
|------|------|----------|
| **Backend Proxy** | Admin Backend에서 Jaeger API 호출 후 전달 | 채택 |
| Direct Frontend | Admin UI에서 Jaeger API 직접 호출 | 미채택 (CORS/보안 문제) |

**장점:**
- 기존 Admin 세션 인증 재사용 (쿠키 기반)
- Jaeger를 Docker 내부망에서만 노출 (외부 미노출)
- 응답 데이터 가공/필터링 가능
- 에러 핸들링 일원화

---

## Authentication

모든 Traces API는 Admin 인증이 필요합니다. 요청 시 유효한 `admin_session` 쿠키가 포함되어야 합니다.

> **보안 참고**: Jaeger UI(`:16686`)는 Docker 내부망에서만 접근 가능합니다. 
> 외부에서는 반드시 Admin UI를 통해 인증된 경로로만 트레이스 데이터에 접근할 수 있습니다.

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `JAEGER_QUERY_URL` | `http://jaeger:16686` | Jaeger Query API base URL |
| `JAEGER_TIMEOUT_SECONDS` | `10` | Jaeger API 요청 타임아웃 |

### docker-compose.prod.yml

```yaml
admin-dashboard:
  environment:
    JAEGER_QUERY_URL: http://jaeger:16686
    JAEGER_TIMEOUT_SECONDS: 10
```

---

## Endpoints

### 1. 서비스 목록 조회

**GET** `/admin/api/traces/services`

Jaeger에 등록된 모든 서비스 목록을 반환합니다.

#### Response

```json
{
  "status": "ok",
  "services": [
    "hololive-bot",
    "mcp-llm-server",
    "twentyq-bot",
    "turtle-soup-bot"
  ]
}
```

#### Jaeger API Mapping

```
GET http://jaeger:16686/api/services
```

---

### 2. 서비스별 Operation 목록

**GET** `/admin/api/traces/operations/:service`

특정 서비스의 모든 Operation(span name) 목록을 반환합니다.

#### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `service` | string | 서비스 이름 (e.g., `hololive-bot`) |

#### Response

```json
{
  "status": "ok",
  "service": "hololive-bot",
  "operations": [
    "HTTP GET /health",
    "HTTP POST /api/holo/members",
    "gRPC /llm.LLMService/Chat",
    "Valkey XREAD"
  ]
}
```

#### Jaeger API Mapping

```
GET http://jaeger:16686/api/services/{service}/operations
```

---

### 3. 트레이스 검색

**GET** `/admin/api/traces`

조건에 맞는 트레이스 목록을 검색합니다.

#### Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `service` | string | **required** | 서비스 이름 |
| `operation` | string | - | Operation 필터 (선택) |
| `limit` | number | 20 | 반환할 최대 트레이스 수 (1-100) |
| `lookback` | string | `1h` | 조회 시간 범위 (e.g., `1h`, `6h`, `24h`, `7d`) |
| `minDuration` | string | - | 최소 Duration 필터 (e.g., `100ms`, `1s`) |
| `maxDuration` | string | - | 최대 Duration 필터 |
| `tags` | string | - | 태그 필터 (JSON, e.g., `{"http.status_code":"500"}`) |

#### Response

```json
{
  "status": "ok",
  "traces": [
    {
      "traceId": "abc123def456...",
      "spanCount": 12,
      "services": ["admin-dashboard", "hololive-bot", "mcp-llm-server"],
      "operationName": "HTTP POST /admin/api/holo/members",
      "duration": 1523,
      "startTime": "2026-01-01T15:30:00.000Z",
      "hasError": false
    }
  ],
  "total": 45,
  "limit": 20
}
```

#### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `traceId` | string | 고유 Trace ID (32자 hex) |
| `spanCount` | number | 트레이스 내 Span 개수 |
| `services` | string[] | 트레이스에 포함된 서비스 목록 |
| `operationName` | string | Root Span의 Operation 이름 |
| `duration` | number | 총 Duration (μs, microseconds) |
| `startTime` | timestamp | 시작 시각 (UTC ISO 8601) |
| `hasError` | boolean | 에러 Span 포함 여부 |

#### Jaeger API Mapping

Backend에서 `lookback` 문자열을 파싱하여 Unix Microseconds `start` 타임스탬프로 변환 후 요청:

```
GET http://jaeger:16686/api/traces?service={service}&limit={limit}&start={calculated_start_us}
```

> **주의: Backend 변환 필수**:
> - `lookback="1h"` → `start = time.Now().Add(-1*time.Hour).UnixMicro()`
> - `tags` 파라미터는 JSON 문자열로 직렬화 후 URL Encoding 필요  
>   예: `tags={"error":"true"}` → `tags=%7B%22error%22%3A%22true%22%7D`

---

### 4. 단일 트레이스 상세 조회

**GET** `/admin/api/traces/:traceId`

특정 Trace ID의 전체 Span 트리를 반환합니다.

#### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `traceId` | string | Trace ID (32자 hex) |

#### Response

```json
{
  "status": "ok",
  "traceId": "abc123def456...",
  "spans": [
    {
      "spanId": "span123",
      "traceId": "abc123def456...",
      "operationName": "HTTP POST /admin/api/holo/members",
      "serviceName": "admin-dashboard",
      "duration": 1523000,
      "startTime": 1735745400000000,
      "references": [],
      "tags": [
        {"key": "http.method", "type": "string", "value": "POST"},
        {"key": "http.status_code", "type": "int64", "value": 200}
      ],
      "logs": [
        {"timestamp": 1735745400100000, "fields": [{"key": "event", "value": "request_received"}]}
      ],
      "processId": "p1",
      "hasError": false
    }
  ],
  "processes": {
    "p1": {"serviceName": "hololive-bot", "tags": []}
  }
}
```

#### Span Fields

| Field | Type | Description |
|-------|------|-------------|
| `spanId` | string | Span 고유 ID |
| `traceId` | string | 부모 Trace ID |
| `operationName` | string | Operation 이름 |
| `serviceName` | string | 서비스 이름 (Backend Enrichment로 주입) |
| `duration` | number | Duration (μs, microseconds) |
| `startTime` | number | 시작 시각 (Unix μs, microseconds) |
| `references` | array | 부모/자식 Span 참조 (parentSpanId 포함) |
| `tags` | array | Span 태그 (key-value) |
| `logs` | array | Span 이벤트 로그 |
| `hasError` | boolean | 에러 여부 (`error=true` 태그 존재) |

> **주의: 시간 단위 통일**: 모든 시간 필드는 **Microseconds(μs)** 를 사용합니다.  
> Jaeger JSON API 기본 단위와 동일하며, 프론트엔드 연산 일관성을 보장합니다.

#### Jaeger API Mapping

```
GET http://jaeger:16686/api/traces/{traceId}
```

---

### 5. 서비스 의존성 그래프 조회

**GET** `/admin/api/traces/dependencies`

서비스 간 호출 관계를 그래프 형태로 반환합니다.

#### Query Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `lookback` | string | 선택 | `24h` | 분석 기간 (`1h`, `6h`, `24h`, `7d`) |

#### Response

```json
{
  "status": "ok",
  "dependencies": [
    {
      "parent": "hololive-bot",
      "child": "mcp-llm-server",
      "callCount": 1523
    },
    {
      "parent": "hololive-bot",
      "child": "twentyq-game",
      "callCount": 892
    }
  ],
  "count": 2
}
```

#### Dependency Fields

| Field | Type | Description |
|-------|------|-------------|
| `parent` | string | 호출하는 서비스 |
| `child` | string | 호출받는 서비스 |
| `callCount` | number | 해당 기간 동안의 호출 횟수 |

---

### 6. 서비스 메트릭 조회 (SPM)

**GET** `/admin/api/traces/metrics/:service`

서비스별 RED(Rate, Error, Duration) 메트릭을 반환합니다.

> **요구사항**: Jaeger v2 SPM 활성화 및 Prometheus 연동 필요

#### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `service` | string | 서비스 이름 |

#### Query Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `lookback` | string | 선택 | `1h` | 조회 기간 |
| `quantile` | string | 선택 | `0.95` | Latency percentile (`0.5`, `0.95`, `0.99`) |
| `spanKind` | string | 선택 | `server` | Span 종류 (`server`, `client`) |
| `step` | string | 선택 | - | 시계열 간격 (`60s`, `5m`) |
| `ratePer` | string | 선택 | `second` | Rate 단위 (`second`, `minute`) |
| `groupByOperation` | bool | 선택 | `false` | Operation별 그룹화 |

#### Response

```json
{
  "status": "ok",
  "service": "hololive-bot",
  "metrics": {
    "name": "hololive-bot",
    "callRate": 12.5,
    "errorRate": 0.02,
    "p50Latency": 45.2,
    "p95Latency": 156.8,
    "p99Latency": 342.1,
    "avgDuration": 78.4
  },
  "operations": [],
  "latencies": [
    {"timestamp": 1735689600000, "value": 145.2},
    {"timestamp": 1735689660000, "value": 152.8}
  ],
  "calls": [
    {"timestamp": 1735689600000, "value": 12.3}
  ],
  "errors": [
    {"timestamp": 1735689600000, "value": 0.02}
  ]
}
```

#### Metrics Fields

| Field | Type | Unit | Description |
|-------|------|------|-------------|
| `callRate` | number | calls/sec | 초당 호출 수 |
| `errorRate` | number | ratio | 에러율 (0.0 ~ 1.0) |
| `p50Latency` | number | ms | 50th percentile 지연 시간 |
| `p95Latency` | number | ms | 95th percentile 지연 시간 |
| `p99Latency` | number | ms | 99th percentile 지연 시간 |
| `avgDuration` | number | ms | 평균 처리 시간 |

---

## Error Responses

### 400 Bad Request

```json
{
  "error": "Invalid parameter: service is required"
}
```

필수 파라미터 누락 또는 형식 오류.

### 401 Unauthorized

```json
{
  "error": "Unauthorized"
}
```

Admin 세션 인증 실패.

### 404 Not Found

```json
{
  "error": "Trace not found",
  "traceId": "invalid-trace-id"
}
```

존재하지 않는 Trace ID.

### 500 Internal Server Error

```json
{
  "error": "Failed to fetch from Jaeger",
  "details": "connection refused"
}
```

Jaeger 서비스 연결 실패.

### 503 Service Unavailable

```json
{
  "error": "Jaeger service unavailable"
}
```

Jaeger 서비스가 비정상 상태.

---

## Implementation Notes (Critical)

> **구현 전 필수 확인사항**: Jaeger 원본 API 응답과 Proxy API 스펙 간의 차이를 메우는 로직입니다.

### 1. Data Enrichment (Process-to-Span Mapping)

Jaeger `/api/traces/{id}` 응답은 `spans` 배열과 `processes` 맵이 분리되어 있으며,  
Span 객체에는 `processID`만 존재합니다 (서비스명 미포함).

**Backend 변환 로직 (필수)**:

```go
// Jaeger Raw Response 구조
type JaegerRawTrace struct {
    TraceID   string           `json:"traceID"`
    Spans     []JaegerRawSpan  `json:"spans"`
    Processes map[string]struct {
        ServiceName string `json:"serviceName"`
        Tags        []Tag  `json:"tags"`
    } `json:"processes"`
}

// Enrichment 로직: processID → serviceName 주입
func enrichSpans(raw *JaegerRawTrace) []Span {
    result := make([]Span, len(raw.Spans))
    for i, s := range raw.Spans {
        proc := raw.Processes[s.ProcessID]
        result[i] = Span{
            SpanID:        s.SpanID,
            TraceID:       s.TraceID,
            OperationName: s.OperationName,
            ServiceName:   proc.ServiceName,  // ← 주입
            Duration:      s.Duration,
            StartTime:     s.StartTime,
            References:    s.References,
            Tags:          s.Tags,
            Logs:          s.Logs,
            ProcessID:     s.ProcessID,
            HasError:      hasErrorTag(s.Tags),
        }
    }
    return result
}
```

### 2. Query Parameter Transformation (lookback → start)

Jaeger HTTP API는 `lookback` 문자열 대신 **Unix Microseconds** `start` 타임스탬프를 요구합니다.

**Backend 변환 로직**:

```go
// admin-dashboard/backend/internal/traces/client.go
func parseLookback(lookback string) (int64, error) {
    duration, err := time.ParseDuration(lookback)
    if err != nil {
        // "7d" 같은 형식 수동 파싱
        if strings.HasSuffix(lookback, "d") {
            days, _ := strconv.Atoi(strings.TrimSuffix(lookback, "d"))
            duration = time.Duration(days) * 24 * time.Hour
        }
    }
    start := time.Now().Add(-duration).UnixMicro()
    return start, nil
}

// 사용 예시
start, _ := parseLookback("1h")  // → Unix Microseconds
url := fmt.Sprintf("%s/api/traces?service=%s&start=%d&limit=%d",
    baseURL, service, start, limit)
```

### 3. Tags Parameter Handling

`tags` 파라미터는 **JSON 문자열 포맷**을 요구합니다.

**Backend 직렬화 로직**:

```go
// admin-dashboard/backend/internal/traces/client.go
func buildTagsParam(tags map[string]string) string {
    if len(tags) == 0 {
        return ""
    }
    jsonBytes, _ := json.Marshal(tags)
    return url.QueryEscape(string(jsonBytes))
}

// 사용 예시
// Input: map[string]string{"error": "true"}
// Output (URL encoded): %7B%22error%22%3A%22true%22%7D
```

---

## Implementation Plan

### Phase 1: Backend (Go) 완료

#### 1.1 Jaeger Client Module

**파일**: `admin-dashboard/backend/internal/traces/client.go`

```go
package traces

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "time"
)

// Client: Jaeger Query API 클라이언트
type Client struct {
    baseURL    string
    httpClient *http.Client
}

// NewClient: 새 Jaeger 클라이언트 생성
func NewClient(baseURL string, timeout time.Duration) *Client {
    return &Client{
        baseURL: baseURL,
        httpClient: &http.Client{
            Timeout: timeout,
        },
    }
}

// GetServices: 서비스 목록 조회
func (c *Client) GetServices(ctx context.Context) ([]string, error)

// GetOperations: Operation 목록 조회
func (c *Client) GetOperations(ctx context.Context, service string) ([]string, error)

// SearchTraces: 트레이스 검색
func (c *Client) SearchTraces(ctx context.Context, params TraceSearchParams) (*TraceSearchResult, error)

// GetTrace: 단일 트레이스 상세 조회
func (c *Client) GetTrace(ctx context.Context, traceID string) (*TraceDetail, error)
```

#### 1.2 Types

**파일**: `admin-dashboard/backend/internal/traces/types.go`

```go
package traces

// TraceSearchParams: 검색 파라미터
type TraceSearchParams struct {
    Service     string            `json:"service"`
    Operation   string            `json:"operation,omitempty"`
    Limit       int               `json:"limit"`
    Lookback    string            `json:"lookback"`
    MinDuration string            `json:"minDuration,omitempty"`
    MaxDuration string            `json:"maxDuration,omitempty"`
    Tags        map[string]string `json:"tags,omitempty"`
}

// TraceSummary: 트레이스 요약 (목록용)
type TraceSummary struct {
    TraceID       string   `json:"traceId"`
    SpanCount     int      `json:"spanCount"`
    Services      []string `json:"services"`
    OperationName string   `json:"operationName"`
    Duration      int64    `json:"duration"`
    StartTime     string   `json:"startTime"`
    HasError      bool     `json:"hasError"`
}

// TraceSearchResult: 검색 결과
type TraceSearchResult struct {
    Traces []TraceSummary `json:"traces"`
    Total  int            `json:"total"`
}

// Span: 개별 Span 정보
type Span struct {
    SpanID        string       `json:"spanId"`
    TraceID       string       `json:"traceId"`
    OperationName string       `json:"operationName"`
    ServiceName   string       `json:"serviceName"`
    Duration      int64        `json:"duration"`
    StartTime     int64        `json:"startTime"`
    References    []Reference  `json:"references"`
    Tags          []Tag        `json:"tags"`
    Logs          []Log        `json:"logs"`
    ProcessID     string       `json:"processId"`
    HasError      bool         `json:"hasError"`
}

// TraceDetail: 트레이스 상세
type TraceDetail struct {
    TraceID   string             `json:"traceId"`
    Spans     []Span             `json:"spans"`
    Processes map[string]Process `json:"processes"`
}
```

#### 1.3 Admin Routes

**파일**: `admin-dashboard/backend/internal/server/server.go` (`setupTracesRoutes` 함수)

```go
// /admin/api/traces/*
tracesGroup := authenticated.Group("/traces")
tracesGroup.GET("/health", s.handleTracesHealth)
tracesGroup.GET("/services", s.handleTracesServices)
tracesGroup.GET("/operations/:service", s.handleTracesOperations)
tracesGroup.GET("", s.handleTracesSearch)
tracesGroup.GET("/:traceId", s.handleTraceDetail)
tracesGroup.GET("/dependencies", s.handleTracesDependencies)
tracesGroup.GET("/metrics/:service", s.handleTracesMetrics)
```

#### 1.4 Handlers

**파일**: `admin-dashboard/backend/internal/server/server.go` (HTTP handlers)

```go
// handleTracesServices / handleTracesOperations / handleTracesSearch / handleTraceDetail ...
```

---

### Phase 2: Frontend (React/TypeScript)

> **상세 구현 가이드**: [traces_frontend_implementation.md](./traces_frontend_implementation.md) 참조  
> 타입 정의, API 클라이언트, 컴포넌트 구조, Span Tree 변환, Gantt 차트 시각화까지 포함된 완전한 구현 가이드입니다.

#### 2.1 API Client

**파일**: `src/api/index.ts` (추가)

```typescript
// Traces API
export interface TraceSearchParams {
  service: string
  operation?: string
  limit?: number
  lookback?: string
  minDuration?: string
  maxDuration?: string
  tags?: Record<string, string>
}

export interface TraceSummary {
  traceId: string
  spanCount: number
  services: string[]
  operationName: string
  duration: number
  startTime: string
  hasError: boolean
}

export const tracesApi = {
  getServices: async () => {
    const response = await apiClient.get<{ status: string; services: string[] }>('/traces/services')
    return response.data
  },

  getOperations: async (service: string) => {
    const response = await apiClient.get<{ status: string; operations: string[] }>(
      `/traces/operations/${encodeURIComponent(service)}`
    )
    return response.data
  },

  search: async (params: TraceSearchParams) => {
    const response = await apiClient.get<{
      status: string
      traces: TraceSummary[]
      total: number
    }>('/traces', { params })
    return response.data
  },

  getTrace: async (traceId: string) => {
    const response = await apiClient.get<{ status: string; traceId: string; spans: Span[] }>(
      `/traces/${traceId}`
    )
    return response.data
  },
}
```

#### 2.2 TracesTab Component

**파일**: `src/components/TracesTab.tsx`

```typescript
// 주요 기능
// 1. 서비스 선택 (Dropdown)
// 2. 시간 범위 선택 (1h, 6h, 24h, 7d)
// 3. Operation 필터 (선택)
// 4. 트레이스 목록 테이블 (Virtualized)
// 5. 트레이스 상세 모달 (Span Timeline)

const TracesTab = () => {
  const [selectedService, setSelectedService] = useState<string>('')
  const [lookback, setLookback] = useState<string>('1h')
  const [selectedTraceId, setSelectedTraceId] = useState<string | null>(null)

  // TanStack Query Hooks
  const { data: services } = useQuery(['traces-services'], tracesApi.getServices)
  const { data: traces } = useQuery(
    ['traces', selectedService, lookback],
    () => tracesApi.search({ service: selectedService, lookback }),
    { enabled: !!selectedService }
  )

  return (
    // ... UI 구현
  )
}
```

#### 2.3 SpanTimeline Component

**파일**: `src/components/traces/SpanTimeline.tsx`

Gantt 차트 스타일의 Span Timeline 시각화:
- 각 Span을 수평 막대로 표시

##### Tree 구조 변환 (필수)

상세 조회 API가 평탄화된(Flat) 리스트를 반환하므로, `references` (parentSpanId)를 기반으로 트리 구조 재구성 필요:

```typescript
interface SpanNode extends Span {
  children: SpanNode[]
  depth: number
}

function buildSpanTree(spans: Span[]): SpanNode[] {
  const spanMap = new Map<string, SpanNode>()
  const roots: SpanNode[] = []

  // 1. 모든 Span을 Map에 등록
  spans.forEach(span => {
    spanMap.set(span.spanId, { ...span, children: [], depth: 0 })
  })

  // 2. 부모-자식 관계 구축
  spans.forEach(span => {
    const node = spanMap.get(span.spanId)!
    const parentRef = span.references.find(r => r.refType === 'CHILD_OF')
    
    if (parentRef && spanMap.has(parentRef.spanId)) {
      const parent = spanMap.get(parentRef.spanId)!
      node.depth = parent.depth + 1
      parent.children.push(node)
    } else {
      roots.push(node)
    }
  })

  return roots
}
```

##### 가상화 (Virtualization)

> **성능 고려**: 하나의 트레이스에 **수천 개의 Span**이 포함될 수 있습니다.  
> 렌더링 성능 저하를 방지하기 위해 `react-window` 또는 `@tanstack/react-virtual` 활용을 권장합니다.

```typescript
import { FixedSizeList as List } from 'react-window'

const SpanList = ({ spans }: { spans: SpanNode[] }) => {
  // 트리를 평탄화하여 가상화 리스트로 렌더링
  const flattenedSpans = useMemo(() => flattenTree(spans), [spans])

  return (
    <List
      height={500}
      itemCount={flattenedSpans.length}
      itemSize={40}
      width="100%"
    >
      {({ index, style }) => (
        <SpanRow span={flattenedSpans[index]} style={style} />
      )}
    </List>
  )
}
```

##### 기존 기능:
- 부모-자식 관계를 들여쓰기로 표현
- Duration을 막대 너비로 표현
- 에러 Span은 빨간색 강조

#### 2.4 Routing

**파일**: `src/App.tsx` (추가)

```typescript
const TracesTab = lazy(() => import('@/components/TracesTab'))

// children 배열에 추가
{
  path: "traces",
  element: <LazyRoute><TracesTab /></LazyRoute>
}
```

#### 2.5 Navigation

**파일**: `src/layouts/AppLayout.tsx` (추가)

```typescript
import { Activity } from 'lucide-react'

// navItems 배열에 추가
{ path: '/dashboard/traces', label: 'Traces', icon: Activity }
```

---

### Phase 3: Docker Configuration

#### 3.1 Environment Variables

**파일**: `docker-compose.prod.yml` (수정)

```yaml
admin-dashboard:
  environment:
    # ... 기존 환경변수
    JAEGER_QUERY_URL: http://jaeger:16686
    JAEGER_TIMEOUT_SECONDS: 10
```

#### 3.2 Network Verification

```yaml
# 이미 같은 네트워크에 있음 (변경 불필요)
admin-dashboard:
  networks:
    - llm-bot-net

jaeger:
  networks:
    - llm-bot-net
```

---

## UI Design Specification

### TracesTab Layout

```
┌─────────────────────────────────────────────────────────────────┐
│ Traces                                                    [Refresh]  │
├─────────────────────────────────────────────────────────────────┤
│ ┌──────────────┐ ┌────────────┐ ┌──────────────┐ ┌───────────┐  │
│ │ Service ▼    │ │ Operation ▼│ │ Time Range ▼ │ │  Search   │  │
│ │ hololive-bot │ │ All        │ │ Last 1 hour  │ │           │  │
│ └──────────────┘ └────────────┘ └──────────────┘ └───────────┘  │
├─────────────────────────────────────────────────────────────────┤
│ Trace ID          │ Operation           │ Duration │ Time      │
│ ───────────────── │ ─────────────────── │ ──────── │ ───────── │
│ abc123...   ERR   │ HTTP POST /api/...  │ 1.52s    │ 15:30:00  │
│ def456...         │ gRPC /llm/Chat      │ 523ms    │ 15:29:45  │
│ ghi789...         │ HTTP GET /health    │ 12ms     │ 15:29:30  │
└─────────────────────────────────────────────────────────────────┘
```

### Trace Detail Modal

```
┌─────────────────────────────────────────────────────────────────┐
│ Trace: abc123def456...                                    [X]   │
├─────────────────────────────────────────────────────────────────┤
│ Duration: 1.52s  │  Spans: 12  │  Services: 3                   │
├─────────────────────────────────────────────────────────────────┤
│ ▼ admin-dashboard: HTTP POST /admin/api/holo/members ████████ 1.5s │
│   ├─ mcp-llm-server: gRPC /llm/Chat              ██████   1.2s  │
│   │  └─ mcp-llm-server: gemini.generate          █████    1.0s  │
│   └─ postgres: db.query                          █        50ms  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Testing Checklist

### Backend Tests

- [ ] `TestGetTraceServices` - 서비스 목록 조회
- [ ] `TestGetTraceOperations` - Operation 목록 조회
- [ ] `TestSearchTraces` - 트레이스 검색
- [ ] `TestSearchTraces_WithFilters` - 필터 옵션 테스트
- [ ] `TestGetTrace` - 단일 트레이스 조회
- [ ] `TestGetTrace_NotFound` - 존재하지 않는 Trace ID
- [ ] `TestJaegerUnavailable` - Jaeger 연결 실패 시 에러 처리

### Frontend Tests

- [ ] 서비스 드롭다운 렌더링
- [ ] 트레이스 목록 로딩/에러 상태
- [ ] 트레이스 상세 모달 열기/닫기
- [ ] Span Timeline 렌더링
- [ ] 시간 범위 필터 동작

### Integration Tests

- [ ] Admin 인증 없이 접근 시 401
- [ ] Jaeger 다운 시 503 반환
- [ ] 실제 트레이스 데이터 조회 E2E

---

## Future Enhancements (Optional)

### SPM Dashboard

GET `/admin/api/traces/metrics/services` - (Optional) 전체 서비스 RED 메트릭 (미구현)

```json
{
  "services": [
    {
      "name": "hololive-bot",
      "callRate": 125.5,
      "errorRate": 0.2,
      "p50Latency": 45,
      "p95Latency": 230,
      "p99Latency": 890
    }
  ]
}
```

### Trace-Log Correlation

Trace ID 클릭 시 해당 시간대의 로그 필터링:

```
GET /admin/api/logs?file=combined&lines=200&traceId=abc123  # traceId 필터는 별도 구현 필요
```

### Alert Integration

에러율 임계값 초과 시 알람 연동:

```
POST /admin/api/alerts
{
  "type": "trace_error_rate",
  "service": "hololive-bot",
  "threshold": 0.05
}
```

---

## References

- [Jaeger Query API Documentation](https://www.jaegertracing.io/docs/1.47/apis/#http-json-internal)
- [OpenTelemetry Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/)
- [Traces Frontend Implementation Guide](./traces_frontend_implementation.md)
- [Session Security Documentation](./session_security.md)
