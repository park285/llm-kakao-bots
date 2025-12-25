package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/domain/turtlesoup"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/gemini"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/guard"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/handler/shared"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/llm"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/session"
)

// LLMClient 는 LLM 호출 인터페이스다.
type LLMClient interface {
	Chat(ctx context.Context, req gemini.Request) (string, string, error)
	Structured(ctx context.Context, req gemini.Request, schema map[string]any) (map[string]any, string, error)
}

// TurtleSoupHandler 는 Turtle Soup API 핸들러다.
type TurtleSoupHandler struct {
	cfg     *config.Config
	client  LLMClient
	guard   *guard.InjectionGuard
	store   *session.Store
	prompts *turtlesoup.Prompts
	loader  *turtlesoup.PuzzleLoader
	logger  *slog.Logger
}

// NewTurtleSoupHandler 는 Turtle Soup 핸들러를 생성한다.
func NewTurtleSoupHandler(
	cfg *config.Config,
	client LLMClient,
	guard *guard.InjectionGuard,
	store *session.Store,
	prompts *turtlesoup.Prompts,
	loader *turtlesoup.PuzzleLoader,
	logger *slog.Logger,
) *TurtleSoupHandler {
	return &TurtleSoupHandler{
		cfg:     cfg,
		client:  client,
		guard:   guard,
		store:   store,
		prompts: prompts,
		loader:  loader,
		logger:  logger,
	}
}

// RegisterRoutes 는 Turtle Soup 라우트를 등록한다.
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

func (h *TurtleSoupHandler) resolveSession(
	ctx context.Context,
	sessionID *string,
	chatID *string,
	namespace *string,
) (string, string, int, []llm.HistoryEntry, error) {
	effectiveSessionID, derived := shared.ResolveSessionID(shared.ValueOrEmpty(sessionID), shared.ValueOrEmpty(chatID), shared.ValueOrEmpty(namespace), "turtle-soup")
	if effectiveSessionID != "" && derived && sessionID == nil {
		now := time.Now()
		meta := session.Meta{
			ID:           effectiveSessionID,
			SystemPrompt: "",
			Model:        h.cfg.Gemini.DefaultModel,
			CreatedAt:    now,
			UpdatedAt:    now,
			MessageCount: 0,
		}
		if err := h.store.CreateSession(ctx, meta); err != nil {
			return "", "", 0, nil, fmt.Errorf("create session: %w", err)
		}
	}

	if effectiveSessionID == "" {
		return "", "", 0, nil, nil
	}

	history, err := h.store.GetHistory(ctx, effectiveSessionID)
	if err != nil {
		h.logError(err)
		return effectiveSessionID, "", 0, nil, nil
	}

	pairs := countQAPairs(history)
	context := shared.BuildRecentQAHistoryContext(history, "[이전 질문/답변 기록]", h.cfg.Session.HistoryMaxPairs)
	return effectiveSessionID, context, pairs, history, nil
}

func (h *TurtleSoupHandler) appendHistory(ctx context.Context, sessionID string, question string, answer string) error {
	if sessionID == "" {
		return nil
	}
	if err := h.store.AppendHistory(
		ctx,
		sessionID,
		llm.HistoryEntry{Role: "user", Content: "Q: " + question},
		llm.HistoryEntry{Role: "assistant", Content: "A: " + answer},
	); err != nil {
		return fmt.Errorf("append history: %w", err)
	}
	return nil
}

func countQAPairs(history []llm.HistoryEntry) int {
	pairs := 0
	for i := 0; i+1 < len(history); i++ {
		q := strings.TrimSpace(history[i].Content)
		a := strings.TrimSpace(history[i+1].Content)
		if strings.HasPrefix(q, "Q:") && strings.HasPrefix(a, "A:") {
			pairs++
			i++
		}
	}
	return pairs
}

func buildTurtleHistoryItems(history []llm.HistoryEntry, currentQuestion string, currentAnswer string) []TurtleSoupHistoryItem {
	items := make([]TurtleSoupHistoryItem, 0)

	for i := 0; i+1 < len(history); i++ {
		q := strings.TrimSpace(history[i].Content)
		a := strings.TrimSpace(history[i+1].Content)
		if !strings.HasPrefix(q, "Q:") || !strings.HasPrefix(a, "A:") {
			continue
		}
		items = append(items, TurtleSoupHistoryItem{
			Question: strings.TrimSpace(strings.TrimPrefix(q, "Q:")),
			Answer:   strings.TrimSpace(strings.TrimPrefix(a, "A:")),
		})
		i++
	}

	items = append(items, TurtleSoupHistoryItem{Question: currentQuestion, Answer: currentAnswer})
	return items
}

func (h *TurtleSoupHandler) logError(err error) {
	shared.LogError(h.logger, "turtlesoup", err)
}
