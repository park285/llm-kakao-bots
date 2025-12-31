package mq

import (
	"context"
	"fmt"
	"log/slog"
)

// QueueCoordinatorConfig: QueueCoordinator 동작(로그/판단)에 필요한 게임별 설정을 묶습니다.
type QueueCoordinatorConfig[PendingMessage any, EnqueueResult any] struct {
	// UserID: enqueue 결과 로깅에 사용할 userID 추출 함수
	UserID func(PendingMessage) string
	// IsQueueFull: enqueue 결과가 "큐가 가득 참"인지 판별하는 함수
	IsQueueFull func(EnqueueResult) bool
	// IsDuplicate: enqueue 결과가 "중복"인지 판별하는 함수
	IsDuplicate func(EnqueueResult) bool
}

// QueueCoordinatorStore: QueueCoordinator가 필요로 하는 대기열 저장소 인터페이스입니다.
// 게임 도메인별 Redis Store 구현체에서 공통으로 사용되는 메서드만 노출합니다.
type QueueCoordinatorStore[PendingMessage any, EnqueueResult any, DequeueResult any] interface {
	Enqueue(ctx context.Context, chatID string, msg PendingMessage) (EnqueueResult, error)
	Dequeue(ctx context.Context, chatID string) (DequeueResult, error)
	HasPending(ctx context.Context, chatID string) (bool, error)
	Size(ctx context.Context, chatID string) (int, error)
	GetQueueDetails(ctx context.Context, chatID string) (string, error)
	Clear(ctx context.Context, chatID string) error
}

// QueueCoordinator: Redis 대기열 저장소 접근을 캡슐화하고, 공통 에러 래핑/로깅 정책을 제공합니다.
type QueueCoordinator[PendingMessage any, EnqueueResult any, DequeueResult any] struct {
	store  QueueCoordinatorStore[PendingMessage, EnqueueResult, DequeueResult]
	logger *slog.Logger
	cfg    QueueCoordinatorConfig[PendingMessage, EnqueueResult]
}

// NewQueueCoordinator: 새로운 QueueCoordinator 인스턴스를 생성합니다.
func NewQueueCoordinator[PendingMessage any, EnqueueResult any, DequeueResult any](
	store QueueCoordinatorStore[PendingMessage, EnqueueResult, DequeueResult],
	logger *slog.Logger,
	cfg QueueCoordinatorConfig[PendingMessage, EnqueueResult],
) *QueueCoordinator[PendingMessage, EnqueueResult, DequeueResult] {
	if logger == nil {
		logger = slog.Default()
	}
	return &QueueCoordinator[PendingMessage, EnqueueResult, DequeueResult]{
		store:  store,
		logger: logger,
		cfg:    cfg,
	}
}

func (c *QueueCoordinator[PendingMessage, EnqueueResult, DequeueResult]) logEnqueueFailure(
	chatID string,
	userID string,
	result EnqueueResult,
) {
	if c == nil || c.logger == nil {
		return
	}
	if c.cfg.IsQueueFull != nil && c.cfg.IsQueueFull(result) {
		c.logger.Warn("enqueue_failed", "chat_id", chatID, "user_id", userID, "reason", "QUEUE_FULL")
		return
	}
	if c.cfg.IsDuplicate != nil && c.cfg.IsDuplicate(result) {
		c.logger.Debug("enqueue_failed", "chat_id", chatID, "user_id", userID, "reason", "DUPLICATE")
		return
	}
}

// LogEnqueueFailure: enqueue 변형(예: replaceOnDuplicate) 처리 후 공통 실패 로깅 정책을 적용합니다.
func (c *QueueCoordinator[PendingMessage, EnqueueResult, DequeueResult]) LogEnqueueFailure(
	chatID string,
	msg PendingMessage,
	result EnqueueResult,
) {
	if c == nil {
		return
	}
	userID := ""
	if c.cfg.UserID != nil {
		userID = c.cfg.UserID(msg)
	}
	c.logEnqueueFailure(chatID, userID, result)
}

// Enqueue: 메시지를 대기열에 추가하고 결과를 반환합니다. (실패 시 로그 기록)
func (c *QueueCoordinator[PendingMessage, EnqueueResult, DequeueResult]) Enqueue(
	ctx context.Context,
	chatID string,
	msg PendingMessage,
) (EnqueueResult, error) {
	result, err := c.store.Enqueue(ctx, chatID, msg)
	if err != nil {
		return result, fmt.Errorf("queue enqueue failed: %w", err)
	}
	c.LogEnqueueFailure(chatID, msg, result)
	return result, nil
}

// Dequeue: 대기열에서 가장 오래된 메시지를 꺼냅니다.
func (c *QueueCoordinator[PendingMessage, EnqueueResult, DequeueResult]) Dequeue(
	ctx context.Context,
	chatID string,
) (DequeueResult, error) {
	result, err := c.store.Dequeue(ctx, chatID)
	if err != nil {
		var zero DequeueResult
		return zero, fmt.Errorf("queue dequeue failed: %w", err)
	}
	return result, nil
}

// HasPending: 대기 중인 메시지가 있는지 확인합니다.
func (c *QueueCoordinator[PendingMessage, EnqueueResult, DequeueResult]) HasPending(ctx context.Context, chatID string) (bool, error) {
	ok, err := c.store.HasPending(ctx, chatID)
	if err != nil {
		return false, fmt.Errorf("queue hasPending failed: %w", err)
	}
	return ok, nil
}

// Size: 대기열 크기를 반환합니다.
func (c *QueueCoordinator[PendingMessage, EnqueueResult, DequeueResult]) Size(ctx context.Context, chatID string) (int, error) {
	size, err := c.store.Size(ctx, chatID)
	if err != nil {
		return 0, fmt.Errorf("queue size failed: %w", err)
	}
	return size, nil
}

// GetQueueDetails: 대기열 상세 정보를 반환합니다.
func (c *QueueCoordinator[PendingMessage, EnqueueResult, DequeueResult]) GetQueueDetails(ctx context.Context, chatID string) (string, error) {
	details, err := c.store.GetQueueDetails(ctx, chatID)
	if err != nil {
		return "", fmt.Errorf("queue details failed: %w", err)
	}
	return details, nil
}

// Clear: 대기열을 비웁니다.
func (c *QueueCoordinator[PendingMessage, EnqueueResult, DequeueResult]) Clear(ctx context.Context, chatID string) error {
	if err := c.store.Clear(ctx, chatID); err != nil {
		return fmt.Errorf("queue clear failed: %w", err)
	}
	return nil
}
