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
	payload, _, err := h.client.Structured(c.Request.Context(), gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "answer",
	}, twentyq.AnswerSchema())
	if err != nil {
		return "", "", fmt.Errorf("answer structured: %w", err)
	}

	rawValue, ok := payload["answer"].(string)
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
