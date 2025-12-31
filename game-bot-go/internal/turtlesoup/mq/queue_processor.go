package mq

import (
	"context"
	"log/slog"
	"time"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	commonmq "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mq"
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

	reEnqueue := func(ctx context.Context, chatID string, pending tsmodel.PendingMessage) (tsredis.EnqueueResult, error) {
		return p.queueCoordinator.Enqueue(ctx, chatID, pending)
	}
	return commonmq.ProcessSingleQueuedMessage(
		ctx,
		p.logger,
		p.lockManager,
		p.processingLockService,
		p.notifier,
		reEnqueue,
		p.executor,
		chatID,
		pending,
		emit,
	)
}
