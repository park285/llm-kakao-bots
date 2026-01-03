# Traces Frontend Implementation Guide

> Last Updated: 2026-01-01  
> Backend Status: 완료

이 문서는 Admin UI에서 Jaeger 분산 트레이싱 데이터를 시각화하기 위한 프론트엔드 구현 가이드입니다.

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [API Reference](#api-reference)
3. [Type Definitions](#type-definitions)
4. [Component Structure](#component-structure)
5. [Data Transformations](#data-transformations)
6. [UI/UX Guidelines](#uiux-guidelines)

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Admin UI (React)                               │
├──────────────┬──────────────┬──────────────┬───────────────┬────────────┤
│ TracesTab    │ SpanTree     │ Dependencies │ ServiceMetrics│ Timeline   │
│ (Container)  │ (Gantt)      │ (Graph)      │ (RED Chart)   │ (Detail)   │
└──────────────┴──────────────┴──────────────┴───────────────┴────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    /admin/api/traces/* (Backend Proxy)                   │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
                            Jaeger Query API
```

---

## API Reference

Base Path: `/admin/api/traces`

### 1. Health Check

```http
GET /admin/api/traces/health
```

**Response:**
```json
{
  "status": "ok",
  "available": true
}
```

---

### 2. Services (서비스 목록)

```http
GET /admin/api/traces/services
```

**Response:**
```json
{
  "status": "ok",
  "services": ["hololive-bot", "mcp-llm-server", "twentyq-game", "turtlesoup-game"]
}
```

---

### 3. Operations (서비스별 Operation)

```http
GET /admin/api/traces/operations/:service
```

**Response:**
```json
{
  "status": "ok",
  "service": "hololive-bot",
  "operations": [
    "HTTP POST /callback",
    "grpc.TwentyQ/ProcessMessage",
    "Valkey.XRead"
  ]
}
```

---

### 4. Search Traces (트레이스 검색)

```http
GET /admin/api/traces?service=SERVICE&operation=OP&lookback=1h&limit=20
```

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `service` | 필수 | - | 서비스 이름 |
| `operation` | 선택 | - | Operation 필터 |
| `lookback` | 선택 | `1h` | 검색 범위 (`1h`, `6h`, `24h`, `7d`) |
| `limit` | 선택 | `20` | 결과 개수 (max: 100) |
| `minDuration` | 선택 | - | 최소 Duration (e.g., `100ms`) |
| `maxDuration` | 선택 | - | 최대 Duration |
| `tags` | 선택 | - | JSON 태그 필터 (e.g., `{"error":"true"}`) |

**Response:**
```json
{
  "status": "ok",
  "traces": [
    {
      "traceId": "abc123def456",
      "spanCount": 12,
      "services": ["hololive-bot", "mcp-llm-server"],
      "operationName": "HTTP POST /callback",
      "duration": 245000,      // microseconds (μs)
      "startTime": "2026-01-01T12:00:00Z",
      "hasError": false
    }
  ],
  "total": 15,
  "limit": 20
}
```

---

### 5. Get Trace Detail (트레이스 상세)

```http
GET /admin/api/traces/:traceId
```

**Response:**
```json
{
  "status": "ok",
  "traceId": "abc123def456",
  "spans": [
    {
      "spanId": "span001",
      "traceId": "abc123def456",
      "operationName": "HTTP POST /callback",
      "serviceName": "hololive-bot",    // Backend Enrichment
      "duration": 245000,               // μs
      "startTime": 1735689600000000,    // Unix μs
      "references": [
        { "refType": "CHILD_OF", "traceId": "abc123def456", "spanId": "span000" }
      ],
      "tags": [
        { "key": "http.method", "type": "string", "value": "POST" },
        { "key": "http.status_code", "type": "int64", "value": 200 }
      ],
      "logs": [
        { "timestamp": 1735689600100000, "fields": [{"key": "event", "value": "request_received"}] }
      ],
      "processId": "p1",
      "hasError": false
    }
  ],
  "processes": {
    "p1": {
      "serviceName": "hololive-bot",
      "tags": [{ "key": "hostname", "type": "string", "value": "pod-123" }]
    }
  }
}
```

---

### 6. Dependencies (서비스 의존성 그래프)

```http
GET /admin/api/traces/dependencies?lookback=24h
```

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `lookback` | 선택 | `24h` | 분석 범위 |

**Response:**
```json
{
  "status": "ok",
  "dependencies": [
    { "parent": "hololive-bot", "child": "mcp-llm-server", "callCount": 1523 },
    { "parent": "hololive-bot", "child": "twentyq-game", "callCount": 892 },
    { "parent": "mcp-llm-server", "child": "gemini-api", "callCount": 1245 }
  ],
  "count": 3
}
```

**시각화 예시 (D3.js Force Graph):**
```
          ┌─────────────────┐
          │  hololive-bot   │
          └───────┬─────────┘
                  │
      ┌───────────┼───────────┐
      ▼           ▼           ▼
┌─────────┐ ┌─────────┐ ┌─────────┐
│mcp-llm  │ │twentyq  │ │turtlesoup│
└────┬────┘ └─────────┘ └──────────┘
     │
     ▼
┌──────────┐
│gemini-api│
└──────────┘
```

---

### 7. Service Metrics (SPM - RED 메트릭)

```http
GET /admin/api/traces/metrics/:service?lookback=1h&quantile=0.95
```

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `service` | 필수 (path) | - | 서비스 이름 |
| `lookback` | 선택 | `1h` | 메트릭 범위 |
| `quantile` | 선택 | `0.95` | Latency percentile (`0.5`, `0.95`, `0.99`) |
| `spanKind` | 선택 | `server` | `server` or `client` |
| `step` | 선택 | - | 시계열 step (e.g., `60s`, `5m`) |
| `ratePer` | 선택 | `second` | Rate 단위 (`second`, `minute`) |
| `groupByOperation` | 선택 | `false` | Operation별 그룹화 |

**Response:**
```json
{
  "status": "ok",
  "service": "hololive-bot",
  "metrics": {
    "name": "hololive-bot",
    "callRate": 12.5,      // calls/sec
    "errorRate": 0.02,     // 2%
    "p50Latency": 45.2,    // ms
    "p95Latency": 156.8,   // ms
    "p99Latency": 342.1,   // ms
    "avgDuration": 78.4    // ms
  },
  "operations": [],        // groupByOperation=true 시 포함
  "latencies": [           // 시계열 데이터
    { "timestamp": 1735689600000, "value": 145.2 },
    { "timestamp": 1735689660000, "value": 152.8 }
  ],
  "calls": [
    { "timestamp": 1735689600000, "value": 12.3 },
    { "timestamp": 1735689660000, "value": 14.1 }
  ],
  "errors": [
    { "timestamp": 1735689600000, "value": 0.02 },
    { "timestamp": 1735689660000, "value": 0.01 }
  ]
}
```

---

## Type Definitions

```typescript
// src/types/traces.ts

// === 기본 타입 ===

export interface TraceSummary {
  traceId: string
  spanCount: number
  services: string[]
  operationName: string
  duration: number  // μs
  startTime: string // ISO 8601
  hasError: boolean
}

export interface Reference {
  refType: 'CHILD_OF' | 'FOLLOWS_FROM'
  traceId: string
  spanId: string
}

export interface Tag {
  key: string
  type: 'string' | 'int64' | 'bool' | 'float64'
  value: string | number | boolean
}

export interface LogField {
  key: string
  value: unknown
}

export interface Log {
  timestamp: number  // Unix μs
  fields: LogField[]
}

export interface Span {
  spanId: string
  traceId: string
  operationName: string
  serviceName: string   // Backend enriched
  duration: number      // μs
  startTime: number     // Unix μs
  references: Reference[]
  tags: Tag[]
  logs: Log[]
  processId: string
  hasError: boolean
}

export interface Process {
  serviceName: string
  tags: Tag[]
}

export interface TraceDetail {
  traceId: string
  spans: Span[]
  processes: Record<string, Process>
}

// === Dependencies ===

export interface Dependency {
  parent: string
  child: string
  callCount: number
}

// === Metrics (SPM) ===

export interface MetricPoint {
  timestamp: number  // Unix ms
  value: number
}

export interface ServiceMetrics {
  name: string
  callRate: number    // calls/sec
  errorRate: number   // 0.0 ~ 1.0
  p50Latency: number  // ms
  p95Latency: number  // ms
  p99Latency: number  // ms
  avgDuration: number // ms
}

export interface OperationMetrics {
  operation: string
  callRate: number
  errorRate: number
  p50Latency: number
  p95Latency: number
  p99Latency: number
  avgDuration: number
}

export interface ServiceMetricsResult {
  service: string
  metrics: ServiceMetrics
  operations?: OperationMetrics[]
  latencies?: MetricPoint[]
  calls?: MetricPoint[]
  errors?: MetricPoint[]
}
```

---

## Component Structure

```
src/components/traces/
├── TracesTab.tsx           # 메인 컨테이너
├── ServiceSelector.tsx     # 서비스 선택 드롭다운
├── TraceSearchForm.tsx     # 검색 필터 폼
├── TraceList.tsx           # 트레이스 목록
├── TraceListItem.tsx       # 트레이스 행
├── TraceDetail/
│   ├── TraceDetailModal.tsx  # 상세 모달
│   ├── SpanTree.tsx          # 트리 구조 (계층)
│   ├── SpanTimeline.tsx      # Gantt 차트 (타임라인)
│   ├── SpanRow.tsx           # 개별 Span 행
│   └── SpanDetail.tsx        # Span 클릭 시 상세
├── Dependencies/
│   ├── DependencyGraph.tsx   # D3.js Force-Directed Graph
│   └── DependencyLegend.tsx  # 범례
└── Metrics/
    ├── ServiceMetricsCard.tsx   # RED 메트릭 카드
    ├── MetricsChart.tsx         # 시계열 차트 (Recharts)
    └── OperationsTable.tsx      # Operation별 메트릭 테이블
```

---

## Data Transformations

### 1. Span Tree 변환

Backend에서 받은 flat한 spans 배열을 트리 구조로 변환:

```typescript
interface SpanNode extends Span {
  children: SpanNode[]
  depth: number
  relativeStart: number  // 트레이스 시작 기준 상대 시간
}

function buildSpanTree(spans: Span[]): SpanNode[] {
  // 1. spanId → span 맵 생성
  const spanMap = new Map(spans.map(s => [s.spanId, { ...s, children: [], depth: 0, relativeStart: 0 }]))
  
  // 2. 트레이스 시작 시간 (가장 빠른 startTime)
  const traceStart = Math.min(...spans.map(s => s.startTime))
  
  // 3. 부모-자식 관계 구축 & 상대 시간 계산
  const roots: SpanNode[] = []
  for (const span of spanMap.values()) {
    span.relativeStart = span.startTime - traceStart
    
    const parentRef = span.references.find(r => r.refType === 'CHILD_OF')
    if (parentRef && spanMap.has(parentRef.spanId)) {
      const parent = spanMap.get(parentRef.spanId)!
      parent.children.push(span)
      span.depth = parent.depth + 1
    } else {
      roots.push(span)
    }
  }
  
  // 4. 자식 정렬 (startTime 순)
  const sortChildren = (node: SpanNode) => {
    node.children.sort((a, b) => a.startTime - b.startTime)
    node.children.forEach(sortChildren)
  }
  roots.forEach(sortChildren)
  
  return roots
}
```

### 2. Gantt Chart 데이터 변환

```typescript
interface GanttBar {
  spanId: string
  label: string         // "serviceName • operationName"
  start: number         // relative μs
  duration: number      // μs
  color: string         // 서비스별 색상
  depth: number
  hasError: boolean
}

function spansToGanttBars(spans: Span[]): GanttBar[] {
  const tree = buildSpanTree(spans)
  const traceStart = Math.min(...spans.map(s => s.startTime))
  const bars: GanttBar[] = []
  
  const traverse = (node: SpanNode) => {
    bars.push({
      spanId: node.spanId,
      label: `${node.serviceName} • ${node.operationName}`,
      start: node.startTime - traceStart,
      duration: node.duration,
      color: getServiceColor(node.serviceName),
      depth: node.depth,
      hasError: node.hasError
    })
    node.children.forEach(traverse)
  }
  
  tree.forEach(traverse)
  return bars
}
```

### 3. 서비스 색상 맵

```typescript
const SERVICE_COLORS: Record<string, string> = {
  'hololive-bot': '#3B82F6',      // blue-500
  'mcp-llm-server': '#10B981',    // emerald-500
  'twentyq-game': '#F59E0B',      // amber-500
  'turtlesoup-game': '#8B5CF6',   // violet-500
  'jaeger': '#6B7280',            // gray-500
}

function getServiceColor(service: string): string {
  return SERVICE_COLORS[service] ?? '#6B7280'
}
```

---

## UI/UX Guidelines

### 1. Trace List

- **Duration Badge**: 색상으로 성능 표시
  - `< 100ms`: green
  - `100ms ~ 500ms`: yellow  
  - `> 500ms`: red
  
- **Error 표시**: `hasError: true`인 경우 빨간색 경고 아이콘

### 2. Span Timeline (Gantt Chart)

- **가상화 필수**: 100+ spans 지원을 위해 `@tanstack/react-virtual` 사용
- **줌/팬**: 마우스 휠로 타임라인 확대/축소
- **호버 효과**: 관련 부모/자식 span 하이라이트

### 3. Dependencies Graph

- **D3.js Force-Directed Layout** 또는 **Dagre**
- **노드 크기**: callCount에 비례
- **엣지 두께**: 호출 빈도에 비례

### 4. Metrics Charts

- **Recharts** 또는 **Chart.js** 사용
- **시간 범위 선택**: 1h, 6h, 24h, 7d 버튼
- **실시간 갱신**: 30초 자동 새로고침

---

## API Client Example

```typescript
// src/api/traces.ts
import { api } from './index'

export const tracesApi = {
  getHealth: () => api.get('/traces/health'),
  
  getServices: () => api.get('/traces/services'),
  
  getOperations: (service: string) => 
    api.get(`/traces/operations/${encodeURIComponent(service)}`),
  
  searchTraces: (params: TraceSearchParams) => 
    api.get('/traces', { params }),
  
  getTrace: (traceId: string) => 
    api.get(`/traces/${traceId}`),
  
  getDependencies: (lookback = '24h') => 
    api.get('/traces/dependencies', { params: { lookback } }),
  
  getMetrics: (service: string, params?: MetricsQueryParams) => 
    api.get(`/traces/metrics/${encodeURIComponent(service)}`, { params }),
}
```

---

## React Query Keys

```typescript
// src/api/queryKeys.ts
export const tracesKeys = {
  all: ['traces'] as const,
  health: () => [...tracesKeys.all, 'health'] as const,
  services: () => [...tracesKeys.all, 'services'] as const,
  operations: (service: string) => [...tracesKeys.all, 'operations', service] as const,
  search: (params: TraceSearchParams) => [...tracesKeys.all, 'search', params] as const,
  detail: (traceId: string) => [...tracesKeys.all, 'detail', traceId] as const,
  dependencies: (lookback: string) => [...tracesKeys.all, 'dependencies', lookback] as const,
  metrics: (service: string, params?: MetricsQueryParams) => 
    [...tracesKeys.all, 'metrics', service, params] as const,
}
```

---

## Performance Considerations

1. **Virtualization**: 100+ spans 시 필수
2. **Debounce**: 검색 입력 500ms debounce
3. **Lazy Loading**: 트레이스 상세는 클릭 시 fetch
4. **Cache**: React Query 5분 stale time
5. **Skeleton UI**: 로딩 중 skeleton 표시

---

## Error Handling

| Status | Message | UI 처리 |
|--------|---------|---------|
| `503` | Jaeger service unavailable | 경고 배너 + 재시도 버튼 |
| `400` | Invalid parameter | 폼 validation 오류 표시 |
| `404` | Trace not found | "트레이스를 찾을 수 없습니다" |
| `500` | Failed to fetch from Jaeger | 토스트 에러 메시지 |

---

## Next Steps

1. [ ] `TracesTab.tsx` 컴포넌트 생성
2. [ ] `tracesApi` 클라이언트 추가
3. [ ] `SpanTimeline` Gantt 차트 구현
4. [ ] `DependencyGraph` D3.js 시각화
5. [ ] `MetricsChart` 시계열 차트
6. [ ] React Query 통합
7. [ ] E2E 테스트
