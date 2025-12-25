package mq

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
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

// NewMessageQueueProcessor: 새로운 MessageQueueProcessor 인스턴스를 생성한다.
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

// HasPending: 대기 중인 메시지가 있는지 확인한다.
func (p *MessageQueueProcessor) HasPending(ctx context.Context, chatID string) (bool, error) {
	return p.queueCoordinator.HasPending(ctx, chatID)
}

// EnqueueAndNotify: 메시지를 대기열에 추가하고 사용자에게 현재 대기열 상태 등을 알린다.
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

// ProcessQueuedMessages: 대기열에 쌓인 메시지들을 순차적으로 처리한다. (최대 반복 횟수 제한)
func (p *MessageQueueProcessor) ProcessQueuedMessages(ctx context.Context, chatID string, emit func(mqmsg.OutboundMessage) error) {
	p.logger.Debug("queue_processing_start", "chat_id", chatID)
	iterations := 0
	for iterations < qconfig.MQMaxQueueIterations {
		iterations++

		dequeueResult, err := p.queueCoordinator.Dequeue(ctx, chatID)
		if err != nil {
			p.logger.Warn("queue_dequeue_failed", "chat_id", chatID, "iteration", iterations, "err", err)
			return
		}
		p.logger.Debug("queue_dequeue_result", "chat_id", chatID, "status", dequeueResult.Status, "hasMessage", dequeueResult.Message != nil)

		switch dequeueResult.Status {
		case qredis.DequeueEmpty:
			return
		case qredis.DequeueSuccess:
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
	pending qmodel.PendingMessage,
	emit func(mqmsg.OutboundMessage) error,
) bool {
	p.logger.Debug("processing_queued_message", "chat_id", chatID, "user_id", pending.UserID)

	// 체인 배치 스킵 체크 (lock 획득 전)
	if p.shouldSkipChainBatch(ctx, chatID, pending, emit) {
		return true
	}

	// NotifyProcessingStart 제거: 대기열 메시지("잠시만 기다려주세요")가 충분한 UX 피드백 제공
	// Lock 실패 시 중복 알림 발생 방지

	holderName := pending.UserID
	if pending.Sender != nil && *pending.Sender != "" {
		holderName = *pending.Sender
	}

	cmd := p.commandParser.Parse(pending.Content)
	requiresWrite := true
	if cmd != nil {
		requiresWrite = cmd.RequiresWriteLock()
	}

	lockFn := p.lockManager.WithLock
	if !requiresWrite {
		lockFn = p.lockManager.WithReadLock
	}

	lockErr := lockFn(ctx, chatID, &holderName, func(ctx context.Context) error {
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
	pending qmodel.PendingMessage,
	emit func(mqmsg.OutboundMessage) error,
) bool {
	p.logger.Debug("queue_processing_lock_failed", "chat_id", chatID, "user_id", pending.UserID)

	replaceOnDuplicate := p.shouldReplaceDuplicate(pending.Content)

	var reEnqueueResult qredis.EnqueueResult
	var err error
	if replaceOnDuplicate {
		reEnqueueResult, err = p.queueCoordinator.EnqueueReplacingDuplicate(ctx, chatID, pending)
	} else {
		reEnqueueResult, err = p.queueCoordinator.Enqueue(ctx, chatID, pending)
	}
	if err != nil {
		p.logger.Warn("queue_requeue_failed", "chat_id", chatID, "user_id", pending.UserID, "err", err)
		return false
	}

	// 재큐잉 알림 제거: 대기열 메시지가 이미 전달됨, 추가 알림은 노이즈
	// 로그만 유지하여 디버깅 가능
	switch reEnqueueResult {
	case qredis.EnqueueSuccess:
		p.logger.Info("queue_requeue_success", "chat_id", chatID, "user_id", pending.UserID)
	case qredis.EnqueueDuplicate:
		p.logger.Info("queue_requeue_duplicate", "chat_id", chatID, "user_id", pending.UserID)
	case qredis.EnqueueQueueFull:
		_ = p.notifier.NotifyFailed(ctx, chatID, pending, emit)
		p.logger.Warn("queue_requeue_full", "chat_id", chatID, "user_id", pending.UserID)
	default:
		_ = p.notifier.NotifyFailed(ctx, chatID, pending, emit)
	}

	return false
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
