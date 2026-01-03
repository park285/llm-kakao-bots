package gemini

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"google.golang.org/genai"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/llm"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/metrics"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/usage"
)

var (
	// ErrMissingAPIKey: Gemini API 키가 없을 때 반환됩니다.
	ErrMissingAPIKey = errors.New("missing gemini api key")
	// ErrInvalidModel: 지원하지 않는 모델일 때 반환됩니다.
	ErrInvalidModel = errors.New("invalid model")
)

// Request: Gemini 요청 데이터입니다.
type Request struct {
	Prompt       string
	SystemPrompt string
	History      []llm.HistoryEntry
	Model        string
	Task         string
}

// Client: Gemini API 호출을 담당하는 클라이언트입니다.
type Client struct {
	cfg           *config.Config
	metrics       *metrics.Store
	usageRecorder *usage.Recorder
	mu            sync.RWMutex // RWMutex로 읽기 경로 락 경합 감소
	clients       map[string]*genai.Client
	apiKeys       []string
	apiKeyIdx     int
}

// NewClient: Gemini 클라이언트를 생성합니다.
func NewClient(cfg *config.Config, metricsStore *metrics.Store, usageRecorder *usage.Recorder) (*Client, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}
	if metricsStore == nil {
		return nil, errors.New("metrics store is nil")
	}
	return &Client{
		cfg:           cfg,
		metrics:       metricsStore,
		usageRecorder: usageRecorder,
		clients:       make(map[string]*genai.Client),
		apiKeys:       cfg.Gemini.APIKeys,
	}, nil
}

// Chat: 텍스트 채팅 요청을 수행합니다.
func (c *Client) Chat(ctx context.Context, req Request) (string, string, error) {
	start := time.Now()
	response, model, err := c.generate(ctx, req, "", nil)
	if err != nil {
		c.metrics.RecordError(time.Since(start))
		return "", model, err
	}

	usageStats := extractUsage(response)
	c.metrics.RecordSuccess(time.Since(start), usageStats)
	c.recordUsage(ctx, usageStats)
	return response.Text(), model, nil
}

// ChatWithUsage: 텍스트 응답과 토큰 사용량을 함께 반환합니다.
func (c *Client) ChatWithUsage(ctx context.Context, req Request) (llm.ChatResult, string, error) {
	start := time.Now()
	response, model, err := c.generate(ctx, req, "", nil)
	if err != nil {
		c.metrics.RecordError(time.Since(start))
		return llm.ChatResult{}, model, err
	}

	textParts, thoughtParts := extractParts(response)
	text := strings.Join(textParts, "")
	reasoning := strings.Join(thoughtParts, "\n")
	usageStats := extractUsage(response)
	result := llm.ChatResult{
		Text:         text,
		Usage:        usageStats,
		Reasoning:    reasoning,
		HasReasoning: reasoning != "",
	}

	c.metrics.RecordSuccess(time.Since(start), usageStats)
	c.recordUsage(ctx, usageStats)
	return result, model, nil
}

// Structured: JSON 스키마 기반 응답을 반환합니다.
func (c *Client) Structured(ctx context.Context, req Request, schema map[string]any) (map[string]any, string, error) {
	parsed, model, _, err := c.structuredInternal(ctx, req, schema, false)
	return parsed, model, err
}

// StructuredResult: 검색 정보를 포함한 응답 결과입니다.
type StructuredResult struct {
	Payload       map[string]any
	Model         string
	SearchQueries []string // Google Search가 사용된 경우 검색 쿼리
}

// StructuredWithSearch: Google Search 도구를 활성화한 JSON 스키마 기반 응답을 반환합니다.
// LLM이 필요하다고 판단하면 자체적으로 검색을 수행합니다.
func (c *Client) StructuredWithSearch(ctx context.Context, req Request, schema map[string]any) (StructuredResult, error) {
	parsed, model, searchQueries, err := c.structuredInternal(ctx, req, schema, true)
	return StructuredResult{
		Payload:       parsed,
		Model:         model,
		SearchQueries: searchQueries,
	}, err
}

// ConsensusVote: 합의 로직에서 수집한 개별 응답입니다.
type ConsensusVote struct {
	Value      string
	Confidence float64
	Reasoning  string
}

type consensusScore struct {
	Count         int
	WeightSum     float64
	MaxConfidence float64
}

