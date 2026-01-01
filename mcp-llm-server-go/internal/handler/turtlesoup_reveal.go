package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	turtlesoupuc "github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/usecase/turtlesoup"
)

func (h *TurtleSoupHandler) handleReveal(c *gin.Context) {
	var req TurtleSoupRevealRequest
	if !bindJSON(c, &req) {
		return
	}

	narrative, err := h.usecase.Reveal(c.Request.Context(), turtlesoupuc.RevealRequest{
		Scenario: req.Scenario,
		Solution: req.Solution,
	})
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, TurtleSoupRevealResponse{Narrative: narrative})
}
