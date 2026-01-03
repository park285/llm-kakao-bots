// Package traces: Jaeger Query API 클라이언트
package traces

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Client: Jaeger Query API 클라이언트
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewClient: Jaeger 클라이언트 생성
func NewClient(baseURL string, timeout time.Duration, logger *slog.Logger) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
		logger: logger.With(slog.String("component", "jaeger-client")),
	}
}

// Available: Jaeger 가용성 확인
func (c *Client) Available(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/services", http.NoBody)
	if err != nil {
		return false
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode == http.StatusOK
}

// GetServices: 서비스 목록 조회
func (c *Client) GetServices(ctx context.Context) ([]string, error) {
	endpoint := c.baseURL + "/api/services"
	body, err := c.doRequest(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("get services: %w", err)
	}

	var resp servicesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse services response: %w", err)
	}
	return resp.Data, nil
}

// GetOperations: 서비스별 Operation 목록 조회
func (c *Client) GetOperations(ctx context.Context, service string) ([]string, error) {
	endpoint := fmt.Sprintf("%s/api/services/%s/operations", c.baseURL, url.PathEscape(service))
	body, err := c.doRequest(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("get operations: %w", err)
	}

	var resp operationsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse operations response: %w", err)
	}
	return resp.Data, nil
}

// SearchTraces: 트레이스 검색
func (c *Client) SearchTraces(ctx context.Context, params TraceSearchParams) (*TraceSearchResult, error) {
	start, err := parseLookback(params.Lookback)
	if err != nil {
		return nil, fmt.Errorf("parse lookback: %w", err)
	}

	queryParams := url.Values{}
	queryParams.Set("service", params.Service)
	queryParams.Set("start", strconv.FormatInt(start, 10))
	queryParams.Set("limit", strconv.Itoa(params.Limit))

	if params.Operation != "" {
		queryParams.Set("operation", params.Operation)
	}
	if params.MinDuration != "" {
		queryParams.Set("minDuration", params.MinDuration)
	}
	if params.MaxDuration != "" {
		queryParams.Set("maxDuration", params.MaxDuration)
	}
	if len(params.Tags) > 0 {
		// Jaeger v2 API: each tag as separate "tag=key:value" parameter
		for k, v := range params.Tags {
			queryParams.Add("tag", fmt.Sprintf("%s:%s", k, v))
		}
	}

	endpoint := c.baseURL + "/api/traces?" + queryParams.Encode()
	body, err := c.doRequest(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("search traces: %w", err)
	}

	var resp tracesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse traces response: %w", err)
	}

	traces := make([]TraceSummary, 0, len(resp.Data))
	for i := range resp.Data {
		summary := c.traceToSummary(&resp.Data[i])
		traces = append(traces, summary)
	}

	return &TraceSearchResult{
		Traces: traces,
		Total:  len(traces),
		Limit:  params.Limit,
	}, nil
}

// GetTrace: 트레이스 상세 조회
func (c *Client) GetTrace(ctx context.Context, traceID string) (*TraceDetail, error) {
	endpoint := fmt.Sprintf("%s/api/traces/%s", c.baseURL, url.PathEscape(traceID))
	body, err := c.doRequest(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("get trace: %w", err)
	}

	var resp tracesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse trace response: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, ErrTraceNotFound
	}

	raw := &resp.Data[0]
	return c.enrichTrace(raw), nil
}

// GetDependencies: 서비스 의존성 그래프 조회
func (c *Client) GetDependencies(ctx context.Context, lookback string) (*DependenciesResult, error) {
	endTs := time.Now().UnixMilli()
	lookbackDuration, err := parseLookbackMillis(lookback)
	if err != nil {
		return nil, fmt.Errorf("parse lookback: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/dependencies?endTs=%d&lookback=%d",
		c.baseURL, endTs, lookbackDuration)

	body, err := c.doRequest(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("get dependencies: %w", err)
	}

	var resp dependenciesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse dependencies response: %w", err)
	}

	return &DependenciesResult{
		Dependencies: resp.Data,
	}, nil
}

