package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/domain/turtlesoup"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/gemini"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/handler/shared"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/httperror"
)

func (h *TurtleSoupHandler) handleGenerate(c *gin.Context) {
	var req TurtleSoupPuzzleGenerationRequest
	if !bindJSONAllowEmpty(c, &req) {
		return
	}

	category := strings.TrimSpace(shared.ValueOrEmpty(req.Category))
	if category == "" {
		category = "MYSTERY"
	}

	difficulty := 3
	if req.Difficulty != nil {
		difficulty = *req.Difficulty
	}
	if difficulty < 1 {
		difficulty = 1
	}
	if difficulty > 5 {
		difficulty = 5
	}

	theme := strings.TrimSpace(shared.ValueOrEmpty(req.Theme))
	if theme != "" {
		if err := h.guard.EnsureSafe(theme); err != nil {
			h.logError(err)
			writeError(c, err)
			return
		}
	}

	preset, err := h.loader.GetRandomPuzzleByDifficulty(difficulty)
	if err == nil {
		c.JSON(http.StatusOK, TurtleSoupPuzzleGenerationResponse{
			Title:      preset.Title,
			Scenario:   preset.Question,
			Solution:   preset.Answer,
			Category:   category,
			Difficulty: preset.Difficulty,
			Hints:      []string{},
		})
		return
	}

	puzzle, err := h.generatePuzzleLLM(c.Request.Context(), category, difficulty, theme)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, puzzle)
}

func (h *TurtleSoupHandler) handleRewrite(c *gin.Context) {
	var req TurtleSoupRewriteRequest
	if !bindJSON(c, &req) {
		return
	}

	system, err := h.prompts.RewriteSystem()
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}
	userContent, err := h.prompts.RewriteUser(req.Title, req.Scenario, req.Solution, req.Difficulty)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	scenario, solution, err := h.rewritePuzzle(c.Request.Context(), system, userContent)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, TurtleSoupRewriteResponse{
		Scenario:         scenario,
		Solution:         solution,
		OriginalScenario: req.Scenario,
		OriginalSolution: req.Solution,
	})
}

func (h *TurtleSoupHandler) handlePuzzles(c *gin.Context) {
	all := h.loader.All()
	c.JSON(http.StatusOK, map[string]any{
		"puzzles": all,
		"stats": map[string]any{
			"total":         len(all),
			"by_difficulty": h.loader.CountByDifficulty(),
		},
	})
}

func (h *TurtleSoupHandler) handleRandomPuzzle(c *gin.Context) {
	difficultyRaw := strings.TrimSpace(c.Query("difficulty"))
	if difficultyRaw == "" {
		puzzle, err := h.loader.GetRandomPuzzle()
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, puzzle)
		return
	}

	difficulty, err := strconv.Atoi(difficultyRaw)
	if err != nil {
		writeError(c, httperror.NewInvalidInput("difficulty must be an integer"))
		return
	}
	puzzle, err := h.loader.GetRandomPuzzleByDifficulty(difficulty)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, puzzle)
}

func (h *TurtleSoupHandler) handlePuzzleByID(c *gin.Context) {
	raw := strings.TrimSpace(c.Param("id"))
	id, err := strconv.Atoi(raw)
	if err != nil {
		writeError(c, httperror.NewInvalidInput("id must be an integer"))
		return
	}

	puzzle, ok := h.loader.GetPuzzleByID(id)
	if !ok {
		writeError(c, &httperror.Error{
			Code:    httperror.ErrorCodeInvalidInput,
			Status:  http.StatusNotFound,
			Type:    "NotFoundError",
			Message: fmt.Sprintf("puzzle %d not found", id),
			Details: map[string]any{"id": id},
		})
		return
	}
	c.JSON(http.StatusOK, puzzle)
}

