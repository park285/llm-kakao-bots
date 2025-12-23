package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/domain/twentyq"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/gemini"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/guard"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/handler/shared"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/httperror"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/llm"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/session"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/toon"
)

const (
	twentyqSafetyBlockMessage   = shared.MsgSafetyBlock
	twentyqInvalidQuestionScale = shared.MsgInvalidQuestion
)

// TwentyQHintsRequest 는 힌트 요청 본문이다.
type TwentyQHintsRequest struct {
	Target   string         `json:"target" binding:"required"`
	Category string         `json:"category" binding:"required"`
	Details  map[string]any `json:"details"`
}

// TwentyQHintsResponse 는 힌트 응답 본문이다.
type TwentyQHintsResponse struct {
	Hints            []string `json:"hints"`
	ThoughtSignature *string  `json:"thought_signature"`
}

// TwentyQAnswerRequest 는 정답 요청 본문이다.
type TwentyQAnswerRequest struct {
	SessionID *string        `json:"session_id"`
	ChatID    *string        `json:"chat_id"`
	Namespace *string        `json:"namespace"`
	Target    string         `json:"target" binding:"required"`
	Category  string         `json:"category" binding:"required"`
	Question  string         `json:"question" binding:"required"`
	Details   map[string]any `json:"details"`
}

// TwentyQAnswerResponse 는 정답 응답 본문이다.
type TwentyQAnswerResponse struct {
	Scale            *string `json:"scale"`
	RawText          string  `json:"raw_text"`
	ThoughtSignature *string `json:"thought_signature"`
}

// TwentyQVerifyRequest 는 정답 검증 요청 본문이다.
type TwentyQVerifyRequest struct {
	Target string `json:"target" binding:"required"`
	Guess  string `json:"guess" binding:"required"`
}

// TwentyQVerifyResponse 는 정답 검증 응답 본문이다.
type TwentyQVerifyResponse struct {
	Result  *string `json:"result"`
	RawText string  `json:"raw_text"`
}

// TwentyQNormalizeRequest 는 질문 정규화 요청 본문이다.
type TwentyQNormalizeRequest struct {
	Question string `json:"question" binding:"required"`
}

// TwentyQNormalizeResponse 는 질문 정규화 응답 본문이다.
type TwentyQNormalizeResponse struct {
	Normalized string `json:"normalized"`
	Original   string `json:"original"`
}

// TwentyQSynonymRequest 는 유사어 확인 요청 본문이다.
type TwentyQSynonymRequest struct {
	Target string `json:"target" binding:"required"`
	Guess  string `json:"guess" binding:"required"`
}

// TwentyQSynonymResponse 는 유사어 확인 응답 본문이다.
type TwentyQSynonymResponse struct {
	Result  *string `json:"result"`
	RawText string  `json:"raw_text"`
}

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

func (h *TwentyQHandler) handleHints(c *gin.Context) {
	var req TwentyQHintsRequest
	if !bindJSON(c, &req) {
		return
	}

	system, err := h.prompts.HintsSystem(req.Category)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	secretToon := toon.EncodeSecret(req.Target, req.Category, nil)
	userContent, err := h.prompts.HintsUser(secretToon)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	detailsJSON, err := shared.SerializeDetails(req.Details)
	if err != nil {
		writeError(c, httperror.NewInvalidInput("details must be a JSON object"))
		return
	}
	err = h.ensureSafeDetails(detailsJSON)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}
	if detailsJSON != "" {
		userContent = userContent + "\n\n[추가 정보(JSON)]\n" + detailsJSON
	}

	payload, _, err := h.client.Structured(c.Request.Context(), gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "hints",
	}, twentyq.HintsSchema())
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	hints, err := shared.ParseStringSlice(payload, "hints")
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, TwentyQHintsResponse{
		Hints:            hints,
		ThoughtSignature: nil,
	})
}

func (h *TwentyQHandler) handleAnswer(c *gin.Context) {
	var req TwentyQAnswerRequest
	if !bindJSON(c, &req) {
		return
	}

	if err := h.guard.EnsureSafe(req.Question); err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	detailsJSON, err := shared.SerializeDetails(req.Details)
	if err != nil {
		writeError(c, httperror.NewInvalidInput("details must be a JSON object"))
		return
	}
	err = h.ensureSafeDetails(detailsJSON)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	sessionID, historyContext, historyCount, err := h.resolveAnswerSession(c.Request.Context(), req)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	h.logAnswerRequest(sessionID, historyCount, req.Question)

	system, userContent, err := h.buildAnswerPrompt(req, historyContext, detailsJSON)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	rawText, scaleText, err := h.getAnswerText(c, system, userContent)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}
	if rawText == "" {
		scale := twentyqInvalidQuestionScale
		c.JSON(http.StatusOK, TwentyQAnswerResponse{
			Scale:            &scale,
			RawText:          twentyqSafetyBlockMessage,
			ThoughtSignature: nil,
		})
		return
	}

	if err := h.appendAnswerHistory(c.Request.Context(), sessionID, req.Question, scaleText); err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	var scale *string
	if scaleText != "" {
		scale = &scaleText
	}
	c.JSON(http.StatusOK, TwentyQAnswerResponse{
		Scale:            scale,
		RawText:          rawText,
		ThoughtSignature: nil,
	})
}

