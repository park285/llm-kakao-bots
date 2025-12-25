package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/domain/twentyq"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/gemini"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/handler/shared"
)

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
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	rawValue, parseErr := shared.ParseStringField(payload, "result")
	var result *string
	if parseErr == nil {
		resultName, ok := twentyq.VerifyResultName(rawValue)
		if ok {
			result = &resultName
		}
	}

	c.JSON(http.StatusOK, TwentyQVerifyResponse{
		Result:  result,
		RawText: rawValue,
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
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	rawValue, parseErr := shared.ParseStringField(payload, "result")
	var result *string
	if parseErr == nil {
		resultName, ok := twentyq.SynonymResultName(rawValue)
		if ok {
			result = &resultName
		}
	}

	c.JSON(http.StatusOK, TwentyQSynonymResponse{
		Result:  result,
		RawText: rawValue,
	})
}