func (h *TurtleSoupHandler) handleReloadPuzzles(c *gin.Context) {
	count, err := h.loader.Reload()
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, map[string]any{
		"success":       true,
		"count":         count,
		"by_difficulty": h.loader.CountByDifficulty(),
	})
}

func (h *TurtleSoupHandler) generatePuzzleLLM(ctx context.Context, category string, difficulty int, theme string) (TurtleSoupPuzzleGenerationResponse, error) {
	system, err := h.prompts.GenerateSystem()
	if err != nil {
		return TurtleSoupPuzzleGenerationResponse{}, fmt.Errorf("load puzzle system prompt: %w", err)
	}

	examples := h.loader.GetExamples(difficulty, 3)
	exampleLines := make([]string, 0, len(examples))
	for _, p := range examples {
		exampleLines = append(exampleLines, strings.Join([]string{
			"- 제목: " + p.Title,
			"  시나리오: " + p.Question,
			"  정답: " + p.Answer,
			"  난이도: " + strconv.Itoa(p.Difficulty),
		}, "\n"))
	}
	examplesBlock := strings.Join(exampleLines, "\n\n")

	userContent, err := h.prompts.GenerateUser(category, difficulty, theme, examplesBlock)
	if err != nil {
		return TurtleSoupPuzzleGenerationResponse{}, fmt.Errorf("format puzzle user prompt: %w", err)
	}

	payload, _, err := h.client.Structured(ctx, gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "hints",
	}, turtlesoup.PuzzleSchema())
	if err != nil {
		return TurtleSoupPuzzleGenerationResponse{}, fmt.Errorf("generate puzzle structured: %w", err)
	}

	title, err := shared.ParseStringField(payload, "title")
	if err != nil {
		return TurtleSoupPuzzleGenerationResponse{}, fmt.Errorf("parse title: %w", err)
	}
	scenario, err := shared.ParseStringField(payload, "scenario")
	if err != nil {
		return TurtleSoupPuzzleGenerationResponse{}, fmt.Errorf("parse scenario: %w", err)
	}
	solution, err := shared.ParseStringField(payload, "solution")
	if err != nil {
		return TurtleSoupPuzzleGenerationResponse{}, fmt.Errorf("parse solution: %w", err)
	}
	respCategory := strings.TrimSpace(valueOrEmptyString(payload, "category"))
	if respCategory == "" {
		respCategory = category
	}
	respDifficulty := difficulty
	if value, ok := payload["difficulty"]; ok {
		switch number := value.(type) {
		case float64:
			respDifficulty = int(number)
		case int:
			respDifficulty = number
		}
	}
	hints, err := shared.ParseStringSlice(payload, "hints")
	if err != nil {
		return TurtleSoupPuzzleGenerationResponse{}, fmt.Errorf("parse hints: %w", err)
	}

	return TurtleSoupPuzzleGenerationResponse{
		Title:      strings.TrimSpace(title),
		Scenario:   strings.TrimSpace(scenario),
		Solution:   strings.TrimSpace(solution),
		Category:   respCategory,
		Difficulty: respDifficulty,
		Hints:      hints,
	}, nil
}

func valueOrEmptyString(payload map[string]any, key string) string {
	raw, ok := payload[key]
	if !ok {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return value
}

func (h *TurtleSoupHandler) rewritePuzzle(ctx context.Context, system string, userContent string) (string, string, error) {
	payload, _, err := h.client.Structured(ctx, gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "answer",
	}, turtlesoup.RewriteSchema())
	if err != nil {
		return "", "", fmt.Errorf("rewrite structured: %w", err)
	}

	scenario, sErr := shared.ParseStringField(payload, "scenario")
	solution, aErr := shared.ParseStringField(payload, "solution")
	if sErr != nil || aErr != nil || strings.TrimSpace(scenario) == "" || strings.TrimSpace(solution) == "" {
		return "", "", httperror.NewInternalError("rewrite response invalid")
	}

	return strings.TrimSpace(scenario), strings.TrimSpace(solution), nil
}
