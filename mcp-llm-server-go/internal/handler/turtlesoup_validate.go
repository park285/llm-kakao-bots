package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	turtlesoupuc "github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/usecase/turtlesoup"
)

func (h *TurtleSoupHandler) handleValidate(c *gin.Context) {
	var req TurtleSoupValidateRequest
	if !bindJSON(c, &req) {
		return
	}

	result, err := h.usecase.ValidateSolution(c.Request.Context(), turtlesoupuc.ValidateRequest{
		Solution:     req.Solution,
		PlayerAnswer: req.PlayerAnswer,
	})
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, TurtleSoupValidateResponse{
		Result:  result.Result,
		RawText: result.RawText,
	})
}