// verifyResultPriority: 검증 결과의 명시적 우선순위 맵입니다.
// 동점 시 사용자에게 유리한 방향으로 판정합니다: 정답 > 근접 > 오답
var verifyResultPriority = map[string]int{
	"정답": 3,
	"근접": 2,
	"오답": 1,
}

func pickConsensusWinner(scores map[string]consensusScore) (string, consensusScore) {
	const epsilon = 1e-9
	var winningValue string
	var winningScore consensusScore
	hasWinner := false

	for value, score := range scores {
		if !hasWinner {
			winningValue = value
			winningScore = score
			hasWinner = true
			continue
		}

		// 1순위: Count (다수결) - "확신에 찬 소수"가 다수를 이기지 못하도록 함
		if score.Count > winningScore.Count {
			winningValue = value
			winningScore = score
			continue
		}
		if score.Count < winningScore.Count {
			continue
		}

		// 2순위: WeightSum (신뢰도 합) - 동일 투표 수일 때 신뢰도로 결정
		if score.WeightSum > winningScore.WeightSum+epsilon {
			winningValue = value
			winningScore = score
			continue
		}
		if math.Abs(score.WeightSum-winningScore.WeightSum) > epsilon {
			continue
		}

		// 3순위: MaxConfidence (최대 개별 신뢰도)
		if score.MaxConfidence > winningScore.MaxConfidence+epsilon {
			winningValue = value
			winningScore = score
			continue
		}
		if math.Abs(score.MaxConfidence-winningScore.MaxConfidence) > epsilon {
			continue
		}

		// 4순위: 명시적 우선순위 맵 (정답 > 근접 > 오답)
		// 사전순 대신, 사용자에게 유리한 방향으로 판정
		valuePriority := verifyResultPriority[value]
		winnerPriority := verifyResultPriority[winningValue]
		if valuePriority > winnerPriority {
			winningValue = value
			winningScore = score
		}
	}

	return winningValue, winningScore
}

func parseNormalizedConfidence(payload map[string]any) (float64, bool) {
	if payload == nil {
		return 0, false
	}
	raw, ok := payload["confidence"]
	if !ok {
		return 0, false
	}

	var confidence float64
	switch value := raw.(type) {
	case float64:
		confidence = value
	case float32:
		confidence = float64(value)
	case int:
		confidence = float64(value)
	case int64:
		confidence = float64(value)
	case json.Number:
		parsed, err := value.Float64()
		if err != nil {
			return 0, false
		}
		confidence = parsed
	default:
		return 0, false
	}

	if confidence < 0 {
		confidence = 0
	} else if confidence > 1 {
		confidence = 1
	}
	return confidence, true
}

// ConsensusResult: 합의 로직 결과입니다.
type ConsensusResult struct {
	Payload         map[string]any
	Model           string
	SearchQueries   []string
	ConsensusField  string // 합의 기준 필드
	ConsensusValue  string // 합의된 값
	ConsensusCount  int    // 동의한 호출 수
	TotalCalls      int    // 총 호출 수
	SuccessfulCalls int
	ConsensusWeight float64
	TotalWeight     float64
	Votes           []ConsensusVote
}

// StructuredWithConsensus: 동일 요청을 N번 병렬 호출하여 합의된 결과를 반환합니다.
// fieldName 필드의 값을 기준으로 다수결을 수행합니다.
func (c *Client) StructuredWithConsensus(
	ctx context.Context,
	req Request,
	schema map[string]any,
	fieldName string,
	numCalls int,
) (ConsensusResult, error) {
	if numCalls <= 1 {
		result, err := c.StructuredWithSearch(ctx, req, schema)
		if err != nil {
			return ConsensusResult{}, err
		}
		value := ""
		if v, ok := result.Payload[fieldName].(string); ok {
			value = v
		}
		return ConsensusResult{
			Payload:        result.Payload,
			Model:          result.Model,
			SearchQueries:  result.SearchQueries,
			ConsensusField: fieldName,
			ConsensusValue: value,
			ConsensusCount: 1,
			TotalCalls:     1,
		}, nil
	}

	// 병렬 호출
	type callResult struct {
		result StructuredResult
		err    error
	}
	results := make(chan callResult, numCalls)

	for i := 0; i < numCalls; i++ {
		go func() {
			r, e := c.StructuredWithSearch(ctx, req, schema)
			results <- callResult{result: r, err: e}
		}()
	}

	// 결과 수집
	votes := make(map[string]int)
	payloads := make(map[string]map[string]any)
	var allSearchQueries []string
	var model string
	successCount := 0

	for i := 0; i < numCalls; i++ {
		cr := <-results
		if cr.err != nil {
			continue
		}
		successCount++
		if model == "" {
			model = cr.result.Model
		}
		allSearchQueries = append(allSearchQueries, cr.result.SearchQueries...)

		if value, ok := cr.result.Payload[fieldName].(string); ok {
			votes[value]++
			if _, exists := payloads[value]; !exists {
				payloads[value] = cr.result.Payload
			}
		}
	}

	if successCount == 0 {
		return ConsensusResult{}, errors.New("all consensus calls failed")
	}

	// 다수결
	var winningValue string
	var winningCount int
	for value, count := range votes {
		if count > winningCount {
			winningValue = value
			winningCount = count
		}
	}

	return ConsensusResult{
		Payload:        payloads[winningValue],
		Model:          model,
		SearchQueries:  allSearchQueries,
		ConsensusField: fieldName,
		ConsensusValue: winningValue,
		ConsensusCount: winningCount,
		TotalCalls:     numCalls,
	}, nil
}

