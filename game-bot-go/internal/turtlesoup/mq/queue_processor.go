package mq

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	tsmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/messages"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
	tsredis "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/redis"
)

// CommandExecutor: 대기열에서 꺼낸 메시지(명령어)를 실제로 수행하는 함수 타입 정의 (예: GameCommandHandler.ProcessCommand)
type CommandExecutor func(ctx context.Context, chatID string, pending tsmodel.PendingMessage, emit func(mqmsg.OutboundMessage) error) error

// MessageQueueProcessor: Redis 대기열 기반의 메시지 순차 처리 및 흐름 제어(락, 알림, 재시도 등)를 담당하는 프로세서
type MessageQueueProcessor struct {
	queueCoordinator      *MessageQueueCoordinator
	lockManager           *tsredis.LockManager
	processingLockService *tsredis.ProcessingLockService
	msgProvider           *messageprovider.Provider
	notifier              *MessageQueueNotifier
	executor              CommandExecutor
	logger                *slog.Logger
}

// NewMessageQueueProcessor: MessageQueueProcessor 인스턴스를 생성하고 필요한 종속성을 주입한다.
func NewMessageQueueProcessor(
	queueCoordinator *MessageQueueCoordinator,
	lockManager *tsredis.LockManager,
	processingLockService *tsredis.ProcessingLockService,
	msgProvider *messageprovider.Provider,
	notifier *MessageQueueNotifier,
	executor CommandExecutor,
	logger *slog.Logger,
) *MessageQueueProcessor {
	return &MessageQueueProcessor{
		queueCoordinator:      queueCoordinator,
		lockManager:           lockManager,
		processingLockService: processingLockService,
		msgProvider:           msgProvider,
		notifier:              notifier,
		executor:              executor,
		logger:                logger,
	}
}

// EnqueueAndNotify: 처리 불가능한 메시지를 대기열에 추가하고, 사용자에게 대기 상태 알림 메시지(순번 등)를 전송한다.
func (p *MessageQueueProcessor) EnqueueAndNotify(
	ctx context.Context,
	chatID string,
	userID string,
	content string,
	threadID *string,
	sender *string,
	emit func(mqmsg.OutboundMessage) error,
) error {
	pending := tsmodel.PendingMessage{
		UserID:    userID,
		Content:   content,
		ThreadID:  threadID,
		Sender:    sender,
		Timestamp: time.Now().UnixMilli(),
	}

	result, err := p.queueCoordinator.Enqueue(ctx, chatID, pending)
	if err != nil {
		return err
	}

	userName := pending.DisplayName(chatID, p.msgProvider.Get(tsmessages.UserAnonymous))
	message, err := p.buildQueueMessage(ctx, result, chatID, userName, content)
	if err != nil {
		return err
	}

	return emit(mqmsg.NewWaiting(chatID, message, threadID))
}

func (p *MessageQueueProcessor) buildQueueMessage(
	ctx context.Context,
	result tsredis.EnqueueResult,
	chatID string,
	userName string,
	content string,
) (string, error) {
	switch result {
	case tsredis.EnqueueSuccess:
		rawDetails, err := p.queueCoordinator.GetQueueDetails(ctx, chatID)
		if err != nil {
			return "", err
		}
		queueDetails := rawDetails
		if queueDetails == "" {
			queueDetails = p.msgProvider.Get(tsmessages.QueueEmpty)
		}
		return p.msgProvider.Get(
			tsmessages.QueueMessageQueued,
			messageprovider.P("user", userName),
			messageprovider.P("queueDetails", queueDetails),
		), nil
	case tsredis.EnqueueQueueFull:
		return p.msgProvider.Get(tsmessages.QueueFull), nil
	case tsredis.EnqueueDuplicate:
		return p.msgProvider.Get(
			tsmessages.QueueAlreadyQueued,
			messageprovider.P("user", userName),
			messageprovider.P("content", content),
		), nil
	default:
		return p.msgProvider.Get(tsmessages.ErrorInternal), nil
	}
}

