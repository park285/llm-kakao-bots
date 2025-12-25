package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/domain/turtlesoup"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/gemini"
)

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
