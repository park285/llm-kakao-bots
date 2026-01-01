package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/handler/shared"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/httperror"
	turtlesoupuc "github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/usecase/turtlesoup"
)

func (h *TurtleSoupHandler) handleGenerate(c *gin.Context) {
	var req TurtleSoupPuzzleGenerationRequest
	if !bindJSONAllowEmpty(c, &req) {
		return
	}

	var difficultyPtr *int
	if req.Difficulty != nil {
		d := *req.Difficulty
		difficultyPtr = &d
	}

	puzzle, err := h.usecase.GeneratePuzzle(c.Request.Context(), turtlesoupuc.GeneratePuzzleRequest{
		Category:   shared.ValueOrEmpty(req.Category),
		Difficulty: difficultyPtr,
		Theme:      shared.ValueOrEmpty(req.Theme),
	})
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, TurtleSoupPuzzleGenerationResponse{
		Title:      puzzle.Title,
		Scenario:   puzzle.Scenario,
		Solution:   puzzle.Solution,
		Category:   puzzle.Category,
		Difficulty: puzzle.Difficulty,
		Hints:      puzzle.Hints,
	})
}

func (h *TurtleSoupHandler) handleRewrite(c *gin.Context) {
	var req TurtleSoupRewriteRequest
	if !bindJSON(c, &req) {
		return
	}

	result, err := h.usecase.RewriteScenario(c.Request.Context(), turtlesoupuc.RewriteRequest{
		Title:      req.Title,
		Scenario:   req.Scenario,
		Solution:   req.Solution,
		Difficulty: req.Difficulty,
	})
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, TurtleSoupRewriteResponse{
		Scenario:         result.Scenario,
		Solution:         result.Solution,
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
