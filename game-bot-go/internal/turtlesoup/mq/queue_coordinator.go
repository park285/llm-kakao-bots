package mq

import (
	"context"
	"fmt"
	"log/slog"

	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
	tsredis "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/redis"
)

// MessageQueueCoordinator: Redis 대기열 작업을 추상화하여 제공하는 관리자
type MessageQueueCoordinator struct {
	store  *tsredis.PendingMessageStore
	logger *slog.Logger
}

// NewMessageQueueCoordinator: 새로운 MessageQueueCoordinator 인스턴스를 생성한다.
func NewMessageQueueCoordinator(store *tsredis.PendingMessageStore, logger *slog.Logger) *MessageQueueCoordinator {
	return &MessageQueueCoordinator{
		store:  store,
		logger: logger,
	}
}

// Enqueue: 메시지를 대기열에 추가하고 결과를 반환한다. (성공, 중복, 가득 참 등)
func (c *MessageQueueCoordinator) Enqueue(ctx context.Context, chatID string, msg tsmodel.PendingMessage) (tsredis.EnqueueResult, error) {
	res, err := c.store.Enqueue(ctx, chatID, msg)
	if err != nil {
		return res, fmt.Errorf("queue enqueue failed: %w", err)
	}

	switch res {
	case tsredis.EnqueueQueueFull:
		c.logger.Warn("enqueue_failed", "chat_id", chatID, "user_id", msg.UserID, "reason", "QUEUE_FULL")
	case tsredis.EnqueueDuplicate:
		c.logger.Debug("enqueue_failed", "chat_id", chatID, "user_id", msg.UserID, "reason", "DUPLICATE")
	default:
	}

	return res, nil
}

// Dequeue: 대기열에서 가장 오래된 메시지를 하나 꺼낸다.
func (c *MessageQueueCoordinator) Dequeue(ctx context.Context, chatID string) (tsredis.DequeueResult, error) {
	res, err := c.store.Dequeue(ctx, chatID)
	if err != nil {
		return tsredis.DequeueResult{}, fmt.Errorf("queue dequeue failed: %w", err)
	}
	return res, nil
}

// HasPending: 대기 중인 메시지가 있는지 확인한다.
func (c *MessageQueueCoordinator) HasPending(ctx context.Context, chatID string) (bool, error) {
	ok, err := c.store.HasPending(ctx, chatID)
	if err != nil {
		return false, fmt.Errorf("queue hasPending failed: %w", err)
	}
	return ok, nil
}

// Size: 현재 대기열의 크기(메시지 수)를 반환한다.
func (c *MessageQueueCoordinator) Size(ctx context.Context, chatID string) (int, error) {
	size, err := c.store.Size(ctx, chatID)
	if err != nil {
		return 0, fmt.Errorf("queue size failed: %w", err)
	}
	return size, nil
}

// GetQueueDetails: 대기열 상태(대기 중인 사용자 목록 등)를 문자열로 반환한다.
func (c *MessageQueueCoordinator) GetQueueDetails(ctx context.Context, chatID string) (string, error) {
	details, err := c.store.GetQueueDetails(ctx, chatID)
	if err != nil {
		return "", fmt.Errorf("queue details failed: %w", err)
	}
	return details, nil
}

// Clear: 대기열의 모든 메시지를 제거한다.
func (c *MessageQueueCoordinator) Clear(ctx context.Context, chatID string) error {
	if err := c.store.Clear(ctx, chatID); err != nil {
		return fmt.Errorf("queue clear failed: %w", err)
	}
	return nil
}
