package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/middleware"
	twentyquc "github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/usecase/twentyq"
)

func (h *TwentyQHandler) handleHints(c *gin.Context) {
	var req TwentyQHintsRequest
	if !bindJSON(c, &req) {
		return
	}

	hints, err := h.usecase.GenerateHints(c.Request.Context(), middleware.GetRequestID(c), twentyquc.HintsRequest{
		Target:   req.Target,
		Category: req.Category,
		Details:  req.Details,
	})
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