// GetServiceMetrics: SPM 메트릭 조회
func (c *Client) GetServiceMetrics(ctx context.Context, params MetricsParams) (*ServiceMetricsResult, error) {
	queryParams := url.Values{}
	queryParams.Set("service", params.Service)

	if params.Quantile == "" {
		params.Quantile = "0.95"
	}
	queryParams.Set("quantile", params.Quantile)

	if params.SpanKind == "" {
		params.SpanKind = "server"
	}
	queryParams.Set("spanKind", params.SpanKind)

	if params.Lookback != "" {
		lookbackMillis, err := parseLookbackMillis(params.Lookback)
		if err != nil {
			lookbackMillis = 3600000
		}
		queryParams.Set("lookback", strconv.FormatInt(lookbackMillis, 10))
	}
	if params.Step != "" {
		stepMillis, err := parseLookbackMillis(params.Step)
		if err != nil {
			stepMillis = 60000
		}
		queryParams.Set("step", strconv.FormatInt(stepMillis, 10))
	}
	if params.RatePer != "" {
		ratePerMillis := parseRatePerToMillis(params.RatePer)
		queryParams.Set("ratePer", strconv.FormatInt(ratePerMillis, 10))
	}
	if params.GroupByOp {
		queryParams.Set("groupByOperation", "true")
	}

	result := &ServiceMetricsResult{Service: params.Service}

	// Latency 조회
	latencyBody, _ := c.doRequest(ctx, c.baseURL+"/api/metrics/latencies?"+queryParams.Encode())
	if len(latencyBody) > 0 {
		var latencyResp metricsResponse
		if err := json.Unmarshal(latencyBody, &latencyResp); err == nil {
			result.Latencies = c.extractMetricPoints(latencyResp)
			if avg := c.calculateAverage(result.Latencies); avg > 0 {
				result.Metrics.P95Latency = avg
			}
		}
	}

	// Calls 조회
	callsBody, _ := c.doRequest(ctx, c.baseURL+"/api/metrics/calls?"+queryParams.Encode())
	if len(callsBody) > 0 {
		var callsResp metricsResponse
		if err := json.Unmarshal(callsBody, &callsResp); err == nil {
			result.Calls = c.extractMetricPoints(callsResp)
			if avg := c.calculateAverage(result.Calls); avg > 0 {
				result.Metrics.CallRate = avg
			}
		}
	}

	// Errors 조회
	errorsBody, _ := c.doRequest(ctx, c.baseURL+"/api/metrics/errors?"+queryParams.Encode())
	if len(errorsBody) > 0 {
		var errorsResp metricsResponse
		if err := json.Unmarshal(errorsBody, &errorsResp); err == nil {
			result.Errors = c.extractMetricPoints(errorsResp)
			if avg := c.calculateAverage(result.Errors); avg > 0 {
				result.Metrics.ErrorRate = avg
			}
		}
	}

	result.Metrics.Name = params.Service
	return result, nil
}

func (c *Client) doRequest(ctx context.Context, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Warn("jaeger_api_error",
			slog.String("endpoint", endpoint),
			slog.Int("status", resp.StatusCode))
		return nil, fmt.Errorf("jaeger api error: status %d", resp.StatusCode)
	}

	return body, nil
}

func (c *Client) traceToSummary(raw *rawTrace) TraceSummary {
	if len(raw.Spans) == 0 {
		return TraceSummary{TraceID: raw.TraceID}
	}

	var rootSpan *rawSpan
	for i := range raw.Spans {
		s := &raw.Spans[i]
		if len(s.References) == 0 {
			rootSpan = s
			break
		}
		if rootSpan == nil || s.StartTime < rootSpan.StartTime {
			rootSpan = s
		}
	}

	serviceSet := make(map[string]struct{})
	for i := range raw.Spans {
		s := &raw.Spans[i]
		if proc, ok := raw.Processes[s.ProcessID]; ok {
			serviceSet[proc.ServiceName] = struct{}{}
		}
	}
	services := make([]string, 0, len(serviceSet))
	for svc := range serviceSet {
		services = append(services, svc)
	}

	hasError := false
	for i := range raw.Spans {
		if c.hasErrorTag(raw.Spans[i].Tags) {
			hasError = true
			break
		}
	}

	var duration int64
	var startTimeStr string
	if rootSpan != nil {
		duration = rootSpan.Duration
		startTimeStr = time.UnixMicro(rootSpan.StartTime).UTC().Format(time.RFC3339Nano)
	}

	return TraceSummary{
		TraceID:       raw.TraceID,
		SpanCount:     len(raw.Spans),
		Services:      services,
		OperationName: rootSpan.OperationName,
		Duration:      duration,
		StartTime:     startTimeStr,
		HasError:      hasError,
	}
}

func (c *Client) enrichTrace(raw *rawTrace) *TraceDetail {
	spans := make([]Span, len(raw.Spans))
	for i := range raw.Spans {
		s := &raw.Spans[i]
		proc := raw.Processes[s.ProcessID]

		spans[i] = Span{
			SpanID:        s.SpanID,
			TraceID:       s.TraceID,
			OperationName: s.OperationName,
			ServiceName:   proc.ServiceName,
			Duration:      s.Duration,
			StartTime:     s.StartTime,
			References:    s.References,
			Tags:          SanitizeTags(s.Tags),
			Logs:          SanitizeLogs(s.Logs),
			ProcessID:     s.ProcessID,
			HasError:      c.hasErrorTag(s.Tags),
		}
	}

	processes := make(map[string]Process, len(raw.Processes))
	for k, v := range raw.Processes {
		processes[k] = Process(v)
	}

	return &TraceDetail{
		TraceID:   raw.TraceID,
		Spans:     spans,
		Processes: processes,
	}
}

