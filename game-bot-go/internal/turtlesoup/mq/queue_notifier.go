package mq

import (
	"context"
	"log/slog"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
	tserrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/errors"
	tsmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/messages"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
)

// MessageQueueNotifier 는 타입이다.
type MessageQueueNotifier struct {
	provider *messageprovider.Provider
	logger   *slog.Logger
}

// NewMessageQueueNotifier 는 동작을 수행한다.
func NewMessageQueueNotifier(provider *messageprovider.Provider, logger *slog.Logger) *MessageQueueNotifier {
	return &MessageQueueNotifier{
		provider: provider,
		logger:   logger,
	}
}

// NotifyProcessingStart 는 동작을 수행한다.
func (n *MessageQueueNotifier) NotifyProcessingStart(
	_ context.Context,
	chatID string,
	pending tsmodel.PendingMessage,
	emit func(mqmsg.OutboundMessage) error,
) error {
	userName := pending.DisplayName(chatID, n.provider.Get(tsmessages.UserAnonymous))
	notifyText := n.provider.Get(tsmessages.QueueProcessing, messageprovider.P("user", userName))
	return emit(mqmsg.NewWaiting(chatID, notifyText, pending.ThreadID))
}

// NotifyRetry 는 동작을 수행한다.
func (n *MessageQueueNotifier) NotifyRetry(
	_ context.Context,
	chatID string,
	pending tsmodel.PendingMessage,
	emit func(mqmsg.OutboundMessage) error,
) error {
	userName := pending.DisplayName(chatID, n.provider.Get(tsmessages.UserAnonymous))
	retryText := n.provider.Get(tsmessages.QueueRetry, messageprovider.P("user", userName))
	return emit(mqmsg.NewWaiting(chatID, retryText, pending.ThreadID))
}

// NotifyDuplicate 는 동작을 수행한다.
func (n *MessageQueueNotifier) NotifyDuplicate(
	_ context.Context,
	chatID string,
	pending tsmodel.PendingMessage,
	emit func(mqmsg.OutboundMessage) error,
) error {
	userName := pending.DisplayName(chatID, n.provider.Get(tsmessages.UserAnonymous))
	dupText := n.provider.Get(tsmessages.QueueRetryDuplicate, messageprovider.P("user", userName))
	return emit(mqmsg.NewWaiting(chatID, dupText, pending.ThreadID))
}

// NotifyFailed 는 동작을 수행한다.
func (n *MessageQueueNotifier) NotifyFailed(
	_ context.Context,
	chatID string,
	pending tsmodel.PendingMessage,
	emit func(mqmsg.OutboundMessage) error,
) error {
	userName := pending.DisplayName(chatID, n.provider.Get(tsmessages.UserAnonymous))
	failedText := n.provider.Get(tsmessages.QueueRetryFailed, messageprovider.P("user", userName))
	return emit(mqmsg.NewError(chatID, failedText, pending.ThreadID))
}

// NotifyError 는 동작을 수행한다.
func (n *MessageQueueNotifier) NotifyError(
	_ context.Context,
	chatID string,
	pending tsmodel.PendingMessage,
	err error,
	emit func(mqmsg.OutboundMessage) error,
) error {
	n.logger.Error("queue_processing_error", "chat_id", chatID, "user_id", pending.UserID, "err", err)

	if tserrors.IsExpectedUserBehavior(err) {
		n.logger.Warn("queue_domain_error", "chat_id", chatID, "err", err)
	}

	mapping := GetErrorMapping(err)
	text := n.provider.Get(mapping.Key, mapping.Params...)
	return emit(mqmsg.NewError(chatID, text, pending.ThreadID))
}
