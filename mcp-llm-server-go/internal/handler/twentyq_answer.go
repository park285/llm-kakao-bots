package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/domain/twentyq"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/gemini"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/handler/shared"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/httperror"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/llm"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/middleware"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/prompt"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/session"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/toon"
)

const (
	twentyqSafetyBlockMessage   = shared.MsgSafetyBlock
	twentyqInvalidQuestionScale = shared.MsgInvalidQuestion
)

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

	// 암시적 캐싱 최적화: historyContext (문자열) 대신 history (배열) 사용
	sessionID, historyCount, history, err := h.resolveAnswerSession(c.Request.Context(), req)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	h.logAnswerRequest(sessionID, historyCount, req.Question)

	// Static Prefix: Secret 정보를 System Prompt에 통합
	system, userContent, err := h.buildAnswerPrompt(req, detailsJSON)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	// 암시적 캐싱: Native History 배열 전달
	rawText, scaleText, err := h.getAnswerText(c, system, userContent, history)
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

	// 정책 위반 질문은 히스토리에 추가하지 않음
	if scaleText != string(twentyq.AnswerPolicyViolation) {
		if err := h.appendAnswerHistory(c.Request.Context(), sessionID, req.Question, scaleText); err != nil {
			h.logError(err)
			writeError(c, err)
			return
		}
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

// resolveAnswerSession: 세션 정보를 조회하고 히스토리를 반환합니다.
// 암시적 캐싱 최적화: 문자열 변환 없이 Native History 배열만 반환합니다.
func (h *TwentyQHandler) resolveAnswerSession(
	ctx context.Context,
	req TwentyQAnswerRequest,
) (string, int, []llm.HistoryEntry, error) {
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
			return "", 0, nil, fmt.Errorf("create session: %w", err)
		}
	}

	if sessionID == "" {
		return "", 0, []llm.HistoryEntry{}, nil // Cold Start: 빈 슬라이스 반환
	}

	history, err := h.store.GetHistory(ctx, sessionID)
	if err != nil {
		h.logError(err)
		return sessionID, 0, []llm.HistoryEntry{}, nil // 에러 시 빈 슬라이스
	}
	historyCount := len(history)
	return sessionID, historyCount, history, nil
}

// buildAnswerPrompt: 답변 프롬프트를 구성합니다.
// 암시적 캐싱 최적화: Secret을 System Prompt에 통합하고, 현재 질문만 User Prompt에 포함합니다.
func (h *TwentyQHandler) buildAnswerPrompt(
	req TwentyQAnswerRequest,
	detailsJSON string,
) (string, string, error) {
	// Static Prefix: Secret 정보를 System Prompt에 통합 (세션 내내 불변)
	secretToon := toon.EncodeSecret(req.Target, req.Category, nil)
	system, err := h.prompts.AnswerSystemWithSecret(secretToon)
	if err != nil {
		return "", "", fmt.Errorf("load answer system prompt: %w", err)
	}

	// Cache Miss 최소화: 현재 질문만 포함
	userContent, err := h.prompts.AnswerUser(req.Question)
	if err != nil {
		return "", "", fmt.Errorf("format answer user prompt: %w", err)
	}
	if detailsJSON != "" {
		userContent = userContent + "\n\n[추가 정보(JSON)]\n" + prompt.WrapXML("details_json", detailsJSON)
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

// getAnswerText: LLM 응답을 가져옵니다.
// 암시적 캐싱 최적화: Native History 배열을 전달하여 누적 Cache Hit를 확보합니다.
func (h *TwentyQHandler) getAnswerText(c *gin.Context, system string, userContent string, history []llm.HistoryEntry) (string, string, error) {
	result, err := h.client.StructuredWithSearch(c.Request.Context(), gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		History:      history, // Native History 사용
		Task:         "answer",
	}, twentyq.AnswerSchema())
	if err != nil {
		return "", "", fmt.Errorf("answer structured: %w", err)
	}

	requestID := middleware.GetRequestID(c)

	// Log Chain of Thought reasoning
	if reasoning, ok := result.Payload["reasoning"].(string); ok && reasoning != "" {
		h.logger.Info("twentyq_cot", "request_id", requestID, "reasoning", reasoning)
	}

	// Log Google Search usage if any
	if len(result.SearchQueries) > 0 {
		h.logger.Info("twentyq_search", "request_id", requestID, "queries", result.SearchQueries)
	}

	rawValue, ok := result.Payload["answer"].(string)
	if !ok || rawValue == "" {
		return "", "", nil
	}

	scale, ok := twentyq.ParseAnswerScale(rawValue)
	scaleText := ""
	if ok {
		scaleText = string(scale)
	}
	return rawValue, scaleText, nil
}
