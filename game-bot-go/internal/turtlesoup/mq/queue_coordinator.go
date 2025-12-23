package mq

import (
	"context"
	"fmt"
	"log/slog"

	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
	tsredis "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/redis"
)

// MessageQueueCoordinator 는 타입이다.
type MessageQueueCoordinator struct {
	store  *tsredis.PendingMessageStore
	logger *slog.Logger
}

// NewMessageQueueCoordinator 는 동작을 수행한다.
func NewMessageQueueCoordinator(store *tsredis.PendingMessageStore, logger *slog.Logger) *MessageQueueCoordinator {
	return &MessageQueueCoordinator{
		store:  store,
		logger: logger,
	}
}

// Enqueue 는 동작을 수행한다.
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

// Dequeue 는 동작을 수행한다.
func (c *MessageQueueCoordinator) Dequeue(ctx context.Context, chatID string) (tsredis.DequeueResult, error) {
	res, err := c.store.Dequeue(ctx, chatID)
	if err != nil {
		return tsredis.DequeueResult{}, fmt.Errorf("queue dequeue failed: %w", err)
	}
	return res, nil
}

// HasPending 는 동작을 수행한다.
func (c *MessageQueueCoordinator) HasPending(ctx context.Context, chatID string) (bool, error) {
	ok, err := c.store.HasPending(ctx, chatID)
	if err != nil {
		return false, fmt.Errorf("queue hasPending failed: %w", err)
	}
	return ok, nil
}

// Size 는 동작을 수행한다.
func (c *MessageQueueCoordinator) Size(ctx context.Context, chatID string) (int, error) {
	size, err := c.store.Size(ctx, chatID)
	if err != nil {
		return 0, fmt.Errorf("queue size failed: %w", err)
	}
	return size, nil
}

// GetQueueDetails 는 동작을 수행한다.
func (c *MessageQueueCoordinator) GetQueueDetails(ctx context.Context, chatID string) (string, error) {
	details, err := c.store.GetQueueDetails(ctx, chatID)
	if err != nil {
		return "", fmt.Errorf("queue details failed: %w", err)
	}
	return details, nil
}

// Clear 는 동작을 수행한다.
func (c *MessageQueueCoordinator) Clear(ctx context.Context, chatID string) error {
	if err := c.store.Clear(ctx, chatID); err != nil {
		return fmt.Errorf("queue clear failed: %w", err)
	}
	return nil
}