func (h *TwentyQHandler) handleVerify(c *gin.Context) {
	var req TwentyQVerifyRequest
	if !bindJSON(c, &req) {
		return
	}

	if err := h.guard.EnsureSafe(req.Guess); err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	system, err := h.prompts.VerifySystem()
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}
	userContent, err := h.prompts.VerifyUser(req.Target, req.Guess)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	payload, _, err := h.client.Structured(c.Request.Context(), gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "verify",
	}, twentyq.VerifySchema())
	if err == nil {
		rawValue, parseErr := shared.ParseStringField(payload, "result")
		if parseErr == nil {
			resultName, ok := twentyq.VerifyResultName(rawValue)
			var result *string
			if ok {
				result = &resultName
			}
			c.JSON(http.StatusOK, TwentyQVerifyResponse{
				Result:  result,
				RawText: rawValue,
			})
			return
		}
		h.logError(parseErr)
	} else {
		h.logError(err)
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
	rawText = strings.TrimSpace(rawText)
	parsed := parseVerifyFallback(rawText)
	c.JSON(http.StatusOK, TwentyQVerifyResponse{
		Result:  parsed,
		RawText: rawText,
	})
}

func (h *TwentyQHandler) handleNormalize(c *gin.Context) {
	var req TwentyQNormalizeRequest
	if !bindJSON(c, &req) {
		return
	}

	if err := h.guard.EnsureSafe(req.Question); err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	system, err := h.prompts.NormalizeSystem()
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}
	userContent, err := h.prompts.NormalizeUser(req.Question)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	normalized := req.Question
	payload, _, err := h.client.Structured(c.Request.Context(), gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
	}, twentyq.NormalizeSchema())
	if err == nil {
		if rawValue, parseErr := shared.ParseStringField(payload, "normalized"); parseErr == nil {
			normalized = rawValue
		} else {
			h.logError(parseErr)
		}
	} else {
		h.logError(err)
	}

	c.JSON(http.StatusOK, TwentyQNormalizeResponse{
		Normalized: normalized,
		Original:   req.Question,
	})
}

func (h *TwentyQHandler) handleSynonym(c *gin.Context) {
	var req TwentyQSynonymRequest
	if !bindJSON(c, &req) {
		return
	}

	if err := h.guard.EnsureSafe(req.Guess); err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	system, err := h.prompts.SynonymSystem()
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}
	userContent, err := h.prompts.SynonymUser(req.Target, req.Guess)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	payload, _, err := h.client.Structured(c.Request.Context(), gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
	}, twentyq.SynonymSchema())
	if err == nil {
		rawValue, parseErr := shared.ParseStringField(payload, "result")
		if parseErr == nil {
			resultName, ok := twentyq.SynonymResultName(rawValue)
			var result *string
			if ok {
				result = &resultName
			}
			c.JSON(http.StatusOK, TwentyQSynonymResponse{
				Result:  result,
				RawText: rawValue,
			})
			return
		}
		h.logError(parseErr)
	} else {
		h.logError(err)
	}

	rawText, _, err := h.client.Chat(c.Request.Context(), gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
	})
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}
	rawText = strings.TrimSpace(rawText)
	parsed := parseSynonymFallback(rawText)
	c.JSON(http.StatusOK, TwentyQSynonymResponse{
		Result:  parsed,
		RawText: rawText,
	})
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

func (h *TwentyQHandler) resolveAnswerSession(
	ctx context.Context,
	req TwentyQAnswerRequest,
) (string, string, int, error) {
	sessionID, derived := shared.ResolveSessionID(
		shared.ValueOrEmpty(req.SessionID),
		shared.ValueOrEmpty(req.ChatID),
		shared.ValueOrEmpty(req.Namespace),
		"twentyq",
	)
	if sessionID != "" && derived && req.SessionID == nil {
		now := time.Now()
		meta := session.Meta{
			ID:           sessionID,
			SystemPrompt: "",
			Model:        h.cfg.Gemini.DefaultModel,
			CreatedAt:    now,
			UpdatedAt:    now,
			MessageCount: 0,
		}
		if err := h.store.CreateSession(ctx, meta); err != nil {
			return "", "", 0, fmt.Errorf("create session: %w", err)
		}
	}

	if sessionID == "" {
		return "", "", 0, nil
	}

	history, err := h.store.GetHistory(ctx, sessionID)
	if err != nil {
		h.logError(err)
		return sessionID, "", 0, nil
	}
	historyCount := len(history)
	historyContext := shared.BuildRecentQAHistoryContext(
		history,
		fmt.Sprintf("[이전 질문/답변 기록 - 정답: %s]", req.Target),
		h.cfg.Session.HistoryMaxPairs,
	)
	return sessionID, historyContext, historyCount, nil
}

