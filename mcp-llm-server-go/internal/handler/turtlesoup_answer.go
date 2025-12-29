package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/domain/turtlesoup"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/gemini"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/handler/shared"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/toon"
)

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

	// 암시적 캐싱 최적화: historyContext (문자열) 대신 history (배열)만 사용
	sessionID, historyPairs, history, err := h.resolveSession(c.Request.Context(), req.SessionID, req.ChatID, req.Namespace)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	// Static Prefix: 퍼즐 정보를 System Prompt에 통합 (세션 내내 불변)
	puzzleToon := toon.EncodePuzzle(req.Scenario, req.Solution, "", nil)
	system, err := h.prompts.AnswerSystemWithPuzzle(puzzleToon)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	// Cache Miss 최소화: 현재 질문만 포함
	userContent, err := h.prompts.AnswerUser(req.Question)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	// 암시적 캐싱: Native History 배열 전달 (누적 Cache Hit)
	payload, _, err := h.client.Structured(c.Request.Context(), gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		History:      history, // Native History 사용
		Task:         "answer",
	}, turtlesoup.AnswerSchema())
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	rawAnswer, _ := shared.ParseStringField(payload, "answer")
	isImportant, _ := payload["important"].(bool)

	base := turtlesoup.AnswerType(rawAnswer)
	if rawAnswer == "" {
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
		RawText:       rawAnswer,
		QuestionCount: historyPairs + 1,
		History:       items,
	})
}
