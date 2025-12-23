package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/domain/turtlesoup"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/gemini"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/guard"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/handler/shared"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/httperror"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/llm"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/session"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/toon"
)

// LLMClient 는 LLM 호출 인터페이스다.
type LLMClient interface {
	Chat(ctx context.Context, req gemini.Request) (string, string, error)
	Structured(ctx context.Context, req gemini.Request, schema map[string]any) (map[string]any, string, error)
}

// TurtleSoupAnswerRequest 는 정답 요청 본문이다.
type TurtleSoupAnswerRequest struct {
	SessionID *string `json:"session_id"`
	ChatID    *string `json:"chat_id"`
	Namespace *string `json:"namespace"`
	Scenario  string  `json:"scenario" binding:"required"`
	Solution  string  `json:"solution" binding:"required"`
	Question  string  `json:"question" binding:"required"`
}

// TurtleSoupHistoryItem 은 질문/답변 히스토리 항목이다.
type TurtleSoupHistoryItem struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// TurtleSoupAnswerResponse 는 정답 응답 본문이다.
type TurtleSoupAnswerResponse struct {
	Answer        string                  `json:"answer"`
	RawText       string                  `json:"raw_text"`
	QuestionCount int                     `json:"question_count"`
	History       []TurtleSoupHistoryItem `json:"history"`
}

// TurtleSoupHintRequest 는 힌트 요청 본문이다.
type TurtleSoupHintRequest struct {
	SessionID *string `json:"session_id"`
	ChatID    *string `json:"chat_id"`
	Namespace *string `json:"namespace"`
	Scenario  string  `json:"scenario" binding:"required"`
	Solution  string  `json:"solution" binding:"required"`
	Level     int     `json:"level" binding:"required,gte=1,lte=3"`
}

// TurtleSoupHintResponse 는 힌트 응답 본문이다.
type TurtleSoupHintResponse struct {
	Hint  string `json:"hint"`
	Level int    `json:"level"`
}

// TurtleSoupValidateRequest 는 검증 요청 본문이다.
type TurtleSoupValidateRequest struct {
	SessionID    *string `json:"session_id"`
	ChatID       *string `json:"chat_id"`
	Namespace    *string `json:"namespace"`
	Solution     string  `json:"solution" binding:"required"`
	PlayerAnswer string  `json:"player_answer" binding:"required"`
}

// TurtleSoupValidateResponse 는 검증 응답 본문이다.
type TurtleSoupValidateResponse struct {
	Result  string `json:"result"`
	RawText string `json:"raw_text"`
}

// TurtleSoupRevealRequest 는 해설 요청 본문이다.
type TurtleSoupRevealRequest struct {
	SessionID *string `json:"session_id"`
	ChatID    *string `json:"chat_id"`
	Namespace *string `json:"namespace"`
	Scenario  string  `json:"scenario" binding:"required"`
	Solution  string  `json:"solution" binding:"required"`
}

// TurtleSoupRevealResponse 는 해설 응답 본문이다.
type TurtleSoupRevealResponse struct {
	Narrative string `json:"narrative"`
}

// TurtleSoupPuzzleGenerationRequest 는 퍼즐 생성 요청 본문이다.
type TurtleSoupPuzzleGenerationRequest struct {
	Category   *string `json:"category,omitempty"`
	Difficulty *int    `json:"difficulty,omitempty"`
	Theme      *string `json:"theme,omitempty"`
}

// TurtleSoupPuzzleGenerationResponse 는 퍼즐 생성 응답 본문이다.
type TurtleSoupPuzzleGenerationResponse struct {
	Title      string   `json:"title"`
	Scenario   string   `json:"scenario"`
	Solution   string   `json:"solution"`
	Category   string   `json:"category"`
	Difficulty int      `json:"difficulty"`
	Hints      []string `json:"hints"`
}

// TurtleSoupRewriteRequest 는 리라이트 요청 본문이다.
type TurtleSoupRewriteRequest struct {
	Title      string `json:"title" binding:"required"`
	Scenario   string `json:"scenario" binding:"required"`
	Solution   string `json:"solution" binding:"required"`
	Difficulty int    `json:"difficulty" binding:"required,gte=1,lte=5"`
}

