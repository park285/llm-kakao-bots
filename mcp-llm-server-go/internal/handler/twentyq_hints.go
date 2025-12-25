package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/domain/twentyq"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/gemini"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/handler/shared"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/httperror"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/toon"
)

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
