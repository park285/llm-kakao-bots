package mq

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
	domainmodels "github.com/park285/llm-kakao-bots/game-bot-go/internal/domain/models"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/messages"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
	qredis "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/redis"
	qsecurity "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/security"
)

// GameMessageService: 스무고개/바다거북스프 게임의 메시지 흐름을 총괄하는 오케스트레이터
// 명령어 파싱, 락 관리, 비즈니스 로직 실행, 대기열 처리 및 응답 전송을 조정한다.
type GameMessageService struct {
	commandHandler         *GameCommandHandler
	playerRegistrar        PlayerRegistrar
	messageSender          *MessageSender
	msgProvider            *messageprovider.Provider
	publisher              *ReplyPublisher
	accessControl          *qsecurity.AccessControl
	commandParser          *CommandParser
	lockManager            *qredis.LockManager
	processingLockService  *qredis.ProcessingLockService
	queueProcessor         *MessageQueueProcessor
	restClient             *llmrest.Client
	commandPrefix          string
	processingWaitingDelay time.Duration
	logger                 *slog.Logger
}

// PlayerRegistrar: 게임 참여자(플레이어) 정보 등록 및 세션 존재 여부를 확인하는 인터페이스
type PlayerRegistrar interface {
	RegisterPlayerAsync(ctx context.Context, chatID string, userID string, sender *string)
	HasSession(ctx context.Context, chatID string) (bool, error)
}

type hintAvailabilityChecker interface {
	CanGenerateHint(ctx context.Context, chatID string) (bool, error)
}

// NewGameMessageService: 모든 종속성을 주입받아 GameMessageService 인스턴스를 생성한다.
func NewGameMessageService(
	commandHandler *GameCommandHandler,
	playerRegistrar PlayerRegistrar,
	messageSender *MessageSender,
	msgProvider *messageprovider.Provider,
	publisher *ReplyPublisher,
	accessControl *qsecurity.AccessControl,
	commandParser *CommandParser,
	lockManager *qredis.LockManager,
	processingLockService *qredis.ProcessingLockService,
	queueProcessor *MessageQueueProcessor,
	restClient *llmrest.Client,
	commandPrefix string,
	logger *slog.Logger,
) *GameMessageService {
	return &GameMessageService{
		commandHandler:         commandHandler,
		playerRegistrar:        playerRegistrar,
		messageSender:          messageSender,
		msgProvider:            msgProvider,
		publisher:              publisher,
		accessControl:          accessControl,
		commandParser:          commandParser,
		lockManager:            lockManager,
		processingLockService:  processingLockService,
		queueProcessor:         queueProcessor,
		restClient:             restClient,
		commandPrefix:          strings.TrimSpace(commandPrefix),
		processingWaitingDelay: 5 * time.Second,
		logger:                 logger,
	}
}

// HandleMessage: Kafka/Streams 등으로부터 수신된 인바운드 메시지를 처리한다.
// 명령어를 파싱하고, 권한 및 세션을 확인한 뒤, 적절한 처리 과정(즉시 실행 또는 큐잉)으로 라우팅한다.
func (s *GameMessageService) HandleMessage(ctx context.Context, message mqmsg.InboundMessage) {
	cmd := s.commandParser.Parse(message.Content)
	if cmd == nil {
		s.logger.Debug("message_ignored", "content", message.Content)
		return
	}

	if !s.isAccessAllowed(ctx, message, *cmd) {
		return
	}

	// 세션이 필요한 명령어인데 세션이 없으면 바로 에러 반환 (대기 메시지 없이)
	if requiresExistingSession(*cmd) {
		hasSession, err := s.playerRegistrar.HasSession(ctx, message.ChatID)
		if err != nil {
			s.logger.Warn("session_check_failed", "chat_id", message.ChatID, "err", err)
		}
		if !hasSession {
			s.logger.Warn("message_rejected_no_session", "chat_id", message.ChatID, "user_id", message.UserID)
			noSessionText := s.msgProvider.Get(qmessages.ErrorNoSession, messageprovider.P("prefix", s.commandPrefix))
			_ = s.messageSender.SendFinal(ctx, message, noSessionText)
			return
		}
	}

	s.handleCommand(ctx, message, *cmd)
}