// StructuredWithConsensusWeighted: 동일 요청을 N번 병렬 호출하여 confidence 가중치로 합의된 결과를 반환합니다.
// fieldName 필드 값별 confidence 합산(Weighted Voting)을 사용하고, 동점이면 count -> maxConfidence -> value 순으로 결정합니다.
func (c *Client) StructuredWithConsensusWeighted(
	ctx context.Context,
	req Request,
	schema map[string]any,
	fieldName string,
	numCalls int,
) (ConsensusResult, error) {
	if numCalls <= 1 {
		return c.structuredWithConsensusWeightedSingle(ctx, req, schema, fieldName)
	}
	return c.structuredWithConsensusWeightedMulti(ctx, req, schema, fieldName, numCalls)
}

func (c *Client) structuredWithConsensusWeightedSingle(
	ctx context.Context,
	req Request,
	schema map[string]any,
	fieldName string,
) (ConsensusResult, error) {
	result, err := c.StructuredWithSearch(ctx, req, schema)
	if err != nil {
		return ConsensusResult{}, err
	}

	value := ""
	if v, ok := result.Payload[fieldName].(string); ok {
		value = v
	}
	confidence, _ := parseNormalizedConfidence(result.Payload)
	reasoning, _ := result.Payload["reasoning"].(string)

	votes := make([]ConsensusVote, 0, 1)
	if value != "" {
		votes = append(votes, ConsensusVote{
			Value:      value,
			Confidence: confidence,
			Reasoning:  reasoning,
		})
	}

	return ConsensusResult{
		Payload:         result.Payload,
		Model:           result.Model,
		SearchQueries:   result.SearchQueries,
		ConsensusField:  fieldName,
		ConsensusValue:  value,
		ConsensusCount:  1,
		TotalCalls:      1,
		SuccessfulCalls: 1,
		ConsensusWeight: confidence,
		TotalWeight:     confidence,
		Votes:           votes,
	}, nil
}

type weightedConsensusCollector struct {
	scores             map[string]consensusScore
	payloads           map[string]map[string]any
	payloadConfidences map[string]float64

	votes []ConsensusVote

	allSearchQueries []string
	model            string
	successCount     int

	fallbackPayload    map[string]any
	fallbackConfidence float64
}

func newWeightedConsensusCollector(numCalls int) *weightedConsensusCollector {
	return &weightedConsensusCollector{
		scores:             make(map[string]consensusScore),
		payloads:           make(map[string]map[string]any),
		payloadConfidences: make(map[string]float64),
		votes:              make([]ConsensusVote, 0, numCalls),
	}
}

func (c *weightedConsensusCollector) add(result StructuredResult, fieldName string) {
	c.successCount++
	if c.model == "" {
		c.model = result.Model
	}
	c.allSearchQueries = append(c.allSearchQueries, result.SearchQueries...)

	if c.fallbackPayload == nil {
		c.fallbackPayload = result.Payload
		if confidence, ok := parseNormalizedConfidence(result.Payload); ok {
			c.fallbackConfidence = confidence
		}
	}

	value, ok := result.Payload[fieldName].(string)
	if !ok {
		return
	}

	confidence, _ := parseNormalizedConfidence(result.Payload)
	reasoning, _ := result.Payload["reasoning"].(string)

	c.votes = append(c.votes, ConsensusVote{
		Value:      value,
		Confidence: confidence,
		Reasoning:  reasoning,
	})

	score := c.scores[value]
	score.Count++
	score.WeightSum += confidence
	if confidence > score.MaxConfidence {
		score.MaxConfidence = confidence
	}
	c.scores[value] = score

	bestConfidence, hasBest := c.payloadConfidences[value]
	if !hasBest || confidence > bestConfidence {
		c.payloads[value] = result.Payload
		c.payloadConfidences[value] = confidence
	}
}