// TurtleSoupRewriteResponse 는 리라이트 응답 본문이다.
type TurtleSoupRewriteResponse struct {
	Scenario         string `json:"scenario"`
	Solution         string `json:"solution"`
	OriginalScenario string `json:"original_scenario"`
	OriginalSolution string `json:"original_solution"`
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

func (h *TurtleSoupHandler) handleAnswer(c *gin.Context) {
	var req TurtleSoupAnswerRequest
	if !bindJSON(c, &req) {
		return
	}

	if err := h.guard.EnsureSafe(req.Question); err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	sessionID, historyContext, historyPairs, history, err := h.resolveSession(c.Request.Context(), req.SessionID, req.ChatID, req.Namespace)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	puzzleToon := toon.EncodePuzzle(req.Scenario, req.Solution, "", nil)
	system, err := h.prompts.AnswerSystem()
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}
	userContent, err := h.prompts.AnswerUser(puzzleToon, req.Question, historyContext)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	rawText, _, err := h.client.Chat(c.Request.Context(), gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "answer",
	})
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}
	rawText = strings.TrimSpace(rawText)
	isImportant := turtlesoup.IsImportantAnswer(rawText)
	base, ok := turtlesoup.ParseBaseAnswer(rawText)
	if !ok && isImportant {
		base = turtlesoup.AnswerYes
		ok = true
	}
	if !ok {
		base = turtlesoup.AnswerCannotAnswer
	}
	answerText := turtlesoup.FormatAnswerText(base, isImportant)
	if answerText == "" {
		answerText = string(turtlesoup.AnswerCannotAnswer)
	}

	items := buildTurtleHistoryItems(history, req.Question, answerText)

	if err := h.appendHistory(c.Request.Context(), sessionID, req.Question, answerText); err != nil {
		h.logError(err)
	}

	c.JSON(http.StatusOK, TurtleSoupAnswerResponse{
		Answer:        answerText,
		RawText:       rawText,
		QuestionCount: historyPairs + 1,
		History:       items,
	})
}

func (h *TurtleSoupHandler) handleHint(c *gin.Context) {
	var req TurtleSoupHintRequest
	if !bindJSON(c, &req) {
		return
	}

	puzzleToon := toon.EncodePuzzle(req.Scenario, req.Solution, "", nil)
	system, err := h.prompts.HintSystem()
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}
	userContent, err := h.prompts.HintUser(puzzleToon, req.Level)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	hint, rawText, err := h.generateHint(c.Request.Context(), system, userContent)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, TurtleSoupHintResponse{
		Hint:  hint,
		Level: req.Level,
	})
	_ = rawText
}

func (h *TurtleSoupHandler) handleValidate(c *gin.Context) {
	var req TurtleSoupValidateRequest
	if !bindJSON(c, &req) {
		return
	}

	if err := h.guard.EnsureSafe(req.PlayerAnswer); err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	system, err := h.prompts.ValidateSystem()
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}
	userContent, err := h.prompts.ValidateUser(req.Solution, req.PlayerAnswer)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	rawText, _, err := h.client.Chat(c.Request.Context(), gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "verify",
	})
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	result, ok := turtlesoup.ParseValidationResult(rawText)
	if !ok {
		result = turtlesoup.ValidationNo
	}

	c.JSON(http.StatusOK, TurtleSoupValidateResponse{
		Result:  string(result),
		RawText: strings.TrimSpace(rawText),
	})
}

func (h *TurtleSoupHandler) handleReveal(c *gin.Context) {
	var req TurtleSoupRevealRequest
	if !bindJSON(c, &req) {
		return
	}

	puzzleToon := toon.EncodePuzzle(req.Scenario, req.Solution, "", nil)
	system, err := h.prompts.RevealSystem()
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}
	userContent, err := h.prompts.RevealUser(puzzleToon)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	narrative, _, err := h.client.Chat(c.Request.Context(), gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "reveal",
	})
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, TurtleSoupRevealResponse{Narrative: strings.TrimSpace(narrative)})
}

func (h *TurtleSoupHandler) handleGenerate(c *gin.Context) {
	var req TurtleSoupPuzzleGenerationRequest
	if !bindJSONAllowEmpty(c, &req) {
		return
	}

	category := strings.TrimSpace(shared.ValueOrEmpty(req.Category))
	if category == "" {
		category = "MYSTERY"
	}

	difficulty := 3
	if req.Difficulty != nil {
		difficulty = *req.Difficulty
	}
	if difficulty < 1 {
		difficulty = 1
	}
	if difficulty > 5 {
		difficulty = 5
	}

	theme := strings.TrimSpace(shared.ValueOrEmpty(req.Theme))
	if theme != "" {
		if err := h.guard.EnsureSafe(theme); err != nil {
			h.logError(err)
			writeError(c, err)
			return
		}
	}

	preset, err := h.loader.GetRandomPuzzleByDifficulty(difficulty)
	if err == nil {
		c.JSON(http.StatusOK, TurtleSoupPuzzleGenerationResponse{
			Title:      preset.Title,
			Scenario:   preset.Question,
			Solution:   preset.Answer,
			Category:   category,
			Difficulty: preset.Difficulty,
			Hints:      []string{},
		})
		return
	}

	puzzle, err := h.generatePuzzleLLM(c.Request.Context(), category, difficulty, theme)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, puzzle)
}

