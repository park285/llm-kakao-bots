package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/handler/shared"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/middleware"
	twentyquc "github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/usecase/twentyq"
)

const (
	twentyqSafetyBlockMessage   = shared.MsgSafetyBlock
	twentyqInvalidQuestionScale = shared.MsgInvalidQuestion
)

func (h *TwentyQHandler) handleAnswer(c *gin.Context) {
	var req TwentyQAnswerRequest
	if !bindJSON(c, &req) {
		return
	}

	result, err := h.usecase.AnswerQuestion(c.Request.Context(), middleware.GetRequestID(c), twentyquc.AnswerRequest{
		SessionID: req.SessionID,
		ChatID:    req.ChatID,
		Namespace: req.Namespace,
		Target:    req.Target,
		Category:  req.Category,
		Question:  req.Question,
		Details:   req.Details,
	})
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}
	if result.RawText == "" {
		scale := twentyqInvalidQuestionScale
		c.JSON(http.StatusOK, TwentyQAnswerResponse{
			Scale:            &scale,
			RawText:          twentyqSafetyBlockMessage,
			ThoughtSignature: nil,
		})
		return
	}

	var scale *string
	if result.ScaleText != "" {
		scale = &result.ScaleText
	}

	c.JSON(http.StatusOK, TwentyQAnswerResponse{
		Scale:            scale,
		RawText:          result.RawText,
		ThoughtSignature: nil,
	})
}
