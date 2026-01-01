package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/middleware"
	turtlesoupuc "github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/usecase/turtlesoup"
)

func (h *TurtleSoupHandler) handleAnswer(c *gin.Context) {
	var req TurtleSoupAnswerRequest
	if !bindJSON(c, &req) {
		return
	}

	result, err := h.usecase.AnswerQuestion(c.Request.Context(), middleware.GetRequestID(c), turtlesoupuc.AnswerRequest{
		SessionID: req.SessionID,
		ChatID:    req.ChatID,
		Namespace: req.Namespace,
		Scenario:  req.Scenario,
		Solution:  req.Solution,
		Question:  req.Question,
	})
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	history := make([]TurtleSoupHistoryItem, 0, len(result.History))
	for _, item := range result.History {
		history = append(history, TurtleSoupHistoryItem{
			Question: item.Question,
			Answer:   item.Answer,
		})
	}

	c.JSON(http.StatusOK, TurtleSoupAnswerResponse{
		Answer:        result.Answer,
		RawText:       result.RawText,
		QuestionCount: result.QuestionCount,
		History:       history,
	})
}
