package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/domain/twentyq"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/gemini"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/guard"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/handler/shared"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/session"
	twentyquc "github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/usecase/twentyq"
)

// TwentyQHandler: TwentyQ API 핸들러입니다.
type TwentyQHandler struct {
	cfg         *config.Config
	client      *gemini.Client
	guard       *guard.InjectionGuard
	store       *session.Store
	prompts     *twentyq.Prompts
	topicLoader *twentyq.TopicLoader
	usecase     *twentyquc.Service
	logger      *slog.Logger
}

// NewTwentyQHandler: TwentyQ 핸들러를 생성합니다.
func NewTwentyQHandler(
	cfg *config.Config,
	client *gemini.Client,
	injectionGuard *guard.InjectionGuard,
	store *session.Store,
	prompts *twentyq.Prompts,
	topicLoader *twentyq.TopicLoader,
	logger *slog.Logger,
) *TwentyQHandler {
	h := &TwentyQHandler{
		cfg:         cfg,
		client:      client,
		guard:       injectionGuard,
		store:       store,
		prompts:     prompts,
		topicLoader: topicLoader,
		logger:      logger,
	}
	h.usecase = twentyquc.New(cfg, client, injectionGuard, store, prompts, topicLoader, logger)
	return h
}

// RegisterRoutes: TwentyQ 라우트를 등록합니다.
func (h *TwentyQHandler) RegisterRoutes(router *gin.Engine) {
	group := router.Group("/api/twentyq")
	group.POST("/hints", h.handleHints)
	group.POST("/answers", h.handleAnswer)
	group.POST("/verifications", h.handleVerify)
	group.POST("/normalizations", h.handleNormalize)
	group.POST("/synonym-checks", h.handleSynonym)
	group.POST("/topics/select", h.handleSelectTopic)
	group.GET("/topics/categories", h.handleCategories)
}

func (h *TwentyQHandler) logError(err error) {
	shared.LogError(h.logger, "twentyq", err)
}
