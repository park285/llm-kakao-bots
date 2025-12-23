package mq

import (
	"context"
	"fmt"
	"log/slog"

	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
	qredis "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/redis"
)

// MessageQueueCoordinator 는 타입이다.
type MessageQueueCoordinator struct {
	store  *qredis.PendingMessageStore
	logger *slog.Logger
}

// NewMessageQueueCoordinator 는 동작을 수행한다.
func NewMessageQueueCoordinator(store *qredis.PendingMessageStore, logger *slog.Logger) *MessageQueueCoordinator {
	return &MessageQueueCoordinator{
		store:  store,
		logger: logger,
	}
}

func (c *MessageQueueCoordinator) logEnqueueFailure(chatID string, userID string, res qredis.EnqueueResult) {
	switch res {
	case qredis.EnqueueQueueFull:
		c.logger.Warn("enqueue_failed", "chat_id", chatID, "user_id", userID, "reason", "QUEUE_FULL")
	case qredis.EnqueueDuplicate:
		c.logger.Debug("enqueue_failed", "chat_id", chatID, "user_id", userID, "reason", "DUPLICATE")
	default:
	}
}

// Enqueue 는 동작을 수행한다.
func (c *MessageQueueCoordinator) Enqueue(ctx context.Context, chatID string, msg qmodel.PendingMessage) (qredis.EnqueueResult, error) {
	res, err := c.store.Enqueue(ctx, chatID, msg)
	if err != nil {
		return res, fmt.Errorf("queue enqueue failed: %w", err)
	}
	c.logEnqueueFailure(chatID, msg.UserID, res)
	return res, nil
}

// EnqueueReplacingDuplicate 는 동작을 수행한다.
func (c *MessageQueueCoordinator) EnqueueReplacingDuplicate(ctx context.Context, chatID string, msg qmodel.PendingMessage) (qredis.EnqueueResult, error) {
	res, err := c.store.EnqueueReplacingDuplicate(ctx, chatID, msg)
	if err != nil {
		return res, fmt.Errorf("queue enqueue failed: %w", err)
	}
	c.logEnqueueFailure(chatID, msg.UserID, res)
	return res, nil
}

// Dequeue 는 동작을 수행한다.
func (c *MessageQueueCoordinator) Dequeue(ctx context.Context, chatID string) (qredis.DequeueResult, error) {
	res, err := c.store.Dequeue(ctx, chatID)
	if err != nil {
		return qredis.DequeueResult{}, fmt.Errorf("queue dequeue failed: %w", err)
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

// SetChainSkipFlag 체인 질문 스킵 플래그 설정.
func (c *MessageQueueCoordinator) SetChainSkipFlag(ctx context.Context, chatID string, userID string) error {
	if err := c.store.SetChainSkipFlag(ctx, chatID, userID); err != nil {
		return fmt.Errorf("set chain skip flag failed: %w", err)
	}
	return nil
}

// CheckAndClearChainSkipFlag 체인 질문 스킵 플래그 확인 및 삭제.
func (c *MessageQueueCoordinator) CheckAndClearChainSkipFlag(ctx context.Context, chatID string, userID string) (bool, error) {
	skipped, err := c.store.CheckAndClearChainSkipFlag(ctx, chatID, userID)
	if err != nil {
		return false, fmt.Errorf("check chain skip flag failed: %w", err)
	}
	return skipped, nil
}