func (s *GameMessageService) handleCommand(ctx context.Context, message mqmsg.InboundMessage, command Command) {
	chatID := message.ChatID

	if s.shouldEnqueueImmediately(ctx, chatID) {
		s.enqueueMessage(ctx, message)
		s.processQueuedMessages(ctx, chatID)
		return
	}

	holderName := message.UserID
	if message.Sender != nil && strings.TrimSpace(*message.Sender) != "" {
		holderName = strings.TrimSpace(*message.Sender)
	}

	lockFn := s.lockManager.WithLock
	if !command.RequiresWriteLock() {
		lockFn = s.lockManager.WithReadLock
	}

	lockErr := lockFn(ctx, chatID, &holderName, func(ctx context.Context) error {
		if s.playerRegistrar != nil && command.Kind != CommandUnknown {
			s.playerRegistrar.RegisterPlayerAsync(ctx, message.ChatID, message.UserID, message.Sender)
		}

		if err := s.processingLockService.StartProcessing(ctx, chatID); err != nil {
			return fmt.Errorf("start processing failed: %w", err)
		}
		defer func() {
			_ = s.processingLockService.FinishProcessing(ctx, chatID)
		}()

		s.executeLockedCommand(ctx, message, command)
		return nil
	})
	if lockErr != nil {
		var lockFailure cerrors.LockError
		if errors.As(lockErr, &lockFailure) {
			s.logger.Warn("message_rejected_locked", "chat_id", message.ChatID, "user_id", message.UserID, "mode_write", command.RequiresWriteLock())
			s.enqueueMessage(ctx, message)
			// processQueuedMessages 호출 제거: Lock 보유자가 처리 완료 시 큐 처리됨
			// 즉시 호출 시 Lock 실패 → 재큐잉 → 알림 루프 발생
			return
		}

		s.logger.Error("lock_execute_failed", "chat_id", message.ChatID, "user_id", message.UserID, "err", lockErr)
		_ = s.messageSender.SendError(ctx, message, ErrorMapping{Key: qmessages.ErrorGeneric})
		s.processQueuedMessages(ctx, chatID)
		return
	}

	s.processQueuedMessages(ctx, chatID)
}

// enqueueMessage: 현재 요청을 처리할 수 없는 상황(락 획득 실패, 처리 중 등)일 때 Redis 대기열에 메시지를 추가한다.
func (s *GameMessageService) enqueueMessage(ctx context.Context, message mqmsg.InboundMessage) {
	s.logger.Debug("message_enqueued", "chat_id", message.ChatID, "user_id", message.UserID)
	_ = s.queueProcessor.EnqueueAndNotify(ctx, message.ChatID, message.UserID, message.Content, message.ThreadID, message.Sender, func(out mqmsg.OutboundMessage) error {
		return s.publisher.Publish(ctx, out)
	})
}

func (s *GameMessageService) processQueuedMessages(ctx context.Context, chatID string) {
	s.queueProcessor.ProcessQueuedMessages(ctx, chatID, func(out mqmsg.OutboundMessage) error {
		return s.publisher.Publish(ctx, out)
	})
}

func (s *GameMessageService) shouldEnqueueImmediately(ctx context.Context, chatID string) bool {
	hasPending, err := s.queueProcessor.HasPending(ctx, chatID)
	if err != nil {
		s.logger.Warn("queue_pending_check_failed", "chat_id", chatID, "err", err)
		hasPending = false
	}
	if hasPending {
		s.logger.Warn("message_rejected_pending", "chat_id", chatID)
		return true
	}

	if s.isProcessing(ctx, chatID) {
		s.logger.Debug("message_rejected_processing", "chat_id", chatID)
		return true
	}

	return false
}

func (s *GameMessageService) executeLockedCommand(ctx context.Context, message mqmsg.InboundMessage, command Command) {
	switch command.Kind {
	case CommandModelInfo:
		s.handleModelInfo(ctx, message)
	default:
		s.executeCommand(ctx, message, command)
	}
}