func (h *TwentyQHandler) buildAnswerPrompt(
	req TwentyQAnswerRequest,
	historyContext string,
	detailsJSON string,
) (string, string, error) {
	system, err := h.prompts.AnswerSystem()
	if err != nil {
		return "", "", fmt.Errorf("load answer system prompt: %w", err)
	}
	secretToon := toon.EncodeSecret(req.Target, req.Category, nil)
	userContent, err := h.prompts.AnswerUser(secretToon, req.Question, historyContext)
	if err != nil {
		return "", "", fmt.Errorf("format answer user prompt: %w", err)
	}
	if detailsJSON != "" {
		userContent = userContent + "\n\n[추가 정보(JSON)]\n" + detailsJSON
	}
	return system, userContent, nil
}

func (h *TwentyQHandler) appendAnswerHistory(
	ctx context.Context,
	sessionID string,
	question string,
	scaleText string,
) error {
	if sessionID == "" {
		return nil
	}
	historyScaleText := "UNKNOWN"
	if scaleText != "" {
		historyScaleText = scaleText
	}
	if err := h.store.AppendHistory(
		ctx,
		sessionID,
		llm.HistoryEntry{Role: "user", Content: "Q: " + question},
		llm.HistoryEntry{Role: "assistant", Content: "A: " + historyScaleText},
	); err != nil {
		return fmt.Errorf("append history: %w", err)
	}
	return nil
}

func (h *TwentyQHandler) logAnswerRequest(sessionID string, historyCount int, question string) {
	sessionLabel := sessionID
	if sessionLabel == "" {
		sessionLabel = "stateless"
	}
	questionCount := historyCount/2 + 1
	h.logger.Info(
		"twentyq_answer",
		"session", sessionLabel,
		"count", questionCount,
		"history_count", historyCount,
		"q", question,
	)
}

func (h *TwentyQHandler) getAnswerText(c *gin.Context, system string, userContent string) (string, string, error) {
	rawText, _, err := h.client.Chat(c.Request.Context(), gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "answer",
	})
	if err != nil {
		return "", "", fmt.Errorf("answer chat: %w", err)
	}
	rawText = strings.TrimSpace(rawText)
	if rawText == "" {
		return "", "", nil
	}

	scale, ok := twentyq.ParseAnswerScale(rawText)
	if !ok {
		retryPrompt := userContent + "\n\n반드시 다음 중 하나만 출력: 예 | 아마도 예 | 아마도 아니오 | 아니오"
		rawText, _, err = h.client.Chat(c.Request.Context(), gemini.Request{
			Prompt:       retryPrompt,
			SystemPrompt: system,
			Task:         "answer",
		})
		if err != nil {
			return "", "", fmt.Errorf("answer retry chat: %w", err)
		}
		rawText = strings.TrimSpace(rawText)
		if rawText == "" {
			return "", "", nil
		}
		scale, ok = twentyq.ParseAnswerScale(rawText)
	}

	scaleText := ""
	if ok {
		scaleText = string(scale)
	}
	return rawText, scaleText, nil
}

// parseVerifyFallback 는 검증 결과를 파싱한다 (fallback).
func parseVerifyFallback(rawText string) *string {
	rawUpper := strings.ToUpper(rawText)
	for _, candidate := range []string{"ACCEPT", "CLOSE", "REJECT"} {
		if strings.Contains(rawUpper, candidate) {
			result := candidate
			return &result
		}
	}
	for _, candidate := range []string{string(twentyq.VerifyAccept), string(twentyq.VerifyClose), string(twentyq.VerifyReject)} {
		if strings.Contains(rawText, candidate) {
			result, ok := twentyq.VerifyResultName(candidate)
			if ok {
				return &result
			}
		}
	}
	return nil
}

// parseSynonymFallback 는 유의어 확인 결과를 파싱한다 (fallback).
func parseSynonymFallback(rawText string) *string {
	rawUpper := strings.ToUpper(rawText)
	for _, candidate := range []string{"NOT_EQUIVALENT", "EQUIVALENT"} {
		if strings.Contains(rawUpper, candidate) {
			result := candidate
			return &result
		}
	}
	for _, candidate := range []string{string(twentyq.SynonymEquivalent), string(twentyq.SynonymNotEquivalent)} {
		if strings.Contains(rawText, candidate) {
			result, ok := twentyq.SynonymResultName(candidate)
			if ok {
				return &result
			}
		}
	}
	return nil
}

func (h *TwentyQHandler) logError(err error) {
	shared.LogError(h.logger, "twentyq", err)
}
