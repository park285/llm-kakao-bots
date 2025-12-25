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

	payload, _, err := h.client.Structured(c.Request.Context(), gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
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
