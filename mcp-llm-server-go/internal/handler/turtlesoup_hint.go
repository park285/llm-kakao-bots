package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/domain/turtlesoup"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/gemini"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/handler/shared"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/toon"
)

func (h *TurtleSoupHandler) handleHint(c *gin.Context) {
	var req TurtleSoupHintRequest
	if !bindJSON(c, &req) {
		return
	}

	puzzleToon := toon.EncodePuzzle(req.Scenario, req.Solution, "", nil)
	system, err := h.prompts.HintSystem()
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}
	userContent, err := h.prompts.HintUser(puzzleToon, req.Level)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	hint, rawText, err := h.generateHint(c.Request.Context(), system, userContent)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, TurtleSoupHintResponse{
		Hint:  hint,
		Level: req.Level,
	})
	_ = rawText
}

func (h *TurtleSoupHandler) generateHint(ctx context.Context, system string, userContent string) (string, string, error) {
	payload, _, err := h.client.Structured(ctx, gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "hints",
	}, turtlesoup.HintSchema())
	if err == nil {
		hint, parseErr := shared.ParseStringField(payload, "hint")
		if parseErr == nil && strings.TrimSpace(hint) != "" {
			return strings.TrimSpace(hint), "", nil
		}
	}

	rawText, _, err := h.client.Chat(ctx, gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "hints",
	})
	if err != nil {
		return "", "", fmt.Errorf("hint chat: %w", err)
	}

	parsed := strings.TrimSpace(rawText)
	if strings.HasPrefix(parsed, "```") {
		parsed = strings.TrimSpace(strings.TrimPrefix(parsed, "```json"))
		parsed = strings.TrimSpace(strings.TrimPrefix(parsed, "```"))
		if idx := strings.Index(parsed, "```"); idx >= 0 {
			parsed = strings.TrimSpace(parsed[:idx])
		}
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(parsed), &decoded); err == nil {
		if hint, err := shared.ParseStringField(decoded, "hint"); err == nil && strings.TrimSpace(hint) != "" {
			return strings.TrimSpace(hint), rawText, nil
		}
	}

	return parsed, rawText, nil
}
