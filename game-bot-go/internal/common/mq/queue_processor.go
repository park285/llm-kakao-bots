package mq

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/pending"
	domainmodels "github.com/park285/llm-kakao-bots/game-bot-go/internal/domain/models"
)

// ProcessSingleQueuedMessage: 큐에 대기 중인 단일 메시지를 락 범위에서 실행하고, 락 획득 실패 시 재큐잉합니다.
// 게임별로 달라지는 enqueue 정책/에러 알림은 함수 인자로 주입하여 중복을 줄입니다.
func ProcessSingleQueuedMessage(
	ctx context.Context,
	logger *slog.Logger,
	lockManager interface {
		WithLock(ctx context.Context, chatID string, holderName *string, block func(ctx context.Context) error) error
	},
	processingLockService interface {
		StartProcessing(ctx context.Context, chatID string) error
		FinishProcessing(ctx context.Context, chatID string) error
	},
	notifier interface {
		NotifyFailed(ctx context.Context, chatID string, pending domainmodels.PendingMessage, emit func(mqmsg.OutboundMessage) error) error
		NotifyError(ctx context.Context, chatID string, pending domainmodels.PendingMessage, err error, emit func(mqmsg.OutboundMessage) error) error
	},
	reEnqueue func(ctx context.Context, chatID string, pending domainmodels.PendingMessage) (pending.EnqueueResult, error),
	executor func(ctx context.Context, chatID string, pending domainmodels.PendingMessage, emit func(mqmsg.OutboundMessage) error) error,
	chatID string,
	pendingMessage domainmodels.PendingMessage,
	emit func(mqmsg.OutboundMessage) error,
) bool {
	if logger == nil {
		logger = slog.Default()
	}

	holderName := pendingMessage.UserID
	if pendingMessage.Sender != nil && *pendingMessage.Sender != "" {
		holderName = *pendingMessage.Sender
	}

	lockErr := lockManager.WithLock(ctx, chatID, &holderName, func(ctx context.Context) error {
		if err := processingLockService.StartProcessing(ctx, chatID); err != nil {
			return fmt.Errorf("start processing failed: %w", err)
		}
		defer func() {
			_ = processingLockService.FinishProcessing(ctx, chatID)
		}()

		if err := executor(ctx, chatID, pendingMessage, emit); err != nil && notifier != nil {
			_ = notifier.NotifyError(ctx, chatID, pendingMessage, err, emit)
		}
		return nil
	})
	if lockErr == nil {
		return true
	}

	logger.Debug("queue_processing_lock_failed", "chat_id", chatID, "user_id", pendingMessage.UserID)

	reEnqueueResult, err := reEnqueue(ctx, chatID, pendingMessage)
	if err != nil {
		logger.Warn("queue_requeue_failed", "chat_id", chatID, "user_id", pendingMessage.UserID, "err", err)
		return false
	}

	// 재큐잉 알림 제거: 대기열 메시지가 이미 전달됨, 추가 알림은 노이즈
	// 로그만 유지하여 디버깅 가능
	switch reEnqueueResult {
	case pending.EnqueueSuccess:
		logger.Info("queue_requeue_success", "chat_id", chatID, "user_id", pendingMessage.UserID)
	case pending.EnqueueDuplicate:
		logger.Info("queue_requeue_duplicate", "chat_id", chatID, "user_id", pendingMessage.UserID)
	case pending.EnqueueQueueFull:
		if notifier != nil {
			_ = notifier.NotifyFailed(ctx, chatID, pendingMessage, emit)
		}
		logger.Warn("queue_requeue_full", "chat_id", chatID, "user_id", pendingMessage.UserID)
	default:
		if notifier != nil {
			_ = notifier.NotifyFailed(ctx, chatID, pendingMessage, emit)
		}
	}

	return false
}
