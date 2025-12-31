package mq

import (
	"context"
	"fmt"
	"log/slog"

	commonmq "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mq"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
	qredis "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/redis"
)

// MessageQueueCoordinator: 대기열 저장소(Redis) 접근과 관련 로깅을 담당하는 코디네이터입니다.
type MessageQueueCoordinator struct {
	store *qredis.PendingMessageStore
	base  *commonmq.QueueCoordinator[qmodel.PendingMessage, qredis.EnqueueResult, qredis.DequeueResult]
}

// NewMessageQueueCoordinator: 새로운 MessageQueueCoordinator 인스턴스를 생성합니다.
func NewMessageQueueCoordinator(store *qredis.PendingMessageStore, logger *slog.Logger) *MessageQueueCoordinator {
	return &MessageQueueCoordinator{
		store: store,
		base: commonmq.NewQueueCoordinator[qmodel.PendingMessage, qredis.EnqueueResult, qredis.DequeueResult](
			store,
			logger,
			commonmq.QueueCoordinatorConfig[qmodel.PendingMessage, qredis.EnqueueResult]{
				UserID: func(msg qmodel.PendingMessage) string {
					return msg.UserID
				},
				IsQueueFull: func(result qredis.EnqueueResult) bool {
					return result == qredis.EnqueueQueueFull
				},
				IsDuplicate: func(result qredis.EnqueueResult) bool {
					return result == qredis.EnqueueDuplicate
				},
			},
		),
	}
}

// Enqueue: 메시지를 대기열에 추가하고 결과를 반환합니다. (실패 시 로그 기록)
func (c *MessageQueueCoordinator) Enqueue(ctx context.Context, chatID string, msg qmodel.PendingMessage) (qredis.EnqueueResult, error) {
	result, err := c.base.Enqueue(ctx, chatID, msg)
	if err != nil {
		return result, fmt.Errorf("queue coordinator enqueue failed: %w", err)
	}
	return result, nil
}

// EnqueueReplacingDuplicate: 기존 중복 메시지가 있다면 최신 내용으로 교체하여 대기열에 추가합니다.
func (c *MessageQueueCoordinator) EnqueueReplacingDuplicate(ctx context.Context, chatID string, msg qmodel.PendingMessage) (qredis.EnqueueResult, error) {
	res, err := c.store.EnqueueReplacingDuplicate(ctx, chatID, msg)
	if err != nil {
		return res, fmt.Errorf("queue enqueue failed: %w", err)
	}
	c.base.LogEnqueueFailure(chatID, msg, res)
	return res, nil
}

// Dequeue: 대기열에서 가장 오래된 메시지를 꺼냅니다.
func (c *MessageQueueCoordinator) Dequeue(ctx context.Context, chatID string) (qredis.DequeueResult, error) {
	result, err := c.base.Dequeue(ctx, chatID)
	if err != nil {
		return qredis.DequeueResult{}, fmt.Errorf("queue coordinator dequeue failed: %w", err)
	}
	return result, nil
}

// DequeueBatch: 대기열에서 여러 메시지를 꺼냅니다.
func (c *MessageQueueCoordinator) DequeueBatch(ctx context.Context, chatID string, limit int) ([]qmodel.PendingMessage, error) {
	messages, err := c.store.DequeueBatch(ctx, chatID, limit)
	if err != nil {
		return nil, fmt.Errorf("queue dequeue batch failed: %w", err)
	}
	return messages, nil
}

// HasPending: 대기 중인 메시지가 있는지 확인합니다.
func (c *MessageQueueCoordinator) HasPending(ctx context.Context, chatID string) (bool, error) {
	ok, err := c.base.HasPending(ctx, chatID)
	if err != nil {
		return false, fmt.Errorf("queue coordinator hasPending failed: %w", err)
	}
	return ok, nil
}

// Size: 대기열 크기를 반환합니다.
func (c *MessageQueueCoordinator) Size(ctx context.Context, chatID string) (int, error) {
	size, err := c.base.Size(ctx, chatID)
	if err != nil {
		return 0, fmt.Errorf("queue coordinator size failed: %w", err)
	}
	return size, nil
}

// GetQueueDetails: 대기열 상세 정보를 반환합니다.
func (c *MessageQueueCoordinator) GetQueueDetails(ctx context.Context, chatID string) (string, error) {
	details, err := c.base.GetQueueDetails(ctx, chatID)
	if err != nil {
		return "", fmt.Errorf("queue coordinator queue details failed: %w", err)
	}
	return details, nil
}

// Clear: 대기열을 비웁니다.
func (c *MessageQueueCoordinator) Clear(ctx context.Context, chatID string) error {
	if err := c.base.Clear(ctx, chatID); err != nil {
		return fmt.Errorf("queue coordinator clear failed: %w", err)
	}
	return nil
}

// SetChainSkipFlag: 체인 질문 그룹의 나머지 질문들을 스킵하도록 플래그를 설정합니다.
func (c *MessageQueueCoordinator) SetChainSkipFlag(ctx context.Context, chatID string, userID string) error {
	if err := c.store.SetChainSkipFlag(ctx, chatID, userID); err != nil {
		return fmt.Errorf("set chain skip flag failed: %w", err)
	}
	return nil
}

// CheckAndClearChainSkipFlag: 스킵 플래그가 설정되어 있는지 확인하고, 확인 후 플래그를 제거합니다.
func (c *MessageQueueCoordinator) CheckAndClearChainSkipFlag(ctx context.Context, chatID string, userID string) (bool, error) {
	skipped, err := c.store.CheckAndClearChainSkipFlag(ctx, chatID, userID)
	if err != nil {
		return false, fmt.Errorf("check chain skip flag failed: %w", err)
	}
	return skipped, nil
}
