package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/handler/shared"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/httperror"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/middleware"
)

// SelectTopicRequest: 토픽 선택 요청입니다.
type SelectTopicRequest struct {
	Category           string   `json:"category"`
	BannedTopics       []string `json:"bannedTopics"`
	ExcludedCategories []string `json:"excludedCategories"`
}

// SelectTopicResponse: 토픽 선택 응답입니다.
type SelectTopicResponse struct {
	Name     string         `json:"name"`
	Category string         `json:"category"`
	Details  map[string]any `json:"details"`
}

// CategoriesResponse: 카테고리 목록 응답입니다.
type CategoriesResponse struct {
	Categories []string `json:"categories"`
}

func (h *TwentyQHandler) handleSelectTopic(c *gin.Context) {
	var req SelectTopicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.WriteError(c, httperror.NewValidationError(err))
		return
	}

	topic, err := h.usecase.SelectTopic(c.Request.Context(), middleware.GetRequestID(c), req.Category, req.BannedTopics, req.ExcludedCategories)
	if err != nil {
		h.logError(err)
		shared.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, SelectTopicResponse{
		Name:     topic.Name,
		Category: topic.Category,
		Details:  topic.Details,
	})
}

func (h *TwentyQHandler) handleCategories(c *gin.Context) {
	c.JSON(http.StatusOK, CategoriesResponse{
		Categories: h.usecase.Categories(),
	})
}
