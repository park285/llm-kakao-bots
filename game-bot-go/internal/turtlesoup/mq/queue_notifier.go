package mq

import (
	"log/slog"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	commonmq "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mq"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
	domainmodels "github.com/park285/llm-kakao-bots/game-bot-go/internal/domain/models"
	tserrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/errors"
	tsmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/messages"
)

// MessageQueueNotifier: turtlesoup 전용 큐 알림을 위한 공통 notifier alias 입니다.
type MessageQueueNotifier = commonmq.MessageQueueNotifier

// NewMessageQueueNotifier: 새로운 MessageQueueNotifier 인스턴스를 생성한다.
func NewMessageQueueNotifier(provider *messageprovider.Provider, logger *slog.Logger) *MessageQueueNotifier {
	cfg := commonmq.QueueNotifierConfig{
		UserAnonymousKey: tsmessages.UserAnonymous,
		DefaultErrorKey:  tsmessages.ErrorInternal,

		QueueProcessingKey:     tsmessages.QueueProcessing,
		QueueRetryKey:          tsmessages.QueueRetry,
		QueueRetryDuplicateKey: tsmessages.QueueRetryDuplicate,
		QueueRetryFailedKey:    tsmessages.QueueRetryFailed,

		ProcessingStartFactory: mqmsg.NewWaiting,
		RetryFactory:           mqmsg.NewWaiting,
		DuplicateFactory:       mqmsg.NewWaiting,
		FailedFactory:          mqmsg.NewError,
		ErrorFactory:           mqmsg.NewError,
	}
	mapper := func(err error) (string, []messageprovider.Param) {
		mapping := GetErrorMapping(err)
		return mapping.Key, mapping.Params
	}
	hook := func(logger *slog.Logger, chatID string, _ domainmodels.PendingMessage, err error) {
		if tserrors.IsExpectedUserBehavior(err) {
			logger.Warn("queue_domain_error", "chat_id", chatID, "err", err)
		}
	}
	return commonmq.NewMessageQueueNotifier(provider, logger, cfg, mapper, hook)
}
