package gemini

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"google.golang.org/genai"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/llm"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/metrics"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/usage"
)

var (
	// ErrMissingAPIKey 는 Gemini API 키가 없을 때 반환된다.
	ErrMissingAPIKey = errors.New("missing gemini api key")
	// ErrInvalidModel 는 지원하지 않는 모델일 때 반환된다.
	ErrInvalidModel = errors.New("invalid model")
)

// Request 는 Gemini 요청 데이터다.
type Request struct {
	Prompt       string
	SystemPrompt string
	History      []llm.HistoryEntry
	Model        string
	Task         string
}

// Client 는 Gemini 호출을 담당한다.
type Client struct {
	cfg           *config.Config
	metrics       *metrics.Store
	usageRecorder *usage.Recorder
	mu            sync.RWMutex // RWMutex로 읽기 경로 락 경합 감소
	clients       map[string]*genai.Client
	apiKeys       []string
	apiKeyIdx     int
}

// NewClient 는 Gemini 클라이언트를 생성한다.
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

// Chat 은 텍스트 채팅 요청을 수행한다.
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

// ChatWithUsage 는 텍스트 응답과 사용량을 함께 반환한다.
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

// Structured 는 JSON 스키마 기반 응답을 반환한다.
func (c *Client) Structured(ctx context.Context, req Request, schema map[string]any) (map[string]any, string, error) {
	parsed, model, _, err := c.structuredInternal(ctx, req, schema, false)
	return parsed, model, err
}

// StructuredResult 는 검색 정보를 포함한 응답 결과다.
type StructuredResult struct {
	Payload       map[string]any
	Model         string
	SearchQueries []string // Google Search가 사용된 경우 검색 쿼리
}

// StructuredWithSearch 는 Google Search 도구를 활성화한 JSON 스키마 기반 응답을 반환한다.
// LLM이 필요하다고 판단하면 자체적으로 검색을 수행한다.
func (c *Client) StructuredWithSearch(ctx context.Context, req Request, schema map[string]any) (StructuredResult, error) {
	parsed, model, searchQueries, err := c.structuredInternal(ctx, req, schema, true)
	return StructuredResult{
		Payload:       parsed,
		Model:         model,
		SearchQueries: searchQueries,
	}, err
}

// ConsensusResult 는 합의 로직 결과다.
type ConsensusResult struct {
	Payload        map[string]any
	Model          string
	SearchQueries  []string
	ConsensusField string // 합의 기준 필드
	ConsensusValue string // 합의된 값
	ConsensusCount int    // 동의한 호출 수
	TotalCalls     int    // 총 호출 수
}

// StructuredWithConsensus 는 동일 요청을 N번 병렬 호출하여 합의된 결과를 반환한다.
// fieldName 필드의 값을 기준으로 다수결을 수행한다.
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

	// Extract search queries from grounding metadata
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
	client, err := c.selectClient(ctx)
	if err != nil {
		return nil, "", err
	}

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
	response, err := client.Models.GenerateContent(ctx, model, contents, genConfig)
	if err != nil {
		return nil, model, fmt.Errorf("generate content: %w", err)
	}
	return response, model, nil
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
	client, err := genai.NewClient(context.WithoutCancel(ctx), &genai.ClientConfig{
		APIKey:  key,
		Backend: genai.BackendGeminiAPI,
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
