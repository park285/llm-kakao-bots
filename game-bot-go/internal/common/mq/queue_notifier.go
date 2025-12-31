package mq

import (
	"context"
	"log/slog"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
	domainmodels "github.com/park285/llm-kakao-bots/game-bot-go/internal/domain/models"
)

type outboundFactory func(chatID string, text string, threadID *string) mqmsg.OutboundMessage

// QueueNotifierConfig: 게임별 메시지 키 및 OutboundMessage 타입 차이를 주입받기 위한 설정입니다.
type QueueNotifierConfig struct {
	UserAnonymousKey string
	DefaultErrorKey  string

	QueueProcessingKey     string
	QueueRetryKey          string
	QueueRetryDuplicateKey string
	QueueRetryFailedKey    string

	ProcessingStartFactory outboundFactory
	RetryFactory           outboundFactory
	DuplicateFactory       outboundFactory
	FailedFactory          outboundFactory
	ErrorFactory           outboundFactory
}

type errorMapper func(err error) (key string, params []messageprovider.Param)

type errorLogHook func(logger *slog.Logger, chatID string, pending domainmodels.PendingMessage, err error)

// MessageQueueNotifier: 큐 처리 상태(대기/재시도/실패/에러)를 사용자에게 알리는 공통 컴포넌트입니다.
// 게임별로 달라지는 메시지 키, 에러 매핑, OutboundMessage 타입을 설정으로 주입합니다.
type MessageQueueNotifier struct {
	provider *messageprovider.Provider
	logger   *slog.Logger
	cfg      QueueNotifierConfig

	mapError errorMapper
	logHook  errorLogHook
}

// NewMessageQueueNotifier: 공통 MessageQueueNotifier 인스턴스를 생성합니다.
func NewMessageQueueNotifier(
	provider *messageprovider.Provider,
	logger *slog.Logger,
	cfg QueueNotifierConfig,
	mapError func(err error) (key string, params []messageprovider.Param),
	logHook func(logger *slog.Logger, chatID string, pending domainmodels.PendingMessage, err error),
) *MessageQueueNotifier {
	return &MessageQueueNotifier{
		provider: provider,
		logger:   logger,
		cfg:      cfg,
		mapError: mapError,
		logHook:  logHook,
	}
}

// NotifyProcessingStart: 요청 처리가 시작되었음을 알립니다.
func (n *MessageQueueNotifier) NotifyProcessingStart(
	_ context.Context,
	chatID string,
	pending domainmodels.PendingMessage,
	emit func(mqmsg.OutboundMessage) error,
) error {
	userName := pending.DisplayName(chatID, n.provider.Get(n.cfg.UserAnonymousKey))
	text := n.provider.Get(n.cfg.QueueProcessingKey, messageprovider.P("user", userName))
	return emit(n.cfg.ProcessingStartFactory(chatID, text, pending.ThreadID))
}

// NotifyRetry: 처리 지연(락 실패 등)으로 재시도 중임을 알립니다.
func (n *MessageQueueNotifier) NotifyRetry(
	_ context.Context,
	chatID string,
	pending domainmodels.PendingMessage,
	emit func(mqmsg.OutboundMessage) error,
) error {
	userName := pending.DisplayName(chatID, n.provider.Get(n.cfg.UserAnonymousKey))
	text := n.provider.Get(n.cfg.QueueRetryKey, messageprovider.P("user", userName))
	return emit(n.cfg.RetryFactory(chatID, text, pending.ThreadID))
}

// NotifyDuplicate: 중복된 요청이 큐에 있어 처리 순서를 조정하거나 대기 중임을 알립니다.
func (n *MessageQueueNotifier) NotifyDuplicate(
	_ context.Context,
	chatID string,
	pending domainmodels.PendingMessage,
	emit func(mqmsg.OutboundMessage) error,
) error {
	userName := pending.DisplayName(chatID, n.provider.Get(n.cfg.UserAnonymousKey))
	text := n.provider.Get(n.cfg.QueueRetryDuplicateKey, messageprovider.P("user", userName))
	return emit(n.cfg.DuplicateFactory(chatID, text, pending.ThreadID))
}

// NotifyFailed: 요청 처리가 최종적으로 실패했음을 알립니다. (대기열 가득 참 등)
func (n *MessageQueueNotifier) NotifyFailed(
	_ context.Context,
	chatID string,
	pending domainmodels.PendingMessage,
	emit func(mqmsg.OutboundMessage) error,
) error {
	userName := pending.DisplayName(chatID, n.provider.Get(n.cfg.UserAnonymousKey))
	text := n.provider.Get(n.cfg.QueueRetryFailedKey, messageprovider.P("user", userName))
	return emit(n.cfg.FailedFactory(chatID, text, pending.ThreadID))
}

// NotifyError: 처리 도중 발생한 에러를 사용자 메시지로 변환하여 알립니다.
func (n *MessageQueueNotifier) NotifyError(
	_ context.Context,
	chatID string,
	pending domainmodels.PendingMessage,
	err error,
	emit func(mqmsg.OutboundMessage) error,
) error {
	if n.logger != nil {
		n.logger.Error("queue_processing_error", "chat_id", chatID, "user_id", pending.UserID, "err", err)
		if n.logHook != nil {
			n.logHook(n.logger, chatID, pending, err)
		}
	}

	if n.mapError == nil {
		text := ""
		if n.provider != nil && n.cfg.DefaultErrorKey != "" {
			text = n.provider.Get(n.cfg.DefaultErrorKey)
		}
		return emit(n.cfg.ErrorFactory(chatID, text, pending.ThreadID))
	}

	key, params := n.mapError(err)
	text := n.provider.Get(key, params...)
	return emit(n.cfg.ErrorFactory(chatID, text, pending.ThreadID))
}
