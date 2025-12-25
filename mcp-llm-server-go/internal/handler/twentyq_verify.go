package handler

import (
	"net/http"
	"strings"

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
