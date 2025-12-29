package handler

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/httperror"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/usage"
)

// DailyUsageResponse: 일자별 사용량 응답입니다.
type DailyUsageResponse struct {
	UsageDate       string `json:"usage_date"`
	InputTokens     int64  `json:"input_tokens"`
	OutputTokens    int64  `json:"output_tokens"`
	TotalTokens     int64  `json:"total_tokens"`
	ReasoningTokens int64  `json:"reasoning_tokens"`
	RequestCount    int64  `json:"request_count"`
	Model           string `json:"model"`
}

// UsageListResponse: 사용량 목록 응답입니다.
type UsageListResponse struct {
	Usages            []DailyUsageResponse `json:"usages"`
	TotalInputTokens  int64                `json:"total_input_tokens"`
	TotalOutputTokens int64                `json:"total_output_tokens"`
	TotalTokens       int64                `json:"total_tokens"`
	TotalRequestCount int64                `json:"total_request_count"`
	Model             string               `json:"model"`
}

// UsageHandler: 사용량 API 핸들러입니다.
type UsageHandler struct {
	cfg    *config.Config
	repo   *usage.Repository
	logger *slog.Logger
}

// NewUsageHandler: 사용량 핸들러를 생성합니다.
func NewUsageHandler(cfg *config.Config, repo *usage.Repository, logger *slog.Logger) *UsageHandler {
	return &UsageHandler{
		cfg:    cfg,
		repo:   repo,
		logger: logger,
	}
}

// RegisterRoutes: 사용량 라우트를 등록합니다.
func (h *UsageHandler) RegisterRoutes(router *gin.Engine) {
	group := router.Group("/api/usage")
	group.GET("/daily", h.handleDaily)
	group.GET("/recent", h.handleRecent)
	group.GET("/total", h.handleTotal)
}

func (h *UsageHandler) handleDaily(c *gin.Context) {
	usageRow, err := h.repo.GetDailyUsage(c.Request.Context(), time.Time{})
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, h.buildDailyResponse(usageRow))
}

func (h *UsageHandler) handleRecent(c *gin.Context) {
	days, ok := parseDays(c, 7)
	if !ok {
		return
	}

	usages, err := h.repo.GetRecentUsage(c.Request.Context(), days)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, h.buildUsageListResponse(usages))
}

func (h *UsageHandler) handleTotal(c *gin.Context) {
	days, ok := parseDays(c, 30)
	if !ok {
		return
	}

	usageRow, err := h.repo.GetTotalUsage(c.Request.Context(), days)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, UsageResponse{
		InputTokens:     usageRow.InputTokens,
		OutputTokens:    usageRow.OutputTokens,
		TotalTokens:     usageRow.TotalTokens(),
		ReasoningTokens: usageRow.ReasoningTokens,
		Model:           h.cfg.Gemini.DefaultModel,
	})
}

func (h *UsageHandler) buildDailyResponse(usageRow *usage.DailyUsage) DailyUsageResponse {
	model := h.cfg.Gemini.DefaultModel
	if usageRow == nil {
		return DailyUsageResponse{
			UsageDate:       time.Now().Format("2006-01-02"),
			InputTokens:     0,
			OutputTokens:    0,
			TotalTokens:     0,
			ReasoningTokens: 0,
			RequestCount:    0,
			Model:           model,
		}
	}

	return DailyUsageResponse{
		UsageDate:       usageRow.UsageDate.Format("2006-01-02"),
		InputTokens:     usageRow.InputTokens,
		OutputTokens:    usageRow.OutputTokens,
		TotalTokens:     usageRow.TotalTokens(),
		ReasoningTokens: usageRow.ReasoningTokens,
		RequestCount:    usageRow.RequestCount,
		Model:           model,
	}
}

func (h *UsageHandler) buildUsageListResponse(usages []usage.DailyUsage) UsageListResponse {
	model := h.cfg.Gemini.DefaultModel
	response := UsageListResponse{
		Usages:            make([]DailyUsageResponse, 0, len(usages)),
		TotalInputTokens:  0,
		TotalOutputTokens: 0,
		TotalTokens:       0,
		TotalRequestCount: 0,
		Model:             model,
	}

	for _, row := range usages {
		response.Usages = append(response.Usages, DailyUsageResponse{
			UsageDate:       row.UsageDate.Format("2006-01-02"),
			InputTokens:     row.InputTokens,
			OutputTokens:    row.OutputTokens,
			TotalTokens:     row.TotalTokens(),
			ReasoningTokens: row.ReasoningTokens,
			RequestCount:    row.RequestCount,
			Model:           model,
		})
		response.TotalInputTokens += row.InputTokens
		response.TotalOutputTokens += row.OutputTokens
		response.TotalTokens += row.TotalTokens()
		response.TotalRequestCount += row.RequestCount
	}

	return response
}

func parseDays(c *gin.Context, defaultDays int) (int, bool) {
	raw := c.Query("days")
	if raw == "" {
		return defaultDays, true
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		writeError(c, httperror.NewInvalidInput("days must be a positive integer"))
		return 0, false
	}
	return parsed, true
}

func (h *UsageHandler) logError(err error) {
	if err == nil {
		return
	}
	h.logger.Warn("usage_request_failed", "err", err)
}
