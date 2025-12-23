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
	mu            sync.Mutex
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

	usage := extractUsage(response)
	c.metrics.RecordSuccess(time.Since(start), usage)
	c.recordUsage(ctx, usage)
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
	usage := extractUsage(response)
	result := llm.ChatResult{
		Text:         text,
		Usage:        usage,
		Reasoning:    reasoning,
		HasReasoning: reasoning != "",
	}

	c.metrics.RecordSuccess(time.Since(start), usage)
	c.recordUsage(ctx, usage)
	return result, model, nil
}

// Structured 는 JSON 스키마 기반 응답을 반환한다.
func (c *Client) Structured(ctx context.Context, req Request, schema map[string]any) (map[string]any, string, error) {
	start := time.Now()
	response, model, err := c.generate(ctx, req, "application/json", schema)
	if err != nil {
		c.metrics.RecordError(time.Since(start))
		return nil, model, err
	}

	usage := extractUsage(response)
	c.metrics.RecordSuccess(time.Since(start), usage)
	c.recordUsage(ctx, usage)

	payload := response.Text()
	if strings.TrimSpace(payload) == "" {
		return nil, model, errors.New("empty structured response")
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
		return nil, model, fmt.Errorf("decode structured response: %w", err)
	}

	return parsed, model, nil
}

func (c *Client) recordUsage(ctx context.Context, usage llm.Usage) {
	if c.usageRecorder == nil {
		return
	}
	c.usageRecorder.Record(ctx, int64(usage.InputTokens), int64(usage.OutputTokens), int64(usage.ReasoningTokens))
}

func (c *Client) generate(
	ctx context.Context,
	req Request,
	responseMimeType string,
	responseSchema map[string]any,
) (*genai.GenerateContentResponse, string, error) {
	client, err := c.selectClient(ctx)
	if err != nil {
		return nil, "", err
	}

	model, err := c.resolveModel(req.Model, req.Task)
	if err != nil {
		return nil, model, err
	}

	config := c.buildGenerateConfig(req.SystemPrompt, req.Task, model, responseMimeType, responseSchema)
	contents := buildContents(req.Prompt, req.History)
	response, err := client.Models.GenerateContent(ctx, model, contents, config)
	if err != nil {
		return nil, model, fmt.Errorf("generate content: %w", err)
	}
	return response, model, nil
}

func (c *Client) selectClient(ctx context.Context) (*genai.Client, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.apiKeys) == 0 {
		return nil, ErrMissingAPIKey
	}

	key := c.apiKeys[c.apiKeyIdx%len(c.apiKeys)]
	c.apiKeyIdx++
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
	config := &genai.GenerateContentConfig{
		Temperature:     genai.Ptr(temperature),
		MaxOutputTokens: int32(c.cfg.Gemini.MaxOutputTokens),
	}

	if systemPrompt != "" {
		config.SystemInstruction = genai.NewContentFromText(systemPrompt, genai.RoleUser)
	}
	if responseMimeType != "" {
		config.ResponseMIMEType = responseMimeType
	}
	if responseSchema != nil {
		config.ResponseJsonSchema = responseSchema
	}

	if thinkingLevel, ok := normalizeThinkingLevel(c.cfg.Gemini.Thinking.Level(task)); ok {
		config.ThinkingConfig = &genai.ThinkingConfig{
			IncludeThoughts: true,
			ThinkingLevel:   thinkingLevel,
		}
	}

	return config
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
	usage := response.UsageMetadata
	return llm.Usage{
		InputTokens:     int(usage.PromptTokenCount),
		OutputTokens:    int(usage.CandidatesTokenCount) + int(usage.ThoughtsTokenCount),
		TotalTokens:     int(usage.TotalTokenCount),
		ReasoningTokens: int(usage.ThoughtsTokenCount),
	}
}

func isGemini3(model string) bool {
	return strings.Contains(strings.ToLower(model), "gemini-3")
}
