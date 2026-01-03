package traces

import "encoding/json"

// TraceSearchParams: 트레이스 검색 파라미터
type TraceSearchParams struct {
	Service     string            `json:"service"`
	Operation   string            `json:"operation,omitempty"`
	Limit       int               `json:"limit"`
	Lookback    string            `json:"lookback"`
	MinDuration string            `json:"minDuration,omitempty"`
	MaxDuration string            `json:"maxDuration,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
}

// TraceSummary: 트레이스 요약
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
	Limit  int            `json:"limit"`
}

// Reference: Span 참조
type Reference struct {
	RefType string `json:"refType"`
	TraceID string `json:"traceId"`
	SpanID  string `json:"spanId"`
}

// Tag: Span 태그
type Tag struct {
	Key   string `json:"key"`
	Type  string `json:"type"`
	Value any    `json:"value"`
}

// LogField: 로그 필드
type LogField struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

// Log: Span 로그
type Log struct {
	Timestamp int64      `json:"timestamp"`
	Fields    []LogField `json:"fields"`
}

// Span: 개별 Span 정보
type Span struct {
	SpanID        string      `json:"spanId"`
	TraceID       string      `json:"traceId"`
	OperationName string      `json:"operationName"`
	ServiceName   string      `json:"serviceName"`
	Duration      int64       `json:"duration"`
	StartTime     int64       `json:"startTime"`
	References    []Reference `json:"references"`
	Tags          []Tag       `json:"tags"`
	Logs          []Log       `json:"logs"`
	ProcessID     string      `json:"processId"`
	HasError      bool        `json:"hasError"`
}

// Process: 서비스 프로세스
type Process struct {
	ServiceName string `json:"serviceName"`
	Tags        []Tag  `json:"tags"`
}

// TraceDetail: 트레이스 상세
type TraceDetail struct {
	TraceID   string             `json:"traceId"`
	Spans     []Span             `json:"spans"`
	Processes map[string]Process `json:"processes"`
}

// Dependency: 서비스 의존성
type Dependency struct {
	Parent    string `json:"parent"`
	Child     string `json:"child"`
	CallCount int64  `json:"callCount"`
}

// DependenciesResult: Dependencies API 결과
type DependenciesResult struct {
	Dependencies []Dependency `json:"dependencies"`
}

// MetricsParams: 메트릭 조회 파라미터
type MetricsParams struct {
	Service   string `json:"service"`
	SpanKind  string `json:"spanKind,omitempty"`
	Quantile  string `json:"quantile,omitempty"`
	Lookback  string `json:"lookback,omitempty"`
	Step      string `json:"step,omitempty"`
	RatePer   string `json:"ratePer,omitempty"`
	GroupByOp bool   `json:"groupByOperation"`
}

// ServiceMetrics: 서비스 메트릭
type ServiceMetrics struct {
	Name        string  `json:"name"`
	CallRate    float64 `json:"callRate"`
	ErrorRate   float64 `json:"errorRate"`
	P50Latency  float64 `json:"p50Latency"`
	P95Latency  float64 `json:"p95Latency"`
	P99Latency  float64 `json:"p99Latency"`
	AvgDuration float64 `json:"avgDuration"`
}

// MetricPoint: 시계열 데이터 포인트
type MetricPoint struct {
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

// OperationMetrics: Operation별 메트릭
type OperationMetrics struct {
	Operation   string  `json:"operation"`
	CallRate    float64 `json:"callRate"`
	ErrorRate   float64 `json:"errorRate"`
	P50Latency  float64 `json:"p50Latency"`
	P95Latency  float64 `json:"p95Latency"`
	P99Latency  float64 `json:"p99Latency"`
	AvgDuration float64 `json:"avgDuration"`
}

// ServiceMetricsResult: 서비스 메트릭 결과
type ServiceMetricsResult struct {
	Service    string             `json:"service"`
	Metrics    ServiceMetrics     `json:"metrics"`
	Operations []OperationMetrics `json:"operations,omitempty"`
	Latencies  []MetricPoint      `json:"latencies,omitempty"`
	Calls      []MetricPoint      `json:"calls,omitempty"`
	Errors     []MetricPoint      `json:"errors,omitempty"`
}

// ===== 내부 JSON 응답 타입 =====

type servicesResponse struct {
	Data   []string `json:"data"`
	Total  int      `json:"total"`
	Limit  int      `json:"limit"`
	Offset int      `json:"offset"`
	Errors []any    `json:"errors"`
}

type operationsResponse struct {
	Data   []string `json:"data"`
	Total  int      `json:"total"`
	Limit  int      `json:"limit"`
	Offset int      `json:"offset"`
	Errors []any    `json:"errors"`
}

type rawSpan struct {
	TraceID       string      `json:"traceID"`
	SpanID        string      `json:"spanID"`
	OperationName string      `json:"operationName"`
	References    []Reference `json:"references"`
	StartTime     int64       `json:"startTime"`
	Duration      int64       `json:"duration"`
	Tags          []Tag       `json:"tags"`
	Logs          []Log       `json:"logs"`
	ProcessID     string      `json:"processID"`
}

type rawProcess struct {
	ServiceName string `json:"serviceName"`
	Tags        []Tag  `json:"tags"`
}

type rawTrace struct {
	TraceID   string                `json:"traceID"`
	Spans     []rawSpan             `json:"spans"`
	Processes map[string]rawProcess `json:"processes"`
}

type tracesResponse struct {
	Data   []rawTrace `json:"data"`
	Total  int        `json:"total"`
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
	Errors []any      `json:"errors"`
}

type dependenciesResponse struct {
	Data   []Dependency `json:"data"`
	Errors []any        `json:"errors"`
}

type metricsResponse struct {
	Metrics []struct {
		Labels []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"labels"`
		MetricPoints []metricPoint `json:"metricPoints"`
	} `json:"metrics"`
	Errors []any `json:"errors"`
}

type metricPoint struct {
	Timestamp  json.RawMessage `json:"timestamp"`
	Value      json.RawMessage `json:"value"`
	GaugeValue *struct {
		DoubleValue json.RawMessage `json:"doubleValue"`
	} `json:"gaugeValue,omitempty"`
}