func (c *weightedConsensusCollector) toResult(fieldName string, totalCalls int) ConsensusResult {
	winningValue, winningScore := pickConsensusWinner(c.scores)
	totalWeight := 0.0
	for _, score := range c.scores {
		totalWeight += score.WeightSum
	}

	selectedPayload := c.payloads[winningValue]
	if selectedPayload == nil {
		selectedPayload = c.fallbackPayload
	}

	consensusWeight := winningScore.WeightSum
	if winningValue == "" && winningScore == (consensusScore{}) {
		consensusWeight = c.fallbackConfidence
		totalWeight = c.fallbackConfidence
	}

	return ConsensusResult{
		Payload:         selectedPayload,
		Model:           c.model,
		SearchQueries:   c.allSearchQueries,
		ConsensusField:  fieldName,
		ConsensusValue:  winningValue,
		ConsensusCount:  winningScore.Count,
		TotalCalls:      totalCalls,
		SuccessfulCalls: c.successCount,
		ConsensusWeight: consensusWeight,
		TotalWeight:     totalWeight,
		Votes:           c.votes,
	}
}

func (c *Client) structuredWithConsensusWeightedMulti(
	ctx context.Context,
	req Request,
	schema map[string]any,
	fieldName string,
	numCalls int,
) (ConsensusResult, error) {
	type callResult struct {
		result StructuredResult
		err    error
	}
	results := make(chan callResult, numCalls)

	for i := 0; i < numCalls; i++ {
		go func() {
			r, e := c.StructuredWithSearch(ctx, req, schema)
			results <- callResult{result: r, err: e}
		}()
	}

	collector := newWeightedConsensusCollector(numCalls)
	for i := 0; i < numCalls; i++ {
		cr := <-results
		if cr.err != nil {
			continue
		}
		collector.add(cr.result, fieldName)
	}

	if collector.successCount == 0 {
		return ConsensusResult{}, errors.New("all consensus calls failed")
	}
	return collector.toResult(fieldName, numCalls), nil
}

func (c *Client) structuredInternal(ctx context.Context, req Request, schema map[string]any, enableSearch bool) (map[string]any, string, []string, error) {
	start := time.Now()
	response, model, err := c.generateWithTools(ctx, req, "application/json", schema, enableSearch)
	if err != nil {
		c.metrics.RecordError(time.Since(start))
		return nil, model, nil, err
	}

	usageStats := extractUsage(response)
	c.metrics.RecordSuccess(time.Since(start), usageStats)
	c.recordUsage(ctx, usageStats)

	// grounding metadata에서 검색 쿼리 추출
	searchQueries := extractSearchQueries(response)

	payload := response.Text()
	if strings.TrimSpace(payload) == "" {
		return nil, model, searchQueries, errors.New("empty structured response")
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
		return nil, model, searchQueries, fmt.Errorf("decode structured response: %w", err)
	}

	return parsed, model, searchQueries, nil
}

func (c *Client) recordUsage(ctx context.Context, usageStats llm.Usage) {
	// 캐시 적중 시 DEBUG 로그 출력
	if usageStats.CachedTokens > 0 {
		slog.DebugContext(ctx, "cache_hit",
			"cached_tokens", usageStats.CachedTokens,
			"input_tokens", usageStats.InputTokens,
			"hit_ratio", fmt.Sprintf("%.1f%%", usageStats.CacheHitRatio()*100),
		)
	}

	if c.usageRecorder == nil {
		return
	}
	c.usageRecorder.Record(ctx, int64(usageStats.InputTokens), int64(usageStats.OutputTokens), int64(usageStats.ReasoningTokens))
}

