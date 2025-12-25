package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/gemini"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/toon"
)

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
