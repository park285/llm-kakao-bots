package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/guard"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/httperror"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/session"
)

// SessionHandler 세션 HTTP 핸들러
type SessionHandler struct {
	manager *session.Manager
	guard   *guard.InjectionGuard
	logger  *slog.Logger
}

// NewSessionHandler 세션 핸들러 생성
func NewSessionHandler(manager *session.Manager, injectionGuard *guard.InjectionGuard, logger *slog.Logger) *SessionHandler {
	return &SessionHandler{
		manager: manager,
		guard:   injectionGuard,
		logger:  logger,
	}
}

// RegisterRoutes 세션 라우트 등록
func (h *SessionHandler) RegisterRoutes(router *gin.Engine) {
	group := router.Group("/api/sessions")
	group.POST("", h.handleCreate)
	group.GET("/:id", h.handleGet)
	group.POST("/:id/messages", h.handleChat)
	group.DELETE("/:id", h.handleDelete)
}

// handleCreate 세션 생성
func (h *SessionHandler) handleCreate(c *gin.Context) {
	var req session.CreateSessionRequest
	if !bindJSONAllowEmpty(c, &req) {
		return
	}

	info, err := h.manager.Create(c.Request.Context(), req)
	if err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, info)
}

// handleGet 세션 정보 조회
func (h *SessionHandler) handleGet(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		writeError(c, httperror.NewMissingField("id"))
		return
	}

	info, err := h.manager.Get(c.Request.Context(), sessionID)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(c, httperror.NewSessionNotFound(sessionID))
			return
		}
		h.logError(err)
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, info)
}

// handleChat 세션 기반 채팅
func (h *SessionHandler) handleChat(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		writeError(c, httperror.NewMissingField("id"))
		return
	}

	var req session.ChatRequest
	if !bindJSON(c, &req) {
		return
	}

	if err := h.guard.EnsureSafe(req.Message); err != nil {
		h.logError(err)
		writeError(c, err)
		return
	}

	resp, err := h.manager.Chat(c.Request.Context(), sessionID, req)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(c, httperror.NewSessionNotFound(sessionID))
			return
		}
		h.logError(err)
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// handleDelete 세션 삭제
func (h *SessionHandler) handleDelete(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		writeError(c, httperror.NewMissingField("id"))
		return
	}

	err := h.manager.Delete(c.Request.Context(), sessionID)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(c, httperror.NewSessionNotFound(sessionID))
			return
		}
		h.logError(err)
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "session deleted", "id": sessionID})
}

func (h *SessionHandler) logError(err error) {
	if err == nil {
		return
	}
	h.logger.Warn("session_request_failed", "err", err)
}