func (c *Client) generateWithTools(
	ctx context.Context,
	req Request,
	responseMimeType string,
	responseSchema map[string]any,
	enableSearch bool,
) (*genai.GenerateContentResponse, string, error) {
	model, err := c.resolveModel(req.Model, req.Task)
	if err != nil {
		return nil, model, err
	}

	genConfig := c.buildGenerateConfig(req.SystemPrompt, req.Task, model, responseMimeType, responseSchema)

	// Google Search 도구 활성화
	if enableSearch {
		genConfig.Tools = []*genai.Tool{
			{GoogleSearch: &genai.GoogleSearch{}},
		}
	}

	contents := buildContents(req.Prompt, req.History)

	maxAttempts := max(1, c.cfg.Gemini.MaxRetries)
	if c.cfg.Gemini.FailoverAttempts > 0 && len(c.apiKeys) > 0 {
		maxAttempts = min(maxAttempts, c.cfg.Gemini.FailoverAttempts*len(c.apiKeys))
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			if err := sleepWithContext(ctx, retryDelay(attempt)); err != nil {
				return nil, model, err
			}
		}

		client, err := c.selectClient(ctx)
		if err != nil {
			return nil, model, err
		}

		response, err := client.Models.GenerateContent(ctx, model, contents, genConfig)
		if err == nil {
			return response, model, nil
		}

		lastErr = err
		if !isRetryableGenerateError(err) {
			break
		}
	}

	if lastErr == nil {
		lastErr = errors.New("unknown generate content error")
	}
	return nil, model, fmt.Errorf("generate content: %w", lastErr)
}

func (c *Client) generate(
	ctx context.Context,
	req Request,
	responseMimeType string,
	responseSchema map[string]any,
) (*genai.GenerateContentResponse, string, error) {
	return c.generateWithTools(ctx, req, responseMimeType, responseSchema, false)
}

// selectClient는 라운드로빈으로 API 키를 선택하고 해당 클라이언트를 반환한다.
// Double-checked locking 패턴으로 읽기 경로 최적화.
func (c *Client) selectClient(ctx context.Context) (*genai.Client, error) {
	if len(c.apiKeys) == 0 {
		return nil, ErrMissingAPIKey
	}

	// 먼저 API 키 인덱스를 원자적으로 증가 (락 없이)
	c.mu.Lock()
	keyIdx := c.apiKeyIdx
	c.apiKeyIdx++
	c.mu.Unlock()

	key := c.apiKeys[keyIdx%len(c.apiKeys)]

	// 읽기 락으로 캐시 히트 확인
	c.mu.RLock()
	if client, ok := c.clients[key]; ok {
		c.mu.RUnlock()
		return client, nil
	}
	c.mu.RUnlock()

	// 쓰기 락으로 클라이언트 생성 (Double-checked locking)
	c.mu.Lock()
	defer c.mu.Unlock()

	// 다른 goroutine이 이미 생성했을 수 있으므로 재확인
	if client, ok := c.clients[key]; ok {
		return client, nil
	}

	timeout := time.Duration(c.cfg.Gemini.TimeoutSeconds) * time.Second

	// OTel HTTP Transport: 분산 추적이 활성화되면 HTTP 요청을 추적함
	var httpClient *http.Client
	if c.cfg.Telemetry.Enabled {
		httpClient = &http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport,
				otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
					return "Gemini." + r.Method
				}),
			),
			Timeout: timeout,
		}
	}

	client, err := genai.NewClient(context.WithoutCancel(ctx), &genai.ClientConfig{
		APIKey:     key,
		Backend:    genai.BackendGeminiAPI,
		HTTPClient: httpClient, // nil이면 SDK가 기본 클라이언트 사용
		HTTPOptions: genai.HTTPOptions{
			Timeout: genai.Ptr(timeout),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create genai client: %w", err)
	}

	c.clients[key] = client
	return client, nil
}

func (c *Client) resolveModel(modelOverride string, task string) (string, error) {
	model := modelOverride
	if model == "" {
		model = c.cfg.Gemini.ModelForTask(task)
	}
	if model == "" {
		return "", ErrInvalidModel
	}
	if !isGemini3(model) {
		return model, ErrInvalidModel
	}
	return model, nil
}

func (c *Client) buildGenerateConfig(
	systemPrompt string,
	task string,
	model string,
	responseMimeType string,
	responseSchema map[string]any,
) *genai.GenerateContentConfig {
	temperature := float32(c.cfg.Gemini.TemperatureForModel(model))
	genConfig := &genai.GenerateContentConfig{
		Temperature:     genai.Ptr(temperature),
		MaxOutputTokens: int32(c.cfg.Gemini.MaxOutputTokens),
	}

	if systemPrompt != "" {
		genConfig.SystemInstruction = genai.NewContentFromText(systemPrompt, genai.RoleUser)
	}
	if responseMimeType != "" {
		genConfig.ResponseMIMEType = responseMimeType
	}
	if responseSchema != nil {
		genConfig.ResponseJsonSchema = responseSchema
	}

	if thinkingLevel, ok := normalizeThinkingLevel(c.cfg.Gemini.Thinking.Level(task)); ok {
		genConfig.ThinkingConfig = &genai.ThinkingConfig{
			IncludeThoughts: true,
			ThinkingLevel:   thinkingLevel,
		}
	}

	return genConfig
}

func buildContents(prompt string, history []llm.HistoryEntry) []*genai.Content {
	contents := make([]*genai.Content, 0, len(history)+1)
	for _, entry := range history {
		var role genai.Role = genai.RoleUser
		if strings.EqualFold(entry.Role, "assistant") {
			role = genai.RoleModel
		}
		contents = append(contents, genai.NewContentFromText(entry.Content, role))
	}
	contents = append(contents, genai.NewContentFromText(prompt, genai.RoleUser))
	return contents
}

func normalizeThinkingLevel(level string) (genai.ThinkingLevel, bool) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "low":
		return genai.ThinkingLevelLow, true
	case "medium":
		return genai.ThinkingLevelMedium, true
	case "high":
		return genai.ThinkingLevelHigh, true
	case "minimal":
		return genai.ThinkingLevelMinimal, true
	case "none", "":
		return "", false
	default:
		return "", false
	}
}

