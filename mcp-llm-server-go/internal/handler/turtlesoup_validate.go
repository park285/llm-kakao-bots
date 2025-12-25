package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/domain/turtlesoup"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/gemini"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/handler/shared"
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

	payload, _, err := h.client.Structured(c.Request.Context(), gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "verify",
	}, turtlesoup.ValidateSchema())
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	rawValue, parseErr := shared.ParseStringField(payload, "result")
	result := string(turtlesoup.ValidationNo)
	if parseErr == nil && rawValue != "" {
		result = rawValue
	}

	c.JSON(http.StatusOK, TurtleSoupValidateResponse{
		Result:  result,
		RawText: rawValue,
	})
}
