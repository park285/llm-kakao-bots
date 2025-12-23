package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	qmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/messages"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/redis"
)

// RiddleServiceForAdmin AdminHandler가 의존하는 RiddleService 메서드 인터페이스.
type RiddleServiceForAdmin interface {
	Surrender(ctx context.Context, chatID string) (string, error)
}

// AdminHandler 관리자 명령어 핸들러.
type AdminHandler struct {
	adminUserIDs  []string
	riddleService RiddleServiceForAdmin
	sessionStore  *redis.SessionStore
	msgProvider   *messageprovider.Provider
	logger        *slog.Logger
}

// NewAdminHandler 생성자.
func NewAdminHandler(
	adminUserIDs []string,
	riddleService RiddleServiceForAdmin,
	sessionStore *redis.SessionStore,
	msgProvider *messageprovider.Provider,
	logger *slog.Logger,
) *AdminHandler {
	return &AdminHandler{
		adminUserIDs:  adminUserIDs,
		riddleService: riddleService,
		sessionStore:  sessionStore,
		msgProvider:   msgProvider,
		logger:        logger,
	}
}

// IsAdmin 관리자 여부 확인.
func (h *AdminHandler) IsAdmin(userID string) bool {
	for _, id := range h.adminUserIDs {
		if id == userID {
			return true
		}
	}
	return false
}

// ForceEnd 관리자 강제 종료.
func (h *AdminHandler) ForceEnd(ctx context.Context, chatID string, userID string) (string, error) {
	h.logger.Info("HANDLE_ADMIN_FORCE_END", "chatID", chatID, "userID", userID)
	if !h.IsAdmin(userID) {
		h.logger.Warn("ADMIN_PERMISSION_DENIED", "userID", userID, "chatID", chatID)
		return h.msgProvider.Get(qmessages.ErrorNoPermission), nil
	}

	// 세션 확인
	session, err := h.sessionStore.Get(ctx, chatID)
	if err != nil {
		return "", fmt.Errorf("session store get: %w", err)
	}
	if session == nil {
		return h.msgProvider.Get(qmessages.ErrorNoSessionShort), nil
	}

	result, err := h.riddleService.Surrender(ctx, chatID)
	if err != nil {
		return "", fmt.Errorf("surrender: %w", err)
	}

	h.logger.Info("ADMIN_FORCE_END_SUCCESS", "chatID", chatID, "adminID", userID)
	return h.msgProvider.Get(qmessages.AdminForceEndPrefix) + result, nil
}

// ClearAll 관리자 전체 삭제.
func (h *AdminHandler) ClearAll(ctx context.Context, chatID string, userID string) (string, error) {
	h.logger.Info("HANDLE_ADMIN_CLEAR_ALL", "chatID", chatID, "userID", userID)
	if !h.IsAdmin(userID) {
		h.logger.Warn("ADMIN_PERMISSION_DENIED", "userID", userID, "chatID", chatID)
		return h.msgProvider.Get(qmessages.ErrorNoPermission), nil
	}

	if err := h.sessionStore.ClearAllData(ctx, chatID); err != nil {
		h.logger.Error("session_store_clear_all_failed", "error", err, "chatID", chatID)
		return "", fmt.Errorf("clear all data: %w", err)
	}

	h.logger.Info("ADMIN_CLEAR_ALL_SUCCESS", "chatID", chatID, "adminID", userID)
	return h.msgProvider.Get(qmessages.AdminClearAllSuccess), nil
}
