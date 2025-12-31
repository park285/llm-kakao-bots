package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/middleware"
)

func (h *TwentyQHandler) handleVerify(c *gin.Context) {
	var req TwentyQVerifyRequest
	if !bindJSON(c, &req) {
		return
	}

	result, err := h.usecase.VerifyGuess(c.Request.Context(), middleware.GetRequestID(c), req.Target, req.Guess)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, TwentyQVerifyResponse{
		Result:  result.Result,
		RawText: result.RawText,
	})
}

func (h *TwentyQHandler) handleNormalize(c *gin.Context) {
	var req TwentyQNormalizeRequest
	if !bindJSON(c, &req) {
		return
	}

	result, err := h.usecase.NormalizeQuestion(c.Request.Context(), middleware.GetRequestID(c), req.Question)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, TwentyQNormalizeResponse{
		Normalized: result.Normalized,
		Original:   result.Original,
	})
}

func (h *TwentyQHandler) handleSynonym(c *gin.Context) {
	var req TwentyQSynonymRequest
	if !bindJSON(c, &req) {
		return
	}

	result, err := h.usecase.CheckSynonym(c.Request.Context(), middleware.GetRequestID(c), req.Target, req.Guess)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, TwentyQSynonymResponse{
		Result:  result.Result,
		RawText: result.RawText,
	})
}
