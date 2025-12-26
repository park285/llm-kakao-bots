package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/gemini"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/guard"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/httperror"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/llm"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/metrics"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/usage"
)

// ChatRequest 는 채팅 요청 본문이다.
type ChatRequest struct {
	Prompt       string             `json:"prompt" binding:"required"`
	SystemPrompt string             `json:"system_prompt"`
	History      []llm.HistoryEntry `json:"history"`
	Model        string             `json:"model"`
	Task         string             `json:"task"`
}

// ChatResponse 는 채팅 응답 본문이다.
type ChatResponse struct {
	Response string `json:"response"`
	Model    string `json:"model"`
}

// StructuredRequest 는 JSON 스키마 요청 본문이다.
type StructuredRequest struct {
	Prompt       string             `json:"prompt" binding:"required"`
	JSONSchema   map[string]any     `json:"json_schema" binding:"required"`
	SystemPrompt string             `json:"system_prompt"`
	History      []llm.HistoryEntry `json:"history"`
	Model        string             `json:"model"`
}

// ChatWithUsageResponse 는 사용량 포함 응답이다.
type ChatWithUsageResponse struct {
	Text         string    `json:"text"`
	Usage        llm.Usage `json:"usage"`
	Reasoning    string    `json:"reasoning"`
	HasReasoning bool      `json:"has_reasoning"`
}

// UsageResponse 는 사용량 응답이다.
type UsageResponse struct {
	InputTokens     int64  `json:"input_tokens"`
	OutputTokens    int64  `json:"output_tokens"`
	TotalTokens     int64  `json:"total_tokens"`
	ReasoningTokens int64  `json:"reasoning_tokens"`
	Model           string `json:"model"`
}

// LLMHandler 는 LLM API 핸들러다.
type LLMHandler struct {
	cfg       *config.Config
	client    *gemini.Client
	guard     *guard.InjectionGuard
	metrics   *metrics.Store
	usageRepo *usage.Repository
	logger    *slog.Logger
}

// NewLLMHandler 는 LLM 핸들러를 생성한다.
func NewLLMHandler(
	cfg *config.Config,
	client *gemini.Client,
	injectionGuard *guard.InjectionGuard,
	metricsStore *metrics.Store,
	usageRepo *usage.Repository,
	logger *slog.Logger,
) *LLMHandler {
	return &LLMHandler{
		cfg:       cfg,
		client:    client,
		guard:     injectionGuard,
		metrics:   metricsStore,
		usageRepo: usageRepo,
		logger:    logger,
	}
}

// RegisterRoutes 는 LLM 라우트를 등록한다.
func (h *LLMHandler) RegisterRoutes(router *gin.Engine) {
	group := router.Group("/api/llm")
	group.POST("/chat", h.handleChat)
	group.POST("/chat-with-usage", h.handleChatWithUsage)
	group.POST("/structured", h.handleStructured)
	group.GET("/usage", h.handleUsage)
	group.GET("/usage/total", h.handleUsageTotal)
	group.GET("/metrics", h.handleMetrics)
}

func (h *LLMHandler) handleChat(c *gin.Context) {
	var req ChatRequest
	if !bindJSON(c, &req) {
		return
	}

	if err := h.guard.EnsureSafe(req.Prompt); err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	result, model, err := h.client.Chat(c.Request.Context(), h.toGeminiRequest(req))
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, ChatResponse{Response: result, Model: model})
}

func (h *LLMHandler) handleChatWithUsage(c *gin.Context) {
	var req ChatRequest
	if !bindJSON(c, &req) {
		return
	}

	if err := h.guard.EnsureSafe(req.Prompt); err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	result, _, err := h.client.ChatWithUsage(c.Request.Context(), h.toGeminiRequest(req))
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	response := ChatWithUsageResponse{
		Text:         result.Text,
		Usage:        result.Usage,
		Reasoning:    result.Reasoning,
		HasReasoning: result.HasReasoning,
	}
	c.JSON(http.StatusOK, response)
}

func (h *LLMHandler) handleStructured(c *gin.Context) {
	var req StructuredRequest
	if !bindJSON(c, &req) {
		return
	}
	if req.JSONSchema == nil {
		writeError(c, httperror.NewMissingField("json_schema"))
		return
	}

	if err := h.guard.EnsureSafe(req.Prompt); err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	payload, _, err := h.client.Structured(c.Request.Context(), gemini.Request{
		Prompt:       req.Prompt,
		SystemPrompt: req.SystemPrompt,
		History:      req.History,
		Model:        req.Model,
	}, req.JSONSchema)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, payload)
}

func (h *LLMHandler) handleUsage(c *gin.Context) {
	usageTotals := h.metrics.UsageTotals()
	model := h.cfg.Gemini.DefaultModel

	c.JSON(http.StatusOK, UsageResponse{
		InputTokens:     int64(usageTotals.InputTokens),
		OutputTokens:    int64(usageTotals.OutputTokens),
		TotalTokens:     int64(usageTotals.TotalTokens),
		ReasoningTokens: int64(usageTotals.ReasoningTokens),
		Model:           model,
	})
}

func (h *LLMHandler) handleUsageTotal(c *gin.Context) {
	if h.usageRepo == nil {
		writeError(c, httperror.NewInternalError("usage repository not configured"))
		return
	}

	totalUsage, err := h.usageRepo.GetTotalUsage(c.Request.Context(), 30)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	model := h.cfg.Gemini.DefaultModel
	c.JSON(http.StatusOK, UsageResponse{
		InputTokens:     totalUsage.InputTokens,
		OutputTokens:    totalUsage.OutputTokens,
		TotalTokens:     totalUsage.TotalTokens(),
		ReasoningTokens: totalUsage.ReasoningTokens,
		Model:           model,
	})
}

func (h *LLMHandler) handleMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, h.metrics.Snapshot())
}

func (h *LLMHandler) toGeminiRequest(req ChatRequest) gemini.Request {
	return gemini.Request{
		Prompt:       req.Prompt,
		SystemPrompt: req.SystemPrompt,
		History:      req.History,
		Model:        req.Model,
		Task:         req.Task,
	}
}

func (h *LLMHandler) logError(err error) {
	if err == nil {
		return
	}
	h.logger.Warn("llm_request_failed", "err", err)
}
