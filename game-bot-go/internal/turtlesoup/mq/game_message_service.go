package mq

import (
	"context"
	"errors"
	"log/slog"
	"time"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	tsmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/messages"
	tsredis "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/redis"
	tssecurity "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/security"
)

// GameMessageService 는 타입이다.
type GameMessageService struct {
	commandHandler        *GameCommandHandler
	messageSender         *MessageSender
	msgProvider           *messageprovider.Provider
	publisher             *ReplyPublisher
	accessControl         *tssecurity.AccessControl
	commandParser         *CommandParser
	processingLockService *tsredis.ProcessingLockService
	queueProcessor        *MessageQueueProcessor
	restClient            *llmrest.Client
	logger                *slog.Logger
}

// NewGameMessageService 는 동작을 수행한다.
func NewGameMessageService(
	commandHandler *GameCommandHandler,
	messageSender *MessageSender,
	msgProvider *messageprovider.Provider,
	publisher *ReplyPublisher,
	accessControl *tssecurity.AccessControl,
	commandParser *CommandParser,
	processingLockService *tsredis.ProcessingLockService,
	queueProcessor *MessageQueueProcessor,
	restClient *llmrest.Client,
	logger *slog.Logger,
) *GameMessageService {
	return &GameMessageService{
		commandHandler:        commandHandler,
		messageSender:         messageSender,
		msgProvider:           msgProvider,
		publisher:             publisher,
		accessControl:         accessControl,
		commandParser:         commandParser,
		processingLockService: processingLockService,
		queueProcessor:        queueProcessor,
		restClient:            restClient,
		logger:                logger,
	}
}

// HandleMessage 는 동작을 수행한다.
func (s *GameMessageService) HandleMessage(ctx context.Context, message mqmsg.InboundMessage) {
	if !s.isAccessAllowed(message) {
		return
	}

	cmd := s.commandParser.Parse(message.Content)
	if cmd == nil {
		s.logger.Debug("message_ignored", "content", message.Content)
		return
	}

	s.dispatchCommand(ctx, message, *cmd)
}

func (s *GameMessageService) dispatchCommand(ctx context.Context, message mqmsg.InboundMessage, command Command) {
	chatID := message.ChatID

	switch {
	case !command.RequiresLock():
		s.handleSimpleCommand(ctx, message, command)
	case s.isProcessing(ctx, chatID):
		s.enqueueMessage(ctx, message)
	default:
		s.executeWithQueue(ctx, message, command)
	}
}

func (s *GameMessageService) enqueueMessage(ctx context.Context, message mqmsg.InboundMessage) {
	s.logger.Debug("message_enqueued", "chat_id", message.ChatID, "user_id", message.UserID)
	_ = s.queueProcessor.EnqueueAndNotify(ctx, message.ChatID, message.UserID, message.Content, message.ThreadID, message.Sender, func(out mqmsg.OutboundMessage) error {
		return s.publisher.Publish(ctx, out)
	})
}

func (s *GameMessageService) executeWithQueue(ctx context.Context, message mqmsg.InboundMessage, command Command) {
	chatID := message.ChatID

	if err := s.processingLockService.StartProcessing(ctx, chatID); err != nil {
		var lockErr cerrors.LockError
		if errors.As(err, &lockErr) {
			s.enqueueMessage(ctx, message)
			return
		}
		s.handleDirectFailure(ctx, message, err)
		return
	}
	s.executeCommand(ctx, message, command)
	_ = s.processingLockService.FinishProcessing(ctx, chatID)

	s.queueProcessor.ProcessQueuedMessages(ctx, chatID, func(out mqmsg.OutboundMessage) error {
		return s.publisher.Publish(ctx, out)
	})
}

func (s *GameMessageService) executeCommand(ctx context.Context, message mqmsg.InboundMessage, command Command) {
	_ = s.messageSender.SendWaiting(ctx, message, command)

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(tsconfig.AITimeoutSeconds)*time.Second)
	defer cancel()

	response, err := s.runCommand(timeoutCtx, message, command)
	if err != nil {
		s.handleDirectFailure(ctx, message, err)
		return
	}

	_ = s.messageSender.SendFinal(ctx, message, response)
}

func (s *GameMessageService) runCommand(ctx context.Context, message mqmsg.InboundMessage, command Command) (string, error) {
	return s.commandHandler.ProcessCommand(ctx, message, command)
}

// HandleQueuedCommand 는 동작을 수행한다.
func (s *GameMessageService) HandleQueuedCommand(
	ctx context.Context,
	message mqmsg.InboundMessage,
	command Command,
	emit func(mqmsg.OutboundMessage) error,
) error {
	if key := command.WaitingMessageKey(); key != nil {
		if err := emit(mqmsg.NewWaiting(message.ChatID, s.msgProvider.Get(*key), message.ThreadID)); err != nil {
			return err
		}
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(tsconfig.AITimeoutSeconds)*time.Second)
	defer cancel()

	response, err := s.runCommand(timeoutCtx, message, command)
	if err != nil {
		return s.handleQueuedFailure(message, err, emit)
	}

	return emit(mqmsg.NewFinal(message.ChatID, response, message.ThreadID))
}

func (s *GameMessageService) handleDirectFailure(ctx context.Context, message mqmsg.InboundMessage, err error) {
	if errors.Is(err, context.DeadlineExceeded) {
		_ = s.messageSender.SendError(ctx, message, ErrorMapping{Key: tsmessages.ErrorAICallTimeout})
		return
	}

	var lockErr cerrors.LockError
	if errors.As(err, &lockErr) {
		_ = s.messageSender.SendLockError(ctx, message, lockErr.HolderName)
		return
	}

	mapping := GetErrorMapping(err)
	_ = s.messageSender.SendError(ctx, message, mapping)
}

func (s *GameMessageService) handleQueuedFailure(
	message mqmsg.InboundMessage,
	err error,
	emit func(mqmsg.OutboundMessage) error,
) error {
	if errors.Is(err, context.DeadlineExceeded) {
		text := s.msgProvider.Get(tsmessages.ErrorAICallTimeout)
		return emit(mqmsg.NewError(message.ChatID, text, message.ThreadID))
	}

	mapping := GetErrorMapping(err)
	text := s.msgProvider.Get(mapping.Key, mapping.Params...)
	return emit(mqmsg.NewError(message.ChatID, text, message.ThreadID))
}

func (s *GameMessageService) handleSimpleCommand(ctx context.Context, message mqmsg.InboundMessage, command Command) {
	switch command.Kind {
	case CommandHelp:
		_ = s.messageSender.SendFinal(ctx, message, s.msgProvider.Get(tsmessages.HelpMessage))
	case CommandUnknown:
		_ = s.messageSender.SendFinal(ctx, message, s.msgProvider.Get(tsmessages.ErrorUnknownCommand))
	default:
	}
}

func (s *GameMessageService) isAccessAllowed(message mqmsg.InboundMessage) bool {
	reason := s.accessControl.GetDenialReason(message.UserID, message.ChatID)
	if reason == nil {
		return true
	}
	s.logger.Debug("access_denied", "user_id", message.UserID, "chat_id", message.ChatID, "reason", *reason)
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
