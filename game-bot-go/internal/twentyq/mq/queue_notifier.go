package mq

import (
	"context"
	"log/slog"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
	qmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/messages"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

// MessageQueueNotifier: 처리 상태에 따라 사용자에게 피드백 메시지(대체로 Waiting 타입)를 전송하는 컴포넌트
type MessageQueueNotifier struct {
	provider      *messageprovider.Provider
	commandPrefix string
	logger        *slog.Logger
}

// NewMessageQueueNotifier: 새로운 MessageQueueNotifier 인스턴스를 생성한다.
func NewMessageQueueNotifier(provider *messageprovider.Provider, commandPrefix string, logger *slog.Logger) *MessageQueueNotifier {
	return &MessageQueueNotifier{
		provider:      provider,
		commandPrefix: commandPrefix,
		logger:        logger,
	}
}

// NotifyProcessingStart: 요청 처리가 시작되었음을 알린다. (주로 긴 작업이 예상될 때)
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

// NotifyRetry: 락 획득 실패 등으로 인해 처리가 지연되고 있음을 알린다.
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

// NotifyDuplicate: 중복된 요청이 큐에 있어 처리 순서를 조정하거나 대기 중임을 알린다.
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

// NotifyFailed: 요청 처리가 최종적으로 실패했음을 알린다. (대기열 가득 참 등)
func (n *MessageQueueNotifier) NotifyFailed(
	_ context.Context,
	chatID string,
	pending qmodel.PendingMessage,
	emit func(mqmsg.OutboundMessage) error,
) error {
	userName := pending.DisplayName(chatID, n.provider.Get(qmessages.UserAnonymous))
	failedText := n.provider.Get(qmessages.QueueRetryFailed, messageprovider.P("user", userName))
	return emit(mqmsg.NewFinal(chatID, failedText, pending.ThreadID))
}

// NotifyError: 처리 도중 발생한 에러를 사용자 친화적인 메시지로 변환하여 알린다.
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
	return emit(mqmsg.NewFinal(chatID, text, pending.ThreadID))
}
