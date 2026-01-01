package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	turtlesoupuc "github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/usecase/turtlesoup"
)

func (h *TurtleSoupHandler) handleHint(c *gin.Context) {
	var req TurtleSoupHintRequest
	if !bindJSON(c, &req) {
		return
	}

	hint, err := h.usecase.GenerateHint(c.Request.Context(), turtlesoupuc.HintRequest{
		Scenario: req.Scenario,
		Solution: req.Solution,
		Level:    req.Level,
	})
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, TurtleSoupHintResponse{
		Hint:  hint,
		Level: req.Level,
	})
}
