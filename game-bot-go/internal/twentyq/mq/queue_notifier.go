package mq

import (
	"context"
	"log/slog"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
	qmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/messages"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

// MessageQueueNotifier 는 타입이다.
type MessageQueueNotifier struct {
	provider      *messageprovider.Provider
	commandPrefix string
	logger        *slog.Logger
}

// NewMessageQueueNotifier 는 동작을 수행한다.
func NewMessageQueueNotifier(provider *messageprovider.Provider, commandPrefix string, logger *slog.Logger) *MessageQueueNotifier {
	return &MessageQueueNotifier{
		provider:      provider,
		commandPrefix: commandPrefix,
		logger:        logger,
	}
}

// NotifyProcessingStart 는 동작을 수행한다.
func (n *MessageQueueNotifier) NotifyProcessingStart(
	_ context.Context,
	chatID string,
	pending qmodel.PendingMessage,
	emit func(mqmsg.OutboundMessage) error,
) error {
	userName := pending.DisplayName(chatID, n.provider.Get(qmessages.UserAnonymous))
	notifyText := n.provider.Get(qmessages.QueueProcessing, messageprovider.P("user", userName))
	return emit(mqmsg.NewWaiting(chatID, notifyText, pending.ThreadID))
}

// NotifyRetry 는 동작을 수행한다.
func (n *MessageQueueNotifier) NotifyRetry(
	_ context.Context,
	chatID string,
	pending qmodel.PendingMessage,
	emit func(mqmsg.OutboundMessage) error,
) error {
	userName := pending.DisplayName(chatID, n.provider.Get(qmessages.UserAnonymous))
	retryText := n.provider.Get(qmessages.QueueRetry, messageprovider.P("user", userName))
	return emit(mqmsg.NewWaiting(chatID, retryText, pending.ThreadID))
}

// NotifyDuplicate 는 동작을 수행한다.
func (n *MessageQueueNotifier) NotifyDuplicate(
	_ context.Context,
	chatID string,
	pending qmodel.PendingMessage,
	emit func(mqmsg.OutboundMessage) error,
) error {
	userName := pending.DisplayName(chatID, n.provider.Get(qmessages.UserAnonymous))
	dupText := n.provider.Get(qmessages.QueueRetryDuplicate, messageprovider.P("user", userName))
	return emit(mqmsg.NewWaiting(chatID, dupText, pending.ThreadID))
}

// NotifyFailed 는 동작을 수행한다.
func (n *MessageQueueNotifier) NotifyFailed(
	_ context.Context,
	chatID string,
	pending qmodel.PendingMessage,
	emit func(mqmsg.OutboundMessage) error,
) error {
	userName := pending.DisplayName(chatID, n.provider.Get(qmessages.UserAnonymous))
	failedText := n.provider.Get(qmessages.QueueRetryFailed, messageprovider.P("user", userName))
	return emit(mqmsg.NewError(chatID, failedText, pending.ThreadID))
}

// NotifyError 는 동작을 수행한다.
func (n *MessageQueueNotifier) NotifyError(
	_ context.Context,
	chatID string,
	pending qmodel.PendingMessage,
	err error,
	emit func(mqmsg.OutboundMessage) error,
) error {
	n.logger.Error("queue_processing_error", "chat_id", chatID, "user_id", pending.UserID, "err", err)

	mapping := GetErrorMapping(err, n.commandPrefix)
	text := n.provider.Get(mapping.Key, mapping.Params...)
	return emit(mqmsg.NewError(chatID, text, pending.ThreadID))
}