func (c *Client) hasErrorTag(tags []Tag) bool {
	for _, tag := range tags {
		if tag.Key == "error" {
			switch v := tag.Value.(type) {
			case bool:
				return v
			case string:
				return v == "true"
			case float64:
				return v != 0
			}
		}
	}
	return false
}

func (c *Client) extractMetricPoints(resp metricsResponse) []MetricPoint {
	var points []MetricPoint
	for _, m := range resp.Metrics {
		for _, mp := range m.MetricPoints {
			ts, ok := parseMetricTimestamp(mp.Timestamp)
			if !ok {
				continue
			}
			valueRaw := mp.Value
			if mp.GaugeValue != nil && len(mp.GaugeValue.DoubleValue) > 0 {
				valueRaw = mp.GaugeValue.DoubleValue
			}
			value, ok := parseMetricValue(valueRaw)
			if !ok {
				continue
			}
			points = append(points, MetricPoint{Timestamp: ts, Value: value})
		}
	}
	return points
}

func (c *Client) calculateAverage(points []MetricPoint) float64 {
	if len(points) == 0 {
		return 0
	}
	var sum float64
	for _, p := range points {
		sum += p.Value
	}
	return sum / float64(len(points))
}

// ===== Helper Functions =====

func parseLookback(lookback string) (int64, error) {
	if lookback == "" {
		lookback = "1h"
	}

	if strings.HasSuffix(lookback, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(lookback, "d"))
		if err != nil {
			return 0, fmt.Errorf("invalid days format: %s", lookback)
		}
		duration := time.Duration(days) * 24 * time.Hour
		return time.Now().Add(-duration).UnixMicro(), nil
	}

	duration, err := time.ParseDuration(lookback)
	if err != nil {
		return 0, fmt.Errorf("invalid duration format: %s", lookback)
	}

	return time.Now().Add(-duration).UnixMicro(), nil
}

func parseLookbackMillis(lookback string) (int64, error) {
	if lookback == "" {
		lookback = "1h"
	}

	if strings.HasSuffix(lookback, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(lookback, "d"))
		if err != nil {
			return 0, fmt.Errorf("invalid days format: %s", lookback)
		}
		return int64(days) * 24 * 60 * 60 * 1000, nil
	}

	duration, err := time.ParseDuration(lookback)
	if err != nil {
		return 0, fmt.Errorf("invalid duration format: %s", lookback)
	}

	return duration.Milliseconds(), nil
}

func parseRatePerToMillis(ratePer string) int64 {
	switch strings.ToLower(ratePer) {
	case "second", "s", "1s":
		return 1000
	case "minute", "m", "1m":
		return 60000
	case "hour", "h", "1h":
		return 3600000
	default:
		if ms, err := strconv.ParseInt(ratePer, 10, 64); err == nil && ms > 0 {
			return ms
		}
		return 1000
	}
}

func parseMetricTimestamp(raw json.RawMessage) (int64, bool) {
	if len(raw) == 0 {
		return 0, false
	}

	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if s == "" {
			return 0, false
		}
		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			return t.UnixMilli(), true
		}
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			return normalizeEpochMillis(int64(v)), true
		}
		return 0, false
	}

	var i int64
	if err := json.Unmarshal(raw, &i); err == nil {
		return normalizeEpochMillis(i), true
	}

	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return normalizeEpochMillis(int64(f)), true
	}

	return 0, false
}

func parseMetricValue(raw json.RawMessage) (float64, bool) {
	if len(raw) == 0 {
		return 0, false
	}

	var v float64
	if err := json.Unmarshal(raw, &v); err == nil {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, false
		}
		return v, true
	}

	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if s == "" {
			return 0, false
		}
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, false
		}
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, false
		}
		return v, true
	}

	return 0, false
}

func normalizeEpochMillis(value int64) int64 {
	switch {
	case value > 100_000_000_000_000_000:
		return value / 1_000_000
	case value > 100_000_000_000_000:
		return value / 1_000
	case value > 100_000_000_000:
		return value
	case value > 1_000_000_000:
		return value * 1_000
	default:
		return value
	}
}

// ErrTraceNotFound: 트레이스 미발견
var ErrTraceNotFound = fmt.Errorf("trace not found")
