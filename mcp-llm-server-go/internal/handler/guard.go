package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/guard"
)

// GuardRequest 는 가드 검사 요청이다.
type GuardRequest struct {
	InputText string `json:"input_text" binding:"required"`
}

// GuardResponse 는 가드 평가 응답이다.
type GuardResponse struct {
	Score     float64       `json:"score"`
	Malicious bool          `json:"malicious"`
	Threshold float64       `json:"threshold"`
	Hits      []guard.Match `json:"hits"`
}

// GuardHandler 는 가드 API 핸들러다.
type GuardHandler struct {
	guard *guard.InjectionGuard
}

// NewGuardHandler 는 가드 핸들러를 생성한다.
func NewGuardHandler(guard *guard.InjectionGuard) *GuardHandler {
	return &GuardHandler{guard: guard}
}

// RegisterRoutes 는 가드 라우트를 등록한다.
func (h *GuardHandler) RegisterRoutes(router *gin.Engine) {
	group := router.Group("/api/guard")
	group.POST("/evaluations", h.handleEvaluate)
	group.POST("/checks", h.handleCheck)
}

func (h *GuardHandler) handleEvaluate(c *gin.Context) {
	var req GuardRequest
	if !bindJSON(c, &req) {
		return
	}

	evaluation := h.guard.Evaluate(req.InputText)
	c.JSON(http.StatusOK, GuardResponse{
		Score:     evaluation.Score,
		Malicious: evaluation.Malicious(),
		Threshold: evaluation.Threshold,
		Hits:      evaluation.Hits,
	})
}

func (h *GuardHandler) handleCheck(c *gin.Context) {
	var req GuardRequest
	if !bindJSON(c, &req) {
		return
	}

	malicious := h.guard.IsMalicious(req.InputText)
	c.JSON(http.StatusOK, gin.H{"malicious": malicious})
}