func extractParts(response *genai.GenerateContentResponse) ([]string, []string) {
	if response == nil || len(response.Candidates) == 0 {
		return nil, nil
	}
	content := response.Candidates[0].Content
	if content == nil || len(content.Parts) == 0 {
		return nil, nil
	}

	texts := make([]string, 0)
	thoughts := make([]string, 0)
	for _, part := range content.Parts {
		if part == nil || part.Text == "" {
			continue
		}
		if part.Thought {
			thoughts = append(thoughts, part.Text)
			continue
		}
		texts = append(texts, part.Text)
	}
	return texts, thoughts
}

func extractUsage(response *genai.GenerateContentResponse) llm.Usage {
	if response == nil || response.UsageMetadata == nil {
		return llm.Usage{}
	}
	usageMeta := response.UsageMetadata
	return llm.Usage{
		InputTokens:     int(usageMeta.PromptTokenCount),
		OutputTokens:    int(usageMeta.CandidatesTokenCount) + int(usageMeta.ThoughtsTokenCount),
		TotalTokens:     int(usageMeta.TotalTokenCount),
		ReasoningTokens: int(usageMeta.ThoughtsTokenCount),
		CachedTokens:    int(usageMeta.CachedContentTokenCount), // 암시적 캐싱 토큰 추출
	}
}

func isGemini3(model string) bool {
	return strings.Contains(strings.ToLower(model), "gemini-3")
}

// extractSearchQueries 는 응답에서 Google Search 쿼리를 추출한다.
func extractSearchQueries(response *genai.GenerateContentResponse) []string {
	if response == nil || len(response.Candidates) == 0 {
		return nil
	}
	gm := response.Candidates[0].GroundingMetadata
	if gm == nil {
		return nil
	}
	return gm.WebSearchQueries
}

func isRetryableGenerateError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var apiErr genai.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.Code {
		case 408, 429, 500, 502, 503, 504:
			return true
		default:
			return false
		}
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true
		}
		if temporary, ok := any(netErr).(interface{ Temporary() bool }); ok && temporary.Temporary() {
			return true
		}
	}

	return false
}

func retryDelay(attempt int) time.Duration {
	// attempt=1부터 backoff 적용 (attempt=0은 최초 호출)
	base := 200 * time.Millisecond
	maxDelay := 2 * time.Second

	exp := attempt - 1
	if exp > 6 {
		exp = 6
	}
	delay := base * time.Duration(1<<exp)
	if delay > maxDelay {
		delay = maxDelay
	}

	// ±20% jitter
	jitterRange := int64(delay / 5)
	if jitterRange <= 0 {
		return delay
	}
	jitter := time.Duration(rand.Int64N(jitterRange*2) - jitterRange)
	return delay + jitter
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return fmt.Errorf("sleep canceled: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}