func (s *GameMessageService) executeCommand(ctx context.Context, message mqmsg.InboundMessage, command Command) {
	if command.Kind == CommandChainedQuestion && len(command.ChainQuestions) > 1 {
		queueNotice, _ := s.commandHandler.chainedQuestionHandler.PrepareChainQueue(
			ctx,
			message.ChatID,
			message.UserID,
			message.Sender,
			command.ChainQuestions,
		)
		if queueNotice != "" {
			_ = s.publisher.Publish(ctx, mqmsg.NewWaiting(message.ChatID, queueNotice, message.ThreadID))
		}
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(qconfig.AITimeoutSeconds)*time.Second)
	defer cancel()

	shouldSendWaiting := s.shouldSendWaiting(ctx, message.ChatID, command)
	waitingKey := command.WaitingMessageKey()

	if shouldSendWaiting && waitingKey != nil && *waitingKey == qmessages.ProcessingWaiting {
		delay := s.processingWaitingDelay
		if delay <= 0 {
			delay = 5 * time.Second
		}

		type commandResult struct {
			responses []string
			err       error
		}

		resultCh := make(chan commandResult, 1)
		go func() {
			responses, err := s.runCommand(timeoutCtx, message, command)
			resultCh <- commandResult{responses: responses, err: err}
		}()

		timer := time.NewTimer(delay)
		defer timer.Stop()

		var result commandResult
		select {
		case result = <-resultCh:
		case <-timer.C:
			_ = s.messageSender.SendWaiting(ctx, message, command)
			result = <-resultCh
		}

		if result.err != nil {
			s.handleDirectFailure(ctx, message, result.err)
			return
		}
		_ = s.sendFinalResponses(ctx, message, result.responses)
		return
	}

	if shouldSendWaiting {
		_ = s.messageSender.SendWaiting(ctx, message, command)
	}

	responses, err := s.runCommand(timeoutCtx, message, command)
	if err != nil {
		s.handleDirectFailure(ctx, message, err)
		return
	}

	_ = s.sendFinalResponses(ctx, message, responses)
}

func (s *GameMessageService) runCommand(ctx context.Context, message mqmsg.InboundMessage, command Command) ([]string, error) {
	return s.commandHandler.ProcessCommand(ctx, message, command)
}

func (s *GameMessageService) shouldSendWaiting(ctx context.Context, chatID string, command Command) bool {
	if command.WaitingMessageKey() == nil {
		return false
	}
	if command.Kind == CommandStart {
		if s.playerRegistrar == nil {
			return true
		}
		hasSession, err := s.playerRegistrar.HasSession(ctx, chatID)
		if err != nil {
			if s.logger != nil {
				s.logger.Warn("session_check_failed_waiting", "chat_id", chatID, "err", err)
			}
			return false
		}
		return !hasSession
	}
	if command.Kind != CommandHints {
		return true
	}

	checker, ok := s.playerRegistrar.(hintAvailabilityChecker)
	if !ok || checker == nil {
		return true
	}

	canGenerate, err := checker.CanGenerateHint(ctx, chatID)
	return err == nil && canGenerate
}

func (s *GameMessageService) sendFinalResponses(ctx context.Context, message mqmsg.InboundMessage, responses []string) error {
	sent := 0
	for _, response := range responses {
		if strings.TrimSpace(response) == "" {
			continue
		}
		if err := s.messageSender.SendFinal(ctx, message, response); err != nil {
			return err
		}
		sent++
	}
	if sent == 0 {
		return s.messageSender.SendFinal(ctx, message, "")
	}
	return nil
}

