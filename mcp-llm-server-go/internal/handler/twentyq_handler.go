package handler

import (
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/domain/twentyq"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/gemini"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/guard"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/handler/shared"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/session"
)

// TwentyQHandler 는 TwentyQ API 핸들러다.
type TwentyQHandler struct {
	cfg     *config.Config
	client  *gemini.Client
	guard   *guard.InjectionGuard
	store   *session.Store
	prompts *twentyq.Prompts
	logger  *slog.Logger
}

// NewTwentyQHandler 는 TwentyQ 핸들러를 생성한다.
func NewTwentyQHandler(
	cfg *config.Config,
	client *gemini.Client,
	guard *guard.InjectionGuard,
	store *session.Store,
	prompts *twentyq.Prompts,
	logger *slog.Logger,
) *TwentyQHandler {
	return &TwentyQHandler{
		cfg:     cfg,
		client:  client,
		guard:   guard,
		store:   store,
		prompts: prompts,
		logger:  logger,
	}
}

// RegisterRoutes 는 TwentyQ 라우트를 등록한다.
func (h *TwentyQHandler) RegisterRoutes(router *gin.Engine) {
	group := router.Group("/api/twentyq")
	group.POST("/hints", h.handleHints)
	group.POST("/answers", h.handleAnswer)
	group.POST("/verifications", h.handleVerify)
	group.POST("/normalizations", h.handleNormalize)
	group.POST("/synonym-checks", h.handleSynonym)
}

func (h *TwentyQHandler) ensureSafeDetails(detailsJSON string) error {
	if detailsJSON == "" {
		return nil
	}
	if err := h.guard.EnsureSafe(detailsJSON); err != nil {
		return fmt.Errorf("guard details: %w", err)
	}
	return nil
}

func (h *TwentyQHandler) logError(err error) {
	shared.LogError(h.logger, "twentyq", err)
}