// ProcessQueuedMessages: 해당 채팅방의 대기열에 쌓인 메시지들을 하나씩 꺼내어 순차적으로 처리한다.
// 무한 루프 방지를 위해 최대 반복 횟수 제한(MQMaxQueueIterations)을 둔다.
func (p *MessageQueueProcessor) ProcessQueuedMessages(ctx context.Context, chatID string, emit func(mqmsg.OutboundMessage) error) {
	iterations := 0
	for iterations < tsconfig.MQMaxQueueIterations {
		iterations++

		dequeueResult, err := p.queueCoordinator.Dequeue(ctx, chatID)
		if err != nil {
			p.logger.Warn("queue_dequeue_failed", "chat_id", chatID, "iteration", iterations, "err", err)
			return
		}

		switch dequeueResult.Status {
		case tsredis.DequeueEmpty:
			return
		case tsredis.DequeueExhausted:
			p.logger.Debug("dequeue_exhausted", "chat_id", chatID, "iteration", iterations)
			continue
		case tsredis.DequeueSuccess:
			if dequeueResult.Message == nil {
				return
			}
			if cont := p.processSingleQueuedMessage(ctx, chatID, *dequeueResult.Message, emit); !cont {
				return
			}
		default:
			return
		}
	}

	p.logger.Warn("queue_processing_limit_reached", "chat_id", chatID, "max_iterations", iterations)
}

func (p *MessageQueueProcessor) processSingleQueuedMessage(
	ctx context.Context,
	chatID string,
	pending tsmodel.PendingMessage,
	emit func(mqmsg.OutboundMessage) error,
) bool {
	p.logger.Debug("processing_queued_message", "chat_id", chatID, "user_id", pending.UserID)

	// NotifyProcessingStart 제거: 대기열 메시지("잠시만 기다려주세요")가 충분한 UX 피드백 제공
	// Lock 실패 시 중복 알림 발생 방지

	holderName := pending.UserID
	if pending.Sender != nil && *pending.Sender != "" {
		holderName = *pending.Sender
	}

	lockErr := p.lockManager.WithLock(ctx, chatID, &holderName, func(ctx context.Context) error {
		if err := p.processingLockService.StartProcessing(ctx, chatID); err != nil {
			return fmt.Errorf("start processing failed: %w", err)
		}
		defer func() {
			_ = p.processingLockService.FinishProcessing(ctx, chatID)
		}()

		if err := p.executor(ctx, chatID, pending, emit); err != nil {
			_ = p.notifier.NotifyError(ctx, chatID, pending, err, emit)
		}
		return nil
	})
	if lockErr != nil {
		return p.handleLockAcquisitionFailure(ctx, chatID, pending, emit)
	}

	return true
}

func (p *MessageQueueProcessor) handleLockAcquisitionFailure(
	ctx context.Context,
	chatID string,
	pending tsmodel.PendingMessage,
	emit func(mqmsg.OutboundMessage) error,
) bool {
	p.logger.Debug("queue_processing_lock_failed", "chat_id", chatID, "user_id", pending.UserID)

	reEnqueueResult, err := p.queueCoordinator.Enqueue(ctx, chatID, pending)
	if err != nil {
		p.logger.Warn("queue_requeue_failed", "chat_id", chatID, "user_id", pending.UserID, "err", err)
		return false
	}

	// 재큐잉 알림 제거: 대기열 메시지가 이미 전달됨, 추가 알림은 노이즈
	// 로그만 유지하여 디버깅 가능
	switch reEnqueueResult {
	case tsredis.EnqueueSuccess:
		p.logger.Info("queue_requeue_success", "chat_id", chatID, "user_id", pending.UserID)
	case tsredis.EnqueueDuplicate:
		p.logger.Info("queue_requeue_duplicate", "chat_id", chatID, "user_id", pending.UserID)
	case tsredis.EnqueueQueueFull:
		_ = p.notifier.NotifyFailed(ctx, chatID, pending, emit)
		p.logger.Warn("queue_requeue_full", "chat_id", chatID, "user_id", pending.UserID)
	default:
		_ = p.notifier.NotifyFailed(ctx, chatID, pending, emit)
	}

	return false
}