func (h *TurtleSoupHandler) handleRewrite(c *gin.Context) {
	var req TurtleSoupRewriteRequest
	if !bindJSON(c, &req) {
		return
	}

	system, err := h.prompts.RewriteSystem()
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}
	userContent, err := h.prompts.RewriteUser(req.Title, req.Scenario, req.Solution, req.Difficulty)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	scenario, solution, err := h.rewritePuzzle(c.Request.Context(), system, userContent)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, TurtleSoupRewriteResponse{
		Scenario:         scenario,
		Solution:         solution,
		OriginalScenario: req.Scenario,
		OriginalSolution: req.Solution,
	})
}

func (h *TurtleSoupHandler) handlePuzzles(c *gin.Context) {
	all := h.loader.All()
	c.JSON(http.StatusOK, map[string]any{
		"puzzles": all,
		"stats": map[string]any{
			"total":         len(all),
			"by_difficulty": h.loader.CountByDifficulty(),
		},
	})
}

func (h *TurtleSoupHandler) handleRandomPuzzle(c *gin.Context) {
	difficultyRaw := strings.TrimSpace(c.Query("difficulty"))
	if difficultyRaw == "" {
		puzzle, err := h.loader.GetRandomPuzzle()
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, puzzle)
		return
	}

	difficulty, err := strconv.Atoi(difficultyRaw)
	if err != nil {
		writeError(c, httperror.NewInvalidInput("difficulty must be an integer"))
		return
	}
	puzzle, err := h.loader.GetRandomPuzzleByDifficulty(difficulty)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, puzzle)
}

func (h *TurtleSoupHandler) handlePuzzleByID(c *gin.Context) {
	raw := strings.TrimSpace(c.Param("id"))
	id, err := strconv.Atoi(raw)
	if err != nil {
		writeError(c, httperror.NewInvalidInput("id must be an integer"))
		return
	}

	puzzle, ok := h.loader.GetPuzzleByID(id)
	if !ok {
		writeError(c, &httperror.Error{
			Code:    httperror.ErrorCodeInvalidInput,
			Status:  http.StatusNotFound,
			Type:    "NotFoundError",
			Message: fmt.Sprintf("puzzle %d not found", id),
			Details: map[string]any{"id": id},
		})
		return
	}
	c.JSON(http.StatusOK, puzzle)
}

func (h *TurtleSoupHandler) handleReloadPuzzles(c *gin.Context) {
	count, err := h.loader.Reload()
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, map[string]any{
		"success":       true,
		"count":         count,
		"by_difficulty": h.loader.CountByDifficulty(),
	})
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

func (h *TurtleSoupHandler) generateHint(ctx context.Context, system string, userContent string) (string, string, error) {
	payload, _, err := h.client.Structured(ctx, gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "hints",
	}, turtlesoup.HintSchema())
	if err == nil {
		hint, parseErr := shared.ParseStringField(payload, "hint")
		if parseErr == nil && strings.TrimSpace(hint) != "" {
			return strings.TrimSpace(hint), "", nil
		}
	}

	rawText, _, err := h.client.Chat(ctx, gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "hints",
	})
	if err != nil {
		return "", "", fmt.Errorf("hint chat: %w", err)
	}

	parsed := strings.TrimSpace(rawText)
	if strings.HasPrefix(parsed, "```") {
		parsed = strings.TrimSpace(strings.TrimPrefix(parsed, "```json"))
		parsed = strings.TrimSpace(strings.TrimPrefix(parsed, "```"))
		if idx := strings.Index(parsed, "```"); idx >= 0 {
			parsed = strings.TrimSpace(parsed[:idx])
		}
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(parsed), &decoded); err == nil {
		if hint, err := shared.ParseStringField(decoded, "hint"); err == nil && strings.TrimSpace(hint) != "" {
			return strings.TrimSpace(hint), rawText, nil
		}
	}

	return parsed, rawText, nil
}

