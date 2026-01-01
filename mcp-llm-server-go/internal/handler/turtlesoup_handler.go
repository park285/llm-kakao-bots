package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/domain/turtlesoup"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/gemini"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/guard"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/handler/shared"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/session"
	turtlesoupuc "github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/usecase/turtlesoup"
)

// TurtleSoupHandler: Turtle Soup API 핸들러입니다.
type TurtleSoupHandler struct {
	usecase *turtlesoupuc.Service
	loader  *turtlesoup.PuzzleLoader
	logger  *slog.Logger
}

// NewTurtleSoupHandler: Turtle Soup 핸들러를 생성합니다.
func NewTurtleSoupHandler(
	cfg *config.Config,
	client gemini.LLM,
	injectionGuard *guard.InjectionGuard,
	store *session.Store,
	prompts *turtlesoup.Prompts,
	loader *turtlesoup.PuzzleLoader,
	logger *slog.Logger,
) *TurtleSoupHandler {
	return &TurtleSoupHandler{
		usecase: turtlesoupuc.New(cfg, client, injectionGuard, store, prompts, loader, logger),
		loader:  loader,
		logger:  logger,
	}
}

// RegisterRoutes: Turtle Soup 라우트를 등록합니다.
func (h *TurtleSoupHandler) RegisterRoutes(router *gin.Engine) {
	group := router.Group("/api/turtle-soup")
	group.POST("/answers", h.handleAnswer)
	group.POST("/hints", h.handleHint)
	group.POST("/validations", h.handleValidate)
	group.POST("/reveals", h.handleReveal)
	group.POST("/puzzles", h.handleGenerate)
	group.POST("/rewrites", h.handleRewrite)
	group.GET("/puzzles", h.handlePuzzles)
	group.GET("/puzzles/random", h.handleRandomPuzzle)
	group.GET("/puzzles/:id", h.handlePuzzleByID)
	group.POST("/puzzles/reload", h.handleReloadPuzzles)
}

func (h *TurtleSoupHandler) logError(err error) {
	shared.LogError(h.logger, "turtlesoup", err)
}