// HandleQueuedCommand: 대기열(Pending Queue)에서 꺼낸 명령어를 처리한다.
// 락을 이미 획득했거나 처리 가능한 상태라고 가정하고 실행한다.
func (s *GameMessageService) HandleQueuedCommand(
	ctx context.Context,
	message mqmsg.InboundMessage,
	command Command,
	emit func(mqmsg.OutboundMessage) error,
) error {
	if command.Kind == CommandChainedQuestion && len(command.ChainQuestions) > 1 {
		queueNotice, _ := s.commandHandler.chainedQuestionHandler.PrepareChainQueue(
			ctx,
			message.ChatID,
			message.UserID,
			message.Sender,
			command.ChainQuestions,
		)
		if queueNotice != "" {
			if err := emit(mqmsg.NewWaiting(message.ChatID, queueNotice, message.ThreadID)); err != nil {
				return err
			}
		}
	}

	switch command.Kind {
	case CommandModelInfo:
		return s.handleModelInfoQueued(ctx, message, emit)
	default:
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(qconfig.AITimeoutSeconds)*time.Second)
	defer cancel()

	responses, err := s.runCommand(timeoutCtx, message, command)
	if err != nil {
		return s.handleQueuedFailure(message, err, emit)
	}

	return emitChunkedResponses(message.ChatID, message.ThreadID, responses, emit)
}

func emitChunkedResponses(chatID string, threadID *string, responses []string, emit func(mqmsg.OutboundMessage) error) error {
	sent := 0
	for _, response := range responses {
		if strings.TrimSpace(response) == "" {
			continue
		}
		if err := emitChunkedText(chatID, threadID, response, emit); err != nil {
			return err
		}
		sent++
	}
	if sent == 0 {
		return emitChunkedText(chatID, threadID, "", emit)
	}
	return nil
}

// HandleQueuedChainBatch: 대기열에서 꺼낸 연쇄 질문(Chain Question) 배치 그룹을 처리한다.
func (s *GameMessageService) HandleQueuedChainBatch(
	ctx context.Context,
	chatID string,
	pending qmodel.PendingMessage,
	emit func(mqmsg.OutboundMessage) error,
) error {
	return s.commandHandler.chainedQuestionHandler.ProcessChainBatch(ctx, chatID, pending, emit)
}

func (s *GameMessageService) handleDirectFailure(ctx context.Context, message mqmsg.InboundMessage, err error) {
	var lockErr cerrors.LockError
	if errors.As(err, &lockErr) {
		_ = s.messageSender.SendLockError(ctx, message)
		return
	}

	mapping := GetErrorMapping(err, s.commandPrefix)
	_ = s.messageSender.SendError(ctx, message, mapping)
}

func (s *GameMessageService) handleQueuedFailure(
	message mqmsg.InboundMessage,
	err error,
	emit func(mqmsg.OutboundMessage) error,
) error {
	var lockErr cerrors.LockError
	if errors.As(err, &lockErr) {
		text := s.msgProvider.Get(qmessages.LockRequestInProgress)
		return emit(mqmsg.NewError(message.ChatID, text, message.ThreadID))
	}

	mapping := GetErrorMapping(err, s.commandPrefix)
	text := s.msgProvider.Get(mapping.Key, mapping.Params...)
	return emit(mqmsg.NewError(message.ChatID, text, message.ThreadID))
}

func (s *GameMessageService) handleModelInfo(ctx context.Context, message mqmsg.InboundMessage) {
	cfg, err := s.restClient.GetModelConfig(ctx)
	if err != nil {
		s.logger.Warn("model_info_fetch_failed", "err", err)
		_ = s.messageSender.SendFinal(ctx, message, s.msgProvider.Get(qmessages.ModelInfoFetchFailed))
		return
	}

	_ = s.messageSender.SendFinal(ctx, message, s.formatModelInfo(cfg))
}

func (s *GameMessageService) handleModelInfoQueued(ctx context.Context, message mqmsg.InboundMessage, emit func(mqmsg.OutboundMessage) error) error {
	cfg, err := s.restClient.GetModelConfig(ctx)
	if err != nil {
		s.logger.Warn("model_info_fetch_failed", "err", err)
		return emitChunkedText(message.ChatID, message.ThreadID, s.msgProvider.Get(qmessages.ModelInfoFetchFailed), emit)
	}

	return emitChunkedText(message.ChatID, message.ThreadID, s.formatModelInfo(cfg), emit)
}

