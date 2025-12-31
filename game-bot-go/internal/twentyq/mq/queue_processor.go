package mq

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	commonmq "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mq"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/messages"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
	qredis "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/redis"
)

// CommandExecutor: 실제 비즈니스 로직(명령어 처리)을 수행하는 함수 타입
type CommandExecutor func(ctx context.Context, chatID string, pending qmodel.PendingMessage, emit func(mqmsg.OutboundMessage) error) error

// MessageQueueProcessor: 대기열 메시지 처리 및 락 관리, 알림 전송 등을 총괄하는 프로세서
type MessageQueueProcessor struct {
	queueCoordinator      *MessageQueueCoordinator
	lockManager           *qredis.LockManager
	processingLockService *qredis.ProcessingLockService
	commandParser         *CommandParser
	msgProvider           *messageprovider.Provider
	notifier              *MessageQueueNotifier
	executor              CommandExecutor
	logger                *slog.Logger
}

// NewMessageQueueProcessor: 새로운 MessageQueueProcessor 인스턴스를 생성합니다.
func NewMessageQueueProcessor(
	queueCoordinator *MessageQueueCoordinator,
	lockManager *qredis.LockManager,
	processingLockService *qredis.ProcessingLockService,
	commandParser *CommandParser,
	msgProvider *messageprovider.Provider,
	notifier *MessageQueueNotifier,
	executor CommandExecutor,
	logger *slog.Logger,
) *MessageQueueProcessor {
	return &MessageQueueProcessor{
		queueCoordinator:      queueCoordinator,
		lockManager:           lockManager,
		processingLockService: processingLockService,
		commandParser:         commandParser,
		msgProvider:           msgProvider,
		notifier:              notifier,
		executor:              executor,
		logger:                logger,
	}
}

// HasPending: 대기 중인 메시지가 있는지 확인합니다.
func (p *MessageQueueProcessor) HasPending(ctx context.Context, chatID string) (bool, error) {
	return p.queueCoordinator.HasPending(ctx, chatID)
}

// EnqueueAndNotify: 메시지를 대기열에 추가하고 사용자에게 현재 대기열 상태 등을 알립니다.
func (p *MessageQueueProcessor) EnqueueAndNotify(
	ctx context.Context,
	chatID string,
	userID string,
	content string,
	threadID *string,
	sender *string,
	emit func(mqmsg.OutboundMessage) error,
) error {
	pending := qmodel.PendingMessage{
		UserID:    userID,
		Content:   content,
		ThreadID:  threadID,
		Sender:    sender,
		Timestamp: time.Now().UnixMilli(),
	}

	replaceOnDuplicate := p.shouldReplaceDuplicate(content)

	var result qredis.EnqueueResult
	var err error
	if replaceOnDuplicate {
		result, err = p.queueCoordinator.EnqueueReplacingDuplicate(ctx, chatID, pending)
	} else {
		result, err = p.queueCoordinator.Enqueue(ctx, chatID, pending)
	}
	if err != nil {
		return err
	}

	userName := pending.DisplayName(chatID, p.msgProvider.Get(qmessages.UserAnonymous))
	message, err := p.buildQueueMessage(ctx, result, chatID, userName, content)
	if err != nil {
		return err
	}

	return emit(mqmsg.NewWaiting(chatID, message, threadID))
}

func (p *MessageQueueProcessor) buildQueueMessage(
	ctx context.Context,
	result qredis.EnqueueResult,
	chatID string,
	userName string,
	content string,
) (string, error) {
	switch result {
	case qredis.EnqueueSuccess:
		rawDetails, err := p.queueCoordinator.GetQueueDetails(ctx, chatID)
		if err != nil {
			return "", err
		}
		queueDetails := rawDetails
		if queueDetails == "" {
			queueDetails = p.msgProvider.Get(qmessages.QueueEmpty)
		}
		return p.msgProvider.Get(
			qmessages.LockMessageQueued,
			messageprovider.P("user", userName),
			messageprovider.P("queueDetails", queueDetails),
		), nil
	case qredis.EnqueueQueueFull:
		return p.msgProvider.Get(qmessages.LockQueueFull), nil
	case qredis.EnqueueDuplicate:
		return p.msgProvider.Get(
			qmessages.LockAlreadyQueued,
			messageprovider.P("user", userName),
			messageprovider.P("content", content),
		), nil
	default:
		return p.msgProvider.Get(qmessages.ErrorGeneric), nil
	}
}

