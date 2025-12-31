package mq

import (
	"context"
	"fmt"
	"log/slog"

	commonmq "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mq"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
	tsredis "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/redis"
)

// MessageQueueCoordinator: Redis 대기열 작업을 추상화하여 제공하는 관리자입니다.
type MessageQueueCoordinator struct {
	base *commonmq.QueueCoordinator[tsmodel.PendingMessage, tsredis.EnqueueResult, tsredis.DequeueResult]
}

// NewMessageQueueCoordinator: 새로운 MessageQueueCoordinator 인스턴스를 생성합니다.
func NewMessageQueueCoordinator(store *tsredis.PendingMessageStore, logger *slog.Logger) *MessageQueueCoordinator {
	return &MessageQueueCoordinator{
		base: commonmq.NewQueueCoordinator[tsmodel.PendingMessage, tsredis.EnqueueResult, tsredis.DequeueResult](
			store,
			logger,
			commonmq.QueueCoordinatorConfig[tsmodel.PendingMessage, tsredis.EnqueueResult]{
				UserID: func(msg tsmodel.PendingMessage) string {
					return msg.UserID
				},
				IsQueueFull: func(result tsredis.EnqueueResult) bool {
					return result == tsredis.EnqueueQueueFull
				},
				IsDuplicate: func(result tsredis.EnqueueResult) bool {
					return result == tsredis.EnqueueDuplicate
				},
			},
		),
	}
}

// Enqueue: 메시지를 대기열에 추가하고 결과를 반환합니다. (성공, 중복, 가득 참 등)
func (c *MessageQueueCoordinator) Enqueue(ctx context.Context, chatID string, msg tsmodel.PendingMessage) (tsredis.EnqueueResult, error) {
	result, err := c.base.Enqueue(ctx, chatID, msg)
	if err != nil {
		return result, fmt.Errorf("queue coordinator enqueue failed: %w", err)
	}
	return result, nil
}

// Dequeue: 대기열에서 가장 오래된 메시지를 하나 꺼냅니다.
func (c *MessageQueueCoordinator) Dequeue(ctx context.Context, chatID string) (tsredis.DequeueResult, error) {
	result, err := c.base.Dequeue(ctx, chatID)
	if err != nil {
		return tsredis.DequeueResult{}, fmt.Errorf("queue coordinator dequeue failed: %w", err)
	}
	return result, nil
}

// HasPending: 대기 중인 메시지가 있는지 확인합니다.
func (c *MessageQueueCoordinator) HasPending(ctx context.Context, chatID string) (bool, error) {
	ok, err := c.base.HasPending(ctx, chatID)
	if err != nil {
		return false, fmt.Errorf("queue coordinator hasPending failed: %w", err)
	}
	return ok, nil
}

// Size: 현재 대기열의 크기(메시지 수)를 반환합니다.
func (c *MessageQueueCoordinator) Size(ctx context.Context, chatID string) (int, error) {
	size, err := c.base.Size(ctx, chatID)
	if err != nil {
		return 0, fmt.Errorf("queue coordinator size failed: %w", err)
	}
	return size, nil
}

// GetQueueDetails: 대기열 상태(대기 중인 사용자 목록 등)를 문자열로 반환합니다.
func (c *MessageQueueCoordinator) GetQueueDetails(ctx context.Context, chatID string) (string, error) {
	details, err := c.base.GetQueueDetails(ctx, chatID)
	if err != nil {
		return "", fmt.Errorf("queue coordinator queue details failed: %w", err)
	}
	return details, nil
}

// Clear: 대기열의 모든 메시지를 제거합니다.
func (c *MessageQueueCoordinator) Clear(ctx context.Context, chatID string) error {
	if err := c.base.Clear(ctx, chatID); err != nil {
		return fmt.Errorf("queue coordinator clear failed: %w", err)
	}
	return nil
}