func (s *GameMessageService) formatModelInfo(cfg *llmrest.ModelConfigResponse) string {
	hintsModel := cfg.ModelDefault
	if cfg.ModelHints != nil && strings.TrimSpace(*cfg.ModelHints) != "" {
		hintsModel = strings.TrimSpace(*cfg.ModelHints)
	}

	answerModel := cfg.ModelDefault
	if cfg.ModelAnswer != nil && strings.TrimSpace(*cfg.ModelAnswer) != "" {
		answerModel = strings.TrimSpace(*cfg.ModelAnswer)
	}

	verifyModel := cfg.ModelDefault
	if cfg.ModelVerify != nil && strings.TrimSpace(*cfg.ModelVerify) != "" {
		verifyModel = strings.TrimSpace(*cfg.ModelVerify)
	}

	transportMode := "h1"
	if cfg.TransportMode != nil && strings.TrimSpace(*cfg.TransportMode) != "" {
		transportMode = strings.TrimSpace(*cfg.TransportMode)
	} else if cfg.HTTP2Enabled {
		transportMode = "h2c"
	}

	temperature := fmt.Sprintf("%.2f", cfg.Temperature)

	lines := []string{
		s.msgProvider.Get(qmessages.ModelInfoHeader),
		s.msgProvider.Get(qmessages.ModelInfoDefault, messageprovider.P("model", cfg.ModelDefault)),
		s.msgProvider.Get(qmessages.ModelInfoHints, messageprovider.P("model", hintsModel)),
		s.msgProvider.Get(qmessages.ModelInfoAnswer, messageprovider.P("model", answerModel)),
		s.msgProvider.Get(qmessages.ModelInfoVerify, messageprovider.P("model", verifyModel)),
		s.msgProvider.Get(qmessages.ModelInfoTemperature, messageprovider.P("value", temperature)),
		s.msgProvider.Get(qmessages.ModelInfoMaxRetries, messageprovider.P("value", cfg.MaxRetries)),
		s.msgProvider.Get(qmessages.ModelInfoTimeout, messageprovider.P("value", cfg.TimeoutSeconds)),
		s.msgProvider.Get(qmessages.ModelInfoTransport, messageprovider.P("mode", transportMode)),
	}

	return strings.Join(lines, "\n")
}

func isAccessBypassAdminCommand(command Command) bool {
	switch command.Kind {
	case CommandAdminForceEnd, CommandAdminClearAll:
		return true
	default:
		return false
	}
}

func (s *GameMessageService) isAccessAllowed(ctx context.Context, message mqmsg.InboundMessage, command Command) bool {
	if isAccessBypassAdminCommand(command) {
		return true
	}

	reason := s.accessControl.GetDenialReason(message.UserID, message.ChatID)
	if reason == nil {
		return true
	}

	s.logger.Warn("access_denied", "user_id", message.UserID, "chat_id", message.ChatID, "reason", *reason)
	if *reason == qmessages.ErrorAccessDenied {
		return false
	}

	mapping := ErrorMapping{Key: *reason}
	if *reason == qmessages.ErrorUserBlocked {
		nickname := domainmodels.DisplayName(message.ChatID, message.UserID, message.Sender, s.msgProvider.Get(qmessages.UserAnonymous))
		mapping.Params = []messageprovider.Param{messageprovider.P("nickname", nickname)}
	}

	_ = s.messageSender.SendError(ctx, message, mapping)
	return false
}

func (s *GameMessageService) isProcessing(ctx context.Context, chatID string) bool {
	ok, err := s.processingLockService.IsProcessing(ctx, chatID)
	if err != nil {
		s.logger.Warn("processing_check_failed", "chat_id", chatID, "err", err)
		return false
	}
	return ok
}

// requiresExistingSession 세션이 필요한 명령어인지 확인.
// Start, Help, UserStats, Admin 명령어는 세션 없이도 실행 가능.
func requiresExistingSession(command Command) bool {
	switch command.Kind {
	case CommandStart, CommandHelp, CommandUserStats, CommandRoomStats,
		CommandAdminForceEnd, CommandAdminClearAll, CommandAdminUsage, CommandModelInfo:
		return false
	default:
		return true
	}
}