// ProcessQueuedMessages: 대기열에 쌓인 메시지들을 순차적으로 처리합니다. (최대 반복 횟수 제한)
func (p *MessageQueueProcessor) ProcessQueuedMessages(ctx context.Context, chatID string, emit func(mqmsg.OutboundMessage) error) {
	p.logger.Debug("queue_processing_start", "chat_id", chatID)
	processed := 0
	for processed < qconfig.MQMaxQueueIterations {
		remaining := qconfig.MQMaxQueueIterations - processed
		batchSize := qconfig.QueueDequeueBatchSize
		if batchSize > remaining {
			batchSize = remaining
		}

		messages, err := p.queueCoordinator.DequeueBatch(ctx, chatID, batchSize)
		if err != nil {
			p.logger.Warn("queue_dequeue_failed", "chat_id", chatID, "batch_size", batchSize, "err", err)
			if len(messages) == 0 {
				return
			}
		}
		if len(messages) == 0 {
			return
		}
		p.logger.Debug("queue_dequeue_result", "chat_id", chatID, "count", len(messages))

		for _, message := range messages {
			processed++
			if cont := p.processSingleQueuedMessage(ctx, chatID, message, emit); !cont {
				return
			}
			if processed >= qconfig.MQMaxQueueIterations {
				break
			}
		}
	}

	p.logger.Warn("queue_processing_limit_reached", "chat_id", chatID, "max_iterations", processed)
}

func (p *MessageQueueProcessor) processSingleQueuedMessage(
	ctx context.Context,
	chatID string,
	pending qmodel.PendingMessage,
	emit func(mqmsg.OutboundMessage) error,
) bool {
	p.logger.Debug("processing_queued_message", "chat_id", chatID, "user_id", pending.UserID)

	// 체인 배치 스킵 체크 (lock 획득 전)
	if p.shouldSkipChainBatch(ctx, chatID, pending, emit) {
		return true
	}

	reEnqueue := func(ctx context.Context, chatID string, pending qmodel.PendingMessage) (qredis.EnqueueResult, error) {
		replaceOnDuplicate := p.shouldReplaceDuplicate(pending.Content)
		if replaceOnDuplicate {
			return p.queueCoordinator.EnqueueReplacingDuplicate(ctx, chatID, pending)
		}
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

// shouldSkipChainBatch 체인 배치 스킵 여부 확인 (lock 획득 전).
func (p *MessageQueueProcessor) shouldSkipChainBatch(
	ctx context.Context,
	chatID string,
	pending qmodel.PendingMessage,
	emit func(mqmsg.OutboundMessage) error,
) bool {
	if !pending.IsChainBatch || len(pending.BatchQuestions) == 0 {
		return false
	}

	shouldSkip, err := p.queueCoordinator.CheckAndClearChainSkipFlag(ctx, chatID, pending.UserID)
	if err != nil {
		p.logger.Warn("chain_skip_flag_check_failed", "chat_id", chatID, "user_id", pending.UserID, "err", err)
		return false
	}

	if !shouldSkip {
		return false
	}

	p.logger.Info("chain_batch_skipped", "chat_id", chatID, "user_id", pending.UserID, "reason", "condition_not_met")
	skipMessage := p.msgProvider.Get(qmessages.ChainConditionNotMet,
		messageprovider.P("questions", strings.Join(pending.BatchQuestions, ", ")))
	_ = emit(mqmsg.NewFinal(chatID, skipMessage, pending.ThreadID))
	return true
}

// shouldReplaceDuplicate 항복/동의/거부 명령은 중복 시 교체 허용.
func (p *MessageQueueProcessor) shouldReplaceDuplicate(content string) bool {
	cmd := p.commandParser.Parse(content)
	if cmd == nil {
		return false
	}
	switch cmd.Kind {
	case CommandSurrender, CommandAgree, CommandReject:
		return true
	default:
		return false
	}
}
