package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/domain/twentyq"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/handler/shared"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/httperror"
)

// SelectTopicRequest 는 토픽 선택 요청이다.
type SelectTopicRequest struct {
	Category           string   `json:"category"`
	BannedTopics       []string `json:"bannedTopics"`
	ExcludedCategories []string `json:"excludedCategories"`
}

// SelectTopicResponse 는 토픽 선택 응답이다.
type SelectTopicResponse struct {
	Name     string         `json:"name"`
	Category string         `json:"category"`
	Details  map[string]any `json:"details"`
}

// CategoriesResponse 는 카테고리 목록 응답이다.
type CategoriesResponse struct {
	Categories []string `json:"categories"`
}

func (h *TwentyQHandler) handleSelectTopic(c *gin.Context) {
	var req SelectTopicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.WriteError(c, httperror.NewValidationError(err))
		return
	}

	topic, err := h.topicLoader.SelectTopic(req.Category, req.BannedTopics, req.ExcludedCategories)
	if err != nil {
		h.logError(err)
		shared.WriteError(c, httperror.NewInternalError("topic selection failed"))
		return
	}

	h.logger.Info("twentyq_topic_selected",
		"category", topic.Category,
		"topic", topic.Name,
		"banned_count", len(req.BannedTopics),
		"excluded_categories", len(req.ExcludedCategories),
	)

	c.JSON(http.StatusOK, SelectTopicResponse{
		Name:     topic.Name,
		Category: topic.Category,
		Details:  topic.Details,
	})
}

func (h *TwentyQHandler) handleCategories(c *gin.Context) {
	c.JSON(http.StatusOK, CategoriesResponse{
		Categories: twentyq.AllCategories,
	})
}