func (h *TurtleSoupHandler) generatePuzzleLLM(ctx context.Context, category string, difficulty int, theme string) (TurtleSoupPuzzleGenerationResponse, error) {
	system, err := h.prompts.GenerateSystem()
	if err != nil {
		return TurtleSoupPuzzleGenerationResponse{}, fmt.Errorf("load puzzle system prompt: %w", err)
	}

	examples := h.loader.GetExamples(difficulty, 3)
	exampleLines := make([]string, 0, len(examples))
	for _, p := range examples {
		exampleLines = append(exampleLines, strings.Join([]string{
			"- 제목: " + p.Title,
			"  시나리오: " + p.Question,
			"  정답: " + p.Answer,
			"  난이도: " + strconv.Itoa(p.Difficulty),
		}, "\n"))
	}
	examplesBlock := strings.Join(exampleLines, "\n\n")

	userContent, err := h.prompts.GenerateUser(category, difficulty, theme, examplesBlock)
	if err != nil {
		return TurtleSoupPuzzleGenerationResponse{}, fmt.Errorf("format puzzle user prompt: %w", err)
	}

	payload, _, err := h.client.Structured(ctx, gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "hints",
	}, turtlesoup.PuzzleSchema())
	if err != nil {
		return TurtleSoupPuzzleGenerationResponse{}, fmt.Errorf("generate puzzle structured: %w", err)
	}

	title, err := shared.ParseStringField(payload, "title")
	if err != nil {
		return TurtleSoupPuzzleGenerationResponse{}, fmt.Errorf("parse title: %w", err)
	}
	scenario, err := shared.ParseStringField(payload, "scenario")
	if err != nil {
		return TurtleSoupPuzzleGenerationResponse{}, fmt.Errorf("parse scenario: %w", err)
	}
	solution, err := shared.ParseStringField(payload, "solution")
	if err != nil {
		return TurtleSoupPuzzleGenerationResponse{}, fmt.Errorf("parse solution: %w", err)
	}
	respCategory := strings.TrimSpace(valueOrEmptyString(payload, "category"))
	if respCategory == "" {
		respCategory = category
	}
	respDifficulty := difficulty
	if value, ok := payload["difficulty"]; ok {
		switch number := value.(type) {
		case float64:
			respDifficulty = int(number)
		case int:
			respDifficulty = number
		}
	}
	hints, err := shared.ParseStringSlice(payload, "hints")
	if err != nil {
		return TurtleSoupPuzzleGenerationResponse{}, fmt.Errorf("parse hints: %w", err)
	}

	return TurtleSoupPuzzleGenerationResponse{
		Title:      strings.TrimSpace(title),
		Scenario:   strings.TrimSpace(scenario),
		Solution:   strings.TrimSpace(solution),
		Category:   respCategory,
		Difficulty: respDifficulty,
		Hints:      hints,
	}, nil
}

func valueOrEmptyString(payload map[string]any, key string) string {
	raw, ok := payload[key]
	if !ok {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return value
}

func (h *TurtleSoupHandler) rewritePuzzle(ctx context.Context, system string, userContent string) (string, string, error) {
	payload, _, err := h.client.Structured(ctx, gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "answer",
	}, turtlesoup.RewriteSchema())
	if err == nil {
		scenario, sErr := shared.ParseStringField(payload, "scenario")
		solution, aErr := shared.ParseStringField(payload, "solution")
		if sErr == nil && aErr == nil && strings.TrimSpace(scenario) != "" && strings.TrimSpace(solution) != "" {
			return strings.TrimSpace(scenario), strings.TrimSpace(solution), nil
		}
	}

	rawText, _, err := h.client.Chat(ctx, gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "answer",
	})
	if err != nil {
		return "", "", fmt.Errorf("rewrite chat: %w", err)
	}

	parsed := strings.TrimSpace(rawText)
	if strings.HasPrefix(parsed, "```") {
		parsed = strings.TrimSpace(strings.TrimPrefix(parsed, "```json"))
		parsed = strings.TrimSpace(strings.TrimPrefix(parsed, "```"))
		if idx := strings.Index(parsed, "```"); idx >= 0 {
			parsed = strings.TrimSpace(parsed[:idx])
		}
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(parsed), &decoded); err == nil {
		scenario, sErr := shared.ParseStringField(decoded, "scenario")
		solution, aErr := shared.ParseStringField(decoded, "solution")
		if sErr == nil && aErr == nil && strings.TrimSpace(scenario) != "" && strings.TrimSpace(solution) != "" {
			return strings.TrimSpace(scenario), strings.TrimSpace(solution), nil
		}
	}

	return "", "", httperror.NewInternalError("rewrite response invalid")
}

func (h *TurtleSoupHandler) logError(err error) {
	shared.LogError(h.logger, "turtlesoup", err)
}
